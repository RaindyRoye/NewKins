package engine

import (
	"container/list"
	"testing"
)

func TestManagerAccessors(t *testing.T) {
	m := &Manager{}

	// All accessors should return nil when fields are uninitialized
	if m.BuildEgn() != nil {
		t.Error("BuildEgn() should return nil when uninitialized")
	}
	if m.HRun() != nil {
		t.Error("HRun() should return nil when uninitialized")
	}
	if m.TimerEng() != nil {
		t.Error("TimerEng() should return nil when uninitialized")
	}
}

func TestManagerPlugins_NilJobEgn(t *testing.T) {
	m := &Manager{}
	plugs := m.Plugins()
	if plugs != nil {
		t.Errorf("Plugins() should return nil when jobEgn is nil, got %v", plugs)
	}
}

func TestManagerPlugins_WithJobEgn(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	je.execs["shell"] = &executer{plug: "shell", jobwt: list.New()}
	m := &Manager{jobEgn: je}
	plugs := m.Plugins()
	if len(plugs) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugs))
	}
	if plugs[0] != "shell" {
		t.Errorf("expected plugin 'shell', got %q", plugs[0])
	}
}

func TestManagerBuildEgn_WithValue(t *testing.T) {
	be := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	m := &Manager{buildEgn: be}
	if m.BuildEgn() != be {
		t.Error("BuildEgn() should return the assigned engine")
	}
}

func TestManagerHRun_WithValue(t *testing.T) {
	hr := &HbtpRunner{}
	m := &Manager{hrun: hr}
	if m.HRun() != hr {
		t.Error("HRun() should return the assigned HbtpRunner")
	}
}

func TestManagerTimerEng_WithValue(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	m := &Manager{timerEgn: te}
	if m.TimerEng() != te {
		t.Error("TimerEng() should return the assigned TimerEngine")
	}
}
