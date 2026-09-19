package engine

import (
	"testing"
)

func TestManager_BuildEgn_Nil(t *testing.T) {
	m := &Manager{}
	if m.BuildEgn() != nil {
		t.Error("BuildEgn() should return nil when not initialized")
	}
}

func TestManager_HRun_Nil(t *testing.T) {
	m := &Manager{}
	if m.HRun() != nil {
		t.Error("HRun() should return nil when not initialized")
	}
}

func TestManager_TimerEng_Nil(t *testing.T) {
	m := &Manager{}
	if m.TimerEng() != nil {
		t.Error("TimerEng() should return nil when not initialized")
	}
}

func TestManager_Plugins_NilJobEngine(t *testing.T) {
	m := &Manager{jobEgn: nil}
	plugins := m.Plugins()
	if plugins != nil {
		t.Errorf("Plugins() = %v, want nil when jobEgn is nil", plugins)
	}
}

func TestManager_BuildEgn_Initialized(t *testing.T) {
	m := &Manager{buildEgn: &BuildEngine{}}
	if m.BuildEgn() == nil {
		t.Error("BuildEgn() should not return nil when initialized")
	}
}

func TestManager_HRun_Initialized(t *testing.T) {
	m := &Manager{hrun: &HbtpRunner{}}
	if m.HRun() == nil {
		t.Error("HRun() should not return nil when initialized")
	}
}

func TestManager_TimerEng_Initialized(t *testing.T) {
	m := &Manager{timerEgn: &TimerEngine{}}
	if m.TimerEng() == nil {
		t.Error("TimerEng() should not return nil when initialized")
	}
}
