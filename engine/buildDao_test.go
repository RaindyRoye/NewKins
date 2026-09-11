package engine

import (
	"testing"
	"time"

	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/runner/runners"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupTestDb(t *testing.T) *xorm.Engine {
	t.Helper()

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}

	// Use xorm Sync2 to create schema from model structs
	err = db.Sync2(
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TCmdLine{},
	)
	if err != nil {
		t.Fatalf("sync schema: %v", err)
	}

	// Save and restore global Db
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() {
		comm.Db = origDb
		_ = db.Close()
	})

	return db
}

func TestUpdateBuild_RunningStatus(t *testing.T) {
	db := setupTestDb(t)

	// Insert a build
	buildID := "build-123"
	_, err := db.Exec(`INSERT INTO t_build (id, status) VALUES (?, 'running')`, buildID)
	if err != nil {
		t.Fatalf("insert build: %v", err)
	}

	bt := &BuildTask{}
	build := &runtime.Build{
		Id:      buildID,
		Status:  "running",
		Started: time.Now(),
	}

	bt.updateBuild(build)

	// Verify update
	var b model.TBuild
	ok, err := db.Where("id=?", buildID).Get(&b)
	if err != nil {
		t.Fatalf("query build: %v", err)
	}
	if !ok {
		t.Fatal("build not found")
	}
	if b.Status != "running" {
		t.Errorf("expected status 'running', got %q", b.Status)
	}
}

func TestUpdateBuild_FinishedCascadesCancel(t *testing.T) {
	db := setupTestDb(t)

	// Insert a build with stages and steps
	buildID := "build-finish"
	_, err := db.Exec(`INSERT INTO t_build (id, status) VALUES (?, 'running')`, buildID)
	if err != nil {
		t.Fatalf("insert build: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_stage (id, build_id, status) VALUES ('stage-1', ?, 'running')`, buildID)
	if err != nil {
		t.Fatalf("insert stage: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_step (id, stage_id, build_id, status) VALUES ('step-1', 'stage-1', ?, 'running')`, buildID)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}

	bt := &BuildTask{}
	build := &runtime.Build{
		Id:       buildID,
		Status:   "ok", // Finished status
		Finished: time.Now(),
	}

	bt.updateBuild(build)

	// Verify stage was canceled
	var s model.TStage
	ok, err := db.Where("id=?", "stage-1").Get(&s)
	if err != nil {
		t.Fatalf("query stage: %v", err)
	}
	if !ok {
		t.Fatal("stage not found")
	}
	if s.Status != "cancel" {
		t.Errorf("expected stage status 'cancel', got %q", s.Status)
	}

	// Verify step was canceled
	var stp model.TStep
	ok, err = db.Where("id=?", "step-1").Get(&stp)
	if err != nil {
		t.Fatalf("query step: %v", err)
	}
	if !ok {
		t.Fatal("step not found")
	}
	if stp.Status != "cancel" {
		t.Errorf("expected step status 'cancel', got %q", stp.Status)
	}
}

func TestUpdateStage_RunningStatus(t *testing.T) {
	db := setupTestDb(t)

	stageID := "stage-123"
	_, err := db.Exec(`INSERT INTO t_stage (id, build_id, status) VALUES (?, 'build-1', 'pending')`, stageID)
	if err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	bt := &BuildTask{}
	stage := &runtime.Stage{
		Id:      stageID,
		Status:  "running",
		Started: time.Now(),
	}

	bt.updateStage(stage)

	var s model.TStage
	ok, err := db.Where("id=?", stageID).Get(&s)
	if err != nil {
		t.Fatalf("query stage: %v", err)
	}
	if !ok {
		t.Fatal("stage not found")
	}
	if s.Status != "running" {
		t.Errorf("expected status 'running', got %q", s.Status)
	}
}

func TestUpdateStage_FinishedCascadesCancel(t *testing.T) {
	db := setupTestDb(t)

	stageID := "stage-finish"
	_, err := db.Exec(`INSERT INTO t_stage (id, build_id, status) VALUES (?, 'build-1', 'running')`, stageID)
	if err != nil {
		t.Fatalf("insert stage: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_step (id, stage_id, build_id, status) VALUES ('step-1', ?, 'build-1', 'running')`, stageID)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}

	bt := &BuildTask{}
	stage := &runtime.Stage{
		Id:       stageID,
		Status:   "error", // Finished status
		Finished: time.Now(),
	}

	bt.updateStage(stage)

	var stp model.TStep
	ok, err := db.Where("id=?", "step-1").Get(&stp)
	if err != nil {
		t.Fatalf("query step: %v", err)
	}
	if !ok {
		t.Fatal("step not found")
	}
	if stp.Status != "cancel" {
		t.Errorf("expected step status 'cancel', got %q", stp.Status)
	}
}

func TestUpdateStep_RunningStatus(t *testing.T) {
	db := setupTestDb(t)

	stepID := "step-123"
	_, err := db.Exec(`INSERT INTO t_step (id, stage_id, build_id, status) VALUES (?, 'stage-1', 'build-1', 'pending')`, stepID)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}

	bt := &BuildTask{}
	job := &jobSync{
		step: &runtime.Step{
			Id:      stepID,
			Status:  "running",
			Started: time.Now(),
		},
	}

	bt.updateStep(job)

	var s model.TStep
	ok, err := db.Where("id=?", stepID).Get(&s)
	if err != nil {
		t.Fatalf("query step: %v", err)
	}
	if !ok {
		t.Fatal("step not found")
	}
	if s.Status != "running" {
		t.Errorf("expected status 'running', got %q", s.Status)
	}
}

func TestUpdateStep_FinishedCascadesCancel(t *testing.T) {
	db := setupTestDb(t)

	stepID := "step-finish"
	_, err := db.Exec(`INSERT INTO t_step (id, stage_id, build_id, status) VALUES (?, 'stage-1', 'build-1', 'running')`, stepID)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_cmd_line (id, step_id, build_id, status) VALUES ('cmd-1', ?, 'build-1', 'running')`, stepID)
	if err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	bt := &BuildTask{}
	job := &jobSync{
		step: &runtime.Step{
			Id:       stepID,
			Status:   "ok", // Finished status
			Finished: time.Now(),
		},
	}

	bt.updateStep(job)

	var cmd model.TCmdLine
	ok, err := db.Where("id=?", "cmd-1").Get(&cmd)
	if err != nil {
		t.Fatalf("query cmd: %v", err)
	}
	if !ok {
		t.Fatal("cmd not found")
	}
	if cmd.Status != "cancel" {
		t.Errorf("expected cmd status 'cancel', got %q", cmd.Status)
	}
}

func TestUpdateStepCmd_RunningStatus(t *testing.T) {
	db := setupTestDb(t)

	cmdID := "cmd-123"
	_, err := db.Exec(`INSERT INTO t_cmd_line (id, step_id, build_id, status) VALUES (?, 'step-1', 'build-1', 'pending')`, cmdID)
	if err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	bt := &BuildTask{}
	cmd := &cmdSync{
		cmd: &runners.CmdContent{
			Id: cmdID,
		},
		status:  "running",
		started: time.Now(),
	}

	bt.updateStepCmd(cmd)

	var c model.TCmdLine
	ok, err := db.Where("id=?", cmdID).Get(&c)
	if err != nil {
		t.Fatalf("query cmd: %v", err)
	}
	if !ok {
		t.Fatal("cmd not found")
	}
	if c.Status != "running" {
		t.Errorf("expected status 'running', got %q", c.Status)
	}
	if c.Started.IsZero() {
		t.Error("expected started time to be set")
	}
}

func TestUpdateStepCmd_OkStatus(t *testing.T) {
	db := setupTestDb(t)

	cmdID := "cmd-ok"
	_, err := db.Exec(`INSERT INTO t_cmd_line (id, step_id, build_id, status) VALUES (?, 'step-1', 'build-1', 'running')`, cmdID)
	if err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	bt := &BuildTask{}
	cmd := &cmdSync{
		cmd: &runners.CmdContent{
			Id: cmdID,
		},
		status:   "ok",
		code:     0,
		finished: time.Now(),
	}

	bt.updateStepCmd(cmd)

	var c model.TCmdLine
	ok, err := db.Where("id=?", cmdID).Get(&c)
	if err != nil {
		t.Fatalf("query cmd: %v", err)
	}
	if !ok {
		t.Fatal("cmd not found")
	}
	if c.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", c.Status)
	}
	if c.Finished.IsZero() {
		t.Error("expected finished time to be set")
	}
}
