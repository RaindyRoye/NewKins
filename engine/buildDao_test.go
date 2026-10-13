package engine

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"github.com/gokins/runner/runners"
	"github.com/stretchr/testify/assert"
	"xorm.io/xorm"
)

// setupTestDb creates an in-memory SQLite database with the required tables
// and sets it as comm.Db. Returns a cleanup function that restores the original state.
func setupTestDb(t *testing.T) func() {
	origDb := comm.Db
	origCtx := comm.Ctx

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// Create tables
	_, err = db.Exec(`CREATE TABLE t_build (
		id VARCHAR(64) PRIMARY KEY,
		pipeline_id VARCHAR(64),
		pipeline_version_id VARCHAR(64),
		status VARCHAR(100),
		error VARCHAR(500),
		event VARCHAR(100),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		version VARCHAR(255)
	)`)
	assert.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE t_stage (
		id VARCHAR(64) PRIMARY KEY,
		pipeline_version_id VARCHAR(64),
		build_id VARCHAR(64),
		status VARCHAR(100),
		error VARCHAR(500),
		name VARCHAR(255),
		display_name VARCHAR(255),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		sort INT,
		stage VARCHAR(255)
	)`)
	assert.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE t_step (
		id VARCHAR(64) PRIMARY KEY,
		build_id VARCHAR(64),
		stage_id VARCHAR(100),
		display_name VARCHAR(255),
		pipeline_version_id VARCHAR(64),
		step VARCHAR(255),
		status VARCHAR(100),
		event VARCHAR(100),
		exit_code INT,
		error VARCHAR(500),
		name VARCHAR(100),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		version VARCHAR(255),
		errignore INT,
		commands TEXT,
		waits JSON,
		sort INT
	)`)
	assert.NoError(t, err)

	_, err = db.Exec(`CREATE TABLE t_cmd_line (
		id VARCHAR(64) PRIMARY KEY,
		group_id VARCHAR(64),
		build_id VARCHAR(64),
		step_id VARCHAR(64),
		status VARCHAR(50),
		num INT,
		code INT,
		content TEXT,
		created DATETIME,
		started DATETIME,
		finished DATETIME
	)`)
	assert.NoError(t, err)

	comm.Db = db
	comm.Ctx = context.Background()

	return func() {
		_ = db.Close()
		comm.Db = origDb
		comm.Ctx = origCtx
	}
}

// --- updateBuild tests ---

func TestUpdateBuild_RunningStatus(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	buildId := utils.NewXid()
	// Insert a build
	build := &model.TBuild{
		Id:      buildId,
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(build)
	assert.NoError(t, err)

	bt := &BuildTask{
		build: &runtime.Build{
			Id:      buildId,
			Status:  common.BuildStatusRunning,
			Started: time.Now(),
		},
		ctx: context.Background(),
	}

	bt.updateBuild(bt.build)

	// Verify the build was updated
	var updated model.TBuild
	has, err := comm.Db.ID(buildId).Get(&updated)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusRunning, updated.Status)
}

func TestUpdateBuild_EndedStatusCascadesToStages(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	buildId := utils.NewXid()
	stageId := utils.NewXid()

	// Insert a build
	build := &model.TBuild{
		Id:      buildId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(build)
	assert.NoError(t, err)

	// Insert a stage with pending status
	stage := &model.TStage{
		Id:      stageId,
		BuildId: buildId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = comm.Db.InsertOne(stage)
	assert.NoError(t, err)

	// Insert a step with pending status
	stepId := utils.NewXid()
	step := &model.TStep{
		Id:      stepId,
		BuildId: buildId,
		StageId: stageId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = comm.Db.InsertOne(step)
	assert.NoError(t, err)

	// Insert a cmd line
	cmdId := utils.NewXid()
	cmd := &model.TCmdLine{
		Id:      cmdId,
		BuildId: buildId,
		StepId:  stepId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = comm.Db.InsertOne(cmd)
	assert.NoError(t, err)

	bt := &BuildTask{
		build: &runtime.Build{
			Id:       buildId,
			Status:   common.BuildStatusError,
			Error:    "build failed",
			Finished: time.Now(),
		},
		ctx: context.Background(),
	}

	bt.updateBuild(bt.build)

	// Verify the build was updated
	var updatedBuild model.TBuild
	has, err := comm.Db.ID(buildId).Get(&updatedBuild)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusError, updatedBuild.Status)
	assert.Equal(t, "build failed", updatedBuild.Error)

	// Verify the stage was canceled
	var updatedStage model.TStage
	has, err = comm.Db.ID(stageId).Get(&updatedStage)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusCancel, updatedStage.Status)

	// Verify the step was canceled
	var updatedStep model.TStep
	has, err = comm.Db.ID(stepId).Get(&updatedStep)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusCancel, updatedStep.Status)

	// Verify the cmd line was canceled
	var updatedCmd model.TCmdLine
	has, err = comm.Db.ID(cmdId).Get(&updatedCmd)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusCancel, updatedCmd.Status)
}

func TestUpdateBuild_OkStatusDoesNotCascade(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	buildId := utils.NewXid()
	stageId := utils.NewXid()

	// Insert a build
	build := &model.TBuild{
		Id:      buildId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(build)
	assert.NoError(t, err)

	// Insert a stage with running status
	stage := &model.TStage{
		Id:      stageId,
		BuildId: buildId,
		Status:  common.BuildStatusOk,
		Created: time.Now(),
	}
	_, err = comm.Db.InsertOne(stage)
	assert.NoError(t, err)

	bt := &BuildTask{
		build: &runtime.Build{
			Id:       buildId,
			Status:   common.BuildStatusOk,
			Finished: time.Now(),
		},
		ctx: context.Background(),
	}

	bt.updateBuild(bt.build)

	// Verify the stage status was NOT changed (it's already Ok)
	var updatedStage model.TStage
	has, err := comm.Db.ID(stageId).Get(&updatedStage)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusOk, updatedStage.Status)
}

// --- updateStage tests ---

func TestUpdateStage_RunningStatus(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	stageId := utils.NewXid()
	stage := &model.TStage{
		Id:      stageId,
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(stage)
	assert.NoError(t, err)

	bt := &BuildTask{ctx: context.Background()}
	rtStage := &runtime.Stage{
		Id:      stageId,
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
	}

	bt.updateStage(rtStage)

	var updated model.TStage
	has, err := comm.Db.ID(stageId).Get(&updated)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusRunning, updated.Status)
}

func TestUpdateStage_EndedStatusCascadesToSteps(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	stageId := utils.NewXid()
	stepId := utils.NewXid()

	// Insert a stage
	stage := &model.TStage{
		Id:      stageId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(stage)
	assert.NoError(t, err)

	// Insert a step with running status
	step := &model.TStep{
		Id:      stepId,
		StageId: stageId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = comm.Db.InsertOne(step)
	assert.NoError(t, err)

	bt := &BuildTask{ctx: context.Background()}
	rtStage := &runtime.Stage{
		Id:       stageId,
		Status:   common.BuildStatusError,
		Error:    "stage failed",
		Finished: time.Now(),
	}

	bt.updateStage(rtStage)

	// Verify the stage was updated
	var updatedStage model.TStage
	has, err := comm.Db.ID(stageId).Get(&updatedStage)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusError, updatedStage.Status)
	assert.Equal(t, "stage failed", updatedStage.Error)

	// Verify the step was canceled
	var updatedStep model.TStep
	has, err = comm.Db.ID(stepId).Get(&updatedStep)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusCancel, updatedStep.Status)
}

func TestUpdateStage_OkStatusDoesNotCascade(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	stageId := utils.NewXid()
	stepId := utils.NewXid()

	// Insert a stage
	stage := &model.TStage{
		Id:      stageId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(stage)
	assert.NoError(t, err)

	// Insert a step with ok status (already finished)
	step := &model.TStep{
		Id:      stepId,
		StageId: stageId,
		Status:  common.BuildStatusOk,
		Created: time.Now(),
	}
	_, err = comm.Db.InsertOne(step)
	assert.NoError(t, err)

	bt := &BuildTask{ctx: context.Background()}
	rtStage := &runtime.Stage{
		Id:       stageId,
		Status:   common.BuildStatusOk,
		Finished: time.Now(),
	}

	bt.updateStage(rtStage)

	// Verify the step status was NOT changed
	var updatedStep model.TStep
	has, err := comm.Db.ID(stepId).Get(&updatedStep)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusOk, updatedStep.Status)
}

// --- updateStep tests ---

func TestUpdateStep_RunningStatus(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	stepId := utils.NewXid()
	step := &model.TStep{
		Id:      stepId,
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(step)
	assert.NoError(t, err)

	bt := &BuildTask{ctx: context.Background()}
	job := &jobSync{
		step: &runtime.Step{
			Id:      stepId,
			Status:  common.BuildStatusRunning,
			Started: time.Now(),
		},
	}

	bt.updateStep(job)

	var updated model.TStep
	has, err := comm.Db.ID(stepId).Get(&updated)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusRunning, updated.Status)
}

func TestUpdateStep_EndedStatusCascadesToCmds(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	stepId := utils.NewXid()
	cmdId := utils.NewXid()

	// Insert a step
	step := &model.TStep{
		Id:      stepId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(step)
	assert.NoError(t, err)

	// Insert a cmd line with running status
	cmd := &model.TCmdLine{
		Id:      cmdId,
		StepId:  stepId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = comm.Db.InsertOne(cmd)
	assert.NoError(t, err)

	bt := &BuildTask{ctx: context.Background()}
	job := &jobSync{
		step: &runtime.Step{
			Id:       stepId,
			Status:   common.BuildStatusError,
			Error:    "step failed",
			ExitCode: 1,
			Finished: time.Now(),
		},
	}

	bt.updateStep(job)

	// Verify the step was updated
	var updatedStep model.TStep
	has, err := comm.Db.ID(stepId).Get(&updatedStep)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusError, updatedStep.Status)
	assert.Equal(t, 1, updatedStep.ExitCode)

	// Verify the cmd line was canceled
	var updatedCmd model.TCmdLine
	has, err = comm.Db.ID(cmdId).Get(&updatedCmd)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusCancel, updatedCmd.Status)
}

func TestUpdateStep_OkStatusDoesNotCascade(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	stepId := utils.NewXid()
	cmdId := utils.NewXid()

	// Insert a step
	step := &model.TStep{
		Id:      stepId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(step)
	assert.NoError(t, err)

	// Insert a cmd line with ok status (already finished)
	cmd := &model.TCmdLine{
		Id:      cmdId,
		StepId:  stepId,
		Status:  common.BuildStatusOk,
		Created: time.Now(),
	}
	_, err = comm.Db.InsertOne(cmd)
	assert.NoError(t, err)

	bt := &BuildTask{ctx: context.Background()}
	job := &jobSync{
		step: &runtime.Step{
			Id:       stepId,
			Status:   common.BuildStatusOk,
			Finished: time.Now(),
		},
	}

	bt.updateStep(job)

	// Verify the cmd line status was NOT changed
	var updatedCmd model.TCmdLine
	has, err := comm.Db.ID(cmdId).Get(&updatedCmd)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusOk, updatedCmd.Status)
}

// --- updateStepCmd tests ---

func TestUpdateStepCmd_RunningStatus(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	cmdId := utils.NewXid()
	cmd := &model.TCmdLine{
		Id:      cmdId,
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(cmd)
	assert.NoError(t, err)

	bt := &BuildTask{ctx: context.Background()}
	cs := &cmdSync{
		cmd:     &runners.CmdContent{Id: cmdId},
		status:  common.BuildStatusRunning,
		started: time.Now(),
	}

	bt.updateStepCmd(cs)

	var updated model.TCmdLine
	has, err := comm.Db.ID(cmdId).Get(&updated)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusRunning, updated.Status)
	assert.False(t, updated.Started.IsZero())
}

func TestUpdateStepCmd_OkStatus(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	cmdId := utils.NewXid()
	cmd := &model.TCmdLine{
		Id:      cmdId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(cmd)
	assert.NoError(t, err)

	bt := &BuildTask{ctx: context.Background()}
	cs := &cmdSync{
		cmd:      &runners.CmdContent{Id: cmdId},
		status:   common.BuildStatusOk,
		code:     0,
		finished: time.Now(),
	}

	bt.updateStepCmd(cs)

	var updated model.TCmdLine
	has, err := comm.Db.ID(cmdId).Get(&updated)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusOk, updated.Status)
	assert.Equal(t, 0, updated.Code)
	assert.False(t, updated.Finished.IsZero())
}

func TestUpdateStepCmd_ErrorStatus(t *testing.T) {
	cleanup := setupTestDb(t)
	defer cleanup()

	cmdId := utils.NewXid()
	cmd := &model.TCmdLine{
		Id:      cmdId,
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(cmd)
	assert.NoError(t, err)

	bt := &BuildTask{ctx: context.Background()}
	cs := &cmdSync{
		cmd:      &runners.CmdContent{Id: cmdId},
		status:   common.BuildStatusError,
		code:     1,
		finished: time.Now(),
	}

	bt.updateStepCmd(cs)

	var updated model.TCmdLine
	has, err := comm.Db.ID(cmdId).Get(&updated)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, common.BuildStatusError, updated.Status)
	assert.Equal(t, 1, updated.Code)
}
