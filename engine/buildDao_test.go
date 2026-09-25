package engine

import (
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

// setupBuildDaoTestDB creates an isolated in-memory SQLite DB for buildDao tests.
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

func createTestBuild(t *testing.T, eng *xorm.Engine, id string) {
	t.Helper()
	b := &model.TBuild{
		Id:         id,
		PipelineId: "pipe-1",
		Event:      "push",
		Status:     common.BuildStatusPending,
		Created:    time.Now(),
	}
	if _, err := eng.InsertOne(b); err != nil {
		t.Fatalf("insert build: %v", err)
	}
}

func createTestStage(t *testing.T, eng *xorm.Engine, id, buildId string) *model.TStage {
	t.Helper()
	s := &model.TStage{
		Id:      id,
		BuildId: buildId,
		Name:    "build",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := eng.InsertOne(s); err != nil {
		t.Fatalf("insert stage: %v", err)
	}
	return s
}

func createTestStep(t *testing.T, eng *xorm.Engine, id, stageId string) *model.TStep {
	t.Helper()
	st := &model.TStep{
		Id:      id,
		StageId: stageId,
		Name:    "compile",
		Status:  common.BuildStatusPending,
		Created: time.Now(),
	}
	if _, err := eng.InsertOne(st); err != nil {
		t.Fatalf("insert step: %v", err)
	}
	return st
}

func createTestCmd(t *testing.T, eng *xorm.Engine, id, stepId string) *model.TCmdLine {
	t.Helper()
	cmd := &model.TCmdLine{
		Id:     id,
		StepId: stepId,
		Status: common.BuildStatusPending,
	}
	if _, err := eng.InsertOne(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}
	return cmd
}

// --- updateBuild ---

func TestUpdateBuild_Running(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	createTestBuild(t, eng, "build-1")
	bt := &BuildTask{build: &runtime.Build{Id: "build-1"}}

	rb := &runtime.Build{
		Id:      "build-1",
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
	}
	bt.updateBuild(rb)

	var got model.TBuild
	ok, err := eng.Where("id=?", "build-1").Get(&got)
	if err != nil || !ok {
		t.Fatalf("query build: ok=%v, err=%v", ok, err)
	}
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
	if got.Started.IsZero() {
		t.Error("started should be set")
	}
}

func TestUpdateBuild_Completed(t *testing.T) {
	eng := setupBuildDaoTestDB(t)

	createTestBuild(t, eng, "build-2")
	createTestStage(t, eng, "stage-2", "build-2")

	bt := &BuildTask{build: &runtime.Build{Id: "build-2"}}
	rb := &runtime.Build{
		Id:       "build-2",
		Status:   common.BuildStatusOk,
		Finished: time.Now(),
	}
	bt.updateBuild(rb)

	var b model.TBuild
	if _, err := eng.Where("id=?", "build-2").Get(&b); err != nil {
		t.Fatalf("query build: %v", err)
	}
	if b.Status != common.BuildStatusOk {
		t.Errorf("build status = %q, want %q", b.Status, common.BuildStatusOk)
	}
	if b.Finished.IsZero() {
		t.Error("finished should be set")
	}

	// Stage should be canceled
	var s model.TStage
	if _, err := eng.Where("id=?", "stage-2").Get(&s); err != nil {
		t.Fatalf("query stage: %v", err)
	}
	if s.Status != common.BuildStatusCancel {
		t.Errorf("stage status = %q, want %q", s.Status, common.BuildStatusCancel)
	}
}

// --- updateStage ---

func TestUpdateStage_Running(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	createTestBuild(t, eng, "build-s1")
	stg := createTestStage(t, eng, "stage-s1", "build-s1")

	bt := &BuildTask{build: &runtime.Build{Id: "build-s1"}}
	rs := &runtime.Stage{
		Id:      stg.Id,
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
	}
	bt.updateStage(rs)

	var got model.TStage
	ok, err := eng.Where("id=?", stg.Id).Get(&got)
	if err != nil || !ok {
		t.Fatalf("query stage: ok=%v, err=%v", ok, err)
	}
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStage_Completed(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	createTestBuild(t, eng, "build-s2")
	stg := createTestStage(t, eng, "stage-s2", "build-s2")
	createTestStep(t, eng, "step-s2", stg.Id)

	bt := &BuildTask{build: &runtime.Build{Id: "build-s2"}}
	rs := &runtime.Stage{
		Id:       stg.Id,
		Status:   common.BuildStatusError,
		Error:    "stage failed",
		Finished: time.Now(),
	}
	bt.updateStage(rs)

	var got model.TStage
	if _, err := eng.Where("id=?", stg.Id).Get(&got); err != nil {
		t.Fatalf("query stage: %v", err)
	}
	if got.Status != common.BuildStatusError {
		t.Errorf("stage status = %q, want %q", got.Status, common.BuildStatusError)
	}
	if got.Error != "stage failed" {
		t.Errorf("stage error = %q, want %q", got.Error, "stage failed")
	}

	// Step should be canceled
	var st model.TStep
	if _, err := eng.Where("id=?", "step-s2").Get(&st); err != nil {
		t.Fatalf("query step: %v", err)
	}
	if st.Status != common.BuildStatusCancel {
		t.Errorf("step status = %q, want %q", st.Status, common.BuildStatusCancel)
	}
}

// --- updateStep ---

func TestUpdateStep_Running(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	createTestBuild(t, eng, "build-st1")
	stg := createTestStage(t, eng, "stage-st1", "build-st1")
	step := createTestStep(t, eng, "step-st1", stg.Id)

	bt := &BuildTask{build: &runtime.Build{Id: "build-st1"}}
	job := &jobSync{
		step: &runtime.Step{
			Id:      step.Id,
			Status:  common.BuildStatusRunning,
			Started: time.Now(),
		},
	}
	bt.updateStep(job)

	var got model.TStep
	ok, err := eng.Where("id=?", step.Id).Get(&got)
	if err != nil || !ok {
		t.Fatalf("query step: ok=%v, err=%v", ok, err)
	}
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStep_Completed(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	createTestBuild(t, eng, "build-st2")
	stg := createTestStage(t, eng, "stage-st2", "build-st2")
	step := createTestStep(t, eng, "step-st2", stg.Id)
	createTestCmd(t, eng, "cmd-st2", step.Id)

	bt := &BuildTask{build: &runtime.Build{Id: "build-st2"}}
	job := &jobSync{
		step: &runtime.Step{
			Id:       step.Id,
			Status:   common.BuildStatusOk,
			ExitCode: 0,
			Finished: time.Now(),
		},
	}
	bt.updateStep(job)

	var got model.TStep
	if _, err := eng.Where("id=?", step.Id).Get(&got); err != nil {
		t.Fatalf("query step: %v", err)
	}
	if got.Status != common.BuildStatusOk {
		t.Errorf("step status = %q, want %q", got.Status, common.BuildStatusOk)
	}
	if got.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", got.ExitCode)
	}

	// Cmd should be canceled
	var cmd model.TCmdLine
	if _, err := eng.Where("id=?", "cmd-st2").Get(&cmd); err != nil {
		t.Fatalf("query cmd: %v", err)
	}
	if cmd.Status != common.BuildStatusCancel {
		t.Errorf("cmd status = %q, want %q", cmd.Status, common.BuildStatusCancel)
	}
}

// --- updateStepCmd ---

func TestUpdateStepCmd_Running(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	createTestBuild(t, eng, "build-cmd1")
	stg := createTestStage(t, eng, "stage-cmd1", "build-cmd1")
	step := createTestStep(t, eng, "step-cmd1", stg.Id)
	cmd := createTestCmd(t, eng, "cmd-cmd1", step.Id)

	bt := &BuildTask{build: &runtime.Build{Id: "build-cmd1"}}
	cs := &cmdSync{
		cmd:     &runners.CmdContent{Id: cmd.Id},
		status:  common.BuildStatusRunning,
		started: time.Now(),
	}
	bt.updateStepCmd(cs)

	var got model.TCmdLine
	ok, err := eng.Where("id=?", cmd.Id).Get(&got)
	if err != nil || !ok {
		t.Fatalf("query cmd: ok=%v, err=%v", ok, err)
	}
	if got.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusRunning)
	}
	if got.Started.IsZero() {
		t.Error("started should be set")
	}
}

func TestUpdateStepCmd_Finished(t *testing.T) {
	eng := setupBuildDaoTestDB(t)
	createTestBuild(t, eng, "build-cmd2")
	stg := createTestStage(t, eng, "stage-cmd2", "build-cmd2")
	step := createTestStep(t, eng, "step-cmd2", stg.Id)
	cmd := createTestCmd(t, eng, "cmd-cmd2", step.Id)

	bt := &BuildTask{build: &runtime.Build{Id: "build-cmd2"}}
	cs := &cmdSync{
		cmd:      &runners.CmdContent{Id: cmd.Id},
		status:   common.BuildStatusOk,
		code:     0,
		finished: time.Now(),
	}
	bt.updateStepCmd(cs)

	var got model.TCmdLine
	if _, err := eng.Where("id=?", cmd.Id).Get(&got); err != nil {
		t.Fatalf("query cmd: %v", err)
	}
	if got.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", got.Status, common.BuildStatusOk)
	}
	if got.Code != 0 {
		t.Errorf("code = %d, want 0", got.Code)
	}
	if got.Finished.IsZero() {
		t.Error("finished should be set")
	}
}
