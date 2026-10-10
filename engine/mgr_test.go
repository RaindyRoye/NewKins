package engine

import (
	"testing"

	"github.com/gokins/gokins/comm"
)

func TestMgr_BuildEgn(t *testing.T) {
	mgr := &Manager{}
	if mgr.BuildEgn() != nil {
		t.Error("BuildEgn() should return nil when not initialized")
	}
}

func TestMgr_TimerEng(t *testing.T) {
	mgr := &Manager{}
	if mgr.TimerEng() != nil {
		t.Error("TimerEng() should return nil when not initialized")
	}
}

func TestMgr_HRun(t *testing.T) {
	mgr := &Manager{}
	hrun := mgr.HRun()
	if hrun != nil {
		t.Error("HRun() should return nil when not initialized")
	}
}

func TestMgr_Plugins(t *testing.T) {
	mgr := &Manager{}
	plugins := mgr.Plugins()
	if plugins != nil {
		t.Errorf("Plugins() should return nil when jobEgn is nil, got %v", plugins)
	}

	// Test with initialized jobEgn
	mgr.jobEgn = &JobEngine{
		execs: map[string]*executer{
			"plugin1": {plug: "plugin1"},
			"plugin2": {plug: "plugin2"},
		},
	}
	plugins = mgr.Plugins()
	if len(plugins) != 2 {
		t.Errorf("Plugins() should return 2 plugins, got %d", len(plugins))
	}
}

func TestStart(t *testing.T) {
	// Save original context state
	origCfg := comm.Cfg
	defer func() {
		comm.Cfg = origCfg
	}()

	// Reset context for clean test
	comm.ResetCtx()
	defer comm.ResetCtx()

	// Set minimal config
	comm.Cfg.Server.RunLimit = 2

	// Start should succeed without panic
	err := Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify engines were created
	if Mgr.buildEgn == nil {
		t.Error("Start() should create buildEgn")
	}
	if Mgr.jobEgn == nil {
		t.Error("Start() should create jobEgn")
	}
	if Mgr.timerEgn == nil {
		t.Error("Start() should create timerEgn")
	}
	if Mgr.brun == nil {
		t.Error("Start() should create brun")
	}
	if Mgr.hrun == nil {
		t.Error("Start() should create hrun")
	}
	if Mgr.shellRun == nil {
		t.Error("Start() should create shellRun")
	}

	// Cancel to stop background goroutines
	comm.Cancel()
}
