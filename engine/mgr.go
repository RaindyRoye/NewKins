package engine

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gokins/core/common"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/util"
	"github.com/gokins/runner/runners"
	"github.com/sirupsen/logrus"
)

// Mgr is the global Manager instance.
// Deprecated: New code should prefer NewManager for dependency injection.
var Mgr = &Manager{}

// Manager coordinates build execution, job scheduling, and runner management.
// It holds references to various engines and runners that work together to
// execute CI/CD pipelines.
type Manager struct {
	// workPath is the base directory for build artifacts and temporary files.
	// Used by DI-based constructors instead of the global comm.WorkPath.
	workPath string

	// ctx is the lifecycle context for this manager instance.
	// Used by DI-based constructors instead of the global comm.Ctx.
	ctx context.Context

	buildEgn *BuildEngine
	jobEgn   *JobEngine
	shellRun *runners.Engine
	brun     *baseRunner
	hrun     *HbtpRunner
	tmrEgn   *TimerEngine
}

func Start() error {
	Mgr.buildEgn = StartBuildEngine()
	Mgr.jobEgn = StartJobEngine()
	Mgr.tmrEgn = StartTimerEngine()

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
		defer util.RecoverLog("shell runner goroutine")
		err := Mgr.shellRun.Run(comm.Ctx)
		if err != nil {
			logrus.Errorf("runner err:%v", err)
		}
	}()

	go func() {
		defer util.RecoverLog("shutdown goroutine")
		_ = os.RemoveAll(filepath.Join(comm.WorkPath, common.PathTmp))
		// Block until context is canceled instead of busy-waiting.
		<-comm.Ctx.Done()
		Mgr.buildEgn.Stop()
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
	return c.tmrEgn
}

func (c *Manager) Plugins() []string {
	if c.jobEgn == nil {
		return nil
	}
	return c.jobEgn.Plugins()
}
