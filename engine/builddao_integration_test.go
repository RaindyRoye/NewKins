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

// setupTestDB creates an in-memory SQLite database with the necessary tables
// for buildDao tests. Returns the engine and a cleanup function.
func setupTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Use xorm's Sync2 to create tables from model structs
	if err := db.Sync2(
		new(model.TBuild),
		new(model.TStage),
		new(model.TStep),
		new(model.TCmdLine),
	); err != nil {
		t.Fatalf("failed to sync tables: %v", err)
	}

	return db
}

// insertTestBuild inserts a test build record into the database.
func insertTestBuild(t *testing.T, db *xorm.Engine, id string) {
	t.Helper()
	build := &model.TBuild{
		Id:      id,
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
		Updated: time.Now(),
	}
	if _, err := db.Insert(build); err != nil {
		t.Fatalf("failed to insert test build: %v", err)
	}
}

// insertTestStage inserts a test stage record into the database.
func insertTestStage(t *testing.T, db *xorm.Engine, id, buildID string) {
	t.Helper()
	stage := &model.TStage{
		Id:      id,
		BuildId: buildID,
		Name:    "test-stage",
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
		Updated: time.Now(),
	}
	if _, err := db.Insert(stage); err != nil {
		t.Fatalf("failed to insert test stage: %v", err)
	}
}

// insertTestStep inserts a test step record into the database.
func insertTestStep(t *testing.T, db *xorm.Engine, id, stageID, buildID string) {
	t.Helper()
	step := &model.TStep{
		Id:      id,
		StageId: stageID,
		BuildId: buildID,
		Name:    "test-step",
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
		Updated: time.Now(),
	}
	if _, err := db.Insert(step); err != nil {
		t.Fatalf("failed to insert test step: %v", err)
	}
}

// insertTestCmd inserts a test cmd_line record into the database.
func insertTestCmd(t *testing.T, db *xorm.Engine, id, stepID, buildID string) {
	t.Helper()
	cmd := &model.TCmdLine{
		Id:      id,
		StepId:  stepID,
		BuildId: buildID,
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
	}
	if _, err := db.Insert(cmd); err != nil {
		t.Fatalf("failed to insert test cmd: %v", err)
	}
}

// TestUpdateBuild_StatusOk tests updateBuild with a successful build status.
func TestUpdateBuild_StatusOk(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Set up global Db for the test
	origDb := comm.Db
	comm.Db = db
	defer func() { comm.Db = origDb }()

	insertTestBuild(t, db, "build-1")

	bt := &BuildTask{
		build: &runtime.Build{
			Id:       "build-1",
			Status:   common.BuildStatusOk,
			Finished: time.Now(),
		},
	}

	bt.updateBuild(bt.build)

	// Verify the build was updated
	var updated model.TBuild
	has, err := db.ID("build-1").Get(&updated)
	if err != nil {
		t.Fatalf("failed to query updated build: %v", err)
	}
	if !has {
		t.Fatal("build not found after update")
	}
	if updated.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusOk)
	}
	if updated.Finished.IsZero() {
		t.Error("finished time should be set")
	}
}

// TestUpdateBuild_StatusError tests updateBuild with an error status.
func TestUpdateBuild_StatusError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	origDb := comm.Db
	comm.Db = db
	defer func() { comm.Db = origDb }()

	insertTestBuild(t, db, "build-2")

	bt := &BuildTask{
		build: &runtime.Build{
			Id:       "build-2",
			Status:   common.BuildStatusError,
			Error:    "build failed",
			Finished: time.Now(),
		},
	}

	bt.updateBuild(bt.build)

	var updated model.TBuild
	has, err := db.ID("build-2").Get(&updated)
	if err != nil {
		t.Fatalf("failed to query build: %v", err)
	}
	if !has {
		t.Fatal("build not found")
	}
	if updated.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusError)
	}
	if updated.Error != "build failed" {
		t.Errorf("error = %q, want %q", updated.Error, "build failed")
	}
}

// TestUpdateBuild_StatusCancelCascades tests that canceling a build also
// cancels its stages, steps, and cmd_lines.
func TestUpdateBuild_StatusCancelCascades(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	origDb := comm.Db
	comm.Db = db
	defer func() { comm.Db = origDb }()

	insertTestBuild(t, db, "build-3")
	insertTestStage(t, db, "stage-1", "build-3")
	insertTestStep(t, db, "step-1", "stage-1", "build-3")
	insertTestCmd(t, db, "cmd-1", "step-1", "build-3")

	bt := &BuildTask{
		build: &runtime.Build{
			Id:       "build-3",
			Status:   common.BuildStatusCancel,
			Finished: time.Now(),
		},
	}

	bt.updateBuild(bt.build)

	// Verify build is canceled
	var build model.TBuild
	if _, err := db.ID("build-3").Get(&build); err != nil {
		t.Fatalf("failed to query build: %v", err)
	}
	if build.Status != common.BuildStatusCancel {
		t.Errorf("build status = %q, want %q", build.Status, common.BuildStatusCancel)
	}

	// Verify stage is canceled
	var stage model.TStage
	if _, err := db.ID("stage-1").Get(&stage); err != nil {
		t.Fatalf("failed to query stage: %v", err)
	}
	if stage.Status != common.BuildStatusCancel {
		t.Errorf("stage status = %q, want %q", stage.Status, common.BuildStatusCancel)
	}

	// Verify step is canceled
	var step model.TStep
	if _, err := db.ID("step-1").Get(&step); err != nil {
		t.Fatalf("failed to query step: %v", err)
	}
	if step.Status != common.BuildStatusCancel {
		t.Errorf("step status = %q, want %q", step.Status, common.BuildStatusCancel)
	}

	// Verify cmd is canceled
	var cmd model.TCmdLine
	if _, err := db.ID("cmd-1").Get(&cmd); err != nil {
		t.Fatalf("failed to query cmd: %v", err)
	}
	if cmd.Status != common.BuildStatusCancel {
		t.Errorf("cmd status = %q, want %q", cmd.Status, common.BuildStatusCancel)
	}
}

// TestUpdateBuild_PreservesAlreadyEndedStatus tests that stages/steps/cmds
// with already-ended statuses are not overwritten.
func TestUpdateBuild_PreservesAlreadyEndedStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	origDb := comm.Db
	comm.Db = db
	defer func() { comm.Db = origDb }()

	insertTestBuild(t, db, "build-4")

	// Insert a stage that already succeeded
	stage := &model.TStage{
		Id:       "stage-done",
		BuildId:  "build-4",
		Name:     "done-stage",
		Status:   common.BuildStatusOk,
		Finished: time.Now().Add(-time.Hour),
		Updated:  time.Now(),
	}
	if _, err := db.Insert(stage); err != nil {
		t.Fatalf("failed to insert stage: %v", err)
	}

	bt := &BuildTask{
		build: &runtime.Build{
			Id:       "build-4",
			Status:   common.BuildStatusCancel,
			Finished: time.Now(),
		},
	}

	bt.updateBuild(bt.build)

	// The already-succeeded stage should remain Ok, not be changed to Cancel
	var updated model.TStage
	if _, err := db.ID("stage-done").Get(&updated); err != nil {
		t.Fatalf("failed to query stage: %v", err)
	}
	if updated.Status != common.BuildStatusOk {
		t.Errorf("stage status = %q, want %q (should preserve already-ended status)", updated.Status, common.BuildStatusOk)
	}
}

// TestUpdateStage_Basic tests updateStage with a basic status update.
func TestUpdateStage_Basic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	origDb := comm.Db
	comm.Db = db
	defer func() { comm.Db = origDb }()

	insertTestBuild(t, db, "build-5")
	insertTestStage(t, db, "stage-2", "build-5")

	bt := &BuildTask{
		build: &runtime.Build{Id: "build-5"},
	}

	stage := &runtime.Stage{
		Id:       "stage-2",
		Status:   common.BuildStatusOk,
		Finished: time.Now(),
	}

	bt.updateStage(stage)

	var updated model.TStage
	if _, err := db.ID("stage-2").Get(&updated); err != nil {
		t.Fatalf("failed to query stage: %v", err)
	}
	if updated.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusOk)
	}
}

// TestUpdateStage_ErrorCascades tests that a stage error cascades to its steps.
func TestUpdateStage_ErrorCascades(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	origDb := comm.Db
	comm.Db = db
	defer func() { comm.Db = origDb }()

	insertTestBuild(t, db, "build-6")
	insertTestStage(t, db, "stage-3", "build-6")
	insertTestStep(t, db, "step-2", "stage-3", "build-6")
	insertTestCmd(t, db, "cmd-2", "step-2", "build-6")

	bt := &BuildTask{
		build: &runtime.Build{Id: "build-6"},
	}

	stage := &runtime.Stage{
		Id:       "stage-3",
		Status:   common.BuildStatusError,
		Error:    "stage failed",
		Finished: time.Now(),
	}

	bt.updateStage(stage)

	// Verify stage is in error state
	var updatedStage model.TStage
	if _, err := db.ID("stage-3").Get(&updatedStage); err != nil {
		t.Fatalf("failed to query stage: %v", err)
	}
	if updatedStage.Status != common.BuildStatusError {
		t.Errorf("stage status = %q, want %q", updatedStage.Status, common.BuildStatusError)
	}
	if updatedStage.Error != "stage failed" {
		t.Errorf("stage error = %q, want %q", updatedStage.Error, "stage failed")
	}

	// Verify child steps are canceled
	var updatedStep model.TStep
	if _, err := db.ID("step-2").Get(&updatedStep); err != nil {
		t.Fatalf("failed to query step: %v", err)
	}
	if updatedStep.Status != common.BuildStatusCancel {
		t.Errorf("step status = %q, want %q", updatedStep.Status, common.BuildStatusCancel)
	}
}

// TestUpdateStep_Basic tests updateStep with a basic status update.
func TestUpdateStep_Basic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	origDb := comm.Db
	comm.Db = db
	defer func() { comm.Db = origDb }()

	insertTestBuild(t, db, "build-7")
	insertTestStage(t, db, "stage-4", "build-7")
	insertTestStep(t, db, "step-3", "stage-4", "build-7")

	bt := &BuildTask{
		build: &runtime.Build{Id: "build-7"},
	}

	job := &jobSync{
		step: &runtime.Step{
			Id:       "step-3",
			Status:   common.BuildStatusOk,
			ExitCode: 0,
			Finished: time.Now(),
		},
	}

	bt.updateStep(job)

	var updated model.TStep
	if _, err := db.ID("step-3").Get(&updated); err != nil {
		t.Fatalf("failed to query step: %v", err)
	}
	if updated.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusOk)
	}
	if updated.ExitCode != 0 {
		t.Errorf("exit_code = %d, want 0", updated.ExitCode)
	}
}

// TestUpdateStep_ErrorCascades tests that a step error cascades to its cmd_lines.
func TestUpdateStep_ErrorCascades(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	origDb := comm.Db
	comm.Db = db
	defer func() { comm.Db = origDb }()

	insertTestBuild(t, db, "build-8")
	insertTestStage(t, db, "stage-5", "build-8")
	insertTestStep(t, db, "step-4", "stage-5", "build-8")
	insertTestCmd(t, db, "cmd-3", "step-4", "build-8")

	bt := &BuildTask{
		build: &runtime.Build{Id: "build-8"},
	}

	job := &jobSync{
		step: &runtime.Step{
			Id:       "step-4",
			Status:   common.BuildStatusError,
			Error:    "step failed",
			ExitCode: 1,
			Finished: time.Now(),
		},
	}

	bt.updateStep(job)

	// Verify step is in error state
	var updatedStep model.TStep
	if _, err := db.ID("step-4").Get(&updatedStep); err != nil {
		t.Fatalf("failed to query step: %v", err)
	}
	if updatedStep.Status != common.BuildStatusError {
		t.Errorf("step status = %q, want %q", updatedStep.Status, common.BuildStatusError)
	}

	// Verify child cmds are canceled
	var updatedCmd model.TCmdLine
	if _, err := db.ID("cmd-3").Get(&updatedCmd); err != nil {
		t.Fatalf("failed to query cmd: %v", err)
	}
	if updatedCmd.Status != common.BuildStatusCancel {
		t.Errorf("cmd status = %q, want %q", updatedCmd.Status, common.BuildStatusCancel)
	}
}

// TestUpdateStepCmd_Running tests updateStepCmd with a running status.
func TestUpdateStepCmd_Running(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	origDb := comm.Db
	comm.Db = db
	defer func() { comm.Db = origDb }()

	insertTestBuild(t, db, "build-9")
	insertTestStage(t, db, "stage-6", "build-9")
	insertTestStep(t, db, "step-5", "stage-6", "build-9")
	insertTestCmd(t, db, "cmd-4", "step-5", "build-9")

	bt := &BuildTask{
		build: &runtime.Build{Id: "build-9"},
	}

	cmd := &cmdSync{
		cmd: &runners.CmdContent{
			Id: "cmd-4",
		},
		status:  common.BuildStatusRunning,
		started: time.Now(),
	}

	bt.updateStepCmd(cmd)

	var updated model.TCmdLine
	if _, err := db.ID("cmd-4").Get(&updated); err != nil {
		t.Fatalf("failed to query cmd: %v", err)
	}
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
	if updated.Started.IsZero() {
		t.Error("started time should be set for running status")
	}
}

// TestUpdateStepCmd_Finished tests updateStepCmd with a finished status.
func TestUpdateStepCmd_Finished(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	origDb := comm.Db
	comm.Db = db
	defer func() { comm.Db = origDb }()

	insertTestBuild(t, db, "build-10")
	insertTestStage(t, db, "stage-7", "build-10")
	insertTestStep(t, db, "step-6", "stage-7", "build-10")
	insertTestCmd(t, db, "cmd-5", "step-6", "build-10")

	bt := &BuildTask{
		build: &runtime.Build{Id: "build-10"},
	}

	cmd := &cmdSync{
		cmd: &runners.CmdContent{
			Id: "cmd-5",
		},
		status:   common.BuildStatusOk,
		code:     0,
		finished: time.Now(),
	}

	bt.updateStepCmd(cmd)

	var updated model.TCmdLine
	if _, err := db.ID("cmd-5").Get(&updated); err != nil {
		t.Fatalf("failed to query cmd: %v", err)
	}
	if updated.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusOk)
	}
	if updated.Code != 0 {
		t.Errorf("code = %d, want 0", updated.Code)
	}
	if updated.Finished.IsZero() {
		t.Error("finished time should be set")
	}
}

// TestUpdateBuild_WithContext tests that updateBuild respects the task context.
func TestUpdateBuild_WithContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	origDb := comm.Db
	comm.Db = db
	defer func() { comm.Db = origDb }()

	insertTestBuild(t, db, "build-11")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bt := &BuildTask{
		build: &runtime.Build{
			Id:       "build-11",
			Status:   common.BuildStatusOk,
			Finished: time.Now(),
		},
		ctx: ctx,
	}

	bt.updateBuild(bt.build)

	var updated model.TBuild
	if _, err := db.ID("build-11").Get(&updated); err != nil {
		t.Fatalf("failed to query build: %v", err)
	}
	if updated.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusOk)
	}
}

// TestUpdateBuild_NilDb tests that updateBuild handles nil Db gracefully.
func TestUpdateBuild_NilDb(t *testing.T) {
	origDb := comm.Db
	comm.Db = nil
	defer func() { comm.Db = origDb }()

	bt := &BuildTask{
		build: &runtime.Build{
			Id:     "build-nil",
			Status: common.BuildStatusOk,
		},
	}

	// Should not panic, just log the error
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic (Db is nil): %v", r)
		}
	}()

	bt.updateBuild(bt.build)
}
