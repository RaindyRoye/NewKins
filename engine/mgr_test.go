package engine

import (
	"testing"
)

// TestMgrAccessors tests that Manager accessor methods work correctly
func TestMgrAccessors(t *testing.T) {
	mgr := &Manager{}

	// Test nil accessors before initialization
	if mgr.BuildEgn() != nil {
		t.Error("BuildEgn() should return nil before initialization")
	}
	if mgr.HRun() != nil {
		t.Error("HRun() should return nil before initialization")
	}
	if mgr.TimerEng() != nil {
		t.Error("TimerEng() should return nil before initialization")
	}

	// Test Plugins() with nil jobEgn - should not panic
	plugins := mgr.Plugins()
	if plugins != nil {
		t.Errorf("Plugins() should return nil with nil jobEgn, got %v", plugins)
	}
}

// TestMgrInitialization tests that Manager can be initialized with components
func TestMgrInitialization(t *testing.T) {
	buildEng := &BuildEngine{}
	jobEng := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	timerEng := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	mgr := &Manager{
		buildEgn: buildEng,
		jobEgn:   jobEng,
		timerEgn: timerEng,
	}

	// Verify accessors return the initialized components
	if mgr.BuildEgn() != buildEng {
		t.Error("BuildEgn() should return the initialized BuildEngine")
	}
	if mgr.TimerEng() != timerEng {
		t.Error("TimerEng() should return the initialized TimerEngine")
	}

	// Test Plugins() with initialized jobEgn
	plugins := mgr.Plugins()
	if plugins == nil {
		t.Error("Plugins() should return a slice (even if empty) with initialized jobEgn")
	}
}

// TestGlobalMgrExists verifies the global Mgr variable exists
func TestGlobalMgrExists(t *testing.T) {
	if Mgr == nil {
		t.Fatal("Global Mgr should be initialized")
	}
}

// TestMgrPluginsWithRegisteredExecs verifies Plugins() returns registered plugin names
func TestMgrPluginsWithRegisteredExecs(t *testing.T) {
	jobEng := &JobEngine{
		execs: map[string]*executer{
			"shell":  {plug: "shell"},
			"docker": {plug: "docker"},
		},
		jobs: make(map[string]*jobSync),
	}
	mgr := &Manager{jobEgn: jobEng}

	plugins := mgr.Plugins()
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d: %v", len(plugins), plugins)
	}
	found := make(map[string]bool)
	for _, p := range plugins {
		found[p] = true
	}
	if !found["shell"] || !found["docker"] {
		t.Errorf("expected shell and docker plugins, got %v", plugins)
	}
}
