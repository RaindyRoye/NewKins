package engine

import (
	"container/list"
	"sync"
	"testing"

	"github.com/gokins/core/runtime"
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
	if err.Error() != "step plugin empty" {
		t.Errorf("error = %q, want %q", err.Error(), "step plugin empty")
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
	if err.Error() != "step plugin empty" {
		t.Errorf("error = %q, want %q", err.Error(), "step plugin empty")
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
	if err.Error() != "plugin not found: myplugin" {
		t.Errorf("error = %q, want %q", err.Error(), "plugin not found: myplugin")
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
