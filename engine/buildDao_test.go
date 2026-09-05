package engine

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"github.com/gokins/runner/runners"
	"xorm.io/xorm"
)

func setupBuildDaoTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create t_build table
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
	if err != nil {
		t.Fatalf("create t_build table: %v", err)
	}

	// Create t_stage table
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
		stage VARCHAR(255),
		sort BIGINT
	)`)
	if err != nil {
		t.Fatalf("create t_stage table: %v", err)
	}

	// Create t_step table
	_, err = db.Exec(`CREATE TABLE t_step (
		id VARCHAR(64) PRIMARY KEY,
		build_id VARCHAR(64),
		stage_id VARCHAR(100),
		display_name VARCHAR(255),
		pipeline_version_id VARCHAR(64),
		step VARCHAR(255),
		status VARCHAR(100),
		event VARCHAR(100),
		exit_code BIGINT,
		error VARCHAR(500),
		name VARCHAR(100),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		version VARCHAR(255),
		errignore INT,
		commands TEXT,
		waits TEXT,
		sort BIGINT
	)`)
	if err != nil {
		t.Fatalf("create t_step table: %v", err)
	}

	// Create t_cmd_line table
	_, err = db.Exec(`CREATE TABLE t_cmd_line (
		id VARCHAR(64) PRIMARY KEY,
		group_id VARCHAR(64),
		build_id VARCHAR(64),
		step_id VARCHAR(64),
		status VARCHAR(50),
		code INT,
		num INT,
		content TEXT,
		created DATETIME,
		started DATETIME,
		finished DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_cmd_line table: %v", err)
	}

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	return db
}

// --- updateBuild ---

func TestUpdateBuild_RunningStatus(t *testing.T) {
	db := setupBuildDaoTestDB(t)

	// Insert a build
	build := &model.TBuild{
		Id:      "build-dao-1",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := db.Insert(build); err != nil {
		t.Fatalf("insert build: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-dao-1"},
		ctx:   ctx,
	}

	rtBuild := &runtime.Build{
		Id:       "build-dao-1",
		Status:   common.BuildStatusRunning,
		Started:  time.Now(),
		Finished: time.Time{},
	}
	bt.updateBuild(rtBuild)

	// Verify the build was updated
	got := &model.TBuild{}
	ok, err := db.Where("id=?", "build-dao-1").Get(got)
	if err != nil {
		t.Fatalf("query build: %v", err)
	}
	if !ok {
		t.Fatal("build not found after update")
	}
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
}

func TestUpdateBuild_EndedStatusCascadesToStages(t *testing.T) {
	db := setupBuildDaoTestDB(t)

	// Insert build and stages
	build := &model.TBuild{
		Id:      "build-dao-2",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := db.Insert(build); err != nil {
		t.Fatalf("insert build: %v", err)
	}

	// Insert a pending stage
	stage := &model.TStage{
		Id:      "stage-dao-1",
		BuildId: "build-dao-2",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := db.Insert(stage); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	// Insert a pending step
	step := &model.TStep{
		Id:      "step-dao-1",
		BuildId: "build-dao-2",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := db.Insert(step); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	// Insert a pending cmd
	cmd := &model.TCmdLine{
		Id:      "cmd-dao-1",
		BuildId: "build-dao-2",
		Status:  common.BuildStatusPending,
	}
	if _, err := db.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-dao-2"},
		ctx:   ctx,
	}

	// Set build to error (an ended status)
	rtBuild := &runtime.Build{
		Id:       "build-dao-2",
		Status:   common.BuildStatusError,
		Error:    "something failed",
		Finished: time.Now(),
	}
	bt.updateBuild(rtBuild)

	// Verify build status
	got := &model.TBuild{}
	ok, err := db.Where("id=?", "build-dao-2").Get(got)
	if err != nil {
		t.Fatalf("query build: %v", err)
	}
	if !ok {
		t.Fatal("build not found")
	}
	if got.Status != common.BuildStatusError {
		t.Errorf("build status = %q, want %q", got.Status, common.BuildStatusError)
	}

	// Verify stage cascaded to cancel
	gotStage := &model.TStage{}
	ok, err = db.Where("id=?", "stage-dao-1").Get(gotStage)
	if err != nil {
		t.Fatalf("query stage: %v", err)
	}
	if !ok {
		t.Fatal("stage not found")
	}
	if gotStage.Status != common.BuildStatusCancel {
		t.Errorf("stage status = %q, want %q", gotStage.Status, common.BuildStatusCancel)
	}

	// Verify step cascaded to cancel
	gotStep := &model.TStep{}
	ok, err = db.Where("id=?", "step-dao-1").Get(gotStep)
	if err != nil {
		t.Fatalf("query step: %v", err)
	}
	if !ok {
		t.Fatal("step not found")
	}
	if gotStep.Status != common.BuildStatusCancel {
		t.Errorf("step status = %q, want %q", gotStep.Status, common.BuildStatusCancel)
	}

	// Verify cmd cascaded to cancel
	gotCmd := &model.TCmdLine{}
	ok, err = db.Where("id=?", "cmd-dao-1").Get(gotCmd)
	if err != nil {
		t.Fatalf("query cmd: %v", err)
	}
	if !ok {
		t.Fatal("cmd not found")
	}
	if gotCmd.Status != common.BuildStatusCancel {
		t.Errorf("cmd status = %q, want %q", gotCmd.Status, common.BuildStatusCancel)
	}
}

func TestUpdateBuild_EndedStatusPreservesOkStages(t *testing.T) {
	db := setupBuildDaoTestDB(t)

	build := &model.TBuild{
		Id:      "build-dao-3",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := db.Insert(build); err != nil {
		t.Fatalf("insert build: %v", err)
	}

	// Insert an OK stage (should not be overridden)
	stageOk := &model.TStage{
		Id:      "stage-dao-ok",
		BuildId: "build-dao-3",
		Status:  common.BuildStatusOk,
		Created: time.Now(),
	}
	if _, err := db.Insert(stageOk); err != nil {
		t.Fatalf("insert ok stage: %v", err)
	}

	// Insert a pending stage (should be cancelled)
	stagePending := &model.TStage{
		Id:      "stage-dao-pending",
		BuildId: "build-dao-3",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := db.Insert(stagePending); err != nil {
		t.Fatalf("insert pending stage: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-dao-3"},
		ctx:   ctx,
	}

	rtBuild := &runtime.Build{
		Id:       "build-dao-3",
		Status:   common.BuildStatusError,
		Finished: time.Now(),
	}
	bt.updateBuild(rtBuild)

	// OK stage should remain OK
	gotOk := &model.TStage{}
	ok, err := db.Where("id=?", "stage-dao-ok").Get(gotOk)
	if err != nil {
		t.Fatalf("query ok stage: %v", err)
	}
	if !ok {
		t.Fatal("ok stage not found")
	}
	if gotOk.Status != common.BuildStatusOk {
		t.Errorf("ok stage status = %q, want %q (should be preserved)", gotOk.Status, common.BuildStatusOk)
	}

	// Pending stage should be cancelled
	gotPending := &model.TStage{}
	ok, err = db.Where("id=?", "stage-dao-pending").Get(gotPending)
	if err != nil {
		t.Fatalf("query pending stage: %v", err)
	}
	if !ok {
		t.Fatal("pending stage not found")
	}
	if gotPending.Status != common.BuildStatusCancel {
		t.Errorf("pending stage status = %q, want %q", gotPending.Status, common.BuildStatusCancel)
	}
}

// --- updateStage ---

func TestUpdateStage_RunningStatus(t *testing.T) {
	db := setupBuildDaoTestDB(t)

	stage := &model.TStage{
		Id:      "stage-dao-r1",
		BuildId: "build-dao-r1",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := db.Insert(stage); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-dao-r1"},
		ctx:   ctx,
	}

	rtStage := &runtime.Stage{
		Id:      "stage-dao-r1",
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
	}
	bt.updateStage(rtStage)

	got := &model.TStage{}
	ok, err := db.Where("id=?", "stage-dao-r1").Get(got)
	if err != nil {
		t.Fatalf("query stage: %v", err)
	}
	if !ok {
		t.Fatal("stage not found after update")
	}
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStage_EndedStatusCascadesToSteps(t *testing.T) {
	db := setupBuildDaoTestDB(t)

	stage := &model.TStage{
		Id:      "stage-dao-e1",
		BuildId: "build-dao-e1",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := db.Insert(stage); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	// Pending step
	step := &model.TStep{
		Id:      "step-dao-e1",
		StageId: "stage-dao-e1",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := db.Insert(step); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	// OK step (should not be overridden)
	stepOk := &model.TStep{
		Id:      "step-dao-e1-ok",
		StageId: "stage-dao-e1",
		Status:  common.BuildStatusOk,
		Created: time.Now(),
	}
	if _, err := db.Insert(stepOk); err != nil {
		t.Fatalf("insert ok step: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-dao-e1"},
		ctx:   ctx,
	}

	rtStage := &runtime.Stage{
		Id:       "stage-dao-e1",
		Status:   common.BuildStatusError,
		Error:    "stage failed",
		Finished: time.Now(),
	}
	bt.updateStage(rtStage)

	// Pending step should be cancelled
	got := &model.TStep{}
	ok, err := db.Where("id=?", "step-dao-e1").Get(got)
	if err != nil {
		t.Fatalf("query step: %v", err)
	}
	if !ok {
		t.Fatal("step not found")
	}
	if got.Status != common.BuildStatusCancel {
		t.Errorf("step status = %q, want %q", got.Status, common.BuildStatusCancel)
	}

	// OK step should remain OK
	gotOk := &model.TStep{}
	ok, err = db.Where("id=?", "step-dao-e1-ok").Get(gotOk)
	if err != nil {
		t.Fatalf("query ok step: %v", err)
	}
	if !ok {
		t.Fatal("ok step not found")
	}
	if gotOk.Status != common.BuildStatusOk {
		t.Errorf("ok step status = %q, want %q", gotOk.Status, common.BuildStatusOk)
	}
}

// --- updateStep ---

func TestUpdateStep_RunningStatus(t *testing.T) {
	db := setupBuildDaoTestDB(t)

	step := &model.TStep{
		Id:      "step-dao-r1",
		StageId: "stage-dao-r1",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := db.Insert(step); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-dao-r1"},
		ctx:   ctx,
	}

	job := &jobSync{
		task: bt,
		step: &runtime.Step{
			Id:      "step-dao-r1",
			Status:  common.BuildStatusRunning,
			Started: time.Now(),
		},
	}
	bt.updateStep(job)

	got := &model.TStep{}
	ok, err := db.Where("id=?", "step-dao-r1").Get(got)
	if err != nil {
		t.Fatalf("query step: %v", err)
	}
	if !ok {
		t.Fatal("step not found after update")
	}
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStep_EndedStatusCascadesToCmds(t *testing.T) {
	db := setupBuildDaoTestDB(t)

	step := &model.TStep{
		Id:      "step-dao-c1",
		StageId: "stage-dao-c1",
		Status:  common.BuildStatusRunning,
		Created: time.Now(),
	}
	if _, err := db.Insert(step); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	// Pending cmd
	cmd := &model.TCmdLine{
		Id:     "cmd-dao-c1",
		StepId: "step-dao-c1",
		Status: common.BuildStatusPending,
	}
	if _, err := db.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-dao-c1"},
		ctx:   ctx,
	}

	job := &jobSync{
		task: bt,
		step: &runtime.Step{
			Id:       "step-dao-c1",
			Status:   common.BuildStatusError,
			ExitCode: 1,
			Finished: time.Now(),
		},
	}
	bt.updateStep(job)

	gotCmd := &model.TCmdLine{}
	ok, err := db.Where("id=?", "cmd-dao-c1").Get(gotCmd)
	if err != nil {
		t.Fatalf("query cmd: %v", err)
	}
	if !ok {
		t.Fatal("cmd not found")
	}
	if gotCmd.Status != common.BuildStatusCancel {
		t.Errorf("cmd status = %q, want %q", gotCmd.Status, common.BuildStatusCancel)
	}
}

// --- updateStepCmd ---

func TestUpdateStepCmd_Running(t *testing.T) {
	db := setupBuildDaoTestDB(t)

	cmd := &model.TCmdLine{
		Id:     "cmd-dao-sc1",
		StepId: "step-dao-sc1",
		Status: common.BuildStatusPending,
	}
	if _, err := db.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-dao-sc1"},
		ctx:   ctx,
	}

	now := time.Now()
	csync := &cmdSync{
		cmd:     &runners.CmdContent{Id: "cmd-dao-sc1"},
		status:  common.BuildStatusRunning,
		started: now,
	}
	bt.updateStepCmd(csync)

	got := &model.TCmdLine{}
	ok, err := db.Where("id=?", "cmd-dao-sc1").Get(got)
	if err != nil {
		t.Fatalf("query cmd: %v", err)
	}
	if !ok {
		t.Fatal("cmd not found after update")
	}
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
	if got.Started.IsZero() {
		t.Error("started should be set for running cmd")
	}
}

func TestUpdateStepCmd_Completed(t *testing.T) {
	db := setupBuildDaoTestDB(t)

	cmd := &model.TCmdLine{
		Id:      "cmd-dao-sc2",
		StepId:  "step-dao-sc2",
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
	}
	if _, err := db.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-dao-sc2"},
		ctx:   ctx,
	}

	csync2 := &cmdSync{
		cmd:      &runners.CmdContent{Id: "cmd-dao-sc2"},
		status:   common.BuildStatusOk,
		finished: time.Now(),
	}
	bt.updateStepCmd(csync2)

	got := &model.TCmdLine{}
	ok, err := db.Where("id=?", "cmd-dao-sc2").Get(got)
	if err != nil {
		t.Fatalf("query cmd: %v", err)
	}
	if !ok {
		t.Fatal("cmd not found after update")
	}
	if got.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusOk)
	}
	if got.Finished.IsZero() {
		t.Error("finished should be set for completed cmd")
	}
}

// --- taskCtx fallback ---

func TestTaskCtx_WithNilContext(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-tc-1"},
		ctx:   nil,
	}
	got := bt.taskCtx()
	if got != comm.Ctx {
		t.Error("taskCtx() should fall back to comm.Ctx when ctx is nil")
	}
}
