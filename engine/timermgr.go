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

// timerTickInterval is the polling interval for the timer engine main loop.
// 1 second provides sub-second latency for cron-like triggers while keeping
// CPU usage negligible during idle periods.
const timerTickInterval = time.Second

type TimerEngine struct {
	tasklk sync.RWMutex
	tasks  map[string]*timerExec
}
type timerExec struct {
	tt   *model.TTrigger
	typ  int64
	tms  time.Time
	tick time.Time
}

func StartTimerEngine() *TimerEngine {
	c := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	go func() {
		defer util.RecoverLog("TimerEngine.main")
		c.refresh()
		for !hbtp.EndContext(comm.Ctx) {
			c.run()
			select {
			case <-comm.Ctx.Done():
				return
			case <-time.After(timerTickInterval):
			}
		}
	}()
	return c
}

// run collects all due timer items under a read-lock snapshot, then executes
// them without holding the lock. This prevents lock contention with concurrent
// Delete/Refresh/resetOne calls and allows service.TriggerTimer to run freely.
func (c *TimerEngine) run() {
	defer util.RecoverLog("TimerEngine.run")

	now := time.Now()
	var due []*timerExec

	c.tasklk.RLock()
	for _, v := range c.tasks {
		if now.After(v.tick) || now.Equal(v.tick) {
			due = append(due, v)
		}
	}
	c.tasklk.RUnlock()

	for _, v := range due {
		c.execItem(v)
	}
}

// execItem fires a single due timer, advances its tick time (for recurring
// timers), or removes it (for one-shot timers). All mutations are performed
// under the write lock; the actual service call runs outside the lock.
func (c *TimerEngine) execItem(v *timerExec) {
	defer util.RecoverLog("TimerEngine.execItem")
	now := time.Now()
	if now.Before(v.tick) {
		return
	}
	logrus.Debugf("Timer(%s[%d]:%s) tick on:%s", v.tt.Name, v.typ, now.Format(common.TimeFmt), v.tick.Format(common.TimeFmt))

	c.tasklk.Lock()
	switch v.typ {
	case 0: // one-shot: remove from map
		delete(c.tasks, v.tt.Id)
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
	// snapshot the trigger so we can call TriggerTimer without holding the lock
	tt := v.tt
	c.tasklk.Unlock()

	rb, err := service.TriggerTimer(comm.Ctx, tt)
	if err != nil {
		logrus.Errorf("TriggerTimer err:%v", err)
	} else {
		Mgr.BuildEgn().Put(rb)
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

// TaskCount returns the number of active timers. Intended for testing and diagnostics.
func (c *TimerEngine) TaskCount() int {
	c.tasklk.RLock()
	defer c.tasklk.RUnlock()
	return len(c.tasks)
}

// TaskIDs returns the IDs of all active timers. Intended for testing and diagnostics.
func (c *TimerEngine) TaskIDs() []string {
	c.tasklk.RLock()
	defer c.tasklk.RUnlock()
	ids := make([]string, 0, len(c.tasks))
	for id := range c.tasks {
		ids = append(ids, id)
	}
	return ids
}
