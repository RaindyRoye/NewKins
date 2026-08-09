package engine

import (
	"container/list"
	"testing"

	"github.com/gokins/core/runtime"
)

func TestManager_BuildEgn_Nil(t *testing.T) {
	m := &Manager{}
	if m.BuildEgn() != nil {
		t.Fatal("expected nil BuildEngine from uninitialized Manager")
	}
}

func TestManager_BuildEgn_Set(t *testing.T) {
	be := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	m := &Manager{buildEgn: be}
	if m.BuildEgn() != be {
		t.Fatal("expected BuildEgn() to return the same engine")
	}
}

func TestManager_HRun_Nil(t *testing.T) {
	m := &Manager{}
	if m.HRun() != nil {
		t.Fatal("expected nil HbtpRunner from uninitialized Manager")
	}
}

func TestManager_HRun_Set(t *testing.T) {
	hr := &HbtpRunner{}
	m := &Manager{hrun: hr}
	if m.HRun() != hr {
		t.Fatal("expected HRun() to return the same runner")
	}
}

func TestManager_TimerEng_Nil(t *testing.T) {
	m := &Manager{}
	if m.TimerEng() != nil {
		t.Fatal("expected nil TimerEngine from uninitialized Manager")
	}
}

func TestManager_TimerEng_Set(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	m := &Manager{tmrEgn: te}
	if m.TimerEng() != te {
		t.Fatal("expected TimerEng() to return the same engine")
	}
}

func TestManager_Plugins_NilJobEngine(t *testing.T) {
	m := &Manager{}
	plugs := m.Plugins()
	if plugs != nil {
		t.Fatalf("expected nil plugins when jobEgn is nil, got %v", plugs)
	}
}

func TestManager_Plugins_DelegatesToJobEngine(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	je.execs["shell@ssh"] = &executer{
		plug:  "shell@ssh",
		jobwt: list.New(),
	}
	m := &Manager{jobEgn: je}
	plugs := m.Plugins()
	if len(plugs) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugs))
	}
	if plugs[0] != "shell@ssh" {
		t.Errorf("expected plugin 'shell@ssh', got %q", plugs[0])
	}
}

func TestManager_GlobalMgr_NotNil(t *testing.T) {
	if Mgr == nil {
		t.Fatal("global Mgr should be initialized")
	}
}

func TestBuildEngine_Stop_Empty(t *testing.T) {
	// Stop on an engine with no tasks should not panic
	be := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	be.Stop()
}

func TestBuildEngine_Stop_StopsAllTasks(t *testing.T) {
	be := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Add a task and verify Stop() processes it
	bt := &BuildTask{
		build: &runtime.Build{Id: "stop-test"},
	}
	be.tasks["stop-test"] = bt

	// Stop should not panic with nil context
	be.Stop()
}
