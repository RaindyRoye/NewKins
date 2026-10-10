package engine

import (
	"container/list"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
	"github.com/gokins/core/utils"
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

func TestJobEngineRmExec(t *testing.T) {
	je := newTestJobEngine()

	// Create an executer with some jobs
	job1 := &jobSync{
		step:  &runtime.Step{Step: "plugin1", Id: "job1"},
		cmdmp: make(map[string]*cmdSync),
	}
	job2 := &jobSync{
		step:  &runtime.Step{Step: "plugin1", Id: "job2"},
		cmdmp: make(map[string]*cmdSync),
	}

	ex := &executer{
		plug:  "plugin1",
		jobwt: list.New(),
	}
	ex.jobwt.PushBack(job1)
	ex.jobwt.PushBack(job2)

	je.execs["plugin1"] = ex

	// Call rmExec
	je.rmExec("plugin1", ex)

	// Verify executer was removed
	je.exelk.RLock()
	_, exists := je.execs["plugin1"]
	je.exelk.RUnlock()

	if exists {
		t.Error("expected executer to be removed")
	}

	// Verify jobs were marked as ended
	if !job1.ended {
		t.Error("expected job1 to be marked as ended")
	}
	if !job2.ended {
		t.Error("expected job2 to be marked as ended")
	}
}

func TestJobEngineRun_CleanupOldExecs(t *testing.T) {
	je := newTestJobEngine()
	je.tmr = utils.NewTimer(time.Second * 30)

	// Create an old executer (more than 2 minutes old)
	ex := &executer{
		plug:  "plugin1",
		tms:   time.Now().Add(-3 * time.Minute),
		jobwt: list.New(),
	}
	je.execs["plugin1"] = ex

	// Run cleanup
	je.run()

	// Give goroutines time to execute
	time.Sleep(100 * time.Millisecond)

	// Verify executer was removed
	je.exelk.RLock()
	_, exists := je.execs["plugin1"]
	je.exelk.RUnlock()

	if exists {
		t.Error("expected old executer to be removed")
	}
}

func TestJobEngineRun_CleanupEndedJobs(t *testing.T) {
	je := newTestJobEngine()
	je.tmr = utils.NewTimer(time.Second * 30)

	// Add an ended job
	job := &jobSync{
		step:  &runtime.Step{Step: "plugin1", Id: "job1"},
		cmdmp: make(map[string]*cmdSync),
		ended: true,
	}
	je.jobs["job1"] = job

	// Run cleanup
	je.run()

	// Verify ended job was removed
	je.joblk.RLock()
	_, exists := je.jobs["job1"]
	je.joblk.RUnlock()

	if exists {
		t.Error("expected ended job to be removed")
	}
}
