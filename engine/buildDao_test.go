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

// setupTestDB creates an in-memory SQLite database with the tables needed
// by buildDao methods and wires it into comm.Db. The caller must call the
// returned cleanup function.
func setupTestDB(t *testing.T) func() {
	t.Helper()
	origDb := comm.Db
	origCtx := comm.Ctx

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	comm.Db = db

	// Create the tables that buildDao methods touch.
	stmts := []string{
		`CREATE TABLE t_build (
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
		)`,
		`CREATE TABLE t_stage (
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
		)`,
		`CREATE TABLE t_step (
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
		)`,
		`CREATE TABLE t_cmd_line (
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
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("exec %q: %v", s[:30], err)
		}
	}

	return func() {
		_ = db.Close()
		comm.Db = origDb
		comm.Ctx = origCtx
	}
}

// insertBuildRow inserts a row into t_build so updateBuild can find it.
func insertBuildRow(t *testing.T, id string) {
	t.Helper()
	_, err := comm.Db.Exec(
		"INSERT INTO t_build (id, status, error, event) VALUES (?,?,?,?)",
		id, common.BuildStatusRunning, "", "",
	)
	if err != nil {
		t.Fatalf("insert t_build: %v", err)
	}
}

// insertStageRow inserts a row into t_stage so updateStage can find it.
func insertStageRow(t *testing.T, id, buildID, status string) {
	t.Helper()
	_, err := comm.Db.Exec(
		"INSERT INTO t_stage (id, build_id, status) VALUES (?,?,?)",
		id, buildID, status,
	)
	if err != nil {
		t.Fatalf("insert t_stage: %v", err)
	}
}

// insertStepRow inserts a row into t_step.
func insertStepRow(t *testing.T, id, buildID, stageID, status string) {
	t.Helper()
	_, err := comm.Db.Exec(
		"INSERT INTO t_step (id, build_id, stage_id, status) VALUES (?,?,?,?)",
		id, buildID, stageID, status,
	)
	if err != nil {
		t.Fatalf("insert t_step: %v", err)
	}
}

// insertCmdLineRow inserts a row into t_cmd_line.
func insertCmdLineRow(t *testing.T, id, buildID, stepID, status string) {
	t.Helper()
	_, err := comm.Db.Exec(
		"INSERT INTO t_cmd_line (id, build_id, step_id, status) VALUES (?,?,?,?)",
		id, buildID, stepID, status,
	)
	if err != nil {
		t.Fatalf("insert t_cmd_line: %v", err)
	}
}

// --- updateBuild tests ---

func TestUpdateBuild_Running(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertBuildRow(t, "b1")
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		ctx:   context.Background(),
	}
	bd := &runtime.Build{
		Id:      "b1",
		Status:  common.BuildStatusRunning,
		Error:   "",
		Event:   "get-repo",
		Started: time.Now(),
	}
	bt.updateBuild(bd)

	var row model.TBuild
	ok, err := comm.Db.Where("id=?", "b1").Get(&row)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("build row not found")
	}
	if row.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", row.Status, common.BuildStatusRunning)
	}
	if row.Event != "get-repo" {
		t.Errorf("event = %q, want %q", row.Event, "get-repo")
	}
}

func TestUpdateBuild_Cancelled_CancelsRelated(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertBuildRow(t, "b2")
	// Insert related rows that should be cancelled
	insertStageRow(t, "s1", "b2", common.BuildStatusRunning)
	insertStepRow(t, "st1", "b2", "s1", common.BuildStatusRunning)
	insertCmdLineRow(t, "c1", "b2", "st1", common.BuildStatusRunning)

	bt := &BuildTask{
		build: &runtime.Build{Id: "b2"},
		ctx:   context.Background(),
	}
	bd := &runtime.Build{
		Id:       "b2",
		Status:   common.BuildStatusCancel,
		Finished: time.Now(),
	}
	bt.updateBuild(bd)

	// Stage should be cancelled
	var stg model.TStage
	if _, err := comm.Db.Where("id=?", "s1").Get(&stg); err != nil {
		t.Fatalf("query stage: %v", err)
	}
	if stg.Status != common.BuildStatusCancel {
		t.Errorf("stage status = %q, want %q", stg.Status, common.BuildStatusCancel)
	}

	// Step should be cancelled
	var stp model.TStep
	if _, err := comm.Db.Where("id=?", "st1").Get(&stp); err != nil {
		t.Fatalf("query step: %v", err)
	}
	if stp.Status != common.BuildStatusCancel {
		t.Errorf("step status = %q, want %q", stp.Status, common.BuildStatusCancel)
	}

	// CmdLine should be cancelled
	var cmd model.TCmdLine
	if _, err := comm.Db.Where("id=?", "c1").Get(&cmd); err != nil {
		t.Fatalf("query cmd: %v", err)
	}
	if cmd.Status != common.BuildStatusCancel {
		t.Errorf("cmd status = %q, want %q", cmd.Status, common.BuildStatusCancel)
	}
}

func TestUpdateBuild_Error_PreservesAlreadyEnded(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertBuildRow(t, "b3")
	// Insert a stage already ended with OK — should NOT be overwritten
	insertStageRow(t, "s2", "b3", common.BuildStatusOk)
	// Insert a stage still running — should be cancelled
	insertStageRow(t, "s3", "b3", common.BuildStatusRunning)

	bt := &BuildTask{
		build: &runtime.Build{Id: "b3"},
		ctx:   context.Background(),
	}
	bd := &runtime.Build{
		Id:       "b3",
		Status:   common.BuildStatusError,
		Error:    "build failed",
		Finished: time.Now(),
	}
	bt.updateBuild(bd)

	// s2 (OK) should remain OK
	var stg2 model.TStage
	if _, err := comm.Db.Where("id=?", "s2").Get(&stg2); err != nil {
		t.Fatalf("query s2: %v", err)
	}
	if stg2.Status != common.BuildStatusOk {
		t.Errorf("s2 status = %q, want %q (should not overwrite ended)", stg2.Status, common.BuildStatusOk)
	}

	// s3 (running) should be cancelled
	var stg3 model.TStage
	if _, err := comm.Db.Where("id=?", "s3").Get(&stg3); err != nil {
		t.Fatalf("query s3: %v", err)
	}
	if stg3.Status != common.BuildStatusCancel {
		t.Errorf("s3 status = %q, want %q", stg3.Status, common.BuildStatusCancel)
	}
}

func TestUpdateBuild_NilContext_FallsBackToGlobal(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertBuildRow(t, "b4")
	bt := &BuildTask{
		build: &runtime.Build{Id: "b4"},
		// ctx is nil — taskCtx() should fall back to comm.Ctx
	}
	bd := &runtime.Build{
		Id:     "b4",
		Status: common.BuildStatusOk,
	}
	bt.updateBuild(bd)

	var row model.TBuild
	if _, err := comm.Db.Where("id=?", "b4").Get(&row); err != nil {
		t.Fatalf("query: %v", err)
	}
	if row.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", row.Status, common.BuildStatusOk)
	}
}

// --- updateStage tests ---

func TestUpdateStage_Running(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertStageRow(t, "s10", "b10", common.BuildStatusPending)
	bt := &BuildTask{
		build: &runtime.Build{Id: "b10"},
		ctx:   context.Background(),
	}
	stg := &runtime.Stage{
		Id:      "s10",
		Status:  common.BuildStatusRunning,
		Started: time.Now(),
	}
	bt.updateStage(stg)

	var row model.TStage
	if _, err := comm.Db.Where("id=?", "s10").Get(&row); err != nil {
		t.Fatalf("query: %v", err)
	}
	if row.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", row.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStage_Cancelled_CancelsSteps(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertStageRow(t, "s20", "b20", common.BuildStatusRunning)
	insertStepRow(t, "st20", "b20", "s20", common.BuildStatusRunning)
	insertStepRow(t, "st21", "b20", "s20", common.BuildStatusOk) // already ended

	bt := &BuildTask{
		build: &runtime.Build{Id: "b20"},
		ctx:   context.Background(),
	}
	stg := &runtime.Stage{
		Id:       "s20",
		Status:   common.BuildStatusCancel,
		Finished: time.Now(),
	}
	bt.updateStage(stg)

	// st20 (running) should be cancelled
	var stp20 model.TStep
	if _, err := comm.Db.Where("id=?", "st20").Get(&stp20); err != nil {
		t.Fatalf("query st20: %v", err)
	}
	if stp20.Status != common.BuildStatusCancel {
		t.Errorf("st20 status = %q, want %q", stp20.Status, common.BuildStatusCancel)
	}

	// st21 (OK) should remain OK
	var stp21 model.TStep
	if _, err := comm.Db.Where("id=?", "st21").Get(&stp21); err != nil {
		t.Fatalf("query st21: %v", err)
	}
	if stp21.Status != common.BuildStatusOk {
		t.Errorf("st21 status = %q, want %q", stp21.Status, common.BuildStatusOk)
	}
}

func TestUpdateStage_Error(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertStageRow(t, "s30", "b30", common.BuildStatusRunning)
	bt := &BuildTask{
		build: &runtime.Build{Id: "b30"},
		ctx:   context.Background(),
	}
	stg := &runtime.Stage{
		Id:       "s30",
		Status:   common.BuildStatusError,
		Error:    "stage failed",
		Finished: time.Now(),
	}
	bt.updateStage(stg)

	var row model.TStage
	if _, err := comm.Db.Where("id=?", "s30").Get(&row); err != nil {
		t.Fatalf("query: %v", err)
	}
	if row.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", row.Status, common.BuildStatusError)
	}
	if row.Error != "stage failed" {
		t.Errorf("error = %q, want %q", row.Error, "stage failed")
	}
}

func TestUpdateStage_NilContext(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertStageRow(t, "s40", "b40", common.BuildStatusPending)
	bt := &BuildTask{
		build: &runtime.Build{Id: "b40"},
		// ctx is nil
	}
	stg := &runtime.Stage{
		Id:     "s40",
		Status: common.BuildStatusRunning,
	}
	bt.updateStage(stg)

	var row model.TStage
	if _, err := comm.Db.Where("id=?", "s40").Get(&row); err != nil {
		t.Fatalf("query: %v", err)
	}
	if row.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", row.Status, common.BuildStatusRunning)
	}
}

// --- updateStep tests ---

func TestUpdateStep_Running(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertStepRow(t, "st50", "b50", "s50", common.BuildStatusPending)
	bt := &BuildTask{
		build: &runtime.Build{Id: "b50"},
		ctx:   context.Background(),
	}
	job := &jobSync{
		step: &runtime.Step{
			Id:      "st50",
			Status:  common.BuildStatusRunning,
			Started: time.Now(),
		},
	}
	bt.updateStep(job)

	var row model.TStep
	if _, err := comm.Db.Where("id=?", "st50").Get(&row); err != nil {
		t.Fatalf("query: %v", err)
	}
	if row.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", row.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStep_Cancelled_CancelsCmds(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertStepRow(t, "st60", "b60", "s60", common.BuildStatusRunning)
	insertCmdLineRow(t, "c60", "b60", "st60", common.BuildStatusRunning)
	insertCmdLineRow(t, "c61", "b60", "st60", common.BuildStatusOk) // already ended

	bt := &BuildTask{
		build: &runtime.Build{Id: "b60"},
		ctx:   context.Background(),
	}
	job := &jobSync{
		step: &runtime.Step{
			Id:       "st60",
			Status:   common.BuildStatusCancel,
			Finished: time.Now(),
		},
	}
	bt.updateStep(job)

	// c60 (running) should be cancelled
	var cmd60 model.TCmdLine
	if _, err := comm.Db.Where("id=?", "c60").Get(&cmd60); err != nil {
		t.Fatalf("query c60: %v", err)
	}
	if cmd60.Status != common.BuildStatusCancel {
		t.Errorf("c60 status = %q, want %q", cmd60.Status, common.BuildStatusCancel)
	}

	// c61 (OK) should remain OK
	var cmd61 model.TCmdLine
	if _, err := comm.Db.Where("id=?", "c61").Get(&cmd61); err != nil {
		t.Fatalf("query c61: %v", err)
	}
	if cmd61.Status != common.BuildStatusOk {
		t.Errorf("c61 status = %q, want %q", cmd61.Status, common.BuildStatusOk)
	}
}

func TestUpdateStep_WithError(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertStepRow(t, "st70", "b70", "s70", common.BuildStatusRunning)
	bt := &BuildTask{
		build: &runtime.Build{Id: "b70"},
		ctx:   context.Background(),
	}
	job := &jobSync{
		step: &runtime.Step{
			Id:       "st70",
			Status:   common.BuildStatusError,
			Error:    "step failed",
			ExitCode: 1,
			Finished: time.Now(),
		},
	}
	bt.updateStep(job)

	var row model.TStep
	if _, err := comm.Db.Where("id=?", "st70").Get(&row); err != nil {
		t.Fatalf("query: %v", err)
	}
	if row.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", row.Status, common.BuildStatusError)
	}
	if row.Error != "step failed" {
		t.Errorf("error = %q, want %q", row.Error, "step failed")
	}
	if row.ExitCode != 1 {
		t.Errorf("exitCode = %d, want 1", row.ExitCode)
	}
}

// --- updateStepCmd tests ---

func TestUpdateStepCmd_Running(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertCmdLineRow(t, "c70", "b70", "st70", common.BuildStatusPending)
	bt := &BuildTask{
		build: &runtime.Build{Id: "b70"},
		ctx:   context.Background(),
	}
	now := time.Now()
	cmd := &cmdSync{
		cmd:     &runners.CmdContent{Id: "c70"},
		status:  common.BuildStatusRunning,
		started: now,
	}
	bt.updateStepCmd(cmd)

	var row model.TCmdLine
	if _, err := comm.Db.Where("id=?", "c70").Get(&row); err != nil {
		t.Fatalf("query: %v", err)
	}
	if row.Status != common.BuildStatusRunning {
		t.Errorf("status = %q, want %q", row.Status, common.BuildStatusRunning)
	}
}

func TestUpdateStepCmd_Finished(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertCmdLineRow(t, "c80", "b80", "st80", common.BuildStatusRunning)
	bt := &BuildTask{
		build: &runtime.Build{Id: "b80"},
		ctx:   context.Background(),
	}
	now := time.Now()
	cmd := &cmdSync{
		cmd:      &runners.CmdContent{Id: "c80"},
		status:   common.BuildStatusOk,
		code:     0,
		finished: now,
	}
	bt.updateStepCmd(cmd)

	var row model.TCmdLine
	if _, err := comm.Db.Where("id=?", "c80").Get(&row); err != nil {
		t.Fatalf("query: %v", err)
	}
	if row.Status != common.BuildStatusOk {
		t.Errorf("status = %q, want %q", row.Status, common.BuildStatusOk)
	}
}

func TestUpdateStepCmd_WithError(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	insertCmdLineRow(t, "c90", "b90", "st90", common.BuildStatusRunning)
	bt := &BuildTask{
		build: &runtime.Build{Id: "b90"},
		ctx:   context.Background(),
	}
	cmd := &cmdSync{
		cmd:      &runners.CmdContent{Id: "c90"},
		status:   common.BuildStatusError,
		code:     127,
		finished: time.Now(),
	}
	bt.updateStepCmd(cmd)

	var row model.TCmdLine
	if _, err := comm.Db.Where("id=?", "c90").Get(&row); err != nil {
		t.Fatalf("query: %v", err)
	}
	if row.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", row.Status, common.BuildStatusError)
	}
	// Note: updateStepCmd only persists (status, finished) for non-running states;
	// code is not in the Cols list, so it is NOT written to the DB. Verify finished is set.
	if row.Finished.IsZero() {
		t.Error("finished should be set for non-running status")
	}
}
