package engine

import (
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/gokins/core/common"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/runner/runners"
	"github.com/sirupsen/logrus"
)

var Mgr = &Manager{}

type Manager struct {
	buildEgn *BuildEngine
	jobEgn   *JobEngine
	shellRun *runners.Engine
	brun     *baseRunner
	hrun     *HbtpRunner
	timerEgn *TimerEngine
}

func Start() error {
	Mgr.buildEgn = StartBuildEngine()
	Mgr.jobEgn = StartJobEngine()
	Mgr.timerEgn = StartTimerEngine()

	Mgr.brun = &baseRunner{}
	Mgr.hrun = &HbtpRunner{}
	// runners
	comm.Cfg.Server.Shells = append(comm.Cfg.Server.Shells, "shell@ssh", "gokins@git")
	Mgr.shellRun = runners.NewEngine(runners.Config{
		Name:      "mainRunner",
		Workspace: filepath.Join(comm.WorkPath, common.PathRunner),
		Plugin:    comm.Cfg.Server.Shells,
	}, Mgr.brun)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("shell runner goroutine panic: %v\n%s", r, string(debug.Stack()))
			}
		}()
		err := Mgr.shellRun.Run(comm.Ctx)
		if err != nil {
			logrus.Errorf("runner err:%v", err)
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("shutdown goroutine panic: %v\n%s", r, string(debug.Stack()))
			}
		}()
		_ = os.RemoveAll(filepath.Join(comm.WorkPath, common.PathTmp))
		// Block until context is canceled instead of busy-waiting.
		<-comm.Ctx.Done()
		if Mgr.buildEgn != nil {
			Mgr.buildEgn.Stop()
		}
		if Mgr.shellRun != nil {
			Mgr.shellRun.Stop()
		}
	}()
	return nil
}

func (c *Manager) BuildEgn() *BuildEngine {
	return c.buildEgn
}
func (c *Manager) HRun() *HbtpRunner {
	return c.hrun
}

func (c *Manager) TimerEng() *TimerEngine {
	return c.timerEgn
}

func (c *Manager) Plugins() []string {
	if c.jobEgn == nil {
		return nil
	}
	return c.jobEgn.Plugins()
}
