package engine

import (
	"container/list"
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
)

// --- Manager accessor tests ---

func TestManagerBuildEgn(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	m := &Manager{buildEgn: egn}
	if m.BuildEgn() != egn {
		t.Error("BuildEgn() should return the assigned engine")
	}
}

func TestManagerBuildEgn_Nil(t *testing.T) {
	m := &Manager{}
	if m.BuildEgn() != nil {
		t.Error("BuildEgn() should return nil when not set")
	}
}

func TestManagerHRun(t *testing.T) {
	hr := &HbtpRunner{}
	m := &Manager{hrun: hr}
	if m.HRun() != hr {
		t.Error("HRun() should return the assigned HbtpRunner")
	}
}

func TestManagerHRun_Nil(t *testing.T) {
	m := &Manager{}
	if m.HRun() != nil {
		t.Error("HRun() should return nil when not set")
	}
}

func TestManagerTimerEng(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	m := &Manager{timerEgn: te}
	if m.TimerEng() != te {
		t.Error("TimerEng() should return the assigned TimerEngine")
	}
}

func TestManagerTimerEng_Nil(t *testing.T) {
	m := &Manager{}
	if m.TimerEng() != nil {
		t.Error("TimerEng() should return nil when not set")
	}
}

func TestManagerPlugins_NilJobEngine(t *testing.T) {
	m := &Manager{}
	if m.Plugins() != nil {
		t.Error("Plugins() should return nil when jobEgn is nil")
	}
}

func TestManagerPlugins_WithJobEngine(t *testing.T) {
	je := newTestJobEngine()
	je.execs["shell@ssh"] = &executer{plug: "shell@ssh", jobwt: list.New()}
	je.execs["gokins@git"] = &executer{plug: "gokins@git", jobwt: list.New()}
	m := &Manager{jobEgn: je}
	plugs := m.Plugins()
	if len(plugs) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugs))
	}
	found := map[string]bool{}
	for _, p := range plugs {
		found[p] = true
	}
	if !found["shell@ssh"] || !found["gokins@git"] {
		t.Errorf("expected shell@ssh and gokins@git, got %v", plugs)
	}
}

// --- BuildEngine.run() tests ---

func TestBuildEngineRun_EmptyQueue(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// run() with empty queue should not panic and not start any builds
	c.run()
	if len(c.tasks) != 0 {
		t.Errorf("expected 0 tasks after run with empty queue, got %d", len(c.tasks))
	}
}

func TestBuildEngineRun_QueueNotEmpty(t *testing.T) {
	// Set RunLimit to a value > 0 so run() will process the queue
	origLimit := comm.Cfg.Server.RunLimit
	comm.Cfg.Server.RunLimit = 5
	defer func() { comm.Cfg.Server.RunLimit = origLimit }()

	// Set up comm.WorkPath to a temp dir so build path creation doesn't fail
	origWorkPath := comm.WorkPath
	comm.WorkPath = t.TempDir()
	defer func() { comm.WorkPath = origWorkPath }()

	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bd := &runtime.Build{
		Id:     "build-queue-1",
		Repo:   nil, // will cause check() to fail quickly
		Stages: []*runtime.Stage{},
	}
	c.taskw.PushBack(bd)

	// Run should dequeue one element from the task queue
	c.run()

	// The queue should now be empty (element was removed and spawned as goroutine)
	if c.taskw.Len() != 0 {
		t.Errorf("expected queue to be empty after run, got %d", c.taskw.Len())
	}
	// Task should be in the tasks map
	if len(c.tasks) != 1 {
		t.Errorf("expected 1 task in map, got %d", len(c.tasks))
	}
}

// --- BuildEngine.startBuild() test ---

func TestBuildEngineStartBuild_Cleanup(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Create a BuildTask that will fail quickly due to nil repo
	bt := &BuildTask{
		build: &runtime.Build{
			Id:     "build-start-1",
			Repo:   nil,
			Stages: []*runtime.Stage{},
		},
	}
	c.tasks["build-start-1"] = bt
	c.startBuild(bt)

	// After startBuild, the task should be removed from tasks map
	if _, ok := c.tasks["build-start-1"]; ok {
		t.Error("startBuild should remove task from map after completion")
	}
}

// --- BuildEngine.Get() nil receiver ---

func TestBuildEngineGet_NilReceiver(t *testing.T) {
	var c *BuildEngine
	_, ok := c.Get("anything")
	if ok {
		t.Error("Get on nil BuildEngine should return false")
	}
}

// --- Multiple Put/Get operations ---

func TestBuildEngineMultiplePutGet(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Put several builds
	for i := 0; i < 5; i++ {
		c.Put(&runtime.Build{Id: "multi-" + string(rune('a'+i))})
	}
	if c.taskw.Len() != 5 {
		t.Fatalf("expected 5 in queue, got %d", c.taskw.Len())
	}
}
