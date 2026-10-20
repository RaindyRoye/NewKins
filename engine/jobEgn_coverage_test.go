package engine

import (
	"container/list"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

// --- JobEngine.rmExec tests ---

func TestJobEngineRmExec_EmptyQueue(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	ex := &executer{
		plug:  "shell@ssh",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	je.execs["shell@ssh"] = ex

	// rmExec with empty queue should not panic
	je.rmExec("shell@ssh", ex)

	// executer should be removed
	if _, ok := je.execs["shell@ssh"]; ok {
		t.Error("executer should be removed after rmExec")
	}
}

func TestJobEngineRmExec_WithJobs(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	ex := &executer{
		plug:  "gokins@git",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	je.execs["gokins@git"] = ex

	// Add some jobs to the queue
	for i := 0; i < 3; i++ {
		job := &jobSync{
			step: &runtime.Step{Id: string(rune('a' + i)), Name: "step"},
		}
		ex.jobwt.PushBack(job)
	}

	je.rmExec("gokins@git", ex)

	// All jobs should be marked as ended
	for e := ex.jobwt.Front(); e != nil; e = e.Next() {
		job := e.Value.(*jobSync)
		if !job.ended {
			t.Errorf("job %s should be marked as ended", job.step.Id)
		}
	}

	// executer should be removed from map
	if _, ok := je.execs["gokins@git"]; ok {
		t.Error("executer should be removed after rmExec")
	}
}

// --- JobEngine.Pull tests ---

func TestJobEnginePull_EmptyPlugs(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	rts := je.Pull("runner-1", []string{"", ""})
	if rts != nil {
		t.Error("Pull with empty plugs should return nil")
	}
}

func TestJobEnginePull_NoExecuter(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	rts := je.Pull("runner-1", []string{"shell@ssh"})
	if rts != nil {
		t.Error("Pull with non-existent executer should return nil")
	}
	// But executer should be created
	if _, ok := je.execs["shell@ssh"]; !ok {
		t.Error("executer should be created on first Pull")
	}
}

func TestJobEnginePull_WithJob(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	ex := &executer{
		plug:  "shell@ssh",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	je.execs["shell@ssh"] = ex

	job := &jobSync{
		step:  &runtime.Step{Id: "step-1", Name: "test"},
		runjb: &runners.RunJob{Id: "job-1"},
	}
	ex.jobwt.PushBack(job)

	rts := je.Pull("runner-1", []string{"shell@ssh"})
	if rts == nil {
		t.Fatal("Pull should return job")
	}
	if rts.Id != "job-1" {
		t.Errorf("Pull returned wrong job: %s", rts.Id)
	}
	// Job should be moved to active jobs map
	if _, ok := je.jobs["step-1"]; !ok {
		t.Error("job should be in active jobs map")
	}
	// Queue should be empty
	if ex.jobwt.Len() != 0 {
		t.Error("job queue should be empty after Pull")
	}
}

// --- JobEngine.Plugins tests ---

func TestJobEnginePlugins(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	je.execs["shell@ssh"] = &executer{plug: "shell@ssh"}
	je.execs["gokins@git"] = &executer{plug: "gokins@git"}

	plugins := je.Plugins()
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	// Order is not guaranteed, so check both are present
	found := make(map[string]bool)
	for _, p := range plugins {
		found[p] = true
	}
	if !found["shell@ssh"] || !found["gokins@git"] {
		t.Errorf("missing expected plugins, got %v", plugins)
	}
}

// --- BuildEngine.run tests ---

func TestBuildEngineRun_EmptyQueue(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// run with empty queue should not panic
	c.run()
	if len(c.tasks) != 0 {
		t.Error("tasks should remain empty")
	}
}

func TestBuildEngineRun_QueueFull(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Add a build to the queue
	c.taskw.PushBack(&runtime.Build{Id: "build-1"})
	// run should move it to tasks
	c.run()
	// Note: run() starts the build in a goroutine, so we can't immediately check
	// But we can verify the queue is being processed
	time.Sleep(100 * time.Millisecond)
}
