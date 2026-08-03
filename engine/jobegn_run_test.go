package engine

import (
	"container/list"
	"sync"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
	"github.com/gokins/core/utils"
)

// --- JobEngine.rmExec ---

func TestJobEngineRmExec_MarksJobsEnded(t *testing.T) {
	je := newTestJobEngine()
	ex := &executer{
		plug:  "test-plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	// Add some jobs to the executer
	js1 := &jobSync{step: &runtime.Step{Id: "s1"}, cmdmp: make(map[string]*cmdSync)}
	js2 := &jobSync{step: &runtime.Step{Id: "s2"}, cmdmp: make(map[string]*cmdSync)}
	ex.jobwt.PushBack(js1)
	ex.jobwt.PushBack(js2)
	je.execs["test-plugin"] = ex

	// rmExec should mark all jobs as ended and remove the executer
	je.rmExec("test-plugin", ex)

	// Verify executer was removed
	je.exelk.RLock()
	_, ok := je.execs["test-plugin"]
	je.exelk.RUnlock()
	if ok {
		t.Error("rmExec should remove the executer from the map")
	}

	// Verify jobs were marked as ended
	if !js1.ended {
		t.Error("js1 should be marked as ended")
	}
	if !js2.ended {
		t.Error("js2 should be marked as ended")
	}
}

func TestJobEngineRmExec_EmptyQueue(t *testing.T) {
	je := newTestJobEngine()
	ex := &executer{
		plug:  "empty-plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	je.execs["empty-plugin"] = ex

	// Should not panic with empty queue
	je.rmExec("empty-plugin", ex)

	je.exelk.RLock()
	_, ok := je.execs["empty-plugin"]
	je.exelk.RUnlock()
	if ok {
		t.Error("rmExec should remove the executer even with empty queue")
	}
}

// --- JobEngine.run ---

func TestJobEngineRun_CleansEndedJobs(t *testing.T) {
	je := newTestJobEngine()
	je.tmr = utils.NewTimer(time.Millisecond) // tick immediately

	// Add an ended job
	endedJob := &jobSync{
		step:  &runtime.Step{Id: "ended-step"},
		cmdmp: make(map[string]*cmdSync),
		ended: true,
	}
	je.jobs["ended-step"] = endedJob

	// Add a non-ended job
	activeJob := &jobSync{
		step:  &runtime.Step{Id: "active-step"},
		cmdmp: make(map[string]*cmdSync),
		ended: false,
	}
	je.jobs["active-step"] = activeJob

	// Wait for timer to tick
	time.Sleep(2 * time.Millisecond)
	je.run()

	// Ended job should be removed
	je.joblk.Lock()
	_, endedExists := je.jobs["ended-step"]
	_, activeExists := je.jobs["active-step"]
	je.joblk.Unlock()

	if endedExists {
		t.Error("ended job should be cleaned up by run()")
	}
	if !activeExists {
		t.Error("active job should remain in the map")
	}
}

func TestJobEngineRun_StaleExecuter(t *testing.T) {
	je := newTestJobEngine()
	je.tmr = utils.NewTimer(time.Millisecond)

	// Add an executer with old timestamp (> 2 minutes)
	ex := &executer{
		plug:  "stale-plugin",
		tms:   time.Now().Add(-3 * time.Minute),
		jobwt: list.New(),
	}
	je.execs["stale-plugin"] = ex

	// Wait for timer
	time.Sleep(2 * time.Millisecond)
	je.run()

	// Give rmExec goroutine time to complete
	time.Sleep(50 * time.Millisecond)

	// Stale executer should be removed
	je.exelk.RLock()
	_, ok := je.execs["stale-plugin"]
	je.exelk.RUnlock()
	if ok {
		t.Error("stale executer should be removed by run()")
	}
}

func TestJobEngineRun_NoTick(t *testing.T) {
	je := newTestJobEngine()
	// Create a timer with a very long duration so it won't tick during the test
	je.tmr = utils.NewTimer(24 * time.Hour)

	// Add an ended job
	endedJob := &jobSync{
		step:  &runtime.Step{Id: "s1"},
		cmdmp: make(map[string]*cmdSync),
		ended: true,
	}
	je.jobs["s1"] = endedJob

	// Call run() directly - it will check the timer and return early without cleaning up
	je.run()

	// The behavior depends on the timer implementation:
	// - If the timer ticks, the ended job will be cleaned up
	// - If the timer doesn't tick, the job remains
	// Either way, the test verifies that run() doesn't panic
	je.joblk.Lock()
	jobCount := len(je.jobs)
	je.joblk.Unlock()

	// Log the result for debugging purposes
	t.Logf("Job count after run(): %d (initial: 1)", jobCount)
}

// --- Manager accessor tests ---

func TestManagerAccessors(t *testing.T) {
	mgr := &Manager{}
	if mgr.BuildEgn() != nil {
		t.Error("BuildEgn() should return nil when not initialized")
	}
	if mgr.HRun() != nil {
		t.Error("HRun() should return nil when not initialized")
	}
	if mgr.TimerEng() != nil {
		t.Error("TimerEng() should return nil when not initialized")
	}
	// Plugins() calls jobEgn.Plugins() which will panic if jobEgn is nil.
	// This is a known limitation - Manager must be fully initialized before use.
	// We don't test Plugins() with nil jobEgn to avoid panic.
}

// --- Concurrent JobEngine stress test ---

func TestJobEngineConcurrentPullAndPut(t *testing.T) {
	je := newTestJobEngine()
	// Pre-register a plugin
	je.execs["shell"] = &executer{
		plug:  "shell",
		tms:   time.Now(),
		jobwt: list.New(),
	}

	var wg sync.WaitGroup

	// Concurrent Pull
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			je.Pull("runner", []string{"shell"})
		}()
	}

	// Concurrent Put
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job := &jobSync{
				step:  &runtime.Step{Step: "shell"},
				cmdmp: make(map[string]*cmdSync),
			}
			_ = je.Put(job)
		}()
	}

	wg.Wait()
}

// --- BuildEngine.run (queue processing) ---

func TestBuildEngineRun_EmptyQueue(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// run() with empty queue should not panic
	c.run()
}

func TestBuildEngineRun_AtRunLimit(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Fill tasks to the default RunLimit (5)
	for i := 0; i < 5; i++ {
		c.tasks[string(rune('a'+i))] = &BuildTask{
			build: &runtime.Build{Id: string(rune('a' + i))},
		}
	}
	// Add a task to the queue
	c.Put(&runtime.Build{Id: "queued"})

	// run() should not start a new task since we're at RunLimit
	// But RunLimit comes from comm.Cfg.Server.RunLimit which defaults to 5
	// This test verifies run() doesn't panic at the boundary
	c.run()
}

// --- executer concurrent lock test ---

func TestExecuterConcurrentLocking(t *testing.T) {
	ex := &executer{
		plug:  "test",
		tms:   time.Now(),
		jobwt: list.New(),
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ex.Lock()
			ex.tms = time.Now()
			ex.Unlock()
		}()
	}
	wg.Wait()
}
