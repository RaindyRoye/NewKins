package engine

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	"github.com/gokins/gokins/util"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/sirupsen/logrus"
)

// TimerEngine manages scheduled trigger execution. It uses a ticker-based
// run loop (instead of busy-wait polling) and executes trigger callbacks
// outside the lock to avoid contention.
type TimerEngine struct {
	tasklk sync.RWMutex
	tasks  map[string]*timerExec
	// wakeCh signals the run loop to check timers immediately (e.g. after Refresh).
	// Buffered with capacity 1 so senders never block.
	wakeCh chan struct{}
	// stopCh is closed to signal the run loop to exit.
	stopCh chan struct{}
	// stopOnce ensures Stop is idempotent.
	stopOnce sync.Once
}
type timerExec struct {
	tt   *model.TTrigger
	typ  int64
	tms  time.Time
	tick time.Time
}

func StartTimerEngine() *TimerEngine {
	c := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	go func() {
		defer util.RecoverLog("TimerEngine.main")
		c.refresh()
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		for {
			select {
			case <-c.stopCh:
				return
			case <-comm.Ctx.Done():
				return
			case <-c.wakeCh:
				// Drain any stale signal
				select {
				case <-c.wakeCh:
				default:
				}
			case <-ticker.C:
			}
			c.run()
		}
	}()
	return c
}

// Stop signals the timer run loop to exit. Safe to call multiple times.
func (c *TimerEngine) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
}

// Wake signals the run loop to check timers immediately.
// Non-blocking; drops the signal if one is already pending.
func (c *TimerEngine) Wake() {
	select {
	case c.wakeCh <- struct{}{}:
	default:
	}
}

func (c *TimerEngine) run() {
	defer util.RecoverLog("TimerEngine.run")

	// Phase 1: collect due timers under read lock.
	type dueItem struct {
		tt  *model.TTrigger
		typ int64
		id  string
	}
	var due []dueItem
	func() {
		c.tasklk.RLock()
		defer c.tasklk.RUnlock()
		now := time.Now()
		for _, v := range c.tasks {
			if now.After(v.tick) {
				due = append(due, dueItem{tt: v.tt, typ: v.typ, id: v.tt.Id})
			}
		}
	}()

	// Phase 2: process due timers outside the lock to avoid contention.
	for _, item := range due {
		c.execItem(item.tt, item.typ, item.id)
	}
}

func (c *TimerEngine) execItem(tt *model.TTrigger, typ int64, id string) {
	defer util.RecoverLog("TimerEngine.execItem")

	now := time.Now()
	logrus.Debugf("Timer(%s[%d]:%s) tick on:%s", tt.Name, typ, now.Format(common.TimeFmt), tt.Id)

	// Update next tick time or schedule deletion under write lock.
	switch typ {
	case 0:
		// One-shot timer: delete after firing
		go func() {
			defer util.RecoverLog("TimerEngine.delete")
			c.Delete(id)
		}()
	case 1:
		c.setTick(id, now.Add(time.Minute))
	case 2:
		c.setTick(id, now.Add(time.Hour))
	case 3:
		c.setTick(id, now.Add(time.Hour*24))
	case 4:
		c.setTick(id, now.Add(time.Hour*24*7))
	case 5:
		c.setTick(id, now.Add(time.Hour*24*30))
	}

	// Trigger execution outside the lock
	rb, err := service.TriggerTimer(comm.Ctx, tt)
	if err != nil {
		logrus.Errorf("TriggerTimer err:%v", err)
	} else {
		Mgr.BuildEgn().Put(rb)
	}
}

// setTick updates the tick time for a timer under write lock.
func (c *TimerEngine) setTick(id string, nextTick time.Time) {
	c.tasklk.Lock()
	defer c.tasklk.Unlock()
	if t, ok := c.tasks[id]; ok {
		t.tick = nextTick
	}
}

func (c *TimerEngine) refresh() {
	defer util.RecoverLog("TimerEngine.refresh")
	var ls []*model.TTrigger
	if err := comm.Db.Context(comm.Ctx).Where("enabled = 1 AND types = 'timer'").Find(&ls); err != nil {
		logrus.Errorf("TimerEngine refresh find err: %v", err)
		return
	}

	for _, v := range ls {
		err := c.resetOne(v)
		if err != nil {
			logrus.Errorf("TimerEngine resetOne err:%v", err)
		}
	}
}
func (c *TimerEngine) resetOne(tmr *model.TTrigger) (rterr error) {
	defer util.RecoverResult(&rterr, "TimerEngine.resetOne")
	if tmr.Types != "timer" {
		return fmt.Errorf("%w: expected 'timer', got '%s'", ErrInvalidTriggerType, tmr.Types)
	}
	mp := hbtp.Map{}
	err := json.Unmarshal([]byte(tmr.Params), &mp)
	if err != nil {
		return fmt.Errorf("unmarshal trigger params: %w", err)
	}
	typ, err := mp.GetInt("timerType")
	if err != nil {
		return fmt.Errorf("get timerType: %w", err)
	}
	dates := mp.GetString("dates")
	tms, err := time.ParseInLocation(time.RFC3339Nano, dates, time.Local)
	if err != nil {
		return fmt.Errorf("parse dates %q: %w", dates, err)
	}
	c.tasklk.Lock()
	defer c.tasklk.Unlock()
	switch typ {
	case 0:
		if time.Since(tms) < 0 {
			t, ok := c.tasks[tmr.Id]
			if !ok {
				t = &timerExec{
					tt: tmr,
				}
				c.tasks[tmr.Id] = t
			}
			t.typ = typ
			t.tms = tms
			t.tick = tms
			logrus.Debugf("Timer add(%s[%d]:%s) tick on:%s", tmr.Name, typ, tms.Format(common.TimeFmt), t.tick.Format(common.TimeFmt))
		}
	case 1, 2, 3, 4, 5:
		now := time.Now()
		t, ok := c.tasks[tmr.Id]
		if !ok {
			t = &timerExec{
				tt: tmr,
			}
			c.tasks[tmr.Id] = t
		}
		t.typ = typ
		t.tms = tms
		switch typ {
		case 1:
			t.tick = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), tms.Second(), 0, time.Local)
			if time.Since(t.tick) > 0 {
				t.tick = t.tick.Add(time.Minute)
			}
		case 2:
			t.tick = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), tms.Minute(), tms.Second(), 0, time.Local)
			if time.Since(t.tick) > 0 {
				t.tick = t.tick.Add(time.Hour)
			}
		case 3:
			t.tick = time.Date(now.Year(), now.Month(), now.Day(), tms.Hour(), tms.Minute(), tms.Second(), 0, time.Local)
			if time.Since(t.tick) > 0 {
				t.tick = t.tick.Add(time.Hour * 24)
			}
		case 4:
			t.tick = time.Date(now.Year(), now.Month(), tms.Day(), tms.Hour(), tms.Minute(), tms.Second(), 0, time.Local)
			if time.Since(t.tick) > 0 {
				t.tick = t.tick.Add(time.Hour * 24 * 7)
			}
		case 5:
			t.tick = time.Date(now.Year(), now.Month(), tms.Day(), tms.Hour(), tms.Minute(), tms.Second(), 0, time.Local)
			if time.Since(t.tick) > 0 {
				t.tick = t.tick.Add(time.Hour * 24 * 30)
			}
		}
		logrus.Debugf("Timer add(%s[%d]:%s) tick on:%s", tmr.Name, typ, tms.Format(common.TimeFmt), t.tick.Format(common.TimeFmt))
	}
	return nil
}
func (c *TimerEngine) Refresh(tmrid string) error {
	if tmrid == "" {
		return fmt.Errorf("timer id is empty: %w", ErrEmptyParams)
	}
	tmr := &model.TTrigger{}
	ok, err := comm.Db.Context(comm.Ctx).Where("id=?", tmrid).Get(tmr)
	if err != nil {
		return fmt.Errorf("query trigger %s: %w", tmrid, err)
	}
	if !ok || tmr.Enabled != 1 {
		c.Delete(tmrid)
		return fmt.Errorf("trigger %s not found or disabled: %w", tmrid, ErrBuildNotFound)
	}
	return c.resetOne(tmr)
}
func (c *TimerEngine) Delete(tmrid string) {
	c.tasklk.Lock()
	delete(c.tasks, tmrid)
	c.tasklk.Unlock()
}
