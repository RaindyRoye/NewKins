package route

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/util"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupRuntimeTestRouter(t *testing.T, db *xorm.Engine) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	origKey := comm.Cfg.Server.LoginKey
	comm.Cfg.Server.LoginKey = "test-runtime-key-12345"
	t.Cleanup(func() { comm.Cfg.Server.LoginKey = origKey })

	// Create a test user in the database
	_, err := db.Exec(`INSERT INTO t_user (id, name, pass, nick, active, created, login_time)
		VALUES ('testuser', 'testuser', 'hash', 'Test', 1, ?, ?)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	r := gin.New()
	rc := &RuntimeController{}
	rc.Routes(r.Group("/api/runtime"))
	return r
}

func createRuntimeToken(t *testing.T) string {
	t.Helper()
	token, err := util.CreateToken(map[string]any{"uid": "testuser"}, "test-runtime-key-12345", time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	return token
}

func createRuntimeTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL,
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(100),
		pass VARCHAR(255),
		nick VARCHAR(100),
		avatar VARCHAR(500),
		active INT DEFAULT 1,
		created DATETIME,
		login_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_user: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_stage (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
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
		sort INT
	)`)
	if err != nil {
		t.Fatalf("create t_stage: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_step (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		build_id VARCHAR(64),
		stage_id VARCHAR(64),
		display_name VARCHAR(255),
		pipeline_version_id VARCHAR(64),
		step VARCHAR(255),
		status VARCHAR(100),
		exit_code BIGINT(20),
		error VARCHAR(500),
		name VARCHAR(255),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		errignore INT,
		commands TEXT,
		waits TEXT,
		sort INT
	)`)
	if err != nil {
		t.Fatalf("create t_step: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_cmd_line (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
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
	if err != nil {
		t.Fatalf("create t_cmd_line: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_build (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
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
		t.Fatalf("create t_build: %v", err)
	}

	return db
}

func TestStages_MissingPvId(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)
	token := createRuntimeToken(t)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/stages", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing pvId, got %d", w.Code)
	}
}

func TestStages_EmptyDb(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)
	token := createRuntimeToken(t)

	body, _ := json.Marshal(map[string]any{"pvId": "nonexistent-version"})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/stages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if ids, ok := resp["ids"].([]any); !ok || len(ids) != 0 {
		t.Errorf("expected empty ids, got %v", resp["ids"])
	}
}

func TestStages_WithSteps(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)
	token := createRuntimeToken(t)

	// Insert stages and steps
	_, err := db.Exec(`INSERT INTO t_stage (id, pipeline_version_id, build_id, name, stage, sort)
		VALUES ('stage1', 'pv1', 'build1', 'Build', 'build', 0)`)
	if err != nil {
		t.Fatalf("insert stage: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_step (id, build_id, stage_id, name, step, sort, waits)
		VALUES ('step1', 'build1', 'stage1', 'test', 'test', 0, '[]')`)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_step (id, build_id, stage_id, name, step, sort, waits)
		VALUES ('step2', 'build1', 'stage1', 'lint', 'lint', 1, '[]')`)
	if err != nil {
		t.Fatalf("insert step2: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"pvId": "pv1"})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/stages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	ids, ok := resp["ids"].([]any)
	if !ok || len(ids) != 1 {
		t.Errorf("expected 1 stage id, got %v", resp["ids"])
	}

	steps, ok := resp["steps"].(map[string]any)
	if !ok || len(steps) != 2 {
		t.Errorf("expected 2 steps, got %v", resp["steps"])
	}
}

func TestCmds_MissingStepId(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)
	token := createRuntimeToken(t)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/cmds", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing stepId, got %d", w.Code)
	}
}

func TestCmds_WithCommands(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)
	token := createRuntimeToken(t)

	_, err := db.Exec(`INSERT INTO t_cmd_line (id, step_id, num, content) VALUES ('cmd1', 'step1', 0, 'echo hello')`)
	if err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"stepId": "step1"})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/cmds", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["stepId"] != "step1" {
		t.Errorf("stepId = %v, want 'step1'", resp["stepId"])
	}
	cmds, ok := resp["cmds"].([]any)
	if !ok || len(cmds) != 1 {
		t.Errorf("expected 1 command, got %v", resp["cmds"])
	}
}

func TestBuild_MissingBuildId(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)
	token := createRuntimeToken(t)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/build", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing buildId, got %d", w.Code)
	}
}

func TestBuild_NotInEngine(t *testing.T) {
	t.Skip("Requires engine.Mgr.BuildEgn() initialization - tested in integration tests")
}

func TestCancel_MissingBuildId(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)
	token := createRuntimeToken(t)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/cancel", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing buildId, got %d", w.Code)
	}
}

func TestCancel_BuildNotFound(t *testing.T) {
	t.Skip("Requires engine.Mgr.BuildEgn() initialization - tested in integration tests")
}

func TestLogs_MissingStepId(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)
	token := createRuntimeToken(t)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/logs", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing stepId, got %d", w.Code)
	}
}

func TestLogs_FileNotFound(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)
	token := createRuntimeToken(t)

	// comm.WorkPath is empty/default, so file won't exist
	body, _ := json.Marshal(map[string]any{"stepId": "step1", "buildId": "build1"})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/logs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing log file, got %d", w.Code)
	}
}

// Suppress unused import warnings
var (
	_ = context.Background
	_ hbtp.Map
)
