package engine

import (
	"container/list"
	"testing"
)

// --- Manager accessor tests ---

func TestManagerBuildEgn(t *testing.T) {
	m := &Manager{}
	if m.BuildEgn() != nil {
		t.Error("BuildEgn() should return nil when not set")
	}
	egn := &BuildEngine{taskw: list.New(), tasks: make(map[string]*BuildTask)}
	m.buildEgn = egn
	if m.BuildEgn() != egn {
		t.Error("BuildEgn() should return the assigned engine")
	}
}

func TestManagerHRun(t *testing.T) {
	m := &Manager{}
	if m.HRun() != nil {
		t.Error("HRun() should return nil when not set")
	}
	hr := &HbtpRunner{}
	m.hrun = hr
	if m.HRun() != hr {
		t.Error("HRun() should return the assigned HbtpRunner")
	}
}

func TestManagerTimerEng(t *testing.T) {
	m := &Manager{}
	if m.TimerEng() != nil {
		t.Error("TimerEng() should return nil when not set")
	}
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	m.timerEgn = te
	if m.TimerEng() != te {
		t.Error("TimerEng() should return the assigned timer engine")
	}
}

func TestManagerPlugins_NilJobEngine(t *testing.T) {
	m := &Manager{}
	// jobEgn is nil, should return nil
	rts := m.Plugins()
	if rts != nil {
		t.Errorf("Plugins() with nil jobEgn should return nil, got %v", rts)
	}
}

func TestManagerPlugins_EmptyExecs(t *testing.T) {
	m := &Manager{
		jobEgn: &JobEngine{
			execs: make(map[string]*executer),
			jobs:  make(map[string]*jobSync),
		},
	}
	rts := m.Plugins()
	if len(rts) != 0 {
		t.Errorf("Plugins() with empty execs should return empty, got %v", rts)
	}
}
