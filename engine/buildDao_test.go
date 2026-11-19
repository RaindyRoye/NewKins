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
	"github.com/gokins/runner/runners"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// setupBuildDaoDB creates an in-memory SQLite database with the tables needed
// by buildDao update functions. It returns the engine and a cleanup function.
func setupBuildDaoDB(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Sync2(
		new(model.TBuild),
		new(model.TStage),
		new(model.TStep),
		new(model.TCmdLine),
	))
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })
	return db
}

// --- updateBuild ---

func TestUpdateBuild_Running(t *testing.T) {
	db := setupBuildDaoDB(t)
	buildId := utils.NewXid()
	_, err := db.InsertOne(&model.TBuild{Id: buildId, Status: common.BuildStatusPending})
	require.NoError(t, err)

	bt := &BuildTask{
		ctx:   context.Background(),
		build: &runtime.Build{Id: buildId},
	}
	bt.updateBuild(&runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusRunning,
	})

	var got model.TBuild
	ok, err := db.Where("id=?", buildId).Get(&got)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusRunning, got.Status)
}

func TestUpdateBuild_EndedCascadesCancel(t *testing.T) {
	db := setupBuildDaoDB(t)
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	stepId := utils.NewXid()
	cmdId := utils.NewXid()

	_, err := db.InsertOne(&model.TBuild{Id: buildId, Status: common.BuildStatusPending})
	require.NoError(t, err)
	_, err = db.InsertOne(&model.TStage{
		Id: stageId, BuildId: buildId, Status: common.BuildStatusRunning,
	})
	require.NoError(t, err)
	_, err = db.InsertOne(&model.TStep{
		Id: stepId, BuildId: buildId, StageId: stageId, Status: common.BuildStatusRunning,
	})
	require.NoError(t, err)
	_, err = db.InsertOne(&model.TCmdLine{
		Id: cmdId, BuildId: buildId, StepId: stepId, Status: common.BuildStatusRunning,
	})
	require.NoError(t, err)

	bt := &BuildTask{
		ctx:   context.Background(),
		build: &runtime.Build{Id: buildId},
	}
	bt.updateBuild(&runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusError,
	})

	// Stage should be canceled
	var stage model.TStage
	ok, _ := db.Where("id=?", stageId).Get(&stage)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, stage.Status)

	// Step should be canceled
	var step model.TStep
	ok, _ = db.Where("id=?", stepId).Get(&step)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, step.Status)

	// CmdLine should be canceled
	var cmd model.TCmdLine
	ok, _ = db.Where("id=?", cmdId).Get(&cmd)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, cmd.Status)
}

func TestUpdateBuild_EndedPreservesTerminalStages(t *testing.T) {
	db := setupBuildDaoDB(t)
	buildId := utils.NewXid()
	okStageId := utils.NewXid()
	errStageId := utils.NewXid()

	_, err := db.InsertOne(&model.TBuild{Id: buildId, Status: common.BuildStatusPending})
	require.NoError(t, err)
	// Already-ok stage should NOT be overwritten to cancel
	_, err = db.InsertOne(&model.TStage{
		Id: okStageId, BuildId: buildId, Status: common.BuildStatusOk,
	})
	require.NoError(t, err)
	// Already-error stage should NOT be overwritten to cancel
	_, err = db.InsertOne(&model.TStage{
		Id: errStageId, BuildId: buildId, Status: common.BuildStatusError,
	})
	require.NoError(t, err)

	bt := &BuildTask{
		ctx:   context.Background(),
		build: &runtime.Build{Id: buildId},
	}
	bt.updateBuild(&runtime.Build{
		Id:     buildId,
		Status: common.BuildStatusCancel,
	})

	var okStage model.TStage
	ok, _ := db.Where("id=?", okStageId).Get(&okStage)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusOk, okStage.Status, "ok stage should be preserved")

	var errStage model.TStage
	ok, _ = db.Where("id=?", errStageId).Get(&errStage)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusError, errStage.Status, "error stage should be preserved")
}

// --- updateStage ---

func TestUpdateStage_Running(t *testing.T) {
	db := setupBuildDaoDB(t)
	stageId := utils.NewXid()
	buildId := utils.NewXid()
	_, err := db.InsertOne(&model.TStage{Id: stageId, BuildId: buildId, Status: common.BuildStatusPending})
	require.NoError(t, err)

	bt := &BuildTask{
		ctx:   context.Background(),
		build: &runtime.Build{Id: buildId},
	}
	bt.updateStage(&runtime.Stage{
		Id:     stageId,
		Status: common.BuildStatusRunning,
	})

	var got model.TStage
	ok, _ := db.Where("id=?", stageId).Get(&got)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusRunning, got.Status)
}

func TestUpdateStage_EndedCancelsSteps(t *testing.T) {
	db := setupBuildDaoDB(t)
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	runStepId := utils.NewXid()
	okStepId := utils.NewXid()

	_, err := db.InsertOne(&model.TStage{Id: stageId, BuildId: buildId, Status: common.BuildStatusRunning})
	require.NoError(t, err)
	// Running step should be canceled
	_, err = db.InsertOne(&model.TStep{
		Id: runStepId, BuildId: buildId, StageId: stageId, Status: common.BuildStatusRunning,
	})
	require.NoError(t, err)
	// Already-ok step should be preserved
	_, err = db.InsertOne(&model.TStep{
		Id: okStepId, BuildId: buildId, StageId: stageId, Status: common.BuildStatusOk,
	})
	require.NoError(t, err)

	bt := &BuildTask{
		ctx:   context.Background(),
		build: &runtime.Build{Id: buildId},
	}
	bt.updateStage(&runtime.Stage{
		Id:     stageId,
		Status: common.BuildStatusError,
	})

	var runStep model.TStep
	ok, _ := db.Where("id=?", runStepId).Get(&runStep)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, runStep.Status, "running step should be canceled")

	var okStep model.TStep
	ok, _ = db.Where("id=?", okStepId).Get(&okStep)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusOk, okStep.Status, "ok step should be preserved")
}

// --- updateStep ---

func TestUpdateStep_Running(t *testing.T) {
	db := setupBuildDaoDB(t)
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	stepId := utils.NewXid()
	_, err := db.InsertOne(&model.TStep{
		Id: stepId, BuildId: buildId, StageId: stageId, Status: common.BuildStatusPending,
	})
	require.NoError(t, err)

	job := &jobSync{
		task: &BuildTask{ctx: context.Background(), build: &runtime.Build{Id: buildId}},
		step: &runtime.Step{
			Id:       stepId,
			BuildId:  buildId,
			StageId:  stageId,
			Status:   common.BuildStatusRunning,
			ExitCode: 0,
		},
		cmdmp: make(map[string]*cmdSync),
	}

	job.task.updateStep(job)

	var got model.TStep
	ok, _ := db.Where("id=?", stepId).Get(&got)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusRunning, got.Status)
}

func TestUpdateStep_EndedCancelsCmdLines(t *testing.T) {
	db := setupBuildDaoDB(t)
	buildId := utils.NewXid()
	stageId := utils.NewXid()
	stepId := utils.NewXid()
	runCmdId := utils.NewXid()
	okCmdId := utils.NewXid()

	_, err := db.InsertOne(&model.TStep{
		Id: stepId, BuildId: buildId, StageId: stageId, Status: common.BuildStatusRunning,
	})
	require.NoError(t, err)
	_, err = db.InsertOne(&model.TCmdLine{
		Id: runCmdId, BuildId: buildId, StepId: stepId, Status: common.BuildStatusRunning,
	})
	require.NoError(t, err)
	_, err = db.InsertOne(&model.TCmdLine{
		Id: okCmdId, BuildId: buildId, StepId: stepId, Status: common.BuildStatusOk,
	})
	require.NoError(t, err)

	job := &jobSync{
		task: &BuildTask{ctx: context.Background(), build: &runtime.Build{Id: buildId}},
		step: &runtime.Step{
			Id:      stepId,
			BuildId: buildId,
			StageId: stageId,
			Status:  common.BuildStatusError,
		},
		cmdmp: make(map[string]*cmdSync),
	}

	job.task.updateStep(job)

	var runCmd model.TCmdLine
	ok, _ := db.Where("id=?", runCmdId).Get(&runCmd)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, runCmd.Status, "running cmd should be canceled")

	var okCmd model.TCmdLine
	ok, _ = db.Where("id=?", okCmdId).Get(&okCmd)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusOk, okCmd.Status, "ok cmd should be preserved")
}

// --- updateStepCmd ---

func TestUpdateStepCmd_Running(t *testing.T) {
	db := setupBuildDaoDB(t)
	cmdId := utils.NewXid()
	stepId := utils.NewXid()
	now := time.Now()

	_, err := db.InsertOne(&model.TCmdLine{
		Id: cmdId, StepId: stepId, Status: common.BuildStatusPending,
	})
	require.NoError(t, err)

	bt := &BuildTask{ctx: context.Background(), build: &runtime.Build{}}
	cmd := &cmdSync{
		cmd:     &runners.CmdContent{Id: cmdId},
		status:  common.BuildStatusRunning,
		code:    0,
		started: now,
	}

	bt.updateStepCmd(cmd)

	var got model.TCmdLine
	ok, _ := db.Where("id=?", cmdId).Get(&got)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusRunning, got.Status)
	assert.False(t, got.Started.IsZero(), "Started should be set for running status")
}

func TestUpdateStepCmd_Finished(t *testing.T) {
	db := setupBuildDaoDB(t)
	cmdId := utils.NewXid()
	stepId := utils.NewXid()
	now := time.Now()

	_, err := db.InsertOne(&model.TCmdLine{
		Id: cmdId, StepId: stepId, Status: common.BuildStatusRunning,
	})
	require.NoError(t, err)

	bt := &BuildTask{ctx: context.Background(), build: &runtime.Build{}}
	cmd := &cmdSync{
		cmd:      &runners.CmdContent{Id: cmdId},
		status:   common.BuildStatusOk,
		code:     0,
		finished: now,
	}

	bt.updateStepCmd(cmd)

	var got model.TCmdLine
	ok, _ := db.Where("id=?", cmdId).Get(&got)
	assert.True(t, ok)
	assert.Equal(t, common.BuildStatusOk, got.Status)
	assert.False(t, got.Finished.IsZero(), "Finished should be set for terminal status")
}

// --- taskCtx fallback ---

func TestTaskCtx_FallbackToGlobal(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "ctx-test"}}
	// When ctx is nil, taskCtx should return comm.Ctx
	ctx := bt.taskCtx()
	assert.Equal(t, comm.Ctx, ctx)
}

func TestTaskCtx_UsesOwnContext(t *testing.T) {
	ownCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{
		ctx:   ownCtx,
		build: &runtime.Build{Id: "ctx-test-2"},
	}
	assert.Equal(t, ownCtx, bt.taskCtx())
}
