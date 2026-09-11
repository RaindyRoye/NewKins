package engine

import (
	"testing"
)

func TestManager_BuildEgn(t *testing.T) {
	mgr := &Manager{}

	// Initially should be nil
	if mgr.BuildEgn() != nil {
		t.Error("BuildEgn() should return nil when not initialized")
	}

	// Set a build engine and verify
	be := &BuildEngine{}
	mgr.buildEgn = be
	if mgr.BuildEgn() != be {
		t.Error("BuildEgn() should return the set build engine")
	}
}

func TestManager_HRun(t *testing.T) {
	mgr := &Manager{}

	// Initially should be nil
	if mgr.HRun() != nil {
		t.Error("HRun() should return nil when not initialized")
	}

	// Set an hbtp runner and verify
	hr := &HbtpRunner{}
	mgr.hrun = hr
	if mgr.HRun() != hr {
		t.Error("HRun() should return the set hbtp runner")
	}
}

func TestManager_TimerEng(t *testing.T) {
	mgr := &Manager{}

	// Initially should be nil
	if mgr.TimerEng() != nil {
		t.Error("TimerEng() should return nil when not initialized")
	}

	// Set a timer engine and verify
	te := &TimerEngine{}
	mgr.timerEgn = te
	if mgr.TimerEng() != te {
		t.Error("TimerEng() should return the set timer engine")
	}
}

func TestManager_Plugins(t *testing.T) {
	mgr := &Manager{}

	// With nil jobEgn, should return nil
	if mgr.Plugins() != nil {
		t.Error("Plugins() should return nil when jobEgn is nil")
	}

	// With a job engine but no plugins
	je := &JobEngine{
		execs: make(map[string]*executer),
	}
	mgr.jobEgn = je
	plugins := mgr.Plugins()
	if plugins == nil {
		t.Error("Plugins() should return non-nil slice even when empty")
	}
	if len(plugins) != 0 {
		t.Errorf("Plugins() should return empty slice, got %d items", len(plugins))
	}
}

func TestManager_Plugins_WithData(t *testing.T) {
	mgr := &Manager{}

	// Create a job engine with some executors
	je := &JobEngine{
		execs: map[string]*executer{
			"plugin1": {plug: "shell@ssh"},
			"plugin2": {plug: "gokins@git"},
		},
	}
	mgr.jobEgn = je

	plugins := mgr.Plugins()
	if len(plugins) != 2 {
		t.Errorf("Plugins() should return 2 plugins, got %d", len(plugins))
	}

	// Verify both plugins are present (order may vary due to map)
	found := make(map[string]bool)
	for _, p := range plugins {
		found[p] = true
	}
	if !found["shell@ssh"] {
		t.Error("Plugins() should contain 'shell@ssh'")
	}
	if !found["gokins@git"] {
		t.Error("Plugins() should contain 'gokins@git'")
	}
}
