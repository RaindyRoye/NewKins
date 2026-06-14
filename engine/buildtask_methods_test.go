package engine

import (
	"container/list"
	"context"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
)

// --- NewBuildTask ---

func TestNewBuildTask(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bd := &runtime.Build{Id: "build-100"}
	bt := NewBuildTask(egn, bd)
	if bt == nil {
		t.Fatal("NewBuildTask returned nil")
	}
	if bt.egn != egn {
		t.Error("BuildTask.egn should reference the engine")
	}
	if bt.build != bd {
		t.Error("BuildTask.build should reference the build")
	}
}

// --- BuildTask.status ---

func TestBuildTaskStatus(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	bt.status(common.BuildStatusRunning, "some error")
	if bt.build.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", bt.build.Status, common.BuildStatusRunning)
	}
	if bt.build.Error != "some error" {
		t.Errorf("error = %q, want %q", bt.build.Error, "some error")
	}
}

func TestBuildTaskStatus_WithEvent(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	bt.status(common.BuildStatusError, "err msg", common.BuildEventGetRepo)
	if bt.build.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", bt.build.Status, common.BuildStatusError)
	}
	if bt.build.Event != common.BuildEventGetRepo {
		t.Errorf("event = %q, want %q", bt.build.Event, common.BuildEventGetRepo)
	}
}

func TestBuildTaskStatus_NoEvent(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Event: "previous-event"}}
	bt.status(common.BuildStatusOk, "")
	// Event should remain unchanged when no event is passed
	if bt.build.Event != "previous-event" {
		t.Errorf("event should be preserved, got %q", bt.build.Event)
	}
}

// --- BuildTask.stopd ---

func TestBuildTaskStopd_NilContext(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	// ctx is nil, so stopd() should return true
	if !bt.stopd() {
		t.Error("stopd() should return true when ctx is nil")
	}
}

func TestBuildTaskStopd_ActiveContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{build: &runtime.Build{}, ctx: ctx}
	if bt.stopd() {
		t.Error("stopd() should return false for active context")
	}
}

func TestBuildTaskStopd_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bt := &BuildTask{build: &runtime.Build{}, ctx: ctx}
	if !bt.stopd() {
		t.Error("stopd() should return true for canceled context")
	}
}

// --- BuildTask.stop ---

func TestBuildTaskStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bt := &BuildTask{build: &runtime.Build{}, ctx: ctx, cncl: cancel}
	bt.stop()
	// After stop, context should be canceled
	if !bt.stopd() {
		t.Error("stop() should cancel the context")
	}
	// ctrlendtm should be zeroed
	if !bt.ctrlendtm.IsZero() {
		t.Error("stop() should zero ctrlendtm")
	}
}

func TestBuildTaskStop_NilCancel(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	// Should not panic
	bt.stop()
}

// --- BuildTask.Cancel ---

func TestBuildTaskCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bt := &BuildTask{build: &runtime.Build{}, ctx: ctx, cncl: cancel}
	bt.Cancel()
	// ctrlendtm should be set to a recent time
	if bt.ctrlendtm.IsZero() {
		t.Error("Cancel() should set ctrlendtm")
	}
	if time.Since(bt.ctrlendtm) > time.Second {
		t.Error("Cancel() should set ctrlendtm to now")
	}
	// Context should be canceled
	if !bt.stopd() {
		t.Error("Cancel() should cancel the context")
	}
}

func TestBuildTaskCancel_NilCancel(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	// Should not panic
	bt.Cancel()
	if bt.ctrlendtm.IsZero() {
		t.Error("Cancel() should set ctrlendtm even when cncl is nil")
	}
}

// --- BuildTask.WorkProgress ---

func TestBuildTaskWorkProgress(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}, workpgss: 42}
	if bt.WorkProgress() != 42 {
		t.Errorf("WorkProgress() = %d, want 42", bt.WorkProgress())
	}
}

func TestBuildTaskWorkProgress_Zero(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	if bt.WorkProgress() != 0 {
		t.Errorf("WorkProgress() = %d, want 0", bt.WorkProgress())
	}
}

// --- BuildTask.Write (git progress parsing) ---

func TestBuildTaskWrite_ProgressParsing(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	// Git clone progress output format: "Receiving objects:  50% (100/200)"
	input := "Receiving objects:  50% (100/200)"
	n, err := bt.Write([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(input) {
		t.Errorf("Write returned n=%d, want %d", n, len(input))
	}
	// workpgss should be 50 * 0.8 = 40
	expected := int(float64(50) * 0.8)
	if bt.workpgss != expected {
		t.Errorf("workpgss = %d, want %d (50%% * 0.8)", bt.workpgss, expected)
	}
}

func TestBuildTaskWrite_ProgressParsing100(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	input := "Receiving objects:  100% (200/200)"
	_, err := bt.Write([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// workpgss should be 100 * 0.8 = 80
	expected := int(float64(100) * 0.8)
	if bt.workpgss != expected {
		t.Errorf("workpgss = %d, want %d (100%% * 0.8)", bt.workpgss, expected)
	}
}

func TestBuildTaskWrite_ProgressParsingZero(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	input := "Receiving objects:   0% (0/200)"
	_, err := bt.Write([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bt.workpgss != 0 {
		t.Errorf("workpgss = %d, want 0", bt.workpgss)
	}
}

func TestBuildTaskWrite_NoProgress(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}, workpgss: 25}
	// Non-progress output should not change workpgss
	input := "Cloning into '/tmp/repo'..."
	n, err := bt.Write([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(input) {
		t.Errorf("Write returned n=%d, want %d", n, len(input))
	}
	if bt.workpgss != 25 {
		t.Errorf("workpgss should remain 25 for non-progress output, got %d", bt.workpgss)
	}
}

func TestBuildTaskWrite_EmptyInput(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	n, err := bt.Write([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("Write returned n=%d, want 0", n)
	}
}

// --- regBfb regex ---

func TestRegBfb_Matches(t *testing.T) {
	tests := []struct {
		input   string
		match   bool
		percent string
	}{
		{"Receiving objects:  50% (100/200)", true, "50"},
		{"Receiving objects:   0% (0/200)", true, "0"},
		{"Receiving objects:  100% (200/200)", true, "100"},
		{"Resolving deltas:  75% (3/4)", true, "75"},
		{"no progress here", false, ""},
		{"50%", false, ""},
		{": 50% (1/2)", true, "50"},
		{"", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := regBfb.MatchString(tt.input)
			if got != tt.match {
				t.Errorf("regBfb.MatchString(%q) = %v, want %v", tt.input, got, tt.match)
			}
			if tt.match {
				subs := regBfb.FindStringSubmatch(tt.input)
				if len(subs) < 2 || subs[1] != tt.percent {
					t.Errorf("regBfb percent = %v, want %q", subs, tt.percent)
				}
			}
		})
	}
}

// --- BuildTask.GetJob ---

func TestBuildTaskGetJob_EmptyId(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{},
		jobs:  make(map[string]*jobSync),
	}
	_, ok := bt.GetJob("")
	if ok {
		t.Error("GetJob with empty id should return false")
	}
}

func TestBuildTaskGetJob_NotFound(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{},
		jobs:  make(map[string]*jobSync),
	}
	_, ok := bt.GetJob("nonexistent")
	if ok {
		t.Error("GetJob with nonexistent id should return false")
	}
}

func TestBuildTaskGetJob_Found(t *testing.T) {
	job := &jobSync{
		step: &runtime.Step{Id: "step-1", Name: "build"},
	}
	bt := &BuildTask{
		build: &runtime.Build{},
		jobs: map[string]*jobSync{
			"step-1": job,
		},
	}
	got, ok := bt.GetJob("step-1")
	if !ok {
		t.Fatal("GetJob should find existing job")
	}
	if got != job {
		t.Error("GetJob should return the correct job")
	}
}

// --- taskStage.status ---

func TestTaskStageStatus(t *testing.T) {
	ts := &taskStage{
		stage: &runtime.Stage{Name: "build"},
	}
	ts.status(common.BuildStatusRunning, "")
	if ts.stage.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", ts.stage.Status, common.BuildStatusRunning)
	}
}

func TestTaskStageStatus_WithError(t *testing.T) {
	ts := &taskStage{
		stage: &runtime.Stage{Name: "test"},
	}
	ts.status(common.BuildStatusError, "something failed")
	if ts.stage.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", ts.stage.Status, common.BuildStatusError)
	}
	if ts.stage.Error != "something failed" {
		t.Errorf("error = %q, want %q", ts.stage.Error, "something failed")
	}
}

func TestTaskStageStatus_WithEvent(t *testing.T) {
	ts := &taskStage{
		stage: &runtime.Stage{Name: "deploy"},
	}
	ts.status(common.BuildStatusError, "err", "deploy-event")
	if ts.stage.Event != "deploy-event" {
		t.Errorf("event = %q, want %q", ts.stage.Event, "deploy-event")
	}
}

func TestTaskStageStatus_NoEventPreservesExisting(t *testing.T) {
	ts := &taskStage{
		stage: &runtime.Stage{Name: "build", Event: "original"},
	}
	ts.status(common.BuildStatusOk, "")
	if ts.stage.Event != "original" {
		t.Errorf("event should be preserved, got %q", ts.stage.Event)
	}
}

// --- jobSync.status ---

func TestJobSyncStatus(t *testing.T) {
	js := &jobSync{
		step: &runtime.Step{Name: "compile"},
	}
	js.status(common.BuildStatusRunning, "")
	if js.step.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", js.step.Status, common.BuildStatusRunning)
	}
}

func TestJobSyncStatus_WithEvent(t *testing.T) {
	js := &jobSync{
		step: &runtime.Step{Name: "test"},
	}
	js.status(common.BuildStatusError, "test failed", "test-event")
	if js.step.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", js.step.Status, common.BuildStatusError)
	}
	if js.step.Error != "test failed" {
		t.Errorf("error = %q, want %q", js.step.Error, "test failed")
	}
	if js.step.Event != "test-event" {
		t.Errorf("event = %q, want %q", js.step.Event, "test-event")
	}
}

func TestJobSyncStatus_NoEventPreservesExisting(t *testing.T) {
	js := &jobSync{
		step: &runtime.Step{Name: "build", Event: "prev-event"},
	}
	js.status(common.BuildStatusOk, "")
	if js.step.Event != "prev-event" {
		t.Errorf("event should be preserved, got %q", js.step.Event)
	}
}

// --- BuildEngine.Stop ---

func TestBuildEngineStop_Empty(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Should not panic on empty tasks
	c.Stop()
}

func TestBuildEngineStop_StopsAllTasks(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Add tasks with active contexts
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		id := "build-stop-" + string(rune('a'+i))
		c.tasks[id] = &BuildTask{
			build: &runtime.Build{Id: id},
			ctx:   ctx,
			cncl:  cancel,
		}
	}
	c.Stop()
	// All tasks should have canceled contexts
	for id, task := range c.tasks {
		if !task.stopd() {
			t.Errorf("task %q should be stopped after Stop()", id)
		}
	}
}

// --- cmdSync status transitions via UpJobCmd ---

func TestUpJobCmd_NilCmd(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	// Should not panic with nil cmd
	bt.UpJobCmd(nil, 1, 0)
}

func TestUpJobCmd_NilJob(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	// Should not panic with nil job
	bt.UpJob(nil, common.BuildStatusOk, "", 0)
}

func TestUpJob_EmptyStatus(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-1", Name: "test"},
		cmdmp: make(map[string]*cmdSync),
	}
	// Empty status should be a no-op
	bt.UpJob(job, "", "", 0)
	if job.step.Status != "" {
		t.Errorf("step status should remain empty, got %q", job.step.Status)
	}
}
