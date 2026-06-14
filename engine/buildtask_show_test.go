package engine

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

// --- BuildTask.Show ---

func TestShow_StoppedTask(t *testing.T) {
	// Task with nil context is considered stopped
	bt := &BuildTask{
		build:  &runtime.Build{Id: "b1"},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	show, ok := bt.Show()
	if ok {
		t.Error("Show() should return false for stopped task")
	}
	if show != nil {
		t.Error("Show() should return nil for stopped task")
	}
}

func TestShow_ActiveTask_NoStages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{
			Id:         "build-show-1",
			PipelineId: "pipe-1",
			Status:     common.BuildStatusRunning,
			Error:      "",
			Event:      "test-event",
		},
		ctx:    ctx,
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	show, ok := bt.Show()
	if !ok {
		t.Fatal("Show() should return true for active task")
	}
	if show.Id != "build-show-1" {
		t.Errorf("Id = %q, want %q", show.Id, "build-show-1")
	}
	if show.PipelineId != "pipe-1" {
		t.Errorf("PipelineId = %q, want %q", show.PipelineId, "pipe-1")
	}
	if show.Status != common.BuildStatusRunning {
		t.Errorf("Status = %q, want %q", show.Status, common.BuildStatusRunning)
	}
	if show.Event != "test-event" {
		t.Errorf("Event = %q, want %q", show.Event, "test-event")
	}
	if len(show.Stages) != 0 {
		t.Errorf("Stages should be empty, got %d", len(show.Stages))
	}
}

func TestShow_WithStagesAndSteps(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a stage with a step
	stage := &runtime.Stage{
		Id:      "stage-1",
		BuildId: "build-show-2",
		Name:    "build-stage",
		Status:  common.BuildStatusRunning,
	}
	step := &runtime.Step{
		Id:      "step-1",
		StageId: "stage-1",
		BuildId: "build-show-2",
		Name:    "compile",
		Status:  common.BuildStatusOk,
	}
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-1"},
		status: common.BuildStatusOk,
	}
	job := &jobSync{
		step:  step,
		cmdmp: map[string]*cmdSync{"cmd-1": cmd},
	}
	ts := &taskStage{
		stage: stage,
		jobs:  map[string]*jobSync{"compile": job},
	}

	bt := &BuildTask{
		build: &runtime.Build{
			Id:     "build-show-2",
			Status: common.BuildStatusRunning,
			Stages: []*runtime.Stage{stage},
		},
		ctx:    ctx,
		stages: map[string]*taskStage{"build-stage": ts},
		jobs:   map[string]*jobSync{"step-1": job},
	}
	// Set steps on the stage
	stage.Steps = []*runtime.Step{step}

	show, ok := bt.Show()
	if !ok {
		t.Fatal("Show() should return true for active task")
	}
	if len(show.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(show.Stages))
	}
	rtstg := show.Stages[0]
	if rtstg.Id != "stage-1" {
		t.Errorf("stage Id = %q, want %q", rtstg.Id, "stage-1")
	}
	if rtstg.Status != common.BuildStatusRunning {
		t.Errorf("stage Status = %q, want %q", rtstg.Status, common.BuildStatusRunning)
	}
	if len(rtstg.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(rtstg.Steps))
	}
	rtstp := rtstg.Steps[0]
	if rtstp.Id != "step-1" {
		t.Errorf("step Id = %q, want %q", rtstp.Id, "step-1")
	}
	if rtstp.Status != common.BuildStatusOk {
		t.Errorf("step Status = %q, want %q", rtstp.Status, common.BuildStatusOk)
	}
	if len(rtstp.Cmds) != 1 {
		t.Fatalf("expected 1 cmd, got %d", len(rtstp.Cmds))
	}
	if rtstp.Cmds[0].Id != "cmd-1" {
		t.Errorf("cmd Id = %q, want %q", rtstp.Cmds[0].Id, "cmd-1")
	}
}

func TestShow_MissingStageSkipped(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bt := &BuildTask{
		build: &runtime.Build{
			Id:     "build-show-3",
			Status: common.BuildStatusRunning,
			Stages: []*runtime.Stage{
				{Name: "nonexistent"},
			},
		},
		ctx:    ctx,
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	show, ok := bt.Show()
	if !ok {
		t.Fatal("Show() should return true")
	}
	if len(show.Stages) != 0 {
		t.Errorf("missing stages should be skipped, got %d", len(show.Stages))
	}
}

// --- BuildTask.UpJobCmd (switch cases) ---

func TestUpJobCmd_RunningStatus(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	startTime := time.Now().Add(-time.Second)
	cmd := &cmdSync{
		cmd:     &runners.CmdContent{Id: "cmd-1"},
		status:  common.BuildStatusPending,
		started: startTime,
	}
	// fs=1 means running
	bt.UpJobCmd(cmd, 1, 0)
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", cmd.status, common.BuildStatusRunning)
	}
	// started should be updated to a recent time
	if time.Since(cmd.started) > time.Second {
		t.Error("started time should be updated")
	}
}

func TestUpJobCmd_OkStatus_ZeroCode(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-2"},
		status: common.BuildStatusRunning,
	}
	// fs=2 means ok
	bt.UpJobCmd(cmd, 2, 0)
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", cmd.status, common.BuildStatusOk)
	}
	if cmd.code != 0 {
		t.Errorf("code = %d, want 0", cmd.code)
	}
	if cmd.finished.IsZero() {
		t.Error("finished should be set")
	}
}

func TestUpJobCmd_OkStatus_NonZeroCode(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-3"},
		status: common.BuildStatusRunning,
	}
	// fs=2 with non-zero code should become error
	bt.UpJobCmd(cmd, 2, 1)
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", cmd.status, common.BuildStatusError)
	}
	if cmd.code != 1 {
		t.Errorf("code = %d, want 1", cmd.code)
	}
}

func TestUpJobCmd_CancelStatus(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-4"},
		status: common.BuildStatusRunning,
	}
	// fs=3 means cancel
	bt.UpJobCmd(cmd, 3, 137)
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusCancel {
		t.Errorf("status = %q, want %q", cmd.status, common.BuildStatusCancel)
	}
	if cmd.code != 137 {
		t.Errorf("code = %d, want 137", cmd.code)
	}
	if cmd.finished.IsZero() {
		t.Error("finished should be set for canceled cmd")
	}
}

func TestUpJobCmd_ErrorStatus(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-5"},
		status: common.BuildStatusRunning,
	}
	// fs=-1 means error
	bt.UpJobCmd(cmd, -1, 127)
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", cmd.status, common.BuildStatusError)
	}
	if cmd.code != 127 {
		t.Errorf("code = %d, want 127", cmd.code)
	}
	if cmd.finished.IsZero() {
		t.Error("finished should be set for errored cmd")
	}
}

func TestUpJobCmd_UnknownFs(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-6"},
		status: common.BuildStatusRunning,
		code:   42,
	}
	// fs=99 is unknown, should be a no-op (default branch returns)
	bt.UpJobCmd(cmd, 99, 0)
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusRunning {
		t.Errorf("status should remain %q for unknown fs, got %q", common.BuildStatusRunning, cmd.status)
	}
	if cmd.code != 42 {
		t.Errorf("code should remain 42, got %d", cmd.code)
	}
}

// --- BuildTask.UpJob (expanded) ---

func TestUpJob_SetsStatusAndCode(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-1", Name: "test"},
		cmdmp: make(map[string]*cmdSync),
	}
	bt.UpJob(job, common.BuildStatusError, "something failed", 1)
	job.RLock()
	defer job.RUnlock()
	if job.step.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", job.step.Status, common.BuildStatusError)
	}
	if job.step.Error != "something failed" {
		t.Errorf("error = %q, want %q", job.step.Error, "something failed")
	}
	if job.step.ExitCode != 1 {
		t.Errorf("exit code = %d, want 1", job.step.ExitCode)
	}
}

func TestUpJob_NilJob(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	// Should not panic
	bt.UpJob(nil, common.BuildStatusOk, "", 0)
}

func TestUpJob_EmptyStatusNoOp(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-1", Name: "test", Status: common.BuildStatusRunning},
		cmdmp: make(map[string]*cmdSync),
	}
	// Empty status is a no-op
	bt.UpJob(job, "", "", 0)
	job.RLock()
	defer job.RUnlock()
	if job.step.Status != common.BuildStatusRunning {
		t.Errorf("status should remain %q, got %q", common.BuildStatusRunning, job.step.Status)
	}
}
