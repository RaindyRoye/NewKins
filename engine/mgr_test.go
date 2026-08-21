package engine

import (
	"container/list"
	"testing"

	"github.com/gokins/core/runtime"
)

// TestManagerBuildEgn tests the BuildEgn accessor method.
func TestManagerBuildEgn(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	mgr := &Manager{buildEgn: egn}
	got := mgr.BuildEgn()
	if got != egn {
		t.Error("BuildEgn() should return the build engine")
	}
}

// TestManagerBuildEgn_Nil tests that BuildEgn returns nil when not set.
func TestManagerBuildEgn_Nil(t *testing.T) {
	mgr := &Manager{}
	if mgr.BuildEgn() != nil {
		t.Error("BuildEgn() should return nil when not initialized")
	}
}

// TestManagerHRun tests the HRun accessor method.
func TestManagerHRun(t *testing.T) {
	hr := &HbtpRunner{}
	mgr := &Manager{hrun: hr}
	got := mgr.HRun()
	if got != hr {
		t.Error("HRun() should return the hbtp runner")
	}
}

// TestManagerHRun_Nil tests that HRun returns nil when not set.
func TestManagerHRun_Nil(t *testing.T) {
	mgr := &Manager{}
	if mgr.HRun() != nil {
		t.Error("HRun() should return nil when not initialized")
	}
}

// TestManagerTimerEng tests the TimerEng accessor method.
func TestManagerTimerEng(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	mgr := &Manager{timerEgn: te}
	got := mgr.TimerEng()
	if got != te {
		t.Error("TimerEng() should return the timer engine")
	}
}

// TestManagerTimerEng_Nil tests that TimerEng returns nil when not set.
func TestManagerTimerEng_Nil(t *testing.T) {
	mgr := &Manager{}
	if mgr.TimerEng() != nil {
		t.Error("TimerEng() should return nil when not initialized")
	}
}

// TestManagerPlugins tests the Plugins method.
func TestManagerPlugins(t *testing.T) {
	je := &JobEngine{
		execs: map[string]*executer{
			"shell@ssh":  {plug: "shell@ssh"},
			"gokins@git": {plug: "gokins@git"},
		},
		jobs: make(map[string]*jobSync),
	}
	mgr := &Manager{jobEgn: je}
	plugins := mgr.Plugins()
	if len(plugins) != 2 {
		t.Fatalf("Plugins() returned %d items, want 2", len(plugins))
	}
	// Check all plugins are present
	found := map[string]bool{}
	for _, p := range plugins {
		found[p] = true
	}
	if !found["shell@ssh"] {
		t.Error("expected shell@ssh in plugins")
	}
	if !found["gokins@git"] {
		t.Error("expected gokins@git in plugins")
	}
}

// TestManagerPlugins_NilJobEngine tests Plugins when jobEgn is nil.
func TestManagerPlugins_NilJobEngine(t *testing.T) {
	mgr := &Manager{}
	plugins := mgr.Plugins()
	if plugins != nil {
		t.Errorf("Plugins() should return nil when jobEgn is nil, got %v", plugins)
	}
}

// TestManagerPlugins_EmptyExecs tests Plugins when execs is empty.
func TestManagerPlugins_EmptyExecs(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	mgr := &Manager{jobEgn: je}
	plugins := mgr.Plugins()
	if len(plugins) != 0 {
		t.Errorf("Plugins() should return empty slice, got %v", plugins)
	}
}

// TestBuildEngineRunQueueEmpty tests run() when the task queue is empty.
func TestBuildEngineRunQueueEmpty(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Should not panic when queue is empty
	c.run()
	if c.taskw.Len() != 0 {
		t.Error("queue should still be empty")
	}
}

// TestBuildEngineStartBuild tests the startBuild lifecycle.
func TestBuildEngineStartBuild(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bd := newTestBuild("build-start-1")
	bt := &BuildTask{
		egn:   c,
		build: bd,
	}
	c.tasks["build-start-1"] = bt
	// After startBuild, the task should be removed from the map
	c.startBuild(bt)
	if _, ok := c.tasks["build-start-1"]; ok {
		t.Error("task should be removed from map after startBuild completes")
	}
}

// newTestBuild creates a minimal runtime.Build for testing.
func newTestBuild(id string) *runtime.Build {
	return &runtime.Build{Id: id}
}
