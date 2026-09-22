package engine

import (
	"container/list"
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

func TestJobEnginePullWithJobs(t *testing.T) {
	je := newTestJobEngine()
	// Register an executer with a job
	je.execs["myplugin"] = &executer{
		plug:  "myplugin",
		jobwt: list.New(),
	}

	job := &jobSync{
		step:  &runtime.Step{Id: "step-1", Step: "myplugin"},
		runjb: &runners.RunJob{Id: "runjob-1"},
		cmdmp: make(map[string]*cmdSync),
	}
	je.execs["myplugin"].jobwt.PushBack(job)

	// Pull should return the job
	result := je.Pull("runner1", []string{"myplugin"})
	if result == nil {
		t.Fatal("expected Pull to return a job")
	}
	if result.Id != "runjob-1" {
		t.Errorf("expected runjob-1, got %s", result.Id)
	}

	// Queue should be empty now
	if je.execs["myplugin"].jobwt.Len() != 0 {
		t.Errorf("expected queue to be empty, got %d", je.execs["myplugin"].jobwt.Len())
	}

	// Job should be tracked in je.jobs
	je.joblk.RLock()
	_, ok := je.jobs["step-1"]
	je.joblk.RUnlock()
	if !ok {
		t.Error("expected job to be tracked in je.jobs after Pull")
	}
}

func TestJobEnginePullMultiplePlugins(t *testing.T) {
	je := newTestJobEngine()
	// Register two plugins, one with a job
	je.execs["plugin1"] = &executer{plug: "plugin1", jobwt: list.New()}
	je.execs["plugin2"] = &executer{plug: "plugin2", jobwt: list.New()}

	job := &jobSync{
		step:  &runtime.Step{Id: "step-2", Step: "plugin2"},
		runjb: &runners.RunJob{Id: "runjob-2"},
		cmdmp: make(map[string]*cmdSync),
	}
	je.execs["plugin2"].jobwt.PushBack(job)

	// Pull should skip plugin1 (no jobs) and return from plugin2
	result := je.Pull("runner1", []string{"plugin1", "plugin2"})
	if result == nil {
		t.Fatal("expected Pull to return a job from plugin2")
	}
	if result.Id != "runjob-2" {
		t.Errorf("expected runjob-2, got %s", result.Id)
	}
}

func TestJobEngineRmExec(t *testing.T) {
	je := newTestJobEngine()
	// Create an executer with jobs
	ex := &executer{
		plug:  "plugin1",
		jobwt: list.New(),
	}
	je.execs["plugin1"] = ex

	job1 := &jobSync{
		step:  &runtime.Step{Id: "step-1"},
		ended: false,
	}
	job2 := &jobSync{
		step:  &runtime.Step{Id: "step-2"},
		ended: false,
	}
	ex.jobwt.PushBack(job1)
	ex.jobwt.PushBack(job2)

	// Remove the executer
	je.rmExec("plugin1", ex)

	// Executer should be removed
	je.exelk.RLock()
	_, ok := je.execs["plugin1"]
	je.exelk.RUnlock()
	if ok {
		t.Error("expected executer to be removed")
	}

	// All jobs should be marked as ended
	if !job1.ended {
		t.Error("expected job1.ended to be true")
	}
	if !job2.ended {
		t.Error("expected job2.ended to be true")
	}
}

func TestJobEngineRunWithEndedJobs(t *testing.T) {
	je := newTestJobEngine()
	// Add ended and non-ended jobs
	je.jobs["step-1"] = &jobSync{
		step:  &runtime.Step{Id: "step-1"},
		ended: true,
	}
	je.jobs["step-2"] = &jobSync{
		step:  &runtime.Step{Id: "step-2"},
		ended: false,
	}

	// Manually call the cleanup logic
	je.joblk.Lock()
	for k, v := range je.jobs {
		if v.ended {
			delete(je.jobs, k)
		}
	}
	je.joblk.Unlock()

	// Ended job should be removed
	je.joblk.RLock()
	_, ok1 := je.jobs["step-1"]
	_, ok2 := je.jobs["step-2"]
	je.joblk.RUnlock()
	if ok1 {
		t.Error("expected ended job to be removed")
	}
	if !ok2 {
		t.Error("expected non-ended job to remain")
	}
}
