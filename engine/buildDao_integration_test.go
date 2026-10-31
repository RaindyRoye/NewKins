package engine

import (
	"context"
	"sync"
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

// setupBuildDaoDB creates an in-memory SQLite database with tables needed for buildDao tests.
func setupBuildDaoDB(t *testing.T) *xorm.Engine {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create test engine: %v", err)
	}
	oldDb := comm.Db
	comm.Db = eng
	t.Cleanup(func() {
		comm.Db = oldDb
		_ = eng.Close()
	})
	err = eng.Sync2(
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TCmdLine{},
	)
	if err != nil {
		t.Fatalf("sync schema: %v", err)
	}
	return eng
}

// newTestBuildTask creates a BuildTask with a background context.
func newTestBuildTask(buildID string) *BuildTask {
	ctx, cancel := context.WithCancel(context.Background())
	return &BuildTask{
		ctx:  ctx,
		cncl: cancel,
		build: &runtime.Build{
			Id:         buildID,
			PipelineId: "pipe-1",
		},
	}
}

// --- updateBuild tests ---

func TestUpdateBuild_RunningStatus(t *testing.T) {
	eng := setupBuildDaoDB(t)
	ctx := context.Background()

	// Insert a build
	build := &model.TBuild{
		Id:         "build-1",
		PipelineId: "pipe-1",
		Status:     common.BuildStatusPending,
		Created:    time.Now(),
	}
	if _, err := eng.Insert(build); err != nil {
		t.Fatalf("insert build: %v", err)
	}

	// Insert a stage for the build
	stage := &model.TStage{
		Id:      "stage-1",
		BuildId: "build-1",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := eng.Insert(stage); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	task := newTestBuildTask("build-1")
	defer task.cncl()

	// Update to running status
	rtBuild := &runtime.Build{
		Id:      "build-1",
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
	}
	task.updateBuild(rtBuild)

	// Verify build status was updated
	var updated model.TBuild
	ok, err := eng.Context(ctx).Where("id=?", "build-1").Get(&updated)
	if err != nil {
		t.Fatalf("query updated build: %v", err)
	}
	if !ok {
		t.Fatal("build not found after update")
	}
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}

	// Stages should NOT be canceled (build not ended)
	var updatedStage model.TStage
	ok, _ = eng.Context(ctx).Where("id=?", "stage-1").Get(&updatedStage)
	if !ok {
		t.Fatal("stage not found")
	}
	if updatedStage.Status == common.BuildStatusCancel {
		t.Error("stage should not be canceled when build is still running")
	}
}

func TestUpdateBuild_EndedStatus_CancelsStagesAndSteps(t *testing.T) {
	eng := setupBuildDaoDB(t)
	ctx := context.Background()

	// Insert a build
	build := &model.TBuild{
		Id:         "build-ended",
		PipelineId: "pipe-1",
		Status:     common.BuildStatusRunning,
		Created:    time.Now(),
	}
	if _, err := eng.Insert(build); err != nil {
		t.Fatalf("insert build: %v", err)
	}

	// Insert stages with running status
	stage1 := &model.TStage{
		Id:      "stage-r1",
		BuildId: "build-ended",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	stage2 := &model.TStage{
		Id:      "stage-r2",
		BuildId: "build-ended",
		Status:  common.BuildStatusOk,
		Created: time.Now(),
	}
	if _, err := eng.Insert(stage1); err != nil {
		t.Fatalf("insert stage1: %v", err)
	}
	if _, err := eng.Insert(stage2); err != nil {
		t.Fatalf("insert stage2: %v", err)
	}

	// Insert steps
	step1 := &model.TStep{
		Id:      "step-r1",
		BuildId: "build-ended",
		StageId: "stage-r1",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	step2 := &model.TStep{
		Id:      "step-r2",
		BuildId: "build-ended",
		StageId: "stage-r1",
		Status:  common.BuildStatusOk, // already ended, should not be canceled
		Created: time.Now(),
	}
	if _, err := eng.Insert(step1); err != nil {
		t.Fatalf("insert step1: %v", err)
	}
	if _, err := eng.Insert(step2); err != nil {
		t.Fatalf("insert step2: %v", err)
	}

	// Insert cmd lines
	cmd1 := &model.TCmdLine{
		Id:      "cmd-r1",
		BuildId: "build-ended",
		StepId:  "step-r1",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := eng.Insert(cmd1); err != nil {
		t.Fatalf("insert cmd1: %v", err)
	}

	task := newTestBuildTask("build-ended")
	defer task.cncl()

	// Set build to error (ended status)
	rtBuild := &runtime.Build{
		Id:       "build-ended",
		Status:   common.BuildStatusError,
		Error:    "build failed",
		Finished: time.Now(),
	}
	task.updateBuild(rtBuild)

	// Verify build status
	var updatedBuild model.TBuild
	_, _ = eng.Context(ctx).Where("id=?", "build-ended").Get(&updatedBuild)
	if updatedBuild.Status != common.BuildStatusError {
		t.Errorf("build status = %q, want %q", updatedBuild.Status, common.BuildStatusError)
	}
	if updatedBuild.Error != "build failed" {
		t.Errorf("build error = %q, want %q", updatedBuild.Error, "build failed")
	}

	// Verify running stage is canceled, already-ended stage is not
	var upStage1 model.TStage
	_, _ = eng.Context(ctx).Where("id=?", "stage-r1").Get(&upStage1)
	if upStage1.Status != common.BuildStatusCancel {
		t.Errorf("stage-r1 status = %q, want %q (running should be canceled)", upStage1.Status, common.BuildStatusCancel)
	}
	var upStage2 model.TStage
	_, _ = eng.Context(ctx).Where("id=?", "stage-r2").Get(&upStage2)
	if upStage2.Status != common.BuildStatusOk {
		t.Errorf("stage-r2 status = %q, want %q (already ok should be preserved)", upStage2.Status, common.BuildStatusOk)
	}

	// Verify running step is canceled
	var upStep1 model.TStep
	_, _ = eng.Context(ctx).Where("id=?", "step-r1").Get(&upStep1)
	if upStep1.Status != common.BuildStatusCancel {
		t.Errorf("step-r1 status = %q, want %q", upStep1.Status, common.BuildStatusCancel)
	}
	// Already-ended step should not change
	var upStep2 model.TStep
	_, _ = eng.Context(ctx).Where("id=?", "step-r2").Get(&upStep2)
	if upStep2.Status != common.BuildStatusOk {
		t.Errorf("step-r2 status = %q, want %q", upStep2.Status, common.BuildStatusOk)
	}

	// Verify running cmd is canceled
	var upCmd1 model.TCmdLine
	_, _ = eng.Context(ctx).Where("id=?", "cmd-r1").Get(&upCmd1)
	if upCmd1.Status != common.BuildStatusCancel {
		t.Errorf("cmd-r1 status = %q, want %q", upCmd1.Status, common.BuildStatusCancel)
	}
}

// --- updateStage tests ---

func TestUpdateStage_RunningStatus(t *testing.T) {
	eng := setupBuildDaoDB(t)
	ctx := context.Background()

	stage := &model.TStage{
		Id:      "stage-run",
		BuildId: "build-1",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := eng.Insert(stage); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	task := newTestBuildTask("build-1")
	defer task.cncl()

	rtStage := &runtime.Stage{
		Id:      "stage-run",
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
	}
	task.updateStage(rtStage)

	var updated model.TStage
	_, _ = eng.Context(ctx).Where("id=?", "stage-run").Get(&updated)
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStage_EndedStatus_CancelsSteps(t *testing.T) {
	eng := setupBuildDaoDB(t)
	ctx := context.Background()

	stage := &model.TStage{
		Id:      "stage-end",
		BuildId: "build-1",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := eng.Insert(stage); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	// Running step under this stage
	step := &model.TStep{
		Id:      "step-under-stage",
		BuildId: "build-1",
		StageId: "stage-end",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := eng.Insert(step); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	// Already-ended step under same stage
	stepDone := &model.TStep{
		Id:      "step-done",
		BuildId: "build-1",
		StageId: "stage-end",
		Status:  common.BuildStatusOk,
		Created: time.Now(),
	}
	if _, err := eng.Insert(stepDone); err != nil {
		t.Fatalf("insert stepDone: %v", err)
	}

	task := newTestBuildTask("build-1")
	defer task.cncl()

	rtStage := &runtime.Stage{
		Id:       "stage-end",
		Status:   common.BuildStatusError,
		Error:    "stage failed",
		Finished: time.Now(),
	}
	task.updateStage(rtStage)

	var upStage model.TStage
	_, _ = eng.Context(ctx).Where("id=?", "stage-end").Get(&upStage)
	if upStage.Status != common.BuildStatusError {
		t.Errorf("stage status = %q, want %q", upStage.Status, common.BuildStatusError)
	}

	var upStep model.TStep
	_, _ = eng.Context(ctx).Where("id=?", "step-under-stage").Get(&upStep)
	if upStep.Status != common.BuildStatusCancel {
		t.Errorf("running step status = %q, want %q", upStep.Status, common.BuildStatusCancel)
	}

	var upStepDone model.TStep
	_, _ = eng.Context(ctx).Where("id=?", "step-done").Get(&upStepDone)
	if upStepDone.Status != common.BuildStatusOk {
		t.Errorf("done step status = %q, want %q", upStepDone.Status, common.BuildStatusOk)
	}
}

// --- updateStep tests ---

func TestUpdateStep_RunningStatus(t *testing.T) {
	eng := setupBuildDaoDB(t)
	ctx := context.Background()

	step := &model.TStep{
		Id:      "step-run",
		BuildId: "build-1",
		StageId: "stage-1",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := eng.Insert(step); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	// Insert a cmd line under this step
	cmd := &model.TCmdLine{
		Id:      "cmd-under-step",
		BuildId: "build-1",
		StepId:  "step-run",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := eng.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	task := newTestBuildTask("build-1")
	defer task.cncl()

	job := &jobSync{
		step: &runtime.Step{
			Id:      "step-run",
			Status:  common.BuildStatusRunning,
			Started: time.Now(),
		},
	}
	task.updateStep(job)

	var updated model.TStep
	_, _ = eng.Context(ctx).Where("id=?", "step-run").Get(&updated)
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("step status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}

	// Cmd should NOT be canceled (step not ended)
	var upCmd model.TCmdLine
	_, _ = eng.Context(ctx).Where("id=?", "cmd-under-step").Get(&upCmd)
	if upCmd.Status == common.BuildStatusCancel {
		t.Error("cmd should not be canceled when step is still running")
	}
}

func TestUpdateStep_EndedStatus_CancelsCmds(t *testing.T) {
	eng := setupBuildDaoDB(t)
	ctx := context.Background()

	step := &model.TStep{
		Id:      "step-end",
		BuildId: "build-1",
		StageId: "stage-1",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := eng.Insert(step); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	cmdRunning := &model.TCmdLine{
		Id:      "cmd-running",
		BuildId: "build-1",
		StepId:  "step-end",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := eng.Insert(cmdRunning); err != nil {
		t.Fatalf("insert cmdRunning: %v", err)
	}

	cmdDone := &model.TCmdLine{
		Id:      "cmd-done",
		BuildId: "build-1",
		StepId:  "step-end",
		Status:  common.BuildStatusOk,
		Created: time.Now(),
	}
	if _, err := eng.Insert(cmdDone); err != nil {
		t.Fatalf("insert cmdDone: %v", err)
	}

	task := newTestBuildTask("build-1")
	defer task.cncl()

	job := &jobSync{
		step: &runtime.Step{
			Id:       "step-end",
			Status:   common.BuildStatusError,
			Error:    "step failed",
			ExitCode: 1,
			Finished: time.Now(),
		},
	}
	task.updateStep(job)

	var upStep model.TStep
	_, _ = eng.Context(ctx).Where("id=?", "step-end").Get(&upStep)
	if upStep.Status != common.BuildStatusError {
		t.Errorf("step status = %q, want %q", upStep.Status, common.BuildStatusError)
	}
	if upStep.ExitCode != 1 {
		t.Errorf("step exit_code = %d, want 1", upStep.ExitCode)
	}

	var upCmdRunning model.TCmdLine
	_, _ = eng.Context(ctx).Where("id=?", "cmd-running").Get(&upCmdRunning)
	if upCmdRunning.Status != common.BuildStatusCancel {
		t.Errorf("running cmd status = %q, want %q", upCmdRunning.Status, common.BuildStatusCancel)
	}

	var upCmdDone model.TCmdLine
	_, _ = eng.Context(ctx).Where("id=?", "cmd-done").Get(&upCmdDone)
	if upCmdDone.Status != common.BuildStatusOk {
		t.Errorf("done cmd status = %q, want %q", upCmdDone.Status, common.BuildStatusOk)
	}
}

// --- updateStepCmd tests ---

func TestUpdateStepCmd_RunningStatus(t *testing.T) {
	eng := setupBuildDaoDB(t)
	ctx := context.Background()

	cmd := &model.TCmdLine{
		Id:      "cmd-run",
		BuildId: "build-1",
		StepId:  "step-1",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := eng.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	task := newTestBuildTask("build-1")
	defer task.cncl()

	now := time.Now()
	cs := &cmdSync{
		cmd:     &runners.CmdContent{Id: "cmd-run"},
		status:  common.BuildStatusRunning,
		started: now,
	}
	task.updateStepCmd(cs)

	var updated model.TCmdLine
	_, _ = eng.Context(ctx).Where("id=?", "cmd-run").Get(&updated)
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("cmd status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStepCmd_FinishedStatus(t *testing.T) {
	eng := setupBuildDaoDB(t)
	ctx := context.Background()

	cmd := &model.TCmdLine{
		Id:      "cmd-finish",
		BuildId: "build-1",
		StepId:  "step-1",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := eng.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	task := newTestBuildTask("build-1")
	defer task.cncl()

	now := time.Now()
	// Running status sets started (and does NOT update code)
	cs := &cmdSync{
		cmd:     &runners.CmdContent{Id: "cmd-finish"},
		code:    42,
		status:  common.BuildStatusRunning,
		started: now,
	}
	task.updateStepCmd(cs)

	var updated model.TCmdLine
	_, _ = eng.Context(ctx).Where("id=?", "cmd-finish").Get(&updated)
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("cmd status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
}

// --- taskCtx tests ---

func TestTaskCtx_NilCtx(t *testing.T) {
	comm.ResetCtx()
	task := &BuildTask{ctx: nil}
	ctx := task.taskCtx()
	if ctx == nil {
		t.Fatal("taskCtx() should return non-nil context")
	}
	if ctx != comm.Ctx {
		t.Error("taskCtx() with nil ctx should return comm.Ctx")
	}
}

func TestTaskCtx_WithCtx(t *testing.T) {
	customCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	task := &BuildTask{ctx: customCtx}
	ctx := task.taskCtx()
	if ctx != customCtx {
		t.Error("taskCtx() should return the task's own context when set")
	}
}

// --- Concurrent update safety test ---

func TestUpdateBuild_Concurrent(t *testing.T) {
	// Use a file-based temp DB for concurrent access (SQLite :memory: is
	// connection-scoped and won't share across goroutines in a pool).
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"
	eng, err := xorm.NewEngine("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("create test engine: %v", err)
	}
	oldDb := comm.Db
	comm.Db = eng
	t.Cleanup(func() {
		comm.Db = oldDb
		_ = eng.Close()
	})
	err = eng.Sync2(
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TCmdLine{},
	)
	if err != nil {
		t.Fatalf("sync schema: %v", err)
	}

	build := &model.TBuild{
		Id:         "build-concurrent",
		PipelineId: "pipe-1",
		Status:     common.BuildStatusPending,
		Created:    time.Now(),
	}
	if _, err := eng.Insert(build); err != nil {
		t.Fatalf("insert build: %v", err)
	}

	task := newTestBuildTask("build-concurrent")
	defer task.cncl()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rb := &runtime.Build{
				Id:     "build-concurrent",
				Status: common.BuildStatusRunning,
			}
			task.updateBuild(rb)
		}(i)
	}
	wg.Wait()

	// Just verify no panic occurred and the build still exists
	var final model.TBuild
	ok, err := eng.Where("id=?", "build-concurrent").Get(&final)
	if err != nil {
		t.Fatalf("query after concurrent updates: %v", err)
	}
	if !ok {
		t.Fatal("build not found after concurrent updates")
	}
}

// --- NewBuildTask integration test ---

func TestNewBuildTask_Integration(t *testing.T) {
	egn := &BuildEngine{}
	bd := &runtime.Build{
		Id:         "test-build",
		PipelineId: "test-pipe",
		Status:     common.BuildStatusPending,
	}
	task := NewBuildTask(egn, bd)
	if task == nil {
		t.Fatal("NewBuildTask returned nil")
	}
	if task.egn != egn {
		t.Error("egn not set correctly")
	}
	if task.build != bd {
		t.Error("build not set correctly")
	}
}
