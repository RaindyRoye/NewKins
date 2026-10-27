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

// TestJobEngineRunCleansEndedJobs verifies that run() removes ended jobs
// from the jobs map.
func TestJobEngineRunCleansEndedJobs(t *testing.T) {
	c := &JobEngine{
		tmr:   utils.NewTimer(time.Second * 30),
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}

	// Add an ended job
	c.jobs["job-1"] = &jobSync{ended: true}
	c.jobs["job-2"] = &jobSync{ended: false}

	// Force the timer to tick by using a zero-value timer
	c.tmr = utils.NewTimer(0)
	time.Sleep(time.Millisecond * 50) // let the timer expire
	c.run()

	// job-1 should be cleaned, job-2 should remain
	if _, ok := c.jobs["job-1"]; ok {
		t.Error("expected ended job-1 to be removed")
	}
	if _, ok := c.jobs["job-2"]; !ok {
		t.Error("expected non-ended job-2 to remain")
	}
}

// TestJobEngineRunRemExecExpired verifies that run() removes expired executers.
func TestJobEngineRunRemExecExpired(t *testing.T) {
	c := &JobEngine{
		tmr:   utils.NewTimer(0),
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}

	// Add an expired executer (tms more than 2 minutes ago)
	ex := &executer{
		plug:  "expired-plugin",
		tms:   time.Now().Add(-3 * time.Minute),
		jobwt: list.New(),
	}
	c.execs["expired-plugin"] = ex

	// Allow timer to tick
	time.Sleep(time.Millisecond * 50)
	c.run()

	// Give the goroutine time to complete rmExec
	time.Sleep(time.Millisecond * 200)

	c.exelk.RLock()
	_, stillExists := c.execs["expired-plugin"]
	c.exelk.RUnlock()
	if stillExists {
		t.Error("expected expired executer to be removed")
	}
}

// TestJobEngineRmExec marks jobs as ended when executer is removed.
func TestJobEngineRmExec(t *testing.T) {
	c := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}

	js := &jobSync{
		step:  &runtime.Step{Id: "step-1", Name: "test"},
		cmdmp: make(map[string]*cmdSync),
	}
	ex := &executer{
		plug:  "test-plugin",
		jobwt: list.New(),
	}
	ex.jobwt.PushBack(js)
	c.execs["test-plugin"] = ex

	c.rmExec("test-plugin", ex)

	// Executer should be removed
	c.exelk.RLock()
	_, exists := c.execs["test-plugin"]
	c.exelk.RUnlock()
	if exists {
		t.Error("expected executer to be removed after rmExec")
	}

	// Job should be marked as ended
	if !js.ended {
		t.Error("expected job to be marked as ended after rmExec")
	}
}

// TestJobEnginePullWithQueuedJob verifies Pull dequeues a job when available.
func TestJobEnginePullWithQueuedJob(t *testing.T) {
	c := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}

	rj := &runners.RunJob{
		Id:   "rj-1",
		Step: "plugin-x",
		Name: "test",
	}
	js := &jobSync{
		step:  &runtime.Step{Id: "step-1", Name: "test", Step: "plugin-x"},
		runjb: rj,
		cmdmp: make(map[string]*cmdSync),
	}
	ex := &executer{
		plug:  "plugin-x",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	ex.jobwt.PushBack(js)
	c.execs["plugin-x"] = ex

	result := c.Pull("runner-1", []string{"plugin-x"})
	if result == nil {
		t.Fatal("expected Pull to return a RunJob")
	}
	if result != rj {
		t.Error("expected Pull to return the correct RunJob")
	}

	// Verify job was moved from executer to jobs map
	c.joblk.RLock()
	_, inJobs := c.jobs["step-1"]
	c.joblk.RUnlock()
	if !inJobs {
		t.Error("expected job to be in jobs map after Pull")
	}

	// Verify executer queue is empty
	ex.Lock()
	ql := ex.jobwt.Len()
	ex.Unlock()
	if ql != 0 {
		t.Errorf("expected executer queue to be empty, got %d", ql)
	}
}

// TestJobEnginePullUpdatesExecuterTimestamp verifies that Pull refreshes
// the executer's tms to prevent premature removal.
func TestJobEnginePullUpdatesExecuterTimestamp(t *testing.T) {
	c := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}

	oldTime := time.Now().Add(-5 * time.Minute)
	ex := &executer{
		plug:  "plugin-y",
		tms:   oldTime,
		jobwt: list.New(),
	}
	c.execs["plugin-y"] = ex

	c.Pull("runner-1", []string{"plugin-y"})

	ex.RLock()
	updated := ex.tms
	ex.RUnlock()
	if updated.Equal(oldTime) {
		t.Error("expected Pull to update executer timestamp")
	}
	if time.Since(updated) > time.Second {
		t.Error("expected executer timestamp to be recent")
	}
}

// TestJobEnginePullMultiplePlugins verifies Pull iterates through plugins
// and returns the first available job.
func TestJobEnginePullMultiplePlugins(t *testing.T) {
	c := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}

	rj := &runners.RunJob{
		Id:   "rj-2",
		Step: "plugin-b",
		Name: "test",
	}
	js := &jobSync{
		step:  &runtime.Step{Id: "step-2", Name: "test", Step: "plugin-b"},
		runjb: rj,
		cmdmp: make(map[string]*cmdSync),
	}
	ex := &executer{
		plug:  "plugin-b",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	ex.jobwt.PushBack(js)
	c.execs["plugin-b"] = ex

	// Pull with multiple plugins, only "plugin-b" has a job
	result := c.Pull("runner-1", []string{"plugin-a", "plugin-b", "plugin-c"})
	if result == nil {
		t.Fatal("expected Pull to return a RunJob from plugin-b")
	}
	if result != rj {
		t.Error("expected Pull to return the correct RunJob from plugin-b")
	}
}

// TestJobEngineRmExecConcurrent verifies rmExec is safe under concurrency.
func TestJobEngineRmExecConcurrent(t *testing.T) {
	c := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		name := "plugin-concurrent-" + string(rune('a'+i))
		ex := &executer{
			plug:  name,
			tms:   time.Now().Add(-5 * time.Minute),
			jobwt: list.New(),
		}
		c.execs[name] = ex
		wg.Add(1)
		go func(n string, e *executer) {
			defer wg.Done()
			c.rmExec(n, e)
		}(name, ex)
	}
	wg.Wait()

	c.exelk.RLock()
	remaining := len(c.execs)
	c.exelk.RUnlock()
	if remaining != 0 {
		t.Errorf("expected all executers removed, got %d", remaining)
	}
}
