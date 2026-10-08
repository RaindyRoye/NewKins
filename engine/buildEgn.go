package engine

import (
	"container/list"
	"sync"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/util"
	"github.com/sirupsen/logrus"
)

type BuildEngine struct {
	tskwlk sync.RWMutex
	taskw  *list.List

	tskslk sync.RWMutex
	tasks  map[string]*BuildTask

	// wakeCh is a buffered channel used to notify the run loop that new
	// work has been enqueued via Put. Buffered with capacity 1 so that
	// Put never blocks even if the run loop is busy processing.
	wakeCh chan struct{}
	// stopCh is closed by Stop to signal the run loop to exit immediately.
	stopCh chan struct{}
	// stopOnce ensures Stop is idempotent and safe to call multiple times.
	stopOnce sync.Once
}

func StartBuildEngine() *BuildEngine {
	if comm.Cfg.Server.RunLimit < 2 {
		comm.Cfg.Server.RunLimit = 5
	}
	c := &BuildEngine{
		taskw:  list.New(),
		tasks:  make(map[string]*BuildTask),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	go func() {
		defer util.RecoverLog("BuildEngine.goroutine")
		c.init()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for comm.Ctx.Err() == nil {
			select {
			case <-c.stopCh:
				return
			case <-c.wakeCh:
				// Drain any pending notification to avoid stale signals
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

// Stop signals the run loop to exit and cancels all active builds.
// Safe to call multiple times from concurrent goroutines.
func (c *BuildEngine) Stop() {
	if c.stopCh != nil {
		c.stopOnce.Do(func() {
			close(c.stopCh)
		})
	}
	c.tskslk.RLock()
	defer c.tskslk.RUnlock()
	for _, v := range c.tasks {
		v.stop()
	}
}
func (c *BuildEngine) init() {
	cont := "server restart"
	if _, err := comm.Db.Context(comm.Ctx).Exec(
		"update `t_build` set `status`=?,`error`=? where `status`!=? and `status`!=? and `status`!=?",
		common.BuildStatusCancel, cont, common.BuildStatusOk, common.BuildStatusError, common.BuildStatusCancel,
	); err != nil {
		logrus.Errorf("BuildEngine init: failed to cancel pending builds: %v", err)
	}
	if _, err := comm.Db.Context(comm.Ctx).Exec(
		"update `t_stage` set `status`=?,`error`=? where `status`!=? and `status`!=? and `status`!=?",
		common.BuildStatusCancel, cont, common.BuildStatusOk, common.BuildStatusError, common.BuildStatusCancel,
	); err != nil {
		logrus.Errorf("BuildEngine init: failed to cancel pending stages: %v", err)
	}
	if _, err := comm.Db.Context(comm.Ctx).Exec(
		"update `t_step` set `status`=?,`error`=? where `status`!=? and `status`!=? and `status`!=?",
		common.BuildStatusCancel, cont, common.BuildStatusOk, common.BuildStatusError, common.BuildStatusCancel,
	); err != nil {
		logrus.Errorf("BuildEngine init: failed to cancel pending steps: %v", err)
	}
	if _, err := comm.Db.Context(comm.Ctx).Exec(
		"update `t_cmd_line` set `status`=? where `status`!=? and `status`!=? and `status`!=?",
		common.BuildStatusCancel, common.BuildStatusOk, common.BuildStatusError, common.BuildStatusCancel,
	); err != nil {
		logrus.Errorf("BuildEngine init: failed to cancel pending cmd lines: %v", err)
	}
}

func (c *BuildEngine) run() {
	defer util.RecoverLog("BuildEngine.run")

	c.tskwlk.RLock()
	ln1 := c.taskw.Len()
	c.tskwlk.RUnlock()
	c.tskslk.RLock()
	ln2 := len(c.tasks)
	c.tskslk.RUnlock()
	if ln1 > 0 && ln2 < comm.Cfg.Server.RunLimit {
		c.tskwlk.RLock()
		e := c.taskw.Front()
		c.tskwlk.RUnlock()
		if e == nil {
			return
		}
		c.tskwlk.Lock()
		c.taskw.Remove(e)
		c.tskwlk.Unlock()
		v := NewBuildTask(c, e.Value.(*runtime.Build))
		c.tskslk.Lock()
		c.tasks[v.build.Id] = v
		c.tskslk.Unlock()
		go func(bt *BuildTask) {
			defer util.RecoverLog("BuildEngine.startBuild")
			c.startBuild(bt)
		}(v)
	}
}
func (c *BuildEngine) startBuild(v *BuildTask) {
	v.run()
	c.tskslk.Lock()
	defer c.tskslk.Unlock()
	delete(c.tasks, v.build.Id)
}
func (c *BuildEngine) Put(bd *runtime.Build) {
	c.tskwlk.Lock()
	defer c.tskwlk.Unlock()
	c.taskw.PushBack(bd)
	// Wake up the run loop immediately so it can pick up the new build
	// without waiting for the next ticker tick. The channel is buffered
	// with capacity 1, so this never blocks even if the run loop is busy.
	select {
	case c.wakeCh <- struct{}{}:
	default:
	}
}
func (c *BuildEngine) Get(buildid string) (*BuildTask, bool) {
	if c == nil || buildid == "" {
		return nil, false
	}
	c.tskslk.RLock()
	defer c.tskslk.RUnlock()
	v, ok := c.tasks[buildid]
	return v, ok
}
