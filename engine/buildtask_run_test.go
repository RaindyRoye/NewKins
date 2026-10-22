package engine

import (
	"container/list"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
)

// TestBuildTaskRun_NoStages verifies that run() exits early when check() fails
// due to empty stages, setting the build status to error.
func TestBuildTaskRun_NoStages(t *testing.T) {
	e := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	build := &runtime.Build{
		Id:         "build-no-stages",
		PipelineId: "pipe-1",
		Stages:     []*runtime.Stage{}, // empty stages
		Repo: &runtime.Repository{
			CloneURL: "",
		},
	}
	bt := NewBuildTask(e, build)
	bt.ctx, bt.cncl = context.WithCancel(context.Background())
	defer bt.cncl()

	// Set up a temporary work path
	origWorkPath := comm.WorkPath
	comm.WorkPath = t.TempDir()
	defer func() { comm.WorkPath = origWorkPath }()

	bt.run()

	// Should fail at check() with error about empty stages
	if bt.build.Status != common.BuildStatusError {
		t.Errorf("expected status %q, got %q", common.BuildStatusError, bt.build.Status)
	}
	if bt.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, bt.build.Event)
	}
}

// TestBuildTaskRun_NoRepo verifies that run() exits early when check() fails
// due to nil repo.
func TestBuildTaskRun_NoRepo(t *testing.T) {
	e := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	build := &runtime.Build{
		Id:         "build-no-repo",
		PipelineId: "pipe-1",
		Repo:       nil, // nil repo
		Stages: []*runtime.Stage{
			{
				Id:      "stage-1",
				BuildId: "build-no-repo",
				Name:    "test-stage",
				Steps: []*runtime.Step{
					{
						Id:      "step-1",
						BuildId: "build-no-repo",
						StageId: "stage-1",
						Name:    "test-step",
						Step:    "shell@ssh",
					},
				},
			},
		},
	}
	bt := NewBuildTask(e, build)
	bt.ctx, bt.cncl = context.WithCancel(context.Background())
	defer bt.cncl()

	origWorkPath := comm.WorkPath
	comm.WorkPath = t.TempDir()
	defer func() { comm.WorkPath = origWorkPath }()

	bt.run()

	// Should fail at check() with error about repo
	// Note: check() has a bug where it passes event constant as status
	// After fix, it should set status to BuildStatusError and event to BuildEventCheckParam
	if bt.build.Status != common.BuildStatusError {
		t.Errorf("expected status %q, got %q", common.BuildStatusError, bt.build.Status)
	}
	if bt.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, bt.build.Event)
	}
}

// TestBuildTaskRun_CreatePathError verifies that run() handles errors when
// creating the build path.
func TestBuildTaskRun_CreatePathError(t *testing.T) {
	e := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	build := &runtime.Build{
		Id:         "build-path-err",
		PipelineId: "pipe-1",
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      "stage-1",
				BuildId: "build-path-err",
				Name:    "test-stage",
				Steps: []*runtime.Step{
					{
						Id:      "step-1",
						BuildId: "build-path-err",
						StageId: "stage-1",
						Name:    "test-step",
						Step:    "shell@ssh",
					},
				},
			},
		},
	}
	bt := NewBuildTask(e, build)
	bt.ctx, bt.cncl = context.WithCancel(context.Background())
	defer bt.cncl()

	// Set an invalid work path that can't be created
	origWorkPath := comm.WorkPath
	comm.WorkPath = "/proc/invalid/path"
	defer func() { comm.WorkPath = origWorkPath }()

	bt.run()

	// Should fail when creating build path
	if bt.build.Status != common.BuildStatusError {
		t.Errorf("expected status %q, got %q", common.BuildStatusError, bt.build.Status)
	}
	if bt.build.Event != common.BuildEventPath {
		t.Errorf("expected event %q, got %q", common.BuildEventPath, bt.build.Event)
	}
}

// TestBuildTaskRun_CanceledContext verifies that run() respects a canceled
// context and exits cleanly.
func TestBuildTaskRun_CancelledContext(t *testing.T) {
	e := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	build := &runtime.Build{
		Id:         "build-cancel",
		PipelineId: "pipe-1",
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      "stage-1",
				BuildId: "build-cancel",
				Name:    "test-stage",
				Steps: []*runtime.Step{
					{
						Id:      "step-1",
						BuildId: "build-cancel",
						StageId: "stage-1",
						Name:    "test-step",
						Step:    "shell@ssh",
					},
				},
			},
		},
	}
	bt := NewBuildTask(e, build)
	ctx, cancel := context.WithCancel(context.Background())
	bt.ctx = ctx
	bt.cncl = cancel

	// Cancel before running
	cancel()

	origWorkPath := comm.WorkPath
	comm.WorkPath = t.TempDir()
	defer func() { comm.WorkPath = origWorkPath }()

	// Should exit early due to canceled context
	bt.run()

	// Verify it didn't panic and exited cleanly
	if bt.build.Finished.IsZero() {
		t.Error("expected Finished to be set")
	}
}

// TestBuildTaskRun_Stopd verifies the stopd() method behavior.
func TestBuildTaskRun_Stopd(t *testing.T) {
	// nil context should return true
	bt1 := &BuildTask{ctx: nil}
	if !bt1.stopd() {
		t.Error("stopd() should return true when ctx is nil")
	}

	// active context should return false
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt2 := &BuildTask{ctx: ctx}
	if bt2.stopd() {
		t.Error("stopd() should return false when ctx is active")
	}

	// canceled context should return true
	cancel()
	time.Sleep(10 * time.Millisecond) // give time for cancellation to propagate
	if !bt2.stopd() {
		t.Error("stopd() should return true when ctx is canceled")
	}
}

// TestBuildTaskRun_StopMethod verifies the stop() method.
func TestBuildTaskRun_StopMethod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bt := &BuildTask{
		ctx:       ctx,
		cncl:      cancel,
		ctrlendtm: time.Now(),
	}

	bt.stop()

	// ctrlendtm should be zeroed
	if !bt.ctrlendtm.IsZero() {
		t.Error("stop() should zero ctrlendtm")
	}

	// context should be canceled
	time.Sleep(10 * time.Millisecond)
	if ctx.Err() == nil {
		t.Error("stop() should cancel the context")
	}
}

// TestBuildTaskRun_CancelMethod verifies the Cancel() method.
func TestBuildTaskRun_CancelMethod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bt := &BuildTask{
		ctx:  ctx,
		cncl: cancel,
	}

	before := time.Now()
	bt.Cancel()
	after := time.Now()

	// ctrlendtm should be set to a recent time
	if bt.ctrlendtm.Before(before) || bt.ctrlendtm.After(after) {
		t.Error("Cancel() should set ctrlendtm to current time")
	}

	// context should be canceled
	time.Sleep(10 * time.Millisecond)
	if ctx.Err() == nil {
		t.Error("Cancel() should cancel the context")
	}
}

// TestBuildTaskRun_GetRepo_NoClone verifies that getRepo() returns nil when
// isClone is false.
func TestBuildTaskRun_GetRepo_NoClone(t *testing.T) {
	bt := &BuildTask{
		isClone: false,
	}
	err := bt.getRepo()
	if err != nil {
		t.Errorf("getRepo() should return nil when isClone is false, got: %v", err)
	}
}

// TestBuildTaskRun_GetRepo_CreatePathError verifies that getRepo() handles errors
// when creating the repo path.
func TestBuildTaskRun_GetRepo_CreatePathError(t *testing.T) {
	bt := &BuildTask{
		isClone:   true,
		repoPaths: "/proc/invalid/repo/path",
		repoPath:  "http://example.com/repo.git",
	}
	err := bt.getRepo()
	if err == nil {
		t.Error("getRepo() should return error when path creation fails")
	}
}

// TestBuildTaskRun_GetRepo_CloneError verifies that getRepo() handles errors
// when git clone fails.
func TestBuildTaskRun_GetRepo_CloneError(t *testing.T) {
	tmpDir := t.TempDir()
	bt := &BuildTask{
		isClone:   true,
		repoPaths: filepath.Join(tmpDir, "repo"),
		repoPath:  "http://invalid.local/repo.git",
		ctx:       context.Background(),
		build: &runtime.Build{
			Repo: &runtime.Repository{
				CloneURL: "http://invalid.local/repo.git",
			},
		},
	}
	err := bt.getRepo()
	if err == nil {
		t.Error("getRepo() should return error when clone fails")
	}
}

// TestBuildTaskRun_Clears verifies the clears() method.
func TestBuildTaskRun_Clears(t *testing.T) {
	tmpDir := t.TempDir()
	bt := &BuildTask{
		isClone:   true,
		repoPaths: filepath.Join(tmpDir, "repo"),
		buildPath: filepath.Join(tmpDir, "build"),
		jobs:      make(map[string]*jobSync),
	}

	// Should not panic
	bt.clears()
}

// TestBuildTaskRun_Write verifies the Write() method for git progress parsing.
func TestBuildTaskRun_Write(t *testing.T) {
	bt := &BuildTask{}

	// Test with progress message
	progressMsg := []byte("Receiving objects:  50% (10/20)")
	n, err := bt.Write(progressMsg)
	if err != nil {
		t.Errorf("Write() returned error: %v", err)
	}
	if n != len(progressMsg) {
		t.Errorf("Write() returned wrong length: got %d, want %d", n, len(progressMsg))
	}
	if bt.workpgss != 40 { // 50 * 0.8
		t.Errorf("workpgss = %d, want 40", bt.workpgss)
	}

	// Test with non-progress message
	normalMsg := []byte("normal log message")
	n, err = bt.Write(normalMsg)
	if err != nil {
		t.Errorf("Write() returned error: %v", err)
	}
	if n != len(normalMsg) {
		t.Errorf("Write() returned wrong length: got %d, want %d", n, len(normalMsg))
	}
}

// TestBuildTaskRun_WorkProgress verifies the WorkProgress() method.
func TestBuildTaskRun_WorkProgress(t *testing.T) {
	bt := &BuildTask{workpgss: 75}
	if bt.WorkProgress() != 75 {
		t.Errorf("WorkProgress() = %d, want 75", bt.WorkProgress())
	}
}

// TestBuildTaskRun_UpJob verifies the UpJob() method.
func TestBuildTaskRun_UpJob(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
		jobs:  make(map[string]*jobSync),
	}

	// nil job should not panic
	bt.UpJob(nil, "status", "error", 0)

	// valid job
	job := &jobSync{
		step: &runtime.Step{Id: "step-1"},
	}
	bt.UpJob(job, common.BuildStatusRunning, "", 0)
	if job.step.Status != common.BuildStatusRunning {
		t.Errorf("expected status %q, got %q", common.BuildStatusRunning, job.step.Status)
	}
}

// TestBuildTaskRun_UpJobCmd verifies the UpJobCmd() method.
func TestBuildTaskRun_UpJobCmd(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}

	// nil cmd should not panic
	bt.UpJobCmd(nil, 1, 0)

	// test case 1: start
	cmd := &cmdSync{}
	bt.UpJobCmd(cmd, 1, 0)
	if cmd.status != common.BuildStatusRunning {
		t.Errorf("expected status %q, got %q", common.BuildStatusRunning, cmd.status)
	}
	if cmd.started.IsZero() {
		t.Error("started should be set")
	}

	// test case 2: success
	bt.UpJobCmd(cmd, 2, 0)
	if cmd.status != common.BuildStatusOk {
		t.Errorf("expected status %q, got %q", common.BuildStatusOk, cmd.status)
	}
	if cmd.finished.IsZero() {
		t.Error("finished should be set")
	}

	// test case 2: error with non-zero code
	cmd2 := &cmdSync{}
	bt.UpJobCmd(cmd2, 2, 1)
	if cmd2.status != common.BuildStatusError {
		t.Errorf("expected status %q, got %q", common.BuildStatusError, cmd2.status)
	}
	if cmd2.code != 1 {
		t.Errorf("expected code 1, got %d", cmd2.code)
	}

	// test case 3: cancel
	cmd3 := &cmdSync{}
	bt.UpJobCmd(cmd3, 3, 0)
	if cmd3.status != common.BuildStatusCancel {
		t.Errorf("expected status %q, got %q", common.BuildStatusCancel, cmd3.status)
	}

	// test case -1: error
	cmd4 := &cmdSync{}
	bt.UpJobCmd(cmd4, -1, 1)
	if cmd4.status != common.BuildStatusError {
		t.Errorf("expected status %q, got %q", common.BuildStatusError, cmd4.status)
	}

	// test default case
	cmd5 := &cmdSync{}
	bt.UpJobCmd(cmd5, 99, 0)
	if cmd5.status != "" {
		t.Errorf("expected empty status for unknown fs, got %q", cmd5.status)
	}
}
