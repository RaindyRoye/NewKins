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

// timerRefreshInterval is the fallback polling interval used by the timer
// engine when no tasks are scheduled. It also bounds the maximum latency
// before a newly added timer is picked up by the run loop.
const timerRefreshInterval = 30 * time.Second

type TimerEngine struct {
	tasklk sync.RWMutex
	tasks  map[string]*timerExec
	// wakeCh is a buffered channel used to notify the run loop that a new
	// timer has been added or an existing one refreshed. Buffered with
	// capacity 1 so that Wake never blocks even if the run loop is busy.
	wakeCh chan struct{}
	// stopCh is closed by Stop to signal the run loop to exit immediately.
	stopCh chan struct{}
	// stopOnce ensures Stop is idempotent and safe to call multiple times.
	stopOnce sync.Once
}
type timerExec struct {
	tt   *model.TTrigger
	typ  int64
	tms  time.Time
	tick time.Time
}

// StartTimerEngine creates a new TimerEngine and launches its background run
// loop. The loop uses a dynamic ticker that adjusts to the nearest scheduled
// task, eliminating the previous busy-wait pattern that polled every 10ms.
func StartTimerEngine() *TimerEngine {
	c := &TimerEngine{
		tasks:  make(map[string]*timerExec),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	go func() {
		defer util.RecoverLog("TimerEngine.main")
		c.refresh()
		c.eventLoop()
	}()
	return c
}

// eventLoop is the main run loop. It rebuilds a ticker after every wake-up so
// the interval adapts to the nearest pending task. When no tasks are scheduled
// it falls back to timerRefreshInterval to keep the loop alive for refreshes.
func (c *TimerEngine) eventLoop() {
	for comm.Ctx.Err() == nil {
		interval := c.nextInterval()
		ticker := time.NewTicker(interval)
		select {
		case <-c.stopCh:
			ticker.Stop()
			return
		case <-comm.Ctx.Done():
			ticker.Stop()
			return
		case <-c.wakeCh:
			// Drain any pending notification to avoid stale signals
			select {
			case <-c.wakeCh:
			default:
			}
			ticker.Stop()
			c.run()
		case <-ticker.C:
			ticker.Stop()
			c.run()
		}
	}
}

// nextInterval returns the duration until the nearest pending task tick, or
// timerRefreshInterval when there are no scheduled tasks. The returned value
// is always clamped to [1ms, timerRefreshInterval] to prevent zero-duration
// ticks or excessive latency.
func (c *TimerEngine) nextInterval() time.Duration {
	c.tasklk.RLock()
	defer c.tasklk.RUnlock()
	if len(c.tasks) == 0 {
		return timerRefreshInterval
	}
	nearest := time.Time{}
	now := time.Now()
	for _, v := range c.tasks {
		if v.tick.IsZero() {
			continue
		}
		if nearest.IsZero() || v.tick.Before(nearest) {
			nearest = v.tick
		}
	}
	// If no valid ticks found, return fallback interval
	if nearest.IsZero() {
		return timerRefreshInterval
	}
	// If tick is in the past, use short interval to process it promptly
	if nearest.Before(now) {
		return time.Millisecond
	}
	d := time.Until(nearest)
	if d < time.Millisecond {
		return time.Millisecond
	}
	if d > timerRefreshInterval {
		return timerRefreshInterval
	}
	return d
}

// Stop signals the run loop to exit. Safe to call multiple times.
func (c *TimerEngine) Stop() {
	if c.stopCh != nil {
		c.stopOnce.Do(func() {
			close(c.stopCh)
		})
	}
}

// Wake notifies the run loop that a timer was added or refreshed and should
// be re-evaluated. It is non-blocking and safe to call from any goroutine.
func (c *TimerEngine) Wake() {
	if c.wakeCh != nil {
		select {
		case c.wakeCh <- struct{}{}:
		default:
		}
	}
}

// run iterates over all scheduled tasks and fires those whose tick is due.
// It snapshots the task list under a read-lock, releases the lock, and then
// executes each timer outside the lock to avoid blocking concurrent calls to
// Refresh / Delete / resetOne while slow operations (service.TriggerTimer,
// Mgr.BuildEgn().Put) are in flight.
func (c *TimerEngine) run() {
	defer util.RecoverLog("TimerEngine.run")

	c.tasklk.RLock()
	snap := make([]*timerExec, 0, len(c.tasks))
	for _, v := range c.tasks {
		snap = append(snap, v)
	}
	c.tasklk.RUnlock()

	for _, v := range snap {
		if comm.Ctx.Err() != nil {
			return
		}
		c.execItem(v)
	}
}

func (c *TimerEngine) execItem(v *timerExec) {
	defer util.RecoverLog("TimerEngine.execItem")
	if time.Since(v.tick) <= 0 {
		return
	}
	now := time.Now()
	logrus.Debugf("Timer(%s[%d]:%s) tick on:%s", v.tt.Name, v.typ, now.Format(common.TimeFmt), v.tick.Format(common.TimeFmt))
	switch v.typ {
	case 0:
		// One-shot: fire once then remove. Use a goroutine for the delete
		// so it can acquire the task lock without risking deadlock here.
		go func(id string) {
			defer util.RecoverLog("TimerEngine.delete")
			c.Delete(id)
		}(v.tt.Id)
	case 1:
		v.tick = now.Add(time.Minute)
	case 2:
		v.tick = now.Add(time.Hour)
	case 3:
		v.tick = now.Add(time.Hour * 24)
	case 4:
		v.tick = now.Add(time.Hour * 24 * 7)
	case 5:
		v.tick = now.Add(time.Hour * 24 * 30)
	}

	// Trigger outside of any lock to avoid blocking concurrent operations.
	rb, err := service.TriggerTimer(comm.Ctx, v.tt)
	if err != nil {
		logrus.Errorf("TriggerTimer err:%v", err)
		return
	}
	if Mgr.BuildEgn() == nil {
		logrus.Warnf("Timer(%s) fired but build engine is unavailable", v.tt.Name)
		return
	}
	Mgr.BuildEgn().Put(rb)
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
	if err := c.resetOne(tmr); err != nil {
		return err
	}
	// Notify the run loop so the new/updated timer is picked up immediately
	// rather than waiting for the next fallback tick.
	c.Wake()
	return nil
}
func (c *TimerEngine) Delete(tmrid string) {
	c.tasklk.Lock()
	delete(c.tasks, tmrid)
	c.tasklk.Unlock()
}
