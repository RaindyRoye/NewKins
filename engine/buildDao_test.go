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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// setupBuildDaoTestDB creates an in-memory SQLite database with the tables
// needed by the buildDao functions. It returns the engine and a cleanup function.
func setupBuildDaoTestDB(t *testing.T) *xorm.Engine {
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
	if err := eng.Sync2(
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TCmdLine{},
	); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// newTestBuildTask creates a BuildTask with a context for testing buildDao functions.
func newTestBuildTask(t *testing.T, buildID string) *BuildTask {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &BuildTask{
		build: &runtime.Build{
			Id:         buildID,
			PipelineId: "pipe-1",
			Status:     common.BuildStatusPending,
			Created:    time.Now(),
		},
		ctx:    ctx,
		cncl:   cancel,
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
}

// --- updateBuild ---

func TestUpdateBuild_UpdatesStatus(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-ub1")

	// Insert a build record
	build := &model.TBuild{
		Id:                "build-ub1",
		PipelineId:        "pipe-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusPending,
		Created:           time.Now(),
	}
	_, err := eng.InsertOne(build)
	require.NoError(t, err)

	// Update build status to running
	bt.build.Status = common.BuildStatusRunning
	bt.build.Started = time.Now()
	bt.updateBuild(bt.build)

	// Verify the update
	got := &model.TBuild{}
	ok, err := eng.Where("id=?", "build-ub1").Get(got)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusRunning, got.Status)
	assert.False(t, got.Started.IsZero())
}

func TestUpdateBuild_EndedStatusCancelsRemaining(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-ub2")

	// Insert build + stage + step + cmd
	build := &model.TBuild{
		Id:                "build-ub2",
		PipelineId:        "pipe-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusRunning,
		Created:           time.Now(),
	}
	_, err := eng.InsertOne(build)
	require.NoError(t, err)

	stage := &model.TStage{
		Id:      "stage-ub2",
		BuildId: "build-ub2",
		Name:    "build",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = eng.InsertOne(stage)
	require.NoError(t, err)

	step := &model.TStep{
		Id:      "step-ub2",
		BuildId: "build-ub2",
		StageId: "stage-ub2",
		Name:    "compile",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = eng.InsertOne(step)
	require.NoError(t, err)

	cmd := &model.TCmdLine{
		Id:      "cmd-ub2",
		BuildId: "build-ub2",
		StepId:  "step-ub2",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = eng.InsertOne(cmd)
	require.NoError(t, err)

	// Mark build as ended (error)
	bt.build.Status = common.BuildStatusError
	bt.build.Error = "test error"
	bt.build.Finished = time.Now()
	bt.updateBuild(bt.build)

	// Verify stage was canceled
	gotStage := &model.TStage{}
	ok, err := eng.Where("id=?", "stage-ub2").Get(gotStage)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, gotStage.Status)

	// Verify step was canceled
	gotStep := &model.TStep{}
	ok, err = eng.Where("id=?", "step-ub2").Get(gotStep)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, gotStep.Status)

	// Verify cmd was canceled
	gotCmd := &model.TCmdLine{}
	ok, err = eng.Where("id=?", "cmd-ub2").Get(gotCmd)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, gotCmd.Status)
}

func TestUpdateBuild_NonEndedStatusDoesNotCancel(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-ub3")

	stage := &model.TStage{
		Id:      "stage-ub3",
		BuildId: "build-ub3",
		Name:    "build",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := eng.InsertOne(stage)
	require.NoError(t, err)

	build := &model.TBuild{
		Id:                "build-ub3",
		PipelineId:        "pipe-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusRunning,
		Created:           time.Now(),
	}
	_, err = eng.InsertOne(build)
	require.NoError(t, err)

	// Update with non-ended status
	bt.build.Status = common.BuildStatusRunning
	bt.updateBuild(bt.build)

	// Stage should remain running
	gotStage := &model.TStage{}
	ok, err := eng.Where("id=?", "stage-ub3").Get(gotStage)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusRunning, gotStage.Status)
}

func TestUpdateBuild_PreservesOkStages(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-ub4")

	build := &model.TBuild{
		Id:                "build-ub4",
		PipelineId:        "pipe-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusRunning,
		Created:           time.Now(),
	}
	_, err := eng.InsertOne(build)
	require.NoError(t, err)

	// Insert a stage that already completed OK
	okStage := &model.TStage{
		Id:      "stage-ok",
		BuildId: "build-ub4",
		Name:    "build",
		Status:  common.BuildStatusOk,
		Created: time.Now(),
	}
	_, err = eng.InsertOne(okStage)
	require.NoError(t, err)

	// Insert a stage that's still running
	runStage := &model.TStage{
		Id:      "stage-run",
		BuildId: "build-ub4",
		Name:    "deploy",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = eng.InsertOne(runStage)
	require.NoError(t, err)

	// End the build with error
	bt.build.Status = common.BuildStatusError
	bt.build.Finished = time.Now()
	bt.updateBuild(bt.build)

	// OK stage should remain OK (not canceled)
	gotOk := &model.TStage{}
	ok, err := eng.Where("id=?", "stage-ok").Get(gotOk)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusOk, gotOk.Status, "completed stages should not be canceled")

	// Running stage should be canceled
	gotRun := &model.TStage{}
	ok, err = eng.Where("id=?", "stage-run").Get(gotRun)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, gotRun.Status)
}

// --- updateStage ---

func TestUpdateStage_UpdatesStatus(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-us1")

	stage := &model.TStage{
		Id:      "stage-us1",
		BuildId: "build-us1",
		Name:    "build",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	_, err := eng.InsertOne(stage)
	require.NoError(t, err)

	rtStage := &runtime.Stage{
		Id:      "stage-us1",
		BuildId: "build-us1",
		Name:    "build",
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
	}
	bt.updateStage(rtStage)

	got := &model.TStage{}
	ok, err := eng.Where("id=?", "stage-us1").Get(got)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusRunning, got.Status)
	assert.False(t, got.Started.IsZero())
}

func TestUpdateStage_EndedCancelsSteps(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-us2")

	stage := &model.TStage{
		Id:      "stage-us2",
		BuildId: "build-us2",
		Name:    "build",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := eng.InsertOne(stage)
	require.NoError(t, err)

	step := &model.TStep{
		Id:      "step-us2",
		BuildId: "build-us2",
		StageId: "stage-us2",
		Name:    "compile",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = eng.InsertOne(step)
	require.NoError(t, err)

	// Mark stage as ended with error
	rtStage := &runtime.Stage{
		Id:       "stage-us2",
		BuildId:  "build-us2",
		Name:     "build",
		Status:   common.BuildStatusError,
		Error:    "stage failed",
		Finished: time.Now(),
	}
	bt.updateStage(rtStage)

	// Step should be canceled
	gotStep := &model.TStep{}
	ok, err := eng.Where("id=?", "step-us2").Get(gotStep)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, gotStep.Status)
}

func TestUpdateStage_PreservesOkSteps(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-us3")

	stage := &model.TStage{
		Id:      "stage-us3",
		BuildId: "build-us3",
		Name:    "build",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := eng.InsertOne(stage)
	require.NoError(t, err)

	// Step already OK
	okStep := &model.TStep{
		Id:      "step-ok",
		BuildId: "build-us3",
		StageId: "stage-us3",
		Name:    "compile",
		Status:  common.BuildStatusOk,
		Created: time.Now(),
	}
	_, err = eng.InsertOne(okStep)
	require.NoError(t, err)

	// Step still running
	runStep := &model.TStep{
		Id:      "step-run",
		BuildId: "build-us3",
		StageId: "stage-us3",
		Name:    "test",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = eng.InsertOne(runStep)
	require.NoError(t, err)

	rtStage := &runtime.Stage{
		Id:       "stage-us3",
		BuildId:  "build-us3",
		Name:     "build",
		Status:   common.BuildStatusError,
		Finished: time.Now(),
	}
	bt.updateStage(rtStage)

	// OK step should remain OK
	gotOk := &model.TStep{}
	ok, err := eng.Where("id=?", "step-ok").Get(gotOk)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusOk, gotOk.Status)

	// Running step should be canceled
	gotRun := &model.TStep{}
	ok, err = eng.Where("id=?", "step-run").Get(gotRun)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, gotRun.Status)
}

// --- updateStep ---

func TestUpdateStep_UpdatesStatus(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-ust1")

	step := &model.TStep{
		Id:      "step-ust1",
		BuildId: "build-ust1",
		StageId: "stage-ust1",
		Name:    "compile",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	_, err := eng.InsertOne(step)
	require.NoError(t, err)

	job := &jobSync{
		step: &runtime.Step{
			Id:      "step-ust1",
			Name:    "compile",
			Status:  common.BuildStatusRunning,
			Started: time.Now(),
		},
		cmdmp: make(map[string]*cmdSync),
	}
	bt.updateStep(job)

	got := &model.TStep{}
	ok, err := eng.Where("id=?", "step-ust1").Get(got)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusRunning, got.Status)
}

func TestUpdateStep_EndedCancelsCmds(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-ust2")

	step := &model.TStep{
		Id:      "step-ust2",
		BuildId: "build-ust2",
		StageId: "stage-ust2",
		Name:    "compile",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := eng.InsertOne(step)
	require.NoError(t, err)

	cmd := &model.TCmdLine{
		Id:      "cmd-ust2",
		BuildId: "build-ust2",
		StepId:  "step-ust2",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err = eng.InsertOne(cmd)
	require.NoError(t, err)

	job := &jobSync{
		step: &runtime.Step{
			Id:       "step-ust2",
			Name:     "compile",
			Status:   common.BuildStatusError,
			Error:    "failed",
			Finished: time.Now(),
		},
		cmdmp: make(map[string]*cmdSync),
	}
	bt.updateStep(job)

	// Cmd should be canceled
	gotCmd := &model.TCmdLine{}
	ok, err := eng.Where("id=?", "cmd-ust2").Get(gotCmd)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusCancel, gotCmd.Status)
}

// --- updateStepCmd ---

func TestUpdateStepCmd_RunningStatus(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-usc1")

	cmd := &model.TCmdLine{
		Id:      "cmd-usc1",
		BuildId: "build-usc1",
		StepId:  "step-usc1",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	_, err := eng.InsertOne(cmd)
	require.NoError(t, err)

	cmdSync := &cmdSync{
		cmd:     &runners.CmdContent{Id: "cmd-usc1"},
		status:  common.BuildStatusRunning,
		started: time.Now(),
	}
	bt.updateStepCmd(cmdSync)

	got := &model.TCmdLine{}
	ok, err := eng.Where("id=?", "cmd-usc1").Get(got)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusRunning, got.Status)
	assert.False(t, got.Started.IsZero())
}

func TestUpdateStepCmd_FinishedStatus(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-usc2")

	cmd := &model.TCmdLine{
		Id:      "cmd-usc2",
		BuildId: "build-usc2",
		StepId:  "step-usc2",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := eng.InsertOne(cmd)
	require.NoError(t, err)

	cmdSync := &cmdSync{
		cmd:      &runners.CmdContent{Id: "cmd-usc2"},
		status:   common.BuildStatusOk,
		finished: time.Now(),
	}
	bt.updateStepCmd(cmdSync)

	got := &model.TCmdLine{}
	ok, err := eng.Where("id=?", "cmd-usc2").Get(got)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusOk, got.Status)
	assert.False(t, got.Finished.IsZero())
}

// --- taskCtx ---

func TestTaskCtx_ReturnsOwnCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bt := &BuildTask{ctx: ctx}
	assert.Equal(t, ctx, bt.taskCtx())
}

func TestTaskCtx_FallsBackToGlobal(t *testing.T) {
	bt := &BuildTask{ctx: nil}
	assert.Equal(t, comm.Ctx, bt.taskCtx())
}

func TestUpdateStep_WithExitCode(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	bt := newTestBuildTask(t, "build-ust3")

	step := &model.TStep{
		Id:      "step-ust3",
		BuildId: "build-ust3",
		StageId: "stage-ust3",
		Name:    "compile",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	_, err := eng.InsertOne(step)
	require.NoError(t, err)

	job := &jobSync{
		step: &runtime.Step{
			Id:       "step-ust3",
			Name:     "compile",
			Status:   common.BuildStatusError,
			Error:    "exit code 1",
			ExitCode: 1,
			Finished: time.Now(),
		},
		cmdmp: make(map[string]*cmdSync),
	}
	bt.updateStep(job)

	got := &model.TStep{}
	ok, err := eng.Where("id=?", "step-ust3").Get(got)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, common.BuildStatusError, got.Status)
	assert.Equal(t, 1, got.ExitCode)
}
