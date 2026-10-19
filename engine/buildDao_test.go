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

func setupBuildDaoTest(t *testing.T) func() {
	t.Helper()
	origDb := comm.Db
	origCtx := comm.Ctx
	origCncl := func() {}

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Sync2(&model.TBuild{}, &model.TStage{}, &model.TStep{}, &model.TCmdLine{}); err != nil {
		_ = db.Close()
		t.Fatalf("sync tables: %v", err)
	}
	comm.Db = db
	ctx, cancel := context.WithCancel(context.Background())
	comm.Ctx = ctx

	return func() {
		cancel()
		_ = db.Close()
		comm.Db = origDb
		comm.Ctx = origCtx
		_ = origCncl
	}
}

func newBuildTaskForDao(ctx context.Context) *BuildTask {
	bt := &BuildTask{
		ctx:   ctx,
		build: &runtime.Build{Id: "bd1", PipelineId: "p1"},
	}
	return bt
}

func TestUpdateBuild_RunningStatus(t *testing.T) {
	cleanup := setupBuildDaoTest(t)
	defer cleanup()

	// Insert a build record
	bd := &model.TBuild{Id: "bd1", Status: common.BuildStatusRunning}
	if _, err := comm.Db.InsertOne(bd); err != nil {
		t.Fatalf("insert build: %v", err)
	}

	bt := newBuildTaskForDao(comm.Ctx)
	build := &runtime.Build{
		Id:       "bd1",
		Status:   common.BuildStatusRunning,
		Started:  time.Now(),
		Finished: time.Time{},
	}
	bt.updateBuild(build)

	var updated model.TBuild
	ok, err := comm.Db.ID("bd1").Get(&updated)
	if err != nil {
		t.Fatalf("get build: %v", err)
	}
	if !ok {
		t.Fatal("build not found")
	}
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
}

func TestUpdateBuild_EndedCancelsChildren(t *testing.T) {
	cleanup := setupBuildDaoTest(t)
	defer cleanup()

	bd := &model.TBuild{Id: "bd2", Status: common.BuildStatusRunning}
	if _, err := comm.Db.InsertOne(bd); err != nil {
		t.Fatalf("insert build: %v", err)
	}
	stg := &model.TStage{Id: "stg1", BuildId: "bd2", Status: common.BuildStatusRunning}
	if _, err := comm.Db.InsertOne(stg); err != nil {
		t.Fatalf("insert stage: %v", err)
	}
	stp := &model.TStep{Id: "stp1", BuildId: "bd2", StageId: "stg1", Status: common.BuildStatusRunning}
	if _, err := comm.Db.InsertOne(stp); err != nil {
		t.Fatalf("insert step: %v", err)
	}
	cmd := &model.TCmdLine{Id: "cmd1", BuildId: "bd2", StepId: "stp1", Status: common.BuildStatusRunning}
	if _, err := comm.Db.InsertOne(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	bt := newBuildTaskForDao(comm.Ctx)
	build := &runtime.Build{
		Id:       "bd2",
		Status:   common.BuildStatusOk,
		Started:  time.Now().Add(-time.Minute),
		Finished: time.Now(),
	}
	bt.updateBuild(build)

	var updatedStage model.TStage
	if _, err := comm.Db.ID("stg1").Get(&updatedStage); err != nil {
		t.Fatalf("get stage: %v", err)
	}
	// Running stages should be cancelled when build ends
	if updatedStage.Status != common.BuildStatusCancel {
		t.Errorf("stage status = %q, want %q", updatedStage.Status, common.BuildStatusCancel)
	}

	var updatedStep model.TStep
	if _, err := comm.Db.ID("stp1").Get(&updatedStep); err != nil {
		t.Fatalf("get step: %v", err)
	}
	if updatedStep.Status != common.BuildStatusCancel {
		t.Errorf("step status = %q, want %q", updatedStep.Status, common.BuildStatusCancel)
	}

	var updatedCmd model.TCmdLine
	if _, err := comm.Db.ID("cmd1").Get(&updatedCmd); err != nil {
		t.Fatalf("get cmd: %v", err)
	}
	if updatedCmd.Status != common.BuildStatusCancel {
		t.Errorf("cmd status = %q, want %q", updatedCmd.Status, common.BuildStatusCancel)
	}
}

func TestUpdateBuild_NilDb(t *testing.T) {
	origDb := comm.Db
	comm.Db = nil
	defer func() { comm.Db = origDb }()

	bt := &BuildTask{
		ctx:   context.Background(),
		build: &runtime.Build{Id: "bd-nil"},
	}
	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("updateBuild panicked with nil Db: %v", r)
		}
	}()
	bt.updateBuild(&runtime.Build{Id: "bd-nil", Status: common.BuildStatusOk})
}

func TestUpdateStage_RunningStatus(t *testing.T) {
	cleanup := setupBuildDaoTest(t)
	defer cleanup()

	stg := &model.TStage{Id: "stg10", BuildId: "bd10", Status: common.BuildStatusPending}
	if _, err := comm.Db.InsertOne(stg); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	bt := newBuildTaskForDao(comm.Ctx)
	stage := &runtime.Stage{
		Id:      "stg10",
		BuildId: "bd10",
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
	}
	bt.updateStage(stage)

	var updated model.TStage
	ok, err := comm.Db.ID("stg10").Get(&updated)
	if err != nil {
		t.Fatalf("get stage: %v", err)
	}
	if !ok {
		t.Fatal("stage not found")
	}
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStage_EndedCancelsSteps(t *testing.T) {
	cleanup := setupBuildDaoTest(t)
	defer cleanup()

	stg := &model.TStage{Id: "stg20", BuildId: "bd20", Status: common.BuildStatusRunning}
	if _, err := comm.Db.InsertOne(stg); err != nil {
		t.Fatalf("insert stage: %v", err)
	}
	stp := &model.TStep{Id: "stp20", BuildId: "bd20", StageId: "stg20", Status: common.BuildStatusRunning}
	if _, err := comm.Db.InsertOne(stp); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	bt := newBuildTaskForDao(comm.Ctx)
	stage := &runtime.Stage{
		Id:       "stg20",
		BuildId:  "bd20",
		Status:   common.BuildStatusOk,
		Started:  time.Now().Add(-time.Minute),
		Finished: time.Now(),
	}
	bt.updateStage(stage)

	var updatedStep model.TStep
	if _, err := comm.Db.ID("stp20").Get(&updatedStep); err != nil {
		t.Fatalf("get step: %v", err)
	}
	if updatedStep.Status != common.BuildStatusCancel {
		t.Errorf("step status = %q, want %q", updatedStep.Status, common.BuildStatusCancel)
	}
}

func TestUpdateStage_NilDb(t *testing.T) {
	origDb := comm.Db
	comm.Db = nil
	defer func() { comm.Db = origDb }()

	bt := &BuildTask{ctx: context.Background()}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("updateStage panicked with nil Db: %v", r)
		}
	}()
	bt.updateStage(&runtime.Stage{Id: "stg-nil", Status: common.BuildStatusOk})
}

func TestUpdateStep_RunningStatus(t *testing.T) {
	cleanup := setupBuildDaoTest(t)
	defer cleanup()

	stp := &model.TStep{Id: "stp30", BuildId: "bd30", StageId: "stg30", Status: common.BuildStatusPending}
	if _, err := comm.Db.InsertOne(stp); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	bt := newBuildTaskForDao(comm.Ctx)
	job := &jobSync{
		step: &runtime.Step{
			Id:       "stp30",
			StageId:  "stg30",
			BuildId:  "bd30",
			Status:   common.BuildStatusRunning,
			Started:  time.Now(),
			ExitCode: 0,
		},
		cmdmp: make(map[string]*cmdSync),
	}
	bt.updateStep(job)

	var updated model.TStep
	ok, err := comm.Db.ID("stp30").Get(&updated)
	if err != nil {
		t.Fatalf("get step: %v", err)
	}
	if !ok {
		t.Fatal("step not found")
	}
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStep_EndedCancelsCmds(t *testing.T) {
	cleanup := setupBuildDaoTest(t)
	defer cleanup()

	stp := &model.TStep{Id: "stp40", BuildId: "bd40", StageId: "stg40", Status: common.BuildStatusRunning}
	if _, err := comm.Db.InsertOne(stp); err != nil {
		t.Fatalf("insert step: %v", err)
	}
	cmd := &model.TCmdLine{Id: "cmd40", BuildId: "bd40", StepId: "stp40", Status: common.BuildStatusRunning}
	if _, err := comm.Db.InsertOne(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	bt := newBuildTaskForDao(comm.Ctx)
	job := &jobSync{
		step: &runtime.Step{
			Id:       "stp40",
			StageId:  "stg40",
			BuildId:  "bd40",
			Status:   common.BuildStatusError,
			Error:    "test error",
			Started:  time.Now().Add(-time.Minute),
			Finished: time.Now(),
			ExitCode: 1,
		},
		cmdmp: make(map[string]*cmdSync),
	}
	bt.updateStep(job)

	var updatedCmd model.TCmdLine
	if _, err := comm.Db.ID("cmd40").Get(&updatedCmd); err != nil {
		t.Fatalf("get cmd: %v", err)
	}
	if updatedCmd.Status != common.BuildStatusCancel {
		t.Errorf("cmd status = %q, want %q", updatedCmd.Status, common.BuildStatusCancel)
	}
}

func TestUpdateStepCmd_RunningStatus(t *testing.T) {
	cleanup := setupBuildDaoTest(t)
	defer cleanup()

	cmd := &model.TCmdLine{Id: "cmd50", BuildId: "bd50", StepId: "stp50", Status: common.BuildStatusPending}
	if _, err := comm.Db.InsertOne(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	bt := newBuildTaskForDao(comm.Ctx)
	cs := &cmdSync{
		cmd:     &runners.CmdContent{Id: "cmd50"},
		status:  common.BuildStatusRunning,
		started: time.Now(),
	}
	bt.updateStepCmd(cs)

	var updated model.TCmdLine
	ok, err := comm.Db.ID("cmd50").Get(&updated)
	if err != nil {
		t.Fatalf("get cmd: %v", err)
	}
	if !ok {
		t.Fatal("cmd not found")
	}
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStepCmd_FinishedStatus(t *testing.T) {
	cleanup := setupBuildDaoTest(t)
	defer cleanup()

	cmd := &model.TCmdLine{Id: "cmd60", BuildId: "bd60", StepId: "stp60", Status: common.BuildStatusRunning}
	if _, err := comm.Db.InsertOne(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	bt := newBuildTaskForDao(comm.Ctx)
	cs := &cmdSync{
		cmd:      &runners.CmdContent{Id: "cmd60"},
		status:   common.BuildStatusOk,
		code:     0,
		finished: time.Now(),
	}
	bt.updateStepCmd(cs)

	var updated model.TCmdLine
	if _, err := comm.Db.ID("cmd60").Get(&updated); err != nil {
		t.Fatalf("get cmd: %v", err)
	}
	if updated.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusOk)
	}
}

func TestTaskCtx_FallbackToGlobalCtx(t *testing.T) {
	bt := &BuildTask{}
	ctx := bt.taskCtx()
	if ctx != comm.Ctx {
		t.Error("expected taskCtx to return comm.Ctx when task ctx is nil")
	}
}

func TestTaskCtx_UsesOwnCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{ctx: ctx}
	got := bt.taskCtx()
	if got != ctx {
		t.Error("expected taskCtx to return task's own context")
	}
}
