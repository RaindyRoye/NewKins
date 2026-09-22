package engine

import (
	"testing"
)

// --- Manager accessor tests ---

func TestManager_BuildEgn(t *testing.T) {
	m := &Manager{}
	if m.BuildEgn() != nil {
		t.Error("expected nil BuildEgn for uninitialized Manager")
	}
	be := &BuildEngine{}
	m.buildEgn = be
	if m.BuildEgn() != be {
		t.Error("BuildEgn() should return the configured engine")
	}
}

func TestManager_HRun(t *testing.T) {
	m := &Manager{}
	if m.HRun() != nil {
		t.Error("expected nil HRun for uninitialized Manager")
	}
	hr := &HbtpRunner{}
	m.hrun = hr
	if m.HRun() != hr {
		t.Error("HRun() should return the configured runner")
	}
}

func TestManager_TimerEng(t *testing.T) {
	m := &Manager{}
	if m.TimerEng() != nil {
		t.Error("expected nil TimerEng for uninitialized Manager")
	}
	te := &TimerEngine{}
	m.timerEgn = te
	if m.TimerEng() != te {
		t.Error("TimerEng() should return the configured engine")
	}
}

func TestManager_Plugins_NilJobEngine(t *testing.T) {
	m := &Manager{}
	// Should not panic when jobEgn is nil
	plugs := m.Plugins()
	if plugs != nil {
		t.Errorf("expected nil plugins for nil jobEgn, got %v", plugs)
	}
}
