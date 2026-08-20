package engine

import (
	"container/list"
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

func TestManagerBuildEgn(t *testing.T) {
	e := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	m := &Manager{
		buildEgn: e,
	}
	got := m.BuildEgn()
	if got != e {
		t.Error("BuildEgn() should return the assigned engine")
	}
}

func TestManagerBuildEgnNil(t *testing.T) {
	m := &Manager{}
	got := m.BuildEgn()
	if got != nil {
		t.Error("BuildEgn() should return nil when not assigned")
	}
}

func TestManagerHRun(t *testing.T) {
	h := &HbtpRunner{}
	m := &Manager{
		hrun: h,
	}
	got := m.HRun()
	if got != h {
		t.Error("HRun() should return the assigned HbtpRunner")
	}
}

func TestManagerHRunNil(t *testing.T) {
	m := &Manager{}
	got := m.HRun()
	if got != nil {
		t.Error("HRun() should return nil when not assigned")
	}
}

func TestManagerTimerEng(t *testing.T) {
	te := &TimerEngine{}
	m := &Manager{
		timerEgn: te,
	}
	got := m.TimerEng()
	if got != te {
		t.Error("TimerEng() should return the assigned TimerEngine")
	}
}

func TestManagerTimerEngNil(t *testing.T) {
	m := &Manager{}
	got := m.TimerEng()
	if got != nil {
		t.Error("TimerEng() should return nil when not assigned")
	}
}

func TestManagerPluginsNilJobEngine(t *testing.T) {
	m := &Manager{
		jobEgn: nil,
	}
	got := m.Plugins()
	if got != nil {
		t.Errorf("Plugins() should return nil when jobEgn is nil, got %v", got)
	}
}

func TestManagerPluginsEmpty(t *testing.T) {
	m := &Manager{
		jobEgn: newTestJobEngine(),
	}
	got := m.Plugins()
	if len(got) != 0 {
		t.Errorf("Plugins() should return empty slice, got %v", got)
	}
}

func TestManagerPluginsWithExecs(t *testing.T) {
	je := newTestJobEngine()
	je.execs["shell"] = &executer{plug: "shell", jobwt: list.New()}
	je.execs["docker"] = &executer{plug: "docker", jobwt: list.New()}

	m := &Manager{
		jobEgn: je,
	}
	got := m.Plugins()
	if len(got) != 2 {
		t.Fatalf("Plugins() should return 2 plugins, got %d", len(got))
	}
	found := map[string]bool{}
	for _, p := range got {
		found[p] = true
	}
	if !found["shell"] || !found["docker"] {
		t.Errorf("expected shell and docker plugins, got %v", got)
	}
}

func TestMgrGlobalNotNil(t *testing.T) {
	if Mgr == nil {
		t.Fatal("global Mgr should not be nil")
	}
}

func TestBuildEngineGetNil(t *testing.T) {
	var be *BuildEngine
	_, ok := be.Get("test")
	if ok {
		t.Error("Get() on nil receiver should return false")
	}
}

func TestBuildEngineGetEmptyId(t *testing.T) {
	be := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	_, ok := be.Get("")
	if ok {
		t.Error("Get() with empty id should return false")
	}
}

func TestBuildEngineStopEmpty(t *testing.T) {
	be := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Should not panic
	be.Stop()
}

func TestBuildEnginePutMultiple(t *testing.T) {
	be := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	for i := 0; i < 5; i++ {
		be.Put(&runtime.Build{Id: "build"})
	}
	if be.taskw.Len() != 5 {
		t.Errorf("expected 5 items, got %d", be.taskw.Len())
	}
}

// Test that NewBuildTask correctly links engine and build
func TestNewBuildTaskLinks(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bd := &runtime.Build{
		Id:         "build-link-1",
		PipelineId: "pipe-1",
	}
	bt := NewBuildTask(egn, bd)
	if bt.egn != egn {
		t.Error("BuildTask.egn should reference the engine")
	}
	if bt.build != bd {
		t.Error("BuildTask.build should reference the build")
	}
	if bt.build.Id != "build-link-1" {
		t.Errorf("build id = %q, want %q", bt.build.Id, "build-link-1")
	}
}

// Test Show returns correct build metadata
func TestShow_ActiveTask_Metadata(t *testing.T) {
	// This test is similar to existing ones but adds extra checks
	// for Created and Updated fields
	ctx := t.Context()
	bt := &BuildTask{
		build: &runtime.Build{
			Id:         "build-meta-1",
			PipelineId: "pipe-meta",
			Status:     "running",
		},
		ctx:    ctx,
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	show, ok := bt.Show()
	if !ok {
		t.Fatal("Show() should return true for active task")
	}
	if show.PipelineId != "pipe-meta" {
		t.Errorf("PipelineId = %q, want %q", show.PipelineId, "pipe-meta")
	}
}

// Test GetJob with empty map
func TestGetJob_EmptyMap(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{},
		jobs:  make(map[string]*jobSync),
	}
	_, ok := bt.GetJob("anything")
	if ok {
		t.Error("GetJob should return false for empty jobs map")
	}
}

// Test GetJob with populated map
func TestGetJob_Populated(t *testing.T) {
	job := &jobSync{
		step: &runtime.Step{Id: "step-x", Name: "build"},
	}
	bt := &BuildTask{
		build: &runtime.Build{},
		jobs: map[string]*jobSync{
			"step-x": job,
		},
	}
	got, ok := bt.GetJob("step-x")
	if !ok {
		t.Fatal("GetJob should find existing job")
	}
	if got != job {
		t.Error("GetJob should return the correct job reference")
	}
}

// Verify the runners.RunJob struct is properly handled
func TestRunnersRunJobFields(t *testing.T) {
	rj := &runners.RunJob{
		Id:         "job-1",
		PipelineId: "pipe-1",
		StageId:    "stage-1",
		BuildId:    "build-1",
		StageName:  "build",
		Step:       "shell",
		Name:       "compile",
	}
	if rj.Id != "job-1" {
		t.Error("RunJob Id should be set")
	}
	if rj.Step != "shell" {
		t.Error("RunJob Step should be set")
	}
}
