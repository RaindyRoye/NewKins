package engine

import (
	"container/list"
	"context"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
)

// Test runStage with empty steps
func TestRunStage_EmptySteps(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := t.Context()

	stage := &runtime.Stage{
		Id:      "stage-empty",
		BuildId: "build-empty",
		Name:    "build",
		Steps:   []*runtime.Step{},
	}

	ts := &taskStage{
		stage: stage,
		jobs:  make(map[string]*jobSync),
	}

	bt := &BuildTask{
		build: &runtime.Build{
			Id:        "build-empty",
			Status:    common.BuildStatusRunning,
			Stages:    []*runtime.Stage{stage},
			Started:   time.Now(),
			Finished:  time.Time{},
		},
		ctx:       ctx,
		stages:    map[string]*taskStage{"build": ts},
		jobs:      make(map[string]*jobSync),
		buildPath: tmpDir,
	}

	before := time.Now()
	bt.runStage(stage)
	after := time.Now()

	// Stage should complete successfully with no steps
	if stage.Status != common.BuildStatusOk {
		t.Errorf("stage status = %q, want %q", stage.Status, common.BuildStatusOk)
	}
	if stage.Finished.Before(before) || stage.Finished.After(after.Add(time.Second)) {
		t.Errorf("stage finished time %v is outside expected range", stage.Finished)
	}
}

// Test runStage with missing stage (nil stage map entry)
// This exercises the fix for the nil-pointer dereference when a stage
// exists in build.Stages but not in the BuildTask's stages map.
func TestRunStage_MissingStage(t *testing.T) {
	ctx := t.Context()

	stage := &runtime.Stage{
		Id:      "stage-missing",
		BuildId: "build-missing",
		Name:    "nonexistent",
		Steps:   []*runtime.Step{},
	}

	bt := &BuildTask{
		build: &runtime.Build{
			Id:       "build-missing",
			Status:   common.BuildStatusRunning,
			Stages:   []*runtime.Stage{stage},
			Started:  time.Now(),
			Finished: time.Time{},
		},
		ctx:    ctx,
		stages: make(map[string]*taskStage), // No entry for "nonexistent"
		jobs:   make(map[string]*jobSync),
	}

	// Should not panic - stage should be marked as error
	bt.runStage(stage)

	if stage.Status != common.BuildStatusError {
		t.Errorf("stage status = %q, want %q", stage.Status, common.BuildStatusError)
	}
	if stage.Error == "" {
		t.Error("stage error should be set for missing stage")
	}
}

// Test runStage with missing job (job in step but not in taskStage.jobs map)
// This exercises the fix for the nil-pointer dereference when a step
// references a job that doesn't exist in the taskStage's jobs map.
func TestRunStage_MissingJob(t *testing.T) {
	ctx := t.Context()

	step := &runtime.Step{
		Id:      "step-missing",
		StageId: "stage-missing-job",
		BuildId: "build-missing-job",
		Name:    "nonexistent-step",
	}

	stage := &runtime.Stage{
		Id:      "stage-missing-job",
		BuildId: "build-missing-job",
		Name:    "build",
		Steps:   []*runtime.Step{step},
	}

	ts := &taskStage{
		stage: stage,
		jobs:  make(map[string]*jobSync), // Empty - no job for "nonexistent-step"
	}

	bt := &BuildTask{
		build: &runtime.Build{
			Id:       "build-missing-job",
			Status:   common.BuildStatusRunning,
			Stages:   []*runtime.Stage{stage},
			Started:  time.Now(),
			Finished: time.Time{},
		},
		ctx:    ctx,
		stages: map[string]*taskStage{"build": ts},
		jobs:   make(map[string]*jobSync),
	}

	// Should not panic - step should be marked as error
	bt.runStage(stage)

	// The step should be marked as error
	if step.Status != common.BuildStatusError {
		t.Errorf("step status = %q, want %q", step.Status, common.BuildStatusError)
	}
	if step.Error == "" {
		t.Error("step error should be set for missing job")
	}
}

// Test runStage with canceled context
func TestRunStage_CanceledContext(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	stage := &runtime.Stage{
		Id:      "stage-cancel",
		BuildId: "build-cancel",
		Name:    "build",
		Steps:   []*runtime.Step{},
	}

	ts := &taskStage{
		stage: stage,
		jobs:  make(map[string]*jobSync),
	}

	bt := &BuildTask{
		build: &runtime.Build{
			Id:        "build-cancel",
			Status:    common.BuildStatusRunning,
			Stages:    []*runtime.Stage{stage},
			Started:   time.Now(),
			Finished:  time.Time{},
		},
		ctx:       ctx,
		stages:    map[string]*taskStage{"build": ts},
		jobs:      make(map[string]*jobSync),
		buildPath: tmpDir,
	}

	bt.runStage(stage)

	// Stage should complete (context cancellation is checked in runStep, not runStage)
	if stage.Status != common.BuildStatusOk {
		t.Errorf("stage status = %q, want %q (empty steps complete immediately)", stage.Status, common.BuildStatusOk)
	}
}

// Test BuildTask.run with nil Repo
func TestBuildTaskRun_NilRepoDetailed(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	bt := &BuildTask{
		build: &runtime.Build{
			Id:     "build-nil-repo",
			Repo:   nil,
			Stages: []*runtime.Stage{},
		},
	}

	bt.run()

	if bt.build.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", bt.build.Status, common.BuildStatusError)
	}
	if bt.build.Error != "repo param err" {
		t.Errorf("error = %q, want %q", bt.build.Error, "repo param err")
	}
	if bt.build.Finished.IsZero() {
		t.Error("Finished should be set")
	}
}

// Test BuildTask.run with empty stages but valid repo
func TestBuildTaskRun_EmptyStagesValidRepo(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	bt := &BuildTask{
		build: &runtime.Build{
			Id: "build-empty-stages",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{},
		},
	}

	bt.run()

	if bt.build.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", bt.build.Status, common.BuildStatusError)
	}
	if bt.build.Event != common.BuildEventCheckParam {
		t.Errorf("event = %q, want %q", bt.build.Event, common.BuildEventCheckParam)
	}
	if bt.build.Error != "build Stages is empty" {
		t.Errorf("error = %q, want %q", bt.build.Error, "build Stages is empty")
	}
}

// Test taskStage status transitions
func TestTaskStageStatusTransitions(t *testing.T) {
	ts := &taskStage{
		stage: &runtime.Stage{Name: "test"},
		jobs:  make(map[string]*jobSync),
	}

	// Initial state
	ts.status(common.BuildStatusPending, "")
	if ts.stage.Status != common.BuildStatusPending {
		t.Errorf("status = %q, want %q", ts.stage.Status, common.BuildStatusPending)
	}

	// Transition to running
	ts.status(common.BuildStatusRunning, "")
	if ts.stage.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", ts.stage.Status, common.BuildStatusRunning)
	}

	// Transition to Error with message
	ts.status(common.BuildStatusError, "something failed")
	if ts.stage.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", ts.stage.Status, common.BuildStatusError)
	}
	if ts.stage.Error != "something failed" {
		t.Errorf("error = %q, want %q", ts.stage.Error, "something failed")
	}
}

// Test jobSync status transitions
func TestJobSyncStatusTransitions(t *testing.T) {
	js := &jobSync{
		step: &runtime.Step{Name: "compile"},
	}

	// Initial state
	js.status(common.BuildStatusPending, "")
	if js.step.Status != common.BuildStatusPending {
		t.Errorf("status = %q, want %q", js.step.Status, common.BuildStatusPending)
	}

	// Transition to Running
	js.status(common.BuildStatusRunning, "")
	if js.step.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", js.step.Status, common.BuildStatusRunning)
	}

	// Transition to Ok
	js.status(common.BuildStatusOk, "")
	if js.step.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", js.step.Status, common.BuildStatusOk)
	}
}

// Test BuildEngine with multiple concurrent operations
func TestBuildEngineConcurrentOperations(t *testing.T) {
	be := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	// Add some tasks
	for i := 0; i < 50; i++ {
		id := "build-" + string(rune('a'+i%26))
		be.tasks[id] = &BuildTask{
			build: &runtime.Build{Id: id},
		}
	}

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				be.Get("build-a")
				be.Get("nonexistent")
			}
			done <- true
		}()
	}

	// Wait for all readers
	for i := 0; i < 10; i++ {
		<-done
	}
}
