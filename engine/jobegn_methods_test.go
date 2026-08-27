package engine

import (
	"container/list"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

func TestJobEnginePut_EmptyStep(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	job := &jobSync{
		step: &runtime.Step{Id: "step-1", Step: ""},
	}
	err := je.Put(job)
	if err == nil {
		t.Fatal("expected error for empty step plugin")
	}
}

func TestJobEnginePut_NilJob(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	err := je.Put(nil)
	if err == nil {
		t.Fatal("expected error for nil job")
	}
}

func TestJobEnginePut_PluginNotFound(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	job := &jobSync{
		step: &runtime.Step{Id: "step-1", Step: "nonexistent-plugin"},
	}
	err := je.Put(job)
	if err == nil {
		t.Fatal("expected error for non-existent plugin")
	}
}

func TestJobEnginePut_Success(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	// Pre-register a plugin
	je.execs["test-plugin"] = &executer{
		plug:  "test-plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	job := &jobSync{
		step: &runtime.Step{Id: "step-1", Step: "test-plugin"},
	}
	err := je.Put(job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify job was added to the plugin's queue
	ex := je.execs["test-plugin"]
	if ex.jobwt.Len() != 1 {
		t.Fatalf("expected 1 job in queue, got %d", ex.jobwt.Len())
	}
}

func TestJobEnginePull_EmptyPluginList(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	result := je.Pull("runner-1", []string{})
	if result != nil {
		t.Fatal("expected nil for empty plugin list")
	}
}

func TestJobEnginePull_EmptyPluginName(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	result := je.Pull("runner-1", []string{"", ""})
	if result != nil {
		t.Fatal("expected nil for empty plugin names")
	}
}

func TestJobEnginePull_NoJobs(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	result := je.Pull("runner-1", []string{"test-plugin"})
	if result != nil {
		t.Fatal("expected nil when no jobs available")
	}
	// Verify plugin was auto-registered
	if _, ok := je.execs["test-plugin"]; !ok {
		t.Error("expected plugin to be auto-registered")
	}
}

func TestJobEnginePull_WithJob(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	// Pre-register a plugin with a job
	ex := &executer{
		plug:  "test-plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-1", Step: "test-plugin"},
		runjb: &runners.RunJob{Id: "runjob-1"},
	}
	ex.jobwt.PushBack(job)
	je.execs["test-plugin"] = ex

	result := je.Pull("runner-1", []string{"test-plugin"})
	if result == nil {
		t.Fatal("expected job to be pulled")
	}
	if result.Id != "runjob-1" {
		t.Errorf("expected runjob id 'runjob-1', got %q", result.Id)
	}
	// Verify job was removed from queue
	if ex.jobwt.Len() != 0 {
		t.Errorf("expected empty queue after pull, got %d", ex.jobwt.Len())
	}
	// Verify job was added to jobs map
	if _, ok := je.jobs["step-1"]; !ok {
		t.Error("expected job to be added to jobs map")
	}
}

func TestJobEnginePlugins_Empty(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	plugins := je.Plugins()
	if len(plugins) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestJobEnginePlugins_WithPlugins(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	je.execs["plugin1"] = &executer{plug: "plugin1", jobwt: list.New()}
	je.execs["plugin2"] = &executer{plug: "plugin2", jobwt: list.New()}

	plugins := je.Plugins()
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	// Order is not guaranteed, just check both are present
	found := make(map[string]bool)
	for _, p := range plugins {
		found[p] = true
	}
	if !found["plugin1"] || !found["plugin2"] {
		t.Errorf("expected both plugins, got %v", plugins)
	}
}

func TestJobEngineRmExec(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	ex := &executer{
		plug:  "test-plugin",
		tms:   time.Now().Add(-3 * time.Minute), // 3 minutes ago
		jobwt: list.New(),
	}
	// Add jobs to the queue
	job1 := &jobSync{step: &runtime.Step{Id: "step-1"}}
	job2 := &jobSync{step: &runtime.Step{Id: "step-2"}}
	ex.jobwt.PushBack(job1)
	ex.jobwt.PushBack(job2)
	je.execs["test-plugin"] = ex

	je.rmExec("test-plugin", ex)

	// Verify executor was removed
	if _, ok := je.execs["test-plugin"]; ok {
		t.Error("expected executor to be removed")
	}
	// Verify jobs were marked as ended
	if !job1.ended || !job2.ended {
		t.Error("expected jobs to be marked as ended")
	}
}
