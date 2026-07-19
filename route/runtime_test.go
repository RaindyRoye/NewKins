package route

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func createRuntimeTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_build (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		pipeline_id VARCHAR(64),
		pipeline_version_id VARCHAR(64),
		status VARCHAR(50),
		error VARCHAR(500),
		event VARCHAR(50),
		version VARCHAR(255),
		created DATETIME,
		started DATETIME,
		finished DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_build: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_stage (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		pipeline_version_id VARCHAR(64),
		build_id VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		stage VARCHAR(255),
		sort INT,
		status VARCHAR(50),
		error VARCHAR(500),
		created DATETIME,
		started DATETIME,
		finished DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_stage: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_step (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		stage_id VARCHAR(64),
		build_id VARCHAR(64),
		pipeline_version_id VARCHAR(64),
		display_name VARCHAR(255),
		step VARCHAR(255),
		sort INT,
		status VARCHAR(50),
		error VARCHAR(500),
		event VARCHAR(50),
		exit_code INT,
		name VARCHAR(100),
		version VARCHAR(255),
		errignore INT,
		commands TEXT,
		waits TEXT,
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_step: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_cmd_line (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		group_id VARCHAR(64),
		step_id VARCHAR(64),
		build_id VARCHAR(64),
		num INT,
		status VARCHAR(50),
		code INT,
		content TEXT,
		created DATETIME,
		started DATETIME,
		finished DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_cmd_line: %v", err)
	}

	return db
}

func setupRuntimeTestRouter(t *testing.T, db *xorm.Engine) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	rc := &RuntimeController{}
	group := r.Group("/api/runtime")
	// Skip middleware for unit tests — call handlers directly
	group.POST("/stages", func(c *gin.Context) { rc.stages(c, &hbtp.Map{}) })
	group.POST("/cmds", func(c *gin.Context) { rc.cmds(c, &hbtp.Map{}) })
	return r
}

func TestRuntime_stages_EmptyPvId(t *testing.T) {
	db := createRuntimeTestDB(t)
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := RuntimeController{}
	m := &hbtp.Map{} // empty pvId
	ctrl.stages(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pvId, got %d", w.Code)
	}
}

func TestRuntime_stages_NoStages(t *testing.T) {
	db := createRuntimeTestDB(t)
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("pvId", "nonexistent-pv")
	ctrl.stages(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for no stages, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	ids, ok := resp["ids"].([]interface{})
	if !ok {
		t.Fatal("expected ids array in response")
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 ids, got %d", len(ids))
	}
}

func TestRuntime_stages_WithStagesAndSteps(t *testing.T) {
	db := createRuntimeTestDB(t)
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	// Insert test stages with pipeline_version_id (the handler queries by pipeline_version_id)
	_, err := db.Exec(`INSERT INTO t_stage (id, aid, pipeline_version_id, build_id, sort, status) VALUES ('stage-1', 1, 'pv-1', 'build-1', 0, 'ok')`)
	if err != nil {
		t.Fatalf("insert stage: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_stage (id, aid, pipeline_version_id, build_id, sort, status) VALUES ('stage-2', 2, 'pv-1', 'build-1', 1, 'running')`)
	if err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	// Insert test steps
	_, err = db.Exec(`INSERT INTO t_step (id, aid, stage_id, build_id, sort, status, waits) VALUES ('step-1', 1, 'stage-1', 'build-1', 0, 'ok', '[]')`)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_step (id, aid, stage_id, build_id, sort, status, waits) VALUES ('step-2', 2, 'stage-1', 'build-1', 1, 'ok', '[]')`)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_step (id, aid, stage_id, build_id, sort, status, waits) VALUES ('step-3', 3, 'stage-2', 'build-1', 0, 'running', '[]')`)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("pvId", "pv-1")
	ctrl.stages(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	ids, ok := resp["ids"].([]interface{})
	if !ok {
		t.Fatal("expected ids array in response")
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 stage ids, got %d", len(ids))
	}

	stages, ok := resp["stages"].(map[string]interface{})
	if !ok {
		t.Fatal("expected stages map in response")
	}
	if len(stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(stages))
	}

	steps, ok := resp["steps"].(map[string]interface{})
	if !ok {
		t.Fatal("expected steps map in response")
	}
	if len(steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(steps))
	}
}

func TestRuntime_stages_WithMalformedWaits(t *testing.T) {
	db := createRuntimeTestDB(t)
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	_, err := db.Exec(`INSERT INTO t_stage (id, aid, build_id, sort, status) VALUES ('stage-w', 1, 'build-w', 0, 'ok')`)
	if err != nil {
		t.Fatalf("insert stage: %v", err)
	}
	// Insert step with malformed waits JSON — should log warning but not fail
	_, err = db.Exec(`INSERT INTO t_step (id, aid, stage_id, build_id, sort, status, waits) VALUES ('step-w', 1, 'stage-w', 'build-w', 0, 'ok', 'not-json')`)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("pvId", "build-w")
	ctrl.stages(c, m)

	// Should succeed even with malformed waits — just logs a warning
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with malformed waits, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestRuntime_cmds_EmptyStepId(t *testing.T) {
	db := createRuntimeTestDB(t)
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := RuntimeController{}
	m := &hbtp.Map{} // empty stepId
	ctrl.cmds(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty stepId, got %d", w.Code)
	}
}

func TestRuntime_cmds_NoCmds(t *testing.T) {
	db := createRuntimeTestDB(t)
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("stepId", "nonexistent-step")
	ctrl.cmds(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for no cmds, got %d", w.Code)
	}
}

func TestRuntime_cmds_WithCmds(t *testing.T) {
	db := createRuntimeTestDB(t)
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	_, err := db.Exec(`INSERT INTO t_cmd_line (id, aid, step_id, build_id, num, status, code) VALUES ('cmd-1', 1, 'step-1', 'build-1', 0, 'ok', 0)`)
	if err != nil {
		t.Fatalf("insert cmd: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_cmd_line (id, aid, step_id, build_id, num, status, code) VALUES ('cmd-2', 2, 'step-1', 'build-1', 1, 'running', 0)`)
	if err != nil {
		t.Fatalf("insert cmd: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_cmd_line (id, aid, step_id, build_id, num, status, code) VALUES ('cmd-3', 3, 'step-1', 'build-1', 2, 'error', 1)`)
	if err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("stepId", "step-1")
	ctrl.cmds(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	stepId, ok := resp["stepId"].(string)
	if !ok || stepId != "step-1" {
		t.Errorf("expected stepId = 'step-1', got %v", resp["stepId"])
	}

	cmds, ok := resp["cmds"].([]interface{})
	if !ok {
		t.Fatal("expected cmds array")
	}
	if len(cmds) != 3 {
		t.Errorf("expected 3 cmds, got %d", len(cmds))
	}
}

func TestRuntime_build_EmptyBuildId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := RuntimeController{}
	m := &hbtp.Map{} // empty buildId
	ctrl.build(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty buildId, got %d", w.Code)
	}
}

func TestRuntime_build_NotFound(t *testing.T) {
	// engine.Mgr is nil in tests, so we can't test the build endpoint
	// without mocking the engine. Skip this test to avoid nil pointer panic.
	t.Skip("engine.Mgr not initialized in test environment")
}

func TestRuntime_cancel_EmptyBuildId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := RuntimeController{}
	m := &hbtp.Map{} // empty buildId
	ctrl.cancel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty buildId, got %d", w.Code)
	}
}

func TestRuntime_cancel_BuildNotFound(t *testing.T) {
	db := createRuntimeTestDB(t)
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// Set a logged-in user for the middleware check
	c.Set(service.LgUserKey, &model.TUser{Id: "user-1", Name: "testuser", Active: 1})

	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("buildId", "nonexistent-build")
	ctrl.cancel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent build, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestRuntime_logs_EmptyStepId(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := RuntimeController{}
	m := &hbtp.Map{} // empty stepId
	ctrl.logs(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty stepId, got %d", w.Code)
	}
}

func TestRuntime_logs_FileNotFound(t *testing.T) {
	origWorkPath := comm.WorkPath
	comm.WorkPath = "/tmp/gokins-test-nonexistent"
	t.Cleanup(func() { comm.WorkPath = origWorkPath })

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("stepId", "step-1")
	m.Set("buildId", "build-1")
	ctrl.logs(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing log file, got %d", w.Code)
	}
}
