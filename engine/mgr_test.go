package engine

import (
	"container/list"
	"testing"
)

func TestManagerAccessorsNilSafe(t *testing.T) {
	m := &Manager{}

	if m.BuildEgn() != nil {
		t.Error("expected nil BuildEgn from empty Manager")
	}
	if m.HRun() != nil {
		t.Error("expected nil HRun from empty Manager")
	}
	if m.TimerEng() != nil {
		t.Error("expected nil TimerEng from empty Manager")
	}
}

func TestManagerPluginsNilJobEgn(t *testing.T) {
	m := &Manager{}
	// jobEgn is nil, should return nil without panic
	plugs := m.Plugins()
	if plugs != nil {
		t.Errorf("expected nil plugins when jobEgn is nil, got %v", plugs)
	}
}

func TestManagerPluginsWithJobEgn(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	je.execs["test-plugin"] = &executer{
		plug:  "test-plugin",
		jobwt: list.New(),
	}
	m := &Manager{
		jobEgn: je,
	}
	plugs := m.Plugins()
	if len(plugs) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugs))
	}
	if plugs[0] != "test-plugin" {
		t.Errorf("expected 'test-plugin', got %q", plugs[0])
	}
}

func TestManagerBuildEgnAccessor(t *testing.T) {
	be := &BuildEngine{
		taskw:  list.New(),
		tasks:  make(map[string]*BuildTask),
		wakeCh: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
	}
	m := &Manager{
		buildEgn: be,
	}
	if m.BuildEgn() != be {
		t.Error("expected BuildEgn to return the assigned engine")
	}
}

func TestManagerTimerEngAccessor(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	m := &Manager{
		timerEgn: te,
	}
	if m.TimerEng() != te {
		t.Error("expected TimerEng to return the assigned engine")
	}
}
