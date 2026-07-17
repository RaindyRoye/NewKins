package engine

import (
	"container/list"
	"sync"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/util"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/sirupsen/logrus"
)

type BuildEngine struct {
	app *comm.App // dependency-injected application state

	tskwlk sync.RWMutex
	taskw  *list.List

	tskslk sync.RWMutex
	tasks  map[string]*BuildTask
}

// StartBuildEngine creates a BuildEngine using the default App singleton.
// This is the backward-compatible entry point; prefer StartBuildEngineWithApp
// in new code so the dependency is explicit.
func StartBuildEngine() *BuildEngine {
	return StartBuildEngineWithApp(comm.GetApp())
}

// StartBuildEngineWithApp creates a BuildEngine that reads configuration and
// database state from the supplied *comm.App instead of package-level globals.
// Passing nil falls back to the default App (same as StartBuildEngine).
func StartBuildEngineWithApp(app *comm.App) *BuildEngine {
	if app == nil {
		app = comm.GetApp()
	}
	limit := app.RunLimit()
	c := &BuildEngine{
		app:   app,
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Persist the resolved limit back so callers that read Cfg still see it.
	comm.Cfg.Server.RunLimit = limit
	go func() {
		defer util.RecoverLog("BuildEngine.goroutine")
		c.init()
		for !hbtp.EndContext(app.AppCtx()) {
			c.run()
			time.Sleep(time.Second)
		}
	}()
	return c
}
func (c *BuildEngine) Stop() {
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
		go c.startBuild(v)
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
}
func (c *BuildEngine) Get(buildid string) (*BuildTask, bool) {
	if buildid == "" {
		return nil, false
	}
	c.tskslk.RLock()
	defer c.tskslk.RUnlock()
	v, ok := c.tasks[buildid]
	return v, ok
}
