package engine

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/runner/runners"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupBuildDaoDB creates an isolated in-memory SQLite DB for buildDao tests.
func setupBuildDaoDB(t *testing.T) *xorm.Engine {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	oldDb := comm.Db
	comm.Db = eng
	t.Cleanup(func() {
		comm.Db = oldDb
		_ = eng.Close()
	})
	if err := eng.Sync2(&model.TBuild{}, &model.TStage{}, &model.TStep{}, &model.TCmdLine{}); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// newTestBuildTask creates a BuildTask with an active context and the given build.
func newTestBuildTask(ctx context.Context, bd *runtime.Build) *BuildTask {
	ctx, cncl := context.WithCancel(ctx)
	return &BuildTask{
		build: bd,
		ctx:   ctx,
		cncl:  cncl,
	}
}

// --- updateBuild ---

func TestUpdateBuild_RunningStatus(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	bd := &runtime.Build{Id: "build-1", Status: common.BuildStatusRunning, Started: now}
	// Insert initial record
	_, err := eng.Insert(&model.TBuild{Id: bd.Id, Status: common.BuildStatusPending, Created: now})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	bt := newTestBuildTask(context.Background(), bd)
	bt.updateBuild(bd)

	// Verify update
	got := &model.TBuild{}
	ok, err := eng.Where("id=?", bd.Id).Get(got)
	if err != nil || !ok {
		t.Fatalf("query: %v, found=%v", err, ok)
	}
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
	if got.Started.IsZero() {
		t.Error("Started should be set")
	}
}

func TestUpdateBuild_EndedCascadesCancel(t *testing.T) {
	eng := setupBuildDaoDB(t)
	bd := &runtime.Build{Id: "build-end", Status: common.BuildStatusError, Error: "failed"}
	now := time.Now()

	// Insert build, stage, step, cmd with non-ended statuses
	_, _ = eng.Insert(&model.TBuild{Id: bd.Id, Status: common.BuildStatusRunning, Created: now})
	_, _ = eng.Insert(&model.TStage{Id: "stage-1", BuildId: bd.Id, Status: common.BuildStatusRunning, Created: now})
	_, _ = eng.Insert(&model.TStep{Id: "step-1", BuildId: bd.Id, StageId: "stage-1", Status: common.BuildStatusRunning, Created: now})
	_, _ = eng.Insert(&model.TCmdLine{Id: "cmd-1", BuildId: bd.Id, StepId: "step-1", Status: common.BuildStatusRunning, Created: now})

	bt := newTestBuildTask(context.Background(), bd)
	bt.updateBuild(bd)

	// Build should be updated
	gotBuild := &model.TBuild{}
	_, _ = eng.Where("id=?", bd.Id).Get(gotBuild)
	if gotBuild.Status != common.BuildStatusError {
		t.Errorf("build status = %q, want %q", gotBuild.Status, common.BuildStatusError)
	}

	// Stage should be canceled (non-ended status)
	gotStage := &model.TStage{}
	_, _ = eng.Where("id=?", "stage-1").Get(gotStage)
	if gotStage.Status != common.BuildStatusCancel {
		t.Errorf("stage status = %q, want %q", gotStage.Status, common.BuildStatusCancel)
	}

	// Step should be canceled
	gotStep := &model.TStep{}
	_, _ = eng.Where("id=?", "step-1").Get(gotStep)
	if gotStep.Status != common.BuildStatusCancel {
		t.Errorf("step status = %q, want %q", gotStep.Status, common.BuildStatusCancel)
	}

	// Cmd should be canceled
	gotCmd := &model.TCmdLine{}
	_, _ = eng.Where("id=?", "cmd-1").Get(gotCmd)
	if gotCmd.Status != common.BuildStatusCancel {
		t.Errorf("cmd status = %q, want %q", gotCmd.Status, common.BuildStatusCancel)
	}
}

func TestUpdateBuild_OkStatus_PreservesOkChildren(t *testing.T) {
	eng := setupBuildDaoDB(t)
	bd := &runtime.Build{Id: "build-ok", Status: common.BuildStatusOk}
	now := time.Now()

	_, _ = eng.Insert(&model.TBuild{Id: bd.Id, Status: common.BuildStatusRunning, Created: now})
	_, _ = eng.Insert(&model.TStage{Id: "stage-ok", BuildId: bd.Id, Status: common.BuildStatusOk, Created: now})
	_, _ = eng.Insert(&model.TStage{Id: "stage-pending", BuildId: bd.Id, Status: common.BuildStatusPending, Created: now})

	bt := newTestBuildTask(context.Background(), bd)
	bt.updateBuild(bd)

	// OK stage should remain OK
	got := &model.TStage{}
	_, _ = eng.Where("id=?", "stage-ok").Get(got)
	if got.Status != common.BuildStatusOk {
		t.Errorf("ok stage should remain %q, got %q", common.BuildStatusOk, got.Status)
	}

	// Pending stage should be canceled
	gotP := &model.TStage{}
	_, _ = eng.Where("id=?", "stage-pending").Get(gotP)
	if gotP.Status != common.BuildStatusCancel {
		t.Errorf("pending stage should be canceled, got %q", gotP.Status)
	}
}

func TestUpdateBuild_CancelStatus(t *testing.T) {
	eng := setupBuildDaoDB(t)
	bd := &runtime.Build{Id: "build-cancel", Status: common.BuildStatusCancel}
	now := time.Now()

	_, _ = eng.Insert(&model.TBuild{Id: bd.Id, Status: common.BuildStatusRunning, Created: now})

	bt := newTestBuildTask(context.Background(), bd)
	bt.updateBuild(bd)

	got := &model.TBuild{}
	_, _ = eng.Where("id=?", bd.Id).Get(got)
	if got.Status != common.BuildStatusCancel {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusCancel)
	}
}

// --- updateStage ---

func TestUpdateStage_RunningStatus(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	stage := &runtime.Stage{Id: "stage-r", Status: common.BuildStatusRunning, Started: now}

	_, _ = eng.Insert(&model.TStage{Id: stage.Id, BuildId: "build-1", Status: common.BuildStatusPending, Created: now})

	bt := newTestBuildTask(context.Background(), &runtime.Build{Id: "build-1"})
	bt.updateStage(stage)

	got := &model.TStage{}
	_, _ = eng.Where("id=?", stage.Id).Get(got)
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
	if got.Started.IsZero() {
		t.Error("Started should be set")
	}
}

func TestUpdateStage_EndedCascadesCancelToSteps(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	stage := &runtime.Stage{Id: "stage-end", Status: common.BuildStatusError, Error: "oops", Finished: now}

	_, _ = eng.Insert(&model.TStage{Id: stage.Id, BuildId: "build-1", Status: common.BuildStatusRunning, Created: now})
	_, _ = eng.Insert(&model.TStep{Id: "step-r", StageId: stage.Id, Status: common.BuildStatusRunning, Created: now})
	_, _ = eng.Insert(&model.TStep{Id: "step-ok", StageId: stage.Id, Status: common.BuildStatusOk, Created: now})

	bt := newTestBuildTask(context.Background(), &runtime.Build{Id: "build-1"})
	bt.updateStage(stage)

	// Running step should be canceled
	got := &model.TStep{}
	_, _ = eng.Where("id=?", "step-r").Get(got)
	if got.Status != common.BuildStatusCancel {
		t.Errorf("running step status = %q, want %q", got.Status, common.BuildStatusCancel)
	}

	// OK step should remain OK
	gotOk := &model.TStep{}
	_, _ = eng.Where("id=?", "step-ok").Get(gotOk)
	if gotOk.Status != common.BuildStatusOk {
		t.Errorf("ok step status = %q, want %q", gotOk.Status, common.BuildStatusOk)
	}
}

func TestUpdateStage_WithEvent(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	stage := &runtime.Stage{Id: "stage-evt", Status: common.BuildStatusOk, Event: "completed"}

	_, _ = eng.Insert(&model.TStage{Id: stage.Id, BuildId: "build-1", Status: common.BuildStatusRunning, Created: now})

	bt := newTestBuildTask(context.Background(), &runtime.Build{Id: "build-1"})
	bt.updateStage(stage)

	got := &model.TStage{}
	_, _ = eng.Where("id=?", stage.Id).Get(got)
	if got.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusOk)
	}
}

// --- updateStep ---

func TestUpdateStep_RunningStatus(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	step := &runtime.Step{Id: "step-r", Status: common.BuildStatusRunning, Started: now, ExitCode: 0}
	job := &jobSync{
		step: step,
	}

	_, _ = eng.Insert(&model.TStep{Id: step.Id, BuildId: "build-1", StageId: "stage-1", Status: common.BuildStatusPending, Created: now})

	bt := newTestBuildTask(context.Background(), &runtime.Build{Id: "build-1"})
	bt.updateStep(job)

	got := &model.TStep{}
	_, _ = eng.Where("id=?", step.Id).Get(got)
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStep_EndedCascadesCancelToCmds(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	step := &runtime.Step{Id: "step-end", Status: common.BuildStatusError, Error: "failed", ExitCode: 1}
	job := &jobSync{
		step: step,
	}

	_, _ = eng.Insert(&model.TStep{Id: step.Id, BuildId: "build-1", StageId: "stage-1", Status: common.BuildStatusRunning, Created: now})
	_, _ = eng.Insert(&model.TCmdLine{Id: "cmd-r", StepId: step.Id, Status: common.BuildStatusRunning, Created: now})
	_, _ = eng.Insert(&model.TCmdLine{Id: "cmd-ok", StepId: step.Id, Status: common.BuildStatusOk, Created: now})

	bt := newTestBuildTask(context.Background(), &runtime.Build{Id: "build-1"})
	bt.updateStep(job)

	// Running cmd should be canceled
	got := &model.TCmdLine{}
	_, _ = eng.Where("id=?", "cmd-r").Get(got)
	if got.Status != common.BuildStatusCancel {
		t.Errorf("running cmd status = %q, want %q", got.Status, common.BuildStatusCancel)
	}

	// OK cmd should remain OK
	gotOk := &model.TCmdLine{}
	_, _ = eng.Where("id=?", "cmd-ok").Get(gotOk)
	if gotOk.Status != common.BuildStatusOk {
		t.Errorf("ok cmd status = %q, want %q", gotOk.Status, common.BuildStatusOk)
	}
}

func TestUpdateStep_WithEvent(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	step := &runtime.Step{Id: "step-evt", Status: common.BuildStatusOk, Event: "done", ExitCode: 0}
	job := &jobSync{step: step}

	_, _ = eng.Insert(&model.TStep{Id: step.Id, BuildId: "build-1", StageId: "stage-1", Status: common.BuildStatusRunning, Created: now})

	bt := newTestBuildTask(context.Background(), &runtime.Build{Id: "build-1"})
	bt.updateStep(job)

	got := &model.TStep{}
	_, _ = eng.Where("id=?", step.Id).Get(got)
	if got.Event != "done" {
		t.Errorf("event = %q, want %q", got.Event, "done")
	}
}

// --- updateStepCmd ---

func TestUpdateStepCmd_Running(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	cmd := &cmdSync{
		cmd:     &runners.CmdContent{Id: "cmd-run"},
		status:  common.BuildStatusRunning,
		started: now,
	}

	_, _ = eng.Insert(&model.TCmdLine{Id: cmd.cmd.Id, BuildId: "build-1", StepId: "step-1", Status: common.BuildStatusPending, Created: now})

	bt := newTestBuildTask(context.Background(), &runtime.Build{Id: "build-1"})
	bt.updateStepCmd(cmd)

	got := &model.TCmdLine{}
	_, _ = eng.Where("id=?", cmd.cmd.Id).Get(got)
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
	if got.Started.IsZero() {
		t.Error("Started should be set for running cmd")
	}
}

func TestUpdateStepCmd_Ok(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	cmd := &cmdSync{
		cmd:      &runners.CmdContent{Id: "cmd-ok"},
		status:   common.BuildStatusOk,
		finished: now,
	}

	_, _ = eng.Insert(&model.TCmdLine{Id: cmd.cmd.Id, BuildId: "build-1", StepId: "step-1", Status: common.BuildStatusRunning, Created: now})

	bt := newTestBuildTask(context.Background(), &runtime.Build{Id: "build-1"})
	bt.updateStepCmd(cmd)

	got := &model.TCmdLine{}
	_, _ = eng.Where("id=?", cmd.cmd.Id).Get(got)
	if got.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusOk)
	}
	if got.Finished.IsZero() {
		t.Error("Finished should be set for ok cmd")
	}
}

func TestUpdateStepCmd_Error(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	cmd := &cmdSync{
		cmd:      &runners.CmdContent{Id: "cmd-err"},
		status:   common.BuildStatusError,
		code:     1,
		finished: now,
	}

	_, _ = eng.Insert(&model.TCmdLine{Id: cmd.cmd.Id, BuildId: "build-1", StepId: "step-1", Status: common.BuildStatusRunning, Created: now})

	bt := newTestBuildTask(context.Background(), &runtime.Build{Id: "build-1"})
	bt.updateStepCmd(cmd)

	got := &model.TCmdLine{}
	_, _ = eng.Where("id=?", cmd.cmd.Id).Get(got)
	if got.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusError)
	}
	if got.Code != 1 {
		t.Errorf("code = %d, want 1", got.Code)
	}
	if got.Finished.IsZero() {
		t.Error("Finished should be set for errored cmd")
	}
}

func TestUpdateStepCmd_Cancel(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	cmd := &cmdSync{
		cmd:      &runners.CmdContent{Id: "cmd-cancel"},
		status:   common.BuildStatusCancel,
		code:     137,
		finished: now,
	}

	_, _ = eng.Insert(&model.TCmdLine{Id: cmd.cmd.Id, BuildId: "build-1", StepId: "step-1", Status: common.BuildStatusRunning, Created: now})

	bt := newTestBuildTask(context.Background(), &runtime.Build{Id: "build-1"})
	bt.updateStepCmd(cmd)

	got := &model.TCmdLine{}
	_, _ = eng.Where("id=?", cmd.cmd.Id).Get(got)
	if got.Status != common.BuildStatusCancel {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusCancel)
	}
}

// --- updateBuild with nil DB (recover behavior) ---

func TestUpdateBuild_NilDbNoPanic(t *testing.T) {
	origDb := comm.Db
	comm.Db = nil
	defer func() { comm.Db = origDb }()

	bt := &BuildTask{
		build: &runtime.Build{Id: "build-nodb", Status: common.BuildStatusRunning},
		ctx:   context.Background(),
	}
	// Should not panic (RecoverLog catches it)
	bt.updateBuild(bt.build)
}

func TestUpdateStage_NilDbNoPanic(t *testing.T) {
	origDb := comm.Db
	comm.Db = nil
	defer func() { comm.Db = origDb }()

	bt := &BuildTask{
		build: &runtime.Build{Id: "build-nodb"},
		ctx:   context.Background(),
	}
	bt.updateStage(&runtime.Stage{Id: "stage-nodb", Status: common.BuildStatusRunning})
}

func TestUpdateStep_NilDbNoPanic(t *testing.T) {
	origDb := comm.Db
	comm.Db = nil
	defer func() { comm.Db = origDb }()

	bt := &BuildTask{
		build: &runtime.Build{Id: "build-nodb"},
		ctx:   context.Background(),
	}
	bt.updateStep(&jobSync{step: &runtime.Step{Id: "step-nodb", Status: common.BuildStatusRunning}})
}

func TestUpdateStepCmd_NilDbNoPanic(t *testing.T) {
	origDb := comm.Db
	comm.Db = nil
	defer func() { comm.Db = origDb }()

	bt := &BuildTask{
		build: &runtime.Build{Id: "build-nodb"},
		ctx:   context.Background(),
	}
	bt.updateStepCmd(&cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-nodb"},
		status: common.BuildStatusRunning,
	})
}

// --- Concurrent update safety ---

func TestUpdateBuild_Concurrent(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	bd := &runtime.Build{Id: "build-conc", Status: common.BuildStatusRunning, Started: now}
	_, _ = eng.Insert(&model.TBuild{Id: bd.Id, Status: common.BuildStatusPending, Created: now})

	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			bt := newTestBuildTask(context.Background(), bd)
			bt.updateBuild(bd)
		}()
	}
	for i := 0; i < 5; i++ {
		<-done
	}

	got := &model.TBuild{}
	_, _ = eng.Where("id=?", bd.Id).Get(got)
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
}

// --- taskCtx with nil task context ---

func TestUpdateBuild_TaskCtxFallback(t *testing.T) {
	eng := setupBuildDaoDB(t)
	now := time.Now()
	bd := &runtime.Build{Id: "build-fctx", Status: common.BuildStatusRunning, Started: now}
	_, _ = eng.Insert(&model.TBuild{Id: bd.Id, Status: common.BuildStatusPending, Created: now})

	// BuildTask with nil ctx - should fall back to comm.Ctx
	bt := &BuildTask{
		build: bd,
		// ctx intentionally nil
	}
	bt.updateBuild(bd)

	got := &model.TBuild{}
	_, _ = eng.Where("id=?", bd.Id).Get(got)
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
}
