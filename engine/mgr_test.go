package engine

import (
	"container/list"
	"testing"
)

// --- Manager accessor methods ---

func TestManagerBuildEgn(t *testing.T) {
	origMgr := *Mgr
	defer func() { *Mgr = origMgr }()

	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	m := &Manager{buildEgn: egn}
	if m.BuildEgn() != egn {
		t.Error("BuildEgn() should return the assigned build engine")
	}
}

func TestManagerBuildEgn_Nil(t *testing.T) {
	m := &Manager{}
	if m.BuildEgn() != nil {
		t.Error("BuildEgn() should return nil when not initialized")
	}
}

func TestManagerHRun(t *testing.T) {
	origMgr := *Mgr
	defer func() { *Mgr = origMgr }()

	hr := &HbtpRunner{}
	m := &Manager{hrun: hr}
	if m.HRun() != hr {
		t.Error("HRun() should return the assigned HbtpRunner")
	}
}

func TestManagerHRun_Nil(t *testing.T) {
	m := &Manager{}
	if m.HRun() != nil {
		t.Error("HRun() should return nil when not initialized")
	}
}

func TestManagerTimerEng(t *testing.T) {
	origMgr := *Mgr
	defer func() { *Mgr = origMgr }()

	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	m := &Manager{timerEgn: te}
	if m.TimerEng() != te {
		t.Error("TimerEng() should return the assigned TimerEngine")
	}
}

func TestManagerTimerEng_Nil(t *testing.T) {
	m := &Manager{}
	if m.TimerEng() != nil {
		t.Error("TimerEng() should return nil when not initialized")
	}
}

func TestManagerPlugins_NilJobEgn(t *testing.T) {
	m := &Manager{jobEgn: nil}
	plugs := m.Plugins()
	if plugs != nil {
		t.Errorf("Plugins() with nil jobEgn should return nil, got %v", plugs)
	}
}

func TestManagerPlugins_WithJobEgn(t *testing.T) {
	je := &JobEngine{
		execs: map[string]*executer{
			"a": {plug: "a"},
			"b": {plug: "b"},
		},
		jobs: make(map[string]*jobSync),
	}
	m := &Manager{jobEgn: je}
	plugs := m.Plugins()
	if len(plugs) != 2 {
		t.Fatalf("Plugins() should return 2, got %d", len(plugs))
	}
	found := map[string]bool{}
	for _, p := range plugs {
		found[p] = true
	}
	if !found["a"] || !found["b"] {
		t.Errorf("Plugins() should contain a and b, got %v", plugs)
	}
}

func TestManagerPlugins_EmptyJobEgn(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	m := &Manager{jobEgn: je}
	plugs := m.Plugins()
	if len(plugs) != 0 {
		t.Errorf("Plugins() should return empty, got %v", plugs)
	}
}

// --- Global Mgr singleton ---

func TestGlobalMgrNotNil(t *testing.T) {
	if Mgr == nil {
		t.Fatal("global Mgr should not be nil")
	}
}
