package engine

import (
	"container/list"
	"sync"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
	"github.com/gokins/core/utils"
	"github.com/gokins/runner/runners"
)

func TestJobEngineRmExecMarksEnded(t *testing.T) {
	je := newTestJobEngine()

	// Create an executer with some jobs
	ex := &executer{
		plug:  "test-plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	job1 := &jobSync{
		step:  &runtime.Step{Step: "test-plugin"},
		cmdmp: make(map[string]*cmdSync),
	}
	job2 := &jobSync{
		step:  &runtime.Step{Step: "test-plugin"},
		cmdmp: make(map[string]*cmdSync),
	}
	ex.jobwt.PushBack(job1)
	ex.jobwt.PushBack(job2)
	je.execs["test-plugin"] = ex

	// rmExec should mark all jobs as ended and remove the executer
	je.rmExec("test-plugin", ex)

	// Verify jobs are marked as ended
	if !job1.ended {
		t.Error("expected job1 to be marked as ended")
	}
	if !job2.ended {
		t.Error("expected job2 to be marked as ended")
	}

	// Verify executer is removed
	je.exelk.RLock()
	_, ok := je.execs["test-plugin"]
	je.exelk.RUnlock()
	if ok {
		t.Error("expected executer to be removed from execs map")
	}
}

func TestJobEnginePullDequeuesJob(t *testing.T) {
	je := newTestJobEngine()

	// Create an executer with a job in the queue
	ex := &executer{
		plug:  "test-plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	job := &jobSync{
		step:  &runtime.Step{Step: "test-plugin", Id: "step-1"},
		cmdmp: make(map[string]*cmdSync),
		runjb: &runners.RunJob{Name: "job-1"},
	}
	ex.jobwt.PushBack(job)
	je.execs["test-plugin"] = ex

	// Pull should dequeue the job and return it
	result := je.Pull("runner1", []string{"test-plugin"})
	if result == nil {
		t.Fatal("expected Pull to return a job")
	}
	if result.Name != "job-1" {
		t.Errorf("expected job name 'job-1', got %q", result.Name)
	}

	// Verify job was removed from the queue
	if ex.jobwt.Len() != 0 {
		t.Errorf("expected empty queue after Pull, got %d items", ex.jobwt.Len())
	}

	// Verify job was added to the jobs map
	je.joblk.RLock()
	_, ok := je.jobs["step-1"]
	je.joblk.RUnlock()
	if !ok {
		t.Error("expected job to be added to jobs map")
	}
}

func TestJobEnginePullEmptyQueue(t *testing.T) {
	je := newTestJobEngine()

	// Create an executer with an empty queue
	ex := &executer{
		plug:  "test-plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	je.execs["test-plugin"] = ex

	// Pull should return nil for an empty queue
	result := je.Pull("runner1", []string{"test-plugin"})
	if result != nil {
		t.Error("expected nil result for Pull from empty queue")
	}
}

func TestJobEngineRunCleansEndedJobs(t *testing.T) {
	je := &JobEngine{
		tmr:    utils.NewTimer(time.Millisecond * 10),
		execs:  make(map[string]*executer),
		jobs:   make(map[string]*jobSync),
		wakeCh: make(chan struct{}, 1),
	}

	// Add an ended job
	job := &jobSync{
		step:  &runtime.Step{Step: "test-plugin", Id: "step-1"},
		cmdmp: make(map[string]*cmdSync),
		ended: true,
	}
	je.jobs["step-1"] = job

	// run should clean up ended jobs
	je.run()

	je.joblk.RLock()
	_, ok := je.jobs["step-1"]
	je.joblk.RUnlock()
	if ok {
		t.Error("expected ended job to be removed from jobs map")
	}
}

func TestJobEngineRunRemovesStaleExec(t *testing.T) {
	je := &JobEngine{
		tmr:    utils.NewTimer(time.Millisecond * 10),
		execs:  make(map[string]*executer),
		jobs:   make(map[string]*jobSync),
		wakeCh: make(chan struct{}, 1),
	}

	// Add an executer that has been idle for more than 2 minutes
	ex := &executer{
		plug:  "stale-plugin",
		tms:   time.Now().Add(-3 * time.Minute),
		jobwt: list.New(),
	}
	je.execs["stale-plugin"] = ex

	// run should trigger rmExec for stale executers
	je.run()

	// Give rmExec goroutine time to complete
	time.Sleep(50 * time.Millisecond)

	je.exelk.RLock()
	_, ok := je.execs["stale-plugin"]
	je.exelk.RUnlock()
	if ok {
		t.Error("expected stale executer to be removed")
	}
}

func TestJobEngineConcurrentPullAndPut(t *testing.T) {
	je := &JobEngine{
		execs:  make(map[string]*executer),
		jobs:   make(map[string]*jobSync),
		wakeCh: make(chan struct{}, 1),
	}

	// Pre-register an executer
	je.execs["plugin"] = &executer{
		plug:  "plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}

	var wg sync.WaitGroup

	// Concurrent Put
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			job := &jobSync{
				step:  &runtime.Step{Step: "plugin", Id: "step-" + string(rune('a'+n%26))},
				cmdmp: make(map[string]*cmdSync),
				runjb: &runners.RunJob{Name: "job"},
			}
			_ = je.Put(job)
		}(i)
	}

	// Concurrent Pull
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = je.Pull("runner", []string{"plugin"})
		}()
	}

	wg.Wait()
}
