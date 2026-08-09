package engine

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gokins/core/common"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/util"
	"github.com/gokins/runner/runners"
	"github.com/sirupsen/logrus"
)

// ManagerConfig holds the configuration for creating a Manager instance.
// This enables dependency injection and reduces reliance on global state.
type ManagerConfig struct {
	WorkPath string
	Ctx      context.Context
	Cfg      *comm.Config
	ShellRun *runners.Engine
}

// NewManager creates a new Manager instance with the provided configuration.
// This is the preferred way to create a Manager in new code, as it enables
// dependency injection and makes testing easier.
//
// Example:
//
//	cfg := ManagerConfig{
//	    WorkPath: "/var/lib/gokins",
//	    Ctx:      ctx,
//	    Cfg:      &comm.Cfg,
//	}
//	mgr, err := NewManager(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewManager(cfg ManagerConfig) (*Manager, error) {
	if cfg.Ctx == nil {
		cfg.Ctx = context.Background()
	}
	if cfg.WorkPath == "" {
		cfg.WorkPath = utils.EnvDefault("GOKINS_WORKPATH", "/var/lib/gokins")
	}

	m := &Manager{
		workPath: cfg.WorkPath,
		ctx:      cfg.Ctx,
	}

	// Initialize build engine
	m.buildEgn = StartBuildEngine()

	// Initialize job engine
	m.jobEgn = StartJobEngine()

	// Initialize timer engine
	m.tmrEgn = StartTimerEngine()

	// Initialize base runner
	m.brun = &baseRunner{}

	// Initialize hbtp runner
	m.hrun = &HbtpRunner{}

	// Initialize shell runner with config
	if cfg.Cfg != nil {
		cfg.Cfg.Server.Shells = append(cfg.Cfg.Server.Shells, "shell@ssh", "gokins@git")
		m.shellRun = runners.NewEngine(runners.Config{
			Name:      "mainRunner",
			Workspace: filepath.Join(cfg.WorkPath, common.PathRunner),
			Plugin:    cfg.Cfg.Server.Shells,
		}, m.brun)
	} else if cfg.ShellRun != nil {
		m.shellRun = cfg.ShellRun
	}

	return m, nil
}

// StartWithContext starts the manager with a specific context.
// This is useful for testing or when you need to control the lifecycle
// of the manager independently of the global context.
func (m *Manager) StartWithContext() error {
	if m.ctx == nil {
		m.ctx = context.Background()
	}

	// Start shell runner if configured
	if m.shellRun != nil {
		go func() {
			defer util.RecoverLog("shell runner goroutine")
			err := m.shellRun.Run(m.ctx)
			if err != nil {
				logrus.Errorf("runner err:%v", err)
			}
		}()
	}

	// Start cleanup goroutine
	go func() {
		defer util.RecoverLog("shutdown goroutine")
		if m.workPath != "" {
			_ = os.RemoveAll(filepath.Join(m.workPath, common.PathTmp))
		}
		<-m.ctx.Done()
		if m.buildEgn != nil {
			m.buildEgn.Stop()
		}
		if m.shellRun != nil {
			m.shellRun.Stop()
		}
	}()

	return nil
}

// Stop gracefully shuts down the manager and all its engines.
func (m *Manager) Stop() {
	if m.buildEgn != nil {
		m.buildEgn.Stop()
	}
	if m.shellRun != nil {
		m.shellRun.Stop()
	}
}
