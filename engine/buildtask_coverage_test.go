package engine

import (
	"container/list"
	"testing"

	"github.com/gokins/core/runtime"
)

func TestNewBuildTask_Fields(t *testing.T) {
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bd := &runtime.Build{Id: "test-build"}
	bt := NewBuildTask(egn, bd)
	if bt == nil {
		t.Fatal("NewBuildTask should not return nil")
	}
	if bt.build != bd {
		t.Error("NewBuildTask should assign build field")
	}
	if bt.egn != egn {
		t.Error("NewBuildTask should assign egn field")
	}
}

func TestBuildTaskStopd_NilCtx_New(t *testing.T) {
	bt := &BuildTask{}
	// ctx is nil, so stopd should return true
	if !bt.stopd() {
		t.Error("stopd() should return true when ctx is nil")
	}
}

func TestBuildTaskStop_NilCancel_New(t *testing.T) {
	bt := &BuildTask{}
	// Should not panic when cncl is nil
	bt.stop()
}

func TestBuildTaskCancel_NilCancel_New(t *testing.T) {
	bt := &BuildTask{}
	// Should not panic when cncl is nil
	bt.Cancel()
	// ctrlendtm should be set
	if bt.ctrlendtm.IsZero() {
		t.Error("Cancel() should set ctrlendtm")
	}
}

func TestBuildTaskStatus_SetFields(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test"},
	}
	bt.status("running", "no error")
	if bt.build.Status != "running" {
		t.Errorf("expected status 'running', got %q", bt.build.Status)
	}
	if bt.build.Error != "no error" {
		t.Errorf("expected error 'no error', got %q", bt.build.Error)
	}
}

func TestBuildTaskStatusWithEvent_Field(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test"},
	}
	bt.status("error", "something failed", "event1")
	if bt.build.Event != "event1" {
		t.Errorf("expected event 'event1', got %q", bt.build.Event)
	}
}

func TestBuildTaskWorkProgress_Value(t *testing.T) {
	bt := &BuildTask{
		workpgss: 42,
	}
	if bt.WorkProgress() != 42 {
		t.Errorf("expected 42, got %d", bt.WorkProgress())
	}
}

func TestBuildTaskGetJobEmptyID_False(t *testing.T) {
	bt := &BuildTask{
		jobs: make(map[string]*jobSync),
	}
	_, ok := bt.GetJob("")
	if ok {
		t.Error("GetJob with empty ID should return false")
	}
}

func TestBuildTaskGetJobNonExistent_False(t *testing.T) {
	bt := &BuildTask{
		jobs: make(map[string]*jobSync),
	}
	_, ok := bt.GetJob("nonexistent")
	if ok {
		t.Error("GetJob with non-existent ID should return false")
	}
}

func TestBuildTaskGetJobExisting_True(t *testing.T) {
	bt := &BuildTask{
		jobs: make(map[string]*jobSync),
	}
	job := &jobSync{}
	bt.jobs["job-1"] = job
	got, ok := bt.GetJob("job-1")
	if !ok {
		t.Error("GetJob should find existing job")
	}
	if got != job {
		t.Error("GetJob should return the correct job")
	}
}

func TestBuildTaskUpJob_NilJob(t *testing.T) {
	bt := &BuildTask{}
	// Should not panic
	bt.UpJob(nil, "running", "", 0)
}

func TestBuildTaskUpJob_EmptyStatus(t *testing.T) {
	bt := &BuildTask{}
	job := &jobSync{}
	// Should not panic when status is empty
	bt.UpJob(job, "", "", 0)
}

func TestBuildTaskUpJobCmd_NilCmd(t *testing.T) {
	bt := &BuildTask{}
	// Should not panic
	bt.UpJobCmd(nil, 1, 0)
}

func TestTaskStageStatus_SetFields(t *testing.T) {
	stg := &taskStage{
		stage: &runtime.Stage{Id: "stg-1"},
	}
	stg.status("running", "no error")
	if stg.stage.Status != "running" {
		t.Errorf("expected status 'running', got %q", stg.stage.Status)
	}
	if stg.stage.Error != "no error" {
		t.Errorf("expected error 'no error', got %q", stg.stage.Error)
	}
}

func TestTaskStageStatusWithEvent_Field(t *testing.T) {
	stg := &taskStage{
		stage: &runtime.Stage{Id: "stg-1"},
	}
	stg.status("error", "failed", "event1")
	if stg.stage.Event != "event1" {
		t.Errorf("expected event 'event1', got %q", stg.stage.Event)
	}
}

func TestBuildTaskWrite_Normal(t *testing.T) {
	bt := &BuildTask{}
	// Test normal write
	n, err := bt.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}
}

func TestBuildTaskWrite_WithProgress(t *testing.T) {
	bt := &BuildTask{}
	// Test progress parsing
	n, err := bt.Write([]byte("Receiving objects: 50% (10/20)"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n == 0 {
		t.Error("expected non-zero bytes written")
	}
	// workpgss should be updated (50 * 0.8 = 40)
	if bt.workpgss != 40 {
		t.Errorf("expected workpgss=40, got %d", bt.workpgss)
	}
}
