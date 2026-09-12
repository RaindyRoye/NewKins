package engine

import (
	"testing"
)

func TestManagerPluginsNil(t *testing.T) {
	mgr := &Manager{
		jobEgn: nil,
	}
	plugins := mgr.Plugins()
	if plugins != nil {
		t.Errorf("expected nil plugins when jobEgn is nil, got %v", plugins)
	}
}

func TestManagerBuildEgn(t *testing.T) {
	be := &BuildEngine{}
	mgr := &Manager{
		buildEgn: be,
	}
	if mgr.BuildEgn() != be {
		t.Error("BuildEgn() should return the assigned build engine")
	}
}

func TestManagerHRun(t *testing.T) {
	hr := &HbtpRunner{}
	mgr := &Manager{
		hrun: hr,
	}
	if mgr.HRun() != hr {
		t.Error("HRun() should return the assigned HbtpRunner")
	}
}

func TestManagerTimerEng(t *testing.T) {
	te := &TimerEngine{}
	mgr := &Manager{
		timerEgn: te,
	}
	if mgr.TimerEng() != te {
		t.Error("TimerEng() should return the assigned TimerEngine")
	}
}
