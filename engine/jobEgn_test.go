package engine

import (
	"container/list"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
	"github.com/gokins/core/utils"
	"github.com/gokins/runner/runners"
)

func newTestJobEngine() *JobEngine {
	return &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
}

func TestJobEnginePutNilJob(t *testing.T) {
	je := newTestJobEngine()
	err := je.Put(nil)
	if err == nil {
		t.Fatal("expected error for nil job, got nil")
	}
	if !errors.Is(err, ErrEmptyParams) {
		t.Errorf("expected error to wrap ErrEmptyParams, got: %v", err)
	}
}

func TestJobEnginePutEmptyStep(t *testing.T) {
	je := newTestJobEngine()
	job := &jobSync{
		step: &runtime.Step{Step: ""},
	}
	err := je.Put(job)
	if err == nil {
		t.Fatal("expected error for empty step plugin, got nil")
	}
	if !errors.Is(err, ErrEmptyParams) {
		t.Errorf("expected error to wrap ErrEmptyParams, got: %v", err)
	}
}

func TestJobEnginePutNoExecuter(t *testing.T) {
	je := newTestJobEngine()
	job := &jobSync{
		step: &runtime.Step{Step: "myplugin"},
	}
	err := je.Put(job)
	if err == nil {
		t.Fatal("expected error when no executer registered for plugin")
	}
	if !errors.Is(err, ErrPluginNotFound) {
		t.Errorf("expected error to wrap ErrPluginNotFound, got: %v", err)
	}
}

func TestJobEnginePutSuccess(t *testing.T) {
	je := newTestJobEngine()
	// Register an executer for "myplugin"
	je.execs["myplugin"] = &executer{
		plug:  "myplugin",
		jobwt: list.New(),
	}

	job := &jobSync{
		step:  &runtime.Step{Step: "myplugin"},
		cmdmp: make(map[string]*cmdSync),
	}
	err := je.Put(job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify job was added to the queue
	ex := je.execs["myplugin"]
	if ex.jobwt.Len() != 1 {
		t.Fatalf("expected 1 job in queue, got %d", ex.jobwt.Len())
	}
}

func TestJobEnginePluginsEmpty(t *testing.T) {
	je := newTestJobEngine()
	plugs := je.Plugins()
	if len(plugs) != 0 {
		t.Errorf("expected empty plugins, got %v", plugs)
	}
}

func TestJobEnginePluginsRegistered(t *testing.T) {
	je := newTestJobEngine()
	je.execs["plugin-a"] = &executer{plug: "plugin-a", jobwt: list.New()}
	je.execs["plugin-b"] = &executer{plug: "plugin-b", jobwt: list.New()}

	plugs := je.Plugins()
	if len(plugs) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugs))
	}
	// Check both are present (order not guaranteed)
	found := map[string]bool{}
	for _, p := range plugs {
		found[p] = true
	}
	if !found["plugin-a"] || !found["plugin-b"] {
		t.Errorf("expected plugin-a and plugin-b, got %v", plugs)
	}
}

func TestJobEnginePullNoPlugins(t *testing.T) {
	je := newTestJobEngine()
	result := je.Pull("runner1", nil)
	if result != nil {
		t.Error("expected nil result for Pull with no plugins")
	}
}

func TestJobEnginePullEmptyPluginName(t *testing.T) {
	je := newTestJobEngine()
	result := je.Pull("runner1", []string{"", ""})
	if result != nil {
		t.Error("expected nil result for Pull with empty plugin names")
	}
}

func TestJobEnginePullCreatesExecuter(t *testing.T) {
	je := newTestJobEngine()
	// Pull should create a new executer for unknown plugins
	je.Pull("runner1", []string{"newplugin"})
	je.exelk.RLock()
	_, ok := je.execs["newplugin"]
	je.exelk.RUnlock()
	if !ok {
		t.Error("expected Pull to create executer for new plugin")
	}
}

func TestJobEnginePullReturnsJob(t *testing.T) {
	je := newTestJobEngine()
	// Register an executer with a queued job
	job := &jobSync{
		step:  &runtime.Step{Id: "step-1", Step: "shell"},
		runjb: &runners.RunJob{Id: "runjob-1"},
		cmdmp: make(map[string]*cmdSync),
	}
	je.execs["shell"] = &executer{
		plug:  "shell",
		jobwt: list.New(),
	}
	je.execs["shell"].jobwt.PushBack(job)

	// Pull should return the queued job
	result := je.Pull("runner1", []string{"shell"})
	if result == nil {
		t.Fatal("expected Pull to return a job")
	}
	if result.Id != "runjob-1" {
		t.Errorf("expected job id 'runjob-1', got %q", result.Id)
	}
	// Job should be removed from queue
	if je.execs["shell"].jobwt.Len() != 0 {
		t.Error("expected job to be removed from queue")
	}
	// Job should be tracked in je.jobs
	je.joblk.RLock()
	_, ok := je.jobs["step-1"]
	je.joblk.RUnlock()
	if !ok {
		t.Error("expected job to be tracked in je.jobs")
	}
}

func TestJobEnginePullEmptyQueue(t *testing.T) {
	je := newTestJobEngine()
	je.execs["shell"] = &executer{
		plug:  "shell",
		jobwt: list.New(), // empty queue
	}
	result := je.Pull("runner1", []string{"shell"})
	if result != nil {
		t.Error("expected nil result for Pull with empty queue")
	}
}

func TestJobEngineRmExec(t *testing.T) {
	je := newTestJobEngine()
	// Add an executer with queued jobs
	job1 := &jobSync{step: &runtime.Step{Id: "step-1"}, cmdmp: make(map[string]*cmdSync)}
	job2 := &jobSync{step: &runtime.Step{Id: "step-2"}, cmdmp: make(map[string]*cmdSync)}
	ex := &executer{
		plug:  "shell",
		jobwt: list.New(),
	}
	ex.jobwt.PushBack(job1)
	ex.jobwt.PushBack(job2)
	je.execs["shell"] = ex

	// Remove the executer
	je.rmExec("shell", ex)

	// Executer should be removed
	je.exelk.RLock()
	_, ok := je.execs["shell"]
	je.exelk.RUnlock()
	if ok {
		t.Error("expected executer to be removed")
	}

	// All queued jobs should be marked as ended
	if !job1.ended {
		t.Error("expected job1 to be marked as ended")
	}
	if !job2.ended {
		t.Error("expected job2 to be marked as ended")
	}
}

func TestJobEngineRun_Cleanup(t *testing.T) {
	je := &JobEngine{
		tmr:   utils.NewTimer(time.Second * 30),
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	// Add an old executer (older than 2 minutes)
	oldTime := time.Now().Add(-3 * time.Minute)
	je.execs["old-plugin"] = &executer{
		plug:  "old-plugin",
		tms:   oldTime,
		jobwt: list.New(),
	}
	// Add a recent executer
	je.execs["new-plugin"] = &executer{
		plug:  "new-plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}

	// Run should clean up old executer
	je.run()

	// Wait a bit for goroutine to complete
	time.Sleep(100 * time.Millisecond)

	je.exelk.RLock()
	_, oldExists := je.execs["old-plugin"]
	_, newExists := je.execs["new-plugin"]
	je.exelk.RUnlock()

	if oldExists {
		t.Error("expected old executer to be removed")
	}
	if !newExists {
		t.Error("expected new executer to remain")
	}
}

func TestJobEngineRun_CleanupEndedJobs(t *testing.T) {
	je := &JobEngine{
		tmr:   utils.NewTimer(time.Second * 30),
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	// Add some jobs, some ended and some not
	je.jobs["step-1"] = &jobSync{step: &runtime.Step{Id: "step-1"}, ended: true}
	je.jobs["step-2"] = &jobSync{step: &runtime.Step{Id: "step-2"}, ended: false}

	// Run should clean up ended jobs
	je.run()

	// Wait a bit for goroutine to complete
	time.Sleep(100 * time.Millisecond)

	je.joblk.RLock()
	_, endedExists := je.jobs["step-1"]
	_, activeExists := je.jobs["step-2"]
	je.joblk.RUnlock()

	if endedExists {
		t.Error("expected ended job to be removed")
	}
	if !activeExists {
		t.Error("expected active job to remain")
	}
}

func TestJobEngineConcurrentPutAndPlugins(t *testing.T) {
	je := newTestJobEngine()
	je.execs["p1"] = &executer{plug: "p1", jobwt: list.New()}
	je.execs["p2"] = &executer{plug: "p2", jobwt: list.New()}

	var wg sync.WaitGroup
	// Concurrent Put
	for i := 0; i < 20; i++ {
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
	// Concurrent Plugins reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = je.Plugins()
		}()
	}
	wg.Wait()

	// Verify all jobs were added
	je.exelk.RLock()
	cnt := je.execs["p1"].jobwt.Len()
	je.exelk.RUnlock()
	if cnt != 20 {
		t.Errorf("expected 20 jobs in p1 queue, got %d", cnt)
	}
}
