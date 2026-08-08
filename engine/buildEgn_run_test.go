package engine

import (
	"container/list"
	"sync"
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
)

// TestBuildEngineRun_DequeueAndStart verifies that run() dequeues a build
// from the task queue and adds it to the active tasks map when under RunLimit.
func TestBuildEngineRun_DequeueAndStart(t *testing.T) {
	comm.Cfg.Server.RunLimit = 5

	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bd := &runtime.Build{Id: "run-test-1"}
	egn.Put(bd)

	if egn.taskw.Len() != 1 {
		t.Fatalf("expected 1 queued task, got %d", egn.taskw.Len())
	}

	// run() should dequeue and start the build
	egn.run()

	// Queue should be empty now
	if egn.taskw.Len() != 0 {
		t.Errorf("expected empty queue after run(), got %d", egn.taskw.Len())
	}

	// Task should be in the active tasks map
	_, ok := egn.Get("run-test-1")
	if !ok {
		t.Error("expected build to be in active tasks after run()")
	}
}

// TestBuildEngineRun_QueueEmpty verifies run() handles empty queue gracefully.
func TestBuildEngineRun_QueueEmpty(t *testing.T) {
	comm.Cfg.Server.RunLimit = 5

	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Should not panic
	egn.run()
}

// TestBuildEngineRun_AtRunLimit verifies run() doesn't start new builds
// when the active task count equals RunLimit.
func TestBuildEngineRun_AtRunLimit(t *testing.T) {
	comm.Cfg.Server.RunLimit = 2

	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Fill tasks up to the limit
	for i := 0; i < 2; i++ {
		id := "existing-" + string(rune('a'+i))
		egn.tasks[id] = &BuildTask{build: &runtime.Build{Id: id}}
	}
	// Add a build to the queue
	egn.Put(&runtime.Build{Id: "waiting-build"})

	// run() should not start the queued build because we're at the limit
	egn.run()

	// The queued build should still be in the queue
	if egn.taskw.Len() != 1 {
		t.Errorf("expected 1 queued task (at limit), got %d", egn.taskw.Len())
	}
	_, ok := egn.Get("waiting-build")
	if ok {
		t.Error("waiting-build should not have been started when at limit")
	}
}

// TestBuildEngineRun_MultipleBuilds verifies run() dequeues one build per call.
func TestBuildEngineRun_MultipleBuilds(t *testing.T) {
	comm.Cfg.Server.RunLimit = 10

	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	for i := 0; i < 3; i++ {
		egn.Put(&runtime.Build{Id: "multi-" + string(rune('a'+i))})
	}
	if egn.taskw.Len() != 3 {
		t.Fatalf("expected 3 queued tasks, got %d", egn.taskw.Len())
	}

	// Each run() call should dequeue one build
	egn.run()
	if egn.taskw.Len() != 2 {
		t.Errorf("expected 2 queued tasks after first run(), got %d", egn.taskw.Len())
	}
	if len(egn.tasks) != 1 {
		t.Errorf("expected 1 active task, got %d", len(egn.tasks))
	}

	egn.run()
	if egn.taskw.Len() != 1 {
		t.Errorf("expected 1 queued task after second run(), got %d", egn.taskw.Len())
	}
	if len(egn.tasks) != 2 {
		t.Errorf("expected 2 active tasks, got %d", len(egn.tasks))
	}
}

// TestBuildEngineStartBuild tests that startBuild removes the task after run().
func TestBuildEngineStartBuild(t *testing.T) {
	comm.Cfg.Server.RunLimit = 5

	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Create a task that will complete immediately (run() with nil stages will finish quickly)
	bt := &BuildTask{
		egn:    egn,
		build:  &runtime.Build{Id: "start-test"},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	egn.tasks["start-test"] = bt

	// startBuild should call run() and then remove from tasks
	// Note: run() will try to call updateBuild which needs DB, but with recover it won't panic
	egn.startBuild(bt)

	// Task should be removed from active tasks
	_, ok := egn.Get("start-test")
	if ok {
		t.Error("expected task to be removed after startBuild")
	}
}

// TestBuildEngineGet_NilEngine tests Get on nil receiver.
func TestBuildEngineGet_NilEngine(t *testing.T) {
	var egn *BuildEngine
	_, ok := egn.Get("any")
	if ok {
		t.Error("Get on nil engine should return false")
	}
}

// TestBuildEngineConcurrentRunAndPut tests concurrent access to run() and Put().
func TestBuildEngineConcurrentRunAndPut(t *testing.T) {
	comm.Cfg.Server.RunLimit = 100

	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	var wg sync.WaitGroup

	// Concurrent Put
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			egn.Put(&runtime.Build{Id: "conc-build"})
		}(i)
	}

	// Concurrent run
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			egn.run()
		}()
	}

	wg.Wait()
	// Just verify no panics occurred
}

// TestBuildEngineRunLimitDefault verifies the default RunLimit behavior.
func TestBuildEngineRunLimitDefault(t *testing.T) {
	// Save and restore
	origLimit := comm.Cfg.Server.RunLimit
	defer func() { comm.Cfg.Server.RunLimit = origLimit }()

	comm.Cfg.Server.RunLimit = 0
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	// Put many builds
	for i := 0; i < 10; i++ {
		egn.Put(&runtime.Build{Id: "limit-test"})
	}

	// With RunLimit=0, run() should not start anything (ln2 < 0 is false)
	egn.run()
	// Queue should be unchanged since RunLimit is 0
	if egn.taskw.Len() != 10 {
		t.Errorf("expected 10 queued tasks (limit=0 means no starts), got %d", egn.taskw.Len())
	}
}
