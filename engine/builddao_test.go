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

// setupEngineDB creates an isolated in-memory SQLite DB for engine tests.
func setupEngineDB(t *testing.T) *xorm.Engine {
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

// --- updateBuild ---

func TestUpdateBuild_RunningStatus(t *testing.T) {
	eng := setupEngineDB(t)

	build := &model.TBuild{
		Id:                "build-run-1",
		PipelineId:        "pipe-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusPending,
		Created:           time.Now(),
	}
	if _, err := eng.Insert(build); err != nil {
		t.Fatalf("insert build: %v", err)
	}

	bt := &BuildTask{
		build: &runtime.Build{
			Id:       "build-run-1",
			Status:   common.BuildStatusRunning,
			Error:    "",
			Started:  time.Now(),
			Finished: time.Time{},
		},
	}
	// Should update without error
	bt.updateBuild(bt.build)

	var updated model.TBuild
	ok, err := eng.Where("id=?", "build-run-1").Get(&updated)
	if err != nil {
		t.Fatalf("query updated build: %v", err)
	}
	if !ok {
		t.Fatal("build not found after update")
	}
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
}

func TestUpdateBuild_EndedStatus_CancelsChildren(t *testing.T) {
	eng := setupEngineDB(t)
	ctx := context.Background()

	// Insert build, stage, step, cmd
	build := &model.TBuild{
		Id:                "build-ended-1",
		PipelineId:        "pipe-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusRunning,
		Created:           time.Now(),
	}
	if _, err := eng.Insert(build); err != nil {
		t.Fatalf("insert build: %v", err)
	}

	stage := &model.TStage{
		Id:                "stage-1",
		BuildId:           "build-ended-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusRunning,
		Name:              "build",
		Created:           time.Now(),
	}
	if _, err := eng.Insert(stage); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	step := &model.TStep{
		Id:                "step-1",
		BuildId:           "build-ended-1",
		StageId:           "stage-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusRunning,
		Name:              "compile",
		Created:           time.Now(),
	}
	if _, err := eng.Insert(step); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	cmd := &model.TCmdLine{
		Id:      "cmd-1",
		BuildId: "build-ended-1",
		StepId:  "step-1",
		Status:  common.BuildStatusRunning,
		Num:     1,
		Content: "echo hello",
		Created: time.Now(),
	}
	if _, err := eng.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	// Mark build as error (ended status)
	bt := &BuildTask{
		build: &runtime.Build{
			Id:       "build-ended-1",
			Status:   common.BuildStatusError,
			Error:    "something failed",
			Finished: time.Now(),
		},
		ctx: ctx,
	}
	bt.updateBuild(bt.build)

	// Stage should be canceled
	var updatedStage model.TStage
	if ok, _ := eng.Where("id=?", "stage-1").Get(&updatedStage); ok {
		if updatedStage.Status != common.BuildStatusCancel {
			t.Errorf("stage status = %q, want %q", updatedStage.Status, common.BuildStatusCancel)
		}
	}

	// Step should be canceled
	var updatedStep model.TStep
	if ok, _ := eng.Where("id=?", "step-1").Get(&updatedStep); ok {
		if updatedStep.Status != common.BuildStatusCancel {
			t.Errorf("step status = %q, want %q", updatedStep.Status, common.BuildStatusCancel)
		}
	}

	// Cmd should be canceled
	var updatedCmd model.TCmdLine
	if ok, _ := eng.Where("id=?", "cmd-1").Get(&updatedCmd); ok {
		if updatedCmd.Status != common.BuildStatusCancel {
			t.Errorf("cmd status = %q, want %q", updatedCmd.Status, common.BuildStatusCancel)
		}
	}
}

func TestUpdateBuild_OkStatus_PreservesChildren(t *testing.T) {
	eng := setupEngineDB(t)
	ctx := context.Background()

	build := &model.TBuild{
		Id:                "build-ok-1",
		PipelineId:        "pipe-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusRunning,
		Created:           time.Now(),
	}
	if _, err := eng.Insert(build); err != nil {
		t.Fatalf("insert build: %v", err)
	}

	stage := &model.TStage{
		Id:                "stage-ok-1",
		BuildId:           "build-ok-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusOk, // already ok
		Name:              "build",
		Created:           time.Now(),
	}
	if _, err := eng.Insert(stage); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	bt := &BuildTask{
		build: &runtime.Build{
			Id:       "build-ok-1",
			Status:   common.BuildStatusOk,
			Finished: time.Now(),
		},
		ctx: ctx,
	}
	bt.updateBuild(bt.build)

	// Stage with Ok status should NOT be overwritten to Cancel
	var updatedStage model.TStage
	if ok, _ := eng.Where("id=?", "stage-ok-1").Get(&updatedStage); ok {
		if updatedStage.Status != common.BuildStatusOk {
			t.Errorf("stage status = %q, want %q (should not cancel ok stages)", updatedStage.Status, common.BuildStatusOk)
		}
	}
}

// --- updateStage ---

func TestUpdateStage_RunningStatus(t *testing.T) {
	eng := setupEngineDB(t)

	stage := &model.TStage{
		Id:                "stage-run-1",
		BuildId:           "build-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusPending,
		Name:              "build",
		Created:           time.Now(),
	}
	if _, err := eng.Insert(stage); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	bt := &BuildTask{build: &runtime.Build{Id: "build-1"}}
	rtStage := &runtime.Stage{
		Id:      "stage-run-1",
		BuildId: "build-1",
		Status:  common.BuildStatusRunning,
		Name:    "build",
		Started: time.Now(),
	}
	bt.updateStage(rtStage)

	var updated model.TStage
	ok, err := eng.Where("id=?", "stage-run-1").Get(&updated)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("stage not found")
	}
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStage_ErrorStatus_CancelsSteps(t *testing.T) {
	eng := setupEngineDB(t)
	ctx := context.Background()

	stage := &model.TStage{
		Id:                "stage-err-1",
		BuildId:           "build-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusRunning,
		Name:              "test",
		Created:           time.Now(),
	}
	if _, err := eng.Insert(stage); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	step := &model.TStep{
		Id:                "step-s-1",
		BuildId:           "build-1",
		StageId:           "stage-err-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusRunning,
		Name:              "compile",
		Created:           time.Now(),
	}
	if _, err := eng.Insert(step); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	bt := &BuildTask{build: &runtime.Build{Id: "build-1"}, ctx: ctx}
	rtStage := &runtime.Stage{
		Id:       "stage-err-1",
		BuildId:  "build-1",
		Status:   common.BuildStatusError,
		Error:    "test failed",
		Finished: time.Now(),
	}
	bt.updateStage(rtStage)

	// Step should be canceled
	var updatedStep model.TStep
	if ok, _ := eng.Where("id=?", "step-s-1").Get(&updatedStep); ok {
		if updatedStep.Status != common.BuildStatusCancel {
			t.Errorf("step status = %q, want %q", updatedStep.Status, common.BuildStatusCancel)
		}
	}
}

// --- updateStep ---

func TestUpdateStep_RunningStatus(t *testing.T) {
	eng := setupEngineDB(t)

	step := &model.TStep{
		Id:                "step-up-1",
		BuildId:           "build-1",
		StageId:           "stage-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusPending,
		Name:              "compile",
		Created:           time.Now(),
	}
	if _, err := eng.Insert(step); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	bt := &BuildTask{build: &runtime.Build{Id: "build-1"}}
	job := &jobSync{
		step: &runtime.Step{
			Id:      "step-up-1",
			BuildId: "build-1",
			StageId: "stage-1",
			Status:  common.BuildStatusRunning,
			Name:    "compile",
			Started: time.Now(),
		},
	}
	bt.updateStep(job)

	var updated model.TStep
	ok, err := eng.Where("id=?", "step-up-1").Get(&updated)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("step not found")
	}
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStep_ErrorStatus_CancelsCmds(t *testing.T) {
	eng := setupEngineDB(t)
	ctx := context.Background()

	step := &model.TStep{
		Id:                "step-err-1",
		BuildId:           "build-1",
		StageId:           "stage-1",
		PipelineVersionId: "pv-1",
		Status:            common.BuildStatusRunning,
		Name:              "test",
		Created:           time.Now(),
	}
	if _, err := eng.Insert(step); err != nil {
		t.Fatalf("insert step: %v", err)
	}

	cmd := &model.TCmdLine{
		Id:      "cmd-s-1",
		BuildId: "build-1",
		StepId:  "step-err-1",
		Status:  common.BuildStatusRunning,
		Num:     1,
		Content: "go test",
		Created: time.Now(),
	}
	if _, err := eng.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	bt := &BuildTask{build: &runtime.Build{Id: "build-1"}, ctx: ctx}
	job := &jobSync{
		step: &runtime.Step{
			Id:       "step-err-1",
			BuildId:  "build-1",
			StageId:  "stage-1",
			Status:   common.BuildStatusError,
			Error:    "test failed",
			ExitCode: 1,
			Finished: time.Now(),
		},
	}
	bt.updateStep(job)

	// Cmd should be canceled
	var updatedCmd model.TCmdLine
	if ok, _ := eng.Where("id=?", "cmd-s-1").Get(&updatedCmd); ok {
		if updatedCmd.Status != common.BuildStatusCancel {
			t.Errorf("cmd status = %q, want %q", updatedCmd.Status, common.BuildStatusCancel)
		}
	}
}

// --- updateStepCmd ---

func TestUpdateStepCmd_RunningStatus(t *testing.T) {
	eng := setupEngineDB(t)

	cmd := &model.TCmdLine{
		Id:      "cmd-run-1",
		BuildId: "build-1",
		StepId:  "step-1",
		Status:  common.BuildStatusPending,
		Num:     1,
		Content: "echo hello",
		Created: time.Now(),
	}
	if _, err := eng.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	bt := &BuildTask{build: &runtime.Build{Id: "build-1"}}
	now := time.Now()
	cmds := &cmdSync{
		cmd:     &runners.CmdContent{Id: "cmd-run-1"},
		status:  common.BuildStatusRunning,
		started: now,
	}
	bt.updateStepCmd(cmds)

	var updated model.TCmdLine
	ok, err := eng.Where("id=?", "cmd-run-1").Get(&updated)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("cmd not found")
	}
	if updated.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStepCmd_OkStatus_ZeroExitCode(t *testing.T) {
	eng := setupEngineDB(t)

	cmd := &model.TCmdLine{
		Id:      "cmd-ok-1",
		BuildId: "build-1",
		StepId:  "step-1",
		Status:  common.BuildStatusRunning,
		Num:     1,
		Content: "echo hello",
		Created: time.Now(),
	}
	if _, err := eng.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	bt := &BuildTask{build: &runtime.Build{Id: "build-1"}}
	now := time.Now()
	cmds := &cmdSync{
		cmd:      &runners.CmdContent{Id: "cmd-ok-1"},
		status:   common.BuildStatusOk,
		code:     0,
		finished: now,
	}
	bt.updateStepCmd(cmds)

	var updated model.TCmdLine
	if ok, _ := eng.Where("id=?", "cmd-ok-1").Get(&updated); ok {
		if updated.Status != common.BuildStatusOk {
			t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusOk)
		}
	}
}

func TestUpdateStepCmd_CancelStatus(t *testing.T) {
	eng := setupEngineDB(t)

	cmd := &model.TCmdLine{
		Id:      "cmd-cancel-1",
		BuildId: "build-1",
		StepId:  "step-1",
		Status:  common.BuildStatusRunning,
		Num:     1,
		Content: "sleep 10",
		Created: time.Now(),
	}
	if _, err := eng.Insert(cmd); err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	bt := &BuildTask{build: &runtime.Build{Id: "build-1"}}
	now := time.Now()
	cmds := &cmdSync{
		cmd:      &runners.CmdContent{Id: "cmd-cancel-1"},
		status:   common.BuildStatusCancel,
		code:     130,
		finished: now,
	}
	bt.updateStepCmd(cmds)

	var updated model.TCmdLine
	if ok, _ := eng.Where("id=?", "cmd-cancel-1").Get(&updated); ok {
		if updated.Status != common.BuildStatusCancel {
			t.Errorf("status = %q, want %q", updated.Status, common.BuildStatusCancel)
		}
	}
}
