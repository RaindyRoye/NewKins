package engine

import (
	"container/list"
	"sync"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/model"
	"github.com/gokins/runner/runners"
)

// --- BuildEngine Stop tests ---

func TestBuildEngineStop(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Add some tasks
	for i := 0; i < 5; i++ {
		id := "task-" + string(rune('a'+i))
		c.tasks[id] = &BuildTask{
			build: &runtime.Build{Id: id},
		}
	}
	// Stop should not panic
	c.Stop()
}

func TestBuildEngineStopEmpty(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Stop on empty engine should not panic
	c.Stop()
}

// --- JobEngine Pull tests ---

func TestJobEnginePullWithExistingExecuter(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	// Create executer with a job in queue
	ex := &executer{
		plug:  "plugin1",
		jobwt: list.New(),
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-1", Step: "plugin1"},
		cmdmp: make(map[string]*cmdSync),
		runjb: &runners.RunJob{Id: "job-1"},
	}
	ex.jobwt.PushBack(job)
	je.execs["plugin1"] = ex

	// Pull should return the job
	result := je.Pull("runner1", []string{"plugin1"})
	if result == nil {
		t.Fatal("expected Pull to return a job")
	}
	if result.Id != "job-1" {
		t.Errorf("expected job Id 'job-1', got %q", result.Id)
	}
}

func TestJobEnginePullMultiplePlugins(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	// Create two executeres, one with a job
	ex1 := &executer{plug: "plugin1", jobwt: list.New()}
	ex2 := &executer{plug: "plugin2", jobwt: list.New()}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-1", Step: "plugin2"},
		cmdmp: make(map[string]*cmdSync),
		runjb: &runners.RunJob{Id: "job-2"},
	}
	ex2.jobwt.PushBack(job)
	je.execs["plugin1"] = ex1
	je.execs["plugin2"] = ex2

	// Pull should return job from plugin2
	result := je.Pull("runner1", []string{"plugin1", "plugin2"})
	if result == nil {
		t.Fatal("expected Pull to return a job")
	}
	if result.Id != "job-2" {
		t.Errorf("expected job Id 'job-2', got %q", result.Id)
	}
}

func TestJobEnginePullNoJobs(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	// Create executer with no jobs
	ex := &executer{plug: "plugin1", jobwt: list.New()}
	je.execs["plugin1"] = ex

	// Pull should return nil
	result := je.Pull("runner1", []string{"plugin1"})
	if result != nil {
		t.Error("expected Pull to return nil when no jobs available")
	}
}

// --- Concurrent tests ---

func TestJobEngineConcurrentPullAndPut(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	je.execs["p1"] = &executer{plug: "p1", jobwt: list.New()}

	var wg sync.WaitGroup
	// Concurrent Put
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job := &jobSync{
				step:  &runtime.Step{Step: "p1"},
				cmdmp: make(map[string]*cmdSync),
			}
			_ = je.Put(job)
		}()
	}
	// Concurrent Pull
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = je.Pull("runner1", []string{"p1"})
		}()
	}
	wg.Wait()
}

func TestBuildEngineConcurrentStop(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Add tasks
	for i := 0; i < 20; i++ {
		id := "task-" + string(rune('a'+i))
		c.tasks[id] = &BuildTask{
			build: &runtime.Build{Id: id},
		}
	}

	var wg sync.WaitGroup
	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Get("task-a")
		}()
	}
	// Concurrent Stop
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Stop()
		}()
	}
	wg.Wait()
}

// --- TimerEngine execItem tests ---

func TestTimerEngineExecItem_FutureTick(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	// Create a timer with tick in the future
	task := &timerExec{
		tt:   &model.TTrigger{Id: "future", Name: "future"},
		typ:  1,
		tick: time.Now().Add(time.Hour), // 1 hour in future
	}
	// execItem should not execute because tick is in future
	te.execItem(task)
	// No error expected, tick should remain unchanged
	if task.typ != 1 {
		t.Errorf("expected type unchanged, got %d", task.typ)
	}
}

func TestTimerEngineRun_Empty(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	// run() on empty tasks should not panic
	te.run()
}

func TestTimerEngineRun_WithFutureTasks(t *testing.T) {
	te := &TimerEngine{tasks: make(map[string]*timerExec)}
	// Add tasks with future ticks
	for i := 0; i < 5; i++ {
		id := "task-" + string(rune('a'+i))
		te.tasks[id] = &timerExec{
			tt:   &model.TTrigger{Id: id, Name: id},
			typ:  1,
			tick: time.Now().Add(time.Hour),
		}
	}
	// run() should not execute any tasks
	te.run()
}

// --- BuildTask Show tests ---

func TestBuildTaskShow_NilCtx(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "b1",
			PipelineId: "p1",
			Status:     common.BuildStatusRunning,
		},
		stages: make(map[string]*taskStage),
	}
	// With nil ctx, stopd() returns true, so Show should return nil
	result, ok := task.Show()
	if ok {
		t.Error("expected Show to return false when ctx is nil")
	}
	if result != nil {
		t.Error("expected Show to return nil when ctx is nil")
	}
}

// --- Manager tests ---

func TestManagerAccessors(t *testing.T) {
	mgr := &Manager{}
	// Accessors should return nil when not initialized
	if mgr.BuildEgn() != nil {
		t.Error("expected BuildEgn to be nil")
	}
	if mgr.HRun() != nil {
		t.Error("expected HRun to be nil")
	}
	if mgr.TimerEng() != nil {
		t.Error("expected TimerEng to be nil")
	}
}

func TestManagerPlugins(t *testing.T) {
	mgr := &Manager{
		jobEgn: &JobEngine{
			execs: make(map[string]*executer),
			jobs:  make(map[string]*jobSync),
		},
	}
	plugins := mgr.Plugins()
	if len(plugins) != 0 {
		t.Errorf("expected empty plugins, got %v", plugins)
	}
}
