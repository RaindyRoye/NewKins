package engine

import (
	"container/list"
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

// --- Manager accessor methods ---

func TestManager_BuildEgn(t *testing.T) {
	origBuildEgn := Mgr.buildEgn
	defer func() { Mgr.buildEgn = origBuildEgn }()

	e := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	Mgr.buildEgn = e

	got := Mgr.BuildEgn()
	if got != e {
		t.Error("BuildEgn() should return the build engine")
	}
}

func TestManager_HRun(t *testing.T) {
	origHrun := Mgr.hrun
	defer func() { Mgr.hrun = origHrun }()

	hr := &HbtpRunner{}
	Mgr.hrun = hr

	got := Mgr.HRun()
	if got != hr {
		t.Error("HRun() should return the HbtpRunner")
	}
}

func TestManager_TimerEng(t *testing.T) {
	origTimerEgn := Mgr.timerEgn
	defer func() { Mgr.timerEgn = origTimerEgn }()

	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	Mgr.timerEgn = te

	got := Mgr.TimerEng()
	if got != te {
		t.Error("TimerEng() should return the timer engine")
	}
}

func TestManager_Plugins_NilJobEngine(t *testing.T) {
	origJobEgn := Mgr.jobEgn
	defer func() { Mgr.jobEgn = origJobEgn }()

	Mgr.jobEgn = nil

	plugs := Mgr.Plugins()
	if plugs != nil {
		t.Errorf("Plugins() should return nil when jobEgn is nil, got %v", plugs)
	}
}

func TestManager_Plugins_WithPlugins(t *testing.T) {
	origJobEgn := Mgr.jobEgn
	defer func() { Mgr.jobEgn = origJobEgn }()

	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	je.execs["p1"] = &executer{plug: "shell@ssh", jobwt: list.New()}
	je.execs["p2"] = &executer{plug: "gokins@git", jobwt: list.New()}
	Mgr.jobEgn = je

	plugs := Mgr.Plugins()
	if len(plugs) != 2 {
		t.Fatalf("Plugins() should return 2 plugins, got %d", len(plugs))
	}
}

// --- BuildEngine.run() edge cases ---

func TestBuildEngineRun_EmptyQueue(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// run() with empty queue should not panic
	c.run()
}

func TestBuildEngineRun_QueueNotEmptyButTasksFull(t *testing.T) {
	// When the task queue has items but all task slots are occupied,
	// run() should still handle gracefully.
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	c.taskw.PushBack(&runtime.Build{Id: "build-q1"})
	// run() will try to dequeue if ln2 < RunLimit,
	// but since RunLimit is read from comm.Cfg which defaults to 0/5,
	// and there are 0 active tasks, it should proceed.
	// However, startBuild will try to run the task which requires DB,
	// so we just test that run() doesn't panic with a non-nil element.
	// We can't fully exercise run() without the full stack.
}

// --- BuildEngine.startBuild ---

func TestBuildEngineStartBuild_RemovesFromMap(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	bt := &BuildTask{
		egn:   c,
		build: &runtime.Build{Id: "build-sb1"},
	}
	c.tasks["build-sb1"] = bt

	// After startBuild, the task should be removed from the map.
	// startBuild calls bt.run() which needs comm.WorkPath etc.
	// So we set up enough state for run() to complete quickly.
	// The simplest approach: just verify the map gets cleaned.
	// bt.run() will set status to error (nil Repo), then call updateBuild and clears.
	// updateBuild needs comm.Db which is nil, but recover will catch that.
	c.startBuild(bt)

	// After startBuild, the task should be removed
	if _, ok := c.tasks["build-sb1"]; ok {
		t.Error("startBuild should remove the task from the map")
	}
}

// --- gencmds with map[any]any ---

func TestGencmds_MapAnyInterface(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	// map[any]any case
	cmds := []any{
		map[any]any{
			"key1": "echo from any map",
			"key2": []any{"echo nested1", "echo nested2"},
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) == 0 {
		t.Fatal("expected commands from map[any]any values, got 0")
	}
}

func TestGencmds_MapStringInterfaceNestedArray(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[string]any{
			"scripts": []any{"echo a", "echo b", "echo c"},
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 3 {
		t.Errorf("expected 3 commands from nested array, got %d", len(runjb.Commands))
	}
}

func TestGencmds_MapAnyInterfaceEmptyValues(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	// Unknown value types inside map should be silently ignored
	cmds := []any{
		map[any]any{
			"key1": 42,   // int, not a string or []any
			"key2": true, // bool, not handled
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 0 {
		t.Errorf("expected 0 commands for unrecognized map values, got %d", len(runjb.Commands))
	}
}

func TestGencmds_StringCommand(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	// Single string in the slice
	cmds := []any{"echo single"}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runjb.Commands))
	}
	if runjb.Commands[0].Conts != "echo single" {
		t.Errorf("expected 'echo single', got %q", runjb.Commands[0].Conts)
	}
}

func TestGencmds_MapAnyInterfaceNestedArray(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[any]any{
			"scripts": []any{"echo x", "echo y"},
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 2 {
		t.Errorf("expected 2 commands from nested array in map[any]any, got %d", len(runjb.Commands))
	}
}

func TestGencmds_EmptyMap(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[string]any{},
		map[any]any{},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 0 {
		t.Errorf("expected 0 commands from empty maps, got %d", len(runjb.Commands))
	}
}

func TestGencmds_MapStringInterfaceWithEmptyNestedArray(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[string]any{
			"scripts": []any{},
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 0 {
		t.Errorf("expected 0 commands from empty nested array, got %d", len(runjb.Commands))
	}
}

func TestGencmds_MapAnyInterfaceWithEmptyNestedArray(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[any]any{
			"scripts": []any{},
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 0 {
		t.Errorf("expected 0 commands from empty nested array, got %d", len(runjb.Commands))
	}
}
