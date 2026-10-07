package engine

import (
	"testing"
)

func TestManager_BuildEgn(t *testing.T) {
	m := &Manager{}
	if m.BuildEgn() != nil {
		t.Error("BuildEgn() should return nil when not initialized")
	}
	egn := &BuildEngine{}
	m.buildEgn = egn
	if m.BuildEgn() != egn {
		t.Error("BuildEgn() should return the set engine")
	}
}

func TestManager_HRun(t *testing.T) {
	m := &Manager{}
	if m.HRun() != nil {
		t.Error("HRun() should return nil when not initialized")
	}
	hrun := &HbtpRunner{}
	m.hrun = hrun
	if m.HRun() != hrun {
		t.Error("HRun() should return the set runner")
	}
}

func TestManager_TimerEng(t *testing.T) {
	m := &Manager{}
	if m.TimerEng() != nil {
		t.Error("TimerEng() should return nil when not initialized")
	}
	teng := &TimerEngine{}
	m.timerEgn = teng
	if m.TimerEng() != teng {
		t.Error("TimerEng() should return the set engine")
	}
}

func TestManager_Plugins_NilJobEngine(t *testing.T) {
	m := &Manager{}
	plugins := m.Plugins()
	if plugins != nil {
		t.Errorf("Plugins() with nil jobEgn should return nil, got %v", plugins)
	}
}

func TestManager_Plugins_WithJobEngine(t *testing.T) {
	m := &Manager{}
	m.jobEgn = StartJobEngine()
	
	plugins := m.Plugins()
	if plugins == nil {
		t.Error("Plugins() should return non-nil slice")
	}
}
