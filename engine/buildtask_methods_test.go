package engine

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

func TestBuildTask_Show_Empty(t *testing.T) {
	// BuildTask with no stages should return empty show
	// Show() requires ctx to be set and not done
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{
		ctx: ctx,
		build: &runtime.Build{
			Id:         "test-build",
			PipelineId: "pipe-1",
			Status:     "running",
			Started:    time.Now(),
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}

	show, ok := bt.Show()
	if !ok {
		t.Fatal("expected Show() to return ok=true")
	}
	if show == nil {
		t.Fatal("expected Show() to return non-nil BuildShow")
	}
	if show.Id != "test-build" {
		t.Errorf("expected build ID 'test-build', got %q", show.Id)
	}
	if show.PipelineId != "pipe-1" {
		t.Errorf("expected pipeline ID 'pipe-1', got %q", show.PipelineId)
	}
	if len(show.Stages) != 0 {
		t.Errorf("expected 0 stages, got %d", len(show.Stages))
	}
}

func TestBuildTask_Show_WithStages(t *testing.T) {
	stageID := "stage-1"
	stageName := "build"
	stepID := "step-1"
	stepName := "compile"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{
		ctx: ctx,
		build: &runtime.Build{
			Id:     "test-build",
			Status: "running",
			Stages: []*runtime.Stage{
				{
					Id:      stageID,
					BuildId: "test-build",
					Name:    stageName,
					Status:  "running",
					Steps: []*runtime.Step{
						{
							Id:      stepID,
							StageId: stageID,
							BuildId: "test-build",
							Name:    stepName,
							Status:  "running",
						},
					},
				},
			},
		},
		stages: map[string]*taskStage{
			stageName: {
				stage: &runtime.Stage{
					Id:      stageID,
					BuildId: "test-build",
					Status:  "running",
				},
				jobs: map[string]*jobSync{
					stepName: {
						step: &runtime.Step{
							Id:      stepID,
							StageId: stageID,
							BuildId: "test-build",
							Status:  "running",
						},
						cmdmp: make(map[string]*cmdSync),
					},
				},
			},
		},
		jobs: make(map[string]*jobSync),
	}

	show, ok := bt.Show()
	if !ok {
		t.Fatal("expected Show() to return ok=true")
	}
	if len(show.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(show.Stages))
	}
	if show.Stages[0].Id != stageID {
		t.Errorf("expected stage ID %q, got %q", stageID, show.Stages[0].Id)
	}
	if len(show.Stages[0].Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(show.Stages[0].Steps))
	}
	if show.Stages[0].Steps[0].Id != stepID {
		t.Errorf("expected step ID %q, got %q", stepID, show.Stages[0].Steps[0].Id)
	}
}

func TestBuildTask_Show_CanceledContext(t *testing.T) {
	// Show() should return false when context is canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	bt := &BuildTask{
		ctx:   ctx,
		build: &runtime.Build{Id: "test-build"},
	}

	_, ok := bt.Show()
	if ok {
		t.Error("expected Show() to return false when context is canceled")
	}
}

func TestBuildTask_UpJobCmd_StatusTransitions(t *testing.T) {
	// Note: UpJobCmd spawns a goroutine that calls updateStepCmd, which
	// requires comm.Db to be initialized. In unit tests without a database,
	// the goroutine will fail but recover via util.RecoverLog.
	// We can still test the status transition logic.

	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}

	cmd := &cmdSync{
		cmd: &runners.CmdContent{
			Id: "cmd-1",
		},
	}

	// Test status = 1 (running)
	bt.UpJobCmd(cmd, 1, 0)
	if cmd.status != "running" {
		t.Errorf("expected status 'running', got %q", cmd.status)
	}
	if cmd.started.IsZero() {
		t.Error("expected started to be set")
	}

	// Test status = 2 (success) - note: actual status constant is "ok"
	bt.UpJobCmd(cmd, 2, 0)
	if cmd.status != "ok" {
		t.Errorf("expected status 'ok', got %q", cmd.status)
	}
	if cmd.finished.IsZero() {
		t.Error("expected finished to be set")
	}

	// Test status = 2 with non-zero exit code (error)
	cmd2 := &cmdSync{cmd: &runners.CmdContent{Id: "cmd-2"}}
	bt.UpJobCmd(cmd2, 2, 1)
	if cmd2.status != "error" {
		t.Errorf("expected status 'error' for non-zero exit code, got %q", cmd2.status)
	}
	if cmd2.code != 1 {
		t.Errorf("expected code 1, got %d", cmd2.code)
	}

	// Test status = 3 (cancel)
	cmd3 := &cmdSync{cmd: &runners.CmdContent{Id: "cmd-3"}}
	bt.UpJobCmd(cmd3, 3, 0)
	if cmd3.status != "cancel" {
		t.Errorf("expected status 'cancel', got %q", cmd3.status)
	}

	// Test status = -1 (error)
	cmd4 := &cmdSync{cmd: &runners.CmdContent{Id: "cmd-4"}}
	bt.UpJobCmd(cmd4, -1, 1)
	if cmd4.status != "error" {
		t.Errorf("expected status 'error', got %q", cmd4.status)
	}

	// Test unknown status (should not modify)
	cmd5 := &cmdSync{cmd: &runners.CmdContent{Id: "cmd-5"}, status: "pending"}
	bt.UpJobCmd(cmd5, 99, 0)
	if cmd5.status != "pending" {
		t.Errorf("expected status to remain 'pending', got %q", cmd5.status)
	}

	// Give goroutines time to complete
	time.Sleep(50 * time.Millisecond)
}

func TestBuildTask_Write_GitProgress(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}

	// Test git progress parsing
	progressLine := "Receiving objects:  50% (100/200)\n"
	n, err := bt.Write([]byte(progressLine))
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if n != len(progressLine) {
		t.Errorf("expected n=%d, got %d", len(progressLine), n)
	}
	// workpgss should be updated to 50 * 0.8 = 40
	if bt.WorkProgress() != 40 {
		t.Errorf("expected work progress 40, got %d", bt.WorkProgress())
	}

	// Test with different progress
	progressLine2 := "Receiving objects:  75% (150/200)\n"
	if _, werr := bt.Write([]byte(progressLine2)); werr != nil {
		t.Fatalf("Write() error: %v", werr)
	}
	if bt.WorkProgress() != 60 { // 75 * 0.8 = 60
		t.Errorf("expected work progress 60, got %d", bt.WorkProgress())
	}

	// Test with non-progress line (should not change workpgss)
	regularLine := "Cloning into 'repo'...\n"
	if _, werr := bt.Write([]byte(regularLine)); werr != nil {
		t.Fatalf("Write() error: %v", werr)
	}
	if bt.WorkProgress() != 60 {
		t.Errorf("expected work progress to remain 60, got %d", bt.WorkProgress())
	}

	// Test empty input
	if _, werr := bt.Write([]byte{}); werr != nil {
		t.Fatalf("Write() error: %v", werr)
	}
}

func TestBuildTask_stopd_NilContext(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}

	if !bt.stopd() {
		t.Error("expected stopd() to return true when ctx is nil")
	}
}

func TestBuildTask_stopd_ActiveContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{
		ctx:   ctx,
		build: &runtime.Build{Id: "test-build"},
	}

	if bt.stopd() {
		t.Error("expected stopd() to return false when context is active")
	}
}

func TestBuildTask_stopd_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bt := &BuildTask{
		ctx:   ctx,
		build: &runtime.Build{Id: "test-build"},
	}

	if !bt.stopd() {
		t.Error("expected stopd() to return true when context is canceled")
	}
}

func TestBuildTask_stop_NilCancel(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}

	// Should not panic with nil cancel
	bt.stop()
}

func TestBuildTask_stop_WithCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bt := &BuildTask{
		ctx:   ctx,
		cncl:  cancel,
		build: &runtime.Build{Id: "test-build"},
	}

	bt.stop()

	// After stop(), context should be done
	select {
	case <-ctx.Done():
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("expected context to be canceled after stop()")
	}
}

func TestBuildTask_Cancel_NilCancel(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}

	// Should not panic with nil cancel
	bt.Cancel()
	if bt.ctrlendtm.IsZero() {
		t.Error("expected ctrlendtm to be set")
	}
}

func TestBuildTask_Cancel_WithCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bt := &BuildTask{
		ctx:   ctx,
		cncl:  cancel,
		build: &runtime.Build{Id: "test-build"},
	}

	bt.Cancel()

	// ctrlendtm should be set
	if bt.ctrlendtm.IsZero() {
		t.Error("expected ctrlendtm to be set")
	}

	// Context should be canceled
	select {
	case <-ctx.Done():
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("expected context to be canceled after Cancel()")
	}
}

func TestBuildTask_WorkProgress_Default(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}

	if bt.WorkProgress() != 0 {
		t.Errorf("expected default work progress 0, got %d", bt.WorkProgress())
	}
}

func TestBuildTask_GetJob_EmptyID(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
		jobs:  make(map[string]*jobSync),
	}

	job, ok := bt.GetJob("")
	if ok {
		t.Error("expected GetJob with empty ID to return false")
	}
	if job != nil {
		t.Error("expected GetJob with empty ID to return nil job")
	}
}

func TestBuildTask_GetJob_NotFound(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
		jobs:  make(map[string]*jobSync),
	}

	job, ok := bt.GetJob("nonexistent")
	if ok {
		t.Error("expected GetJob with nonexistent ID to return false")
	}
	if job != nil {
		t.Error("expected GetJob with nonexistent ID to return nil job")
	}
}

func TestBuildTask_GetJob_Found(t *testing.T) {
	jobID := "job-1"
	expectedJob := &jobSync{
		step:  &runtime.Step{Id: jobID},
		cmdmp: make(map[string]*cmdSync),
	}

	bt := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
		jobs: map[string]*jobSync{
			jobID: expectedJob,
		},
	}

	job, ok := bt.GetJob(jobID)
	if !ok {
		t.Error("expected GetJob to find the job")
	}
	if job != expectedJob {
		t.Error("expected GetJob to return the same job instance")
	}
}
