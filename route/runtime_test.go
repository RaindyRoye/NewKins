package route

import (
	"bytes"
	"encoding/json"
	"io"
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

func setupRuntimeTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

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
		sort INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("failed to create t_stage: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_step (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		build_id VARCHAR(64),
		stage_id VARCHAR(100),
		display_name VARCHAR(255),
		pipeline_version_id VARCHAR(64),
		step VARCHAR(255),
		status VARCHAR(100),
		exit_code BIGINT,
		error VARCHAR(500),
		name VARCHAR(100),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		errignore INT,
		waits JSON,
		sort BIGINT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("failed to create t_step: %v", err)
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
		t.Fatalf("failed to create t_cmd_line: %v", err)
	}

	comm.Db = db
	return db
}

// Helper function to create a reader from bytes
func jsonReader(t *testing.T, data []byte) io.Reader {
	t.Helper()
	return bytes.NewReader(data)
}

// makeRuntimeGinContext creates a gin context for testing runtime handlers
func makeRuntimeGinContext(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	} else {
		req = httptest.NewRequest("POST", "/test", nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	if loggedInUser != nil {
		c.Set(service.LgUserKey, loggedInUser)
	}
	return c, w
}

func TestRuntimeController_stages_MissingPvId(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)

	usr := &model.TUser{Id: "testuser", Name: "test", Active: 1}
	m := &hbtp.Map{}

	c, w := makeRuntimeGinContext(t, m, usr)
	ctrl := RuntimeController{}
	ctrl.stages(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestRuntimeController_stages_EmptyResult(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)

	usr := &model.TUser{Id: "testuser", Name: "test", Active: 1}
	m := &hbtp.Map{}
	m.Set("pvId", "nonexistent-pv-id")

	c, w := makeRuntimeGinContext(t, m, usr)
	ctrl := RuntimeController{}
	ctrl.stages(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should have empty arrays
	if ids, ok := result["ids"].([]interface{}); ok {
		if len(ids) != 0 {
			t.Errorf("expected 0 ids, got %d", len(ids))
		}
	} else {
		t.Error("ids field missing or not an array")
	}
}

func TestRuntimeController_stages_WithStages(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)

	// Insert test data - both stages must have steps to be included in results
	_, _ = comm.Db.Exec(`INSERT INTO t_stage (id, pipeline_version_id, build_id, name, status, sort) 
		VALUES ('stage1', 'pv1', 'build1', 'Stage 1', 'pending', 1)`)
	_, _ = comm.Db.Exec(`INSERT INTO t_stage (id, pipeline_version_id, build_id, name, status, sort) 
		VALUES ('stage2', 'pv1', 'build1', 'Stage 2', 'running', 2)`)
	_, _ = comm.Db.Exec(`INSERT INTO t_step (id, stage_id, build_id, name, status, sort, waits) 
		VALUES ('step1', 'stage1', 'build1', 'Step 1', 'pending', 1, '[]')`)
	_, _ = comm.Db.Exec(`INSERT INTO t_step (id, stage_id, build_id, name, status, sort, waits) 
		VALUES ('step2', 'stage1', 'build1', 'Step 2', 'running', 2, '["step1"]')`)
	_, _ = comm.Db.Exec(`INSERT INTO t_step (id, stage_id, build_id, name, status, sort, waits) 
		VALUES ('step3', 'stage2', 'build1', 'Step 3', 'pending', 1, '[]')`)

	usr := &model.TUser{Id: "testuser", Name: "test", Active: 1}
	m := &hbtp.Map{}
	m.Set("pvId", "pv1")

	c, w := makeRuntimeGinContext(t, m, usr)
	ctrl := RuntimeController{}
	ctrl.stages(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should have stages and steps
	if ids, ok := result["ids"].([]interface{}); ok {
		if len(ids) != 2 {
			t.Errorf("expected 2 stage ids, got %d", len(ids))
		}
	} else {
		t.Error("ids field missing or not an array")
	}

	if stages, ok := result["stages"].(map[string]interface{}); ok {
		if len(stages) != 2 {
			t.Errorf("expected 2 stages, got %d", len(stages))
		}
	} else {
		t.Error("stages field missing or not a map")
	}

	if steps, ok := result["steps"].(map[string]interface{}); ok {
		if len(steps) != 3 {
			t.Errorf("expected 3 steps, got %d", len(steps))
		}
	} else {
		t.Error("steps field missing or not a map")
	}
}

func TestRuntimeController_cmds_MissingStepId(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)

	usr := &model.TUser{Id: "testuser", Name: "test", Active: 1}
	m := &hbtp.Map{}

	c, w := makeRuntimeGinContext(t, m, usr)
	ctrl := RuntimeController{}
	ctrl.cmds(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestRuntimeController_cmds_EmptyResult(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)

	usr := &model.TUser{Id: "testuser", Name: "test", Active: 1}
	m := &hbtp.Map{}
	m.Set("stepId", "nonexistent-step-id")

	c, w := makeRuntimeGinContext(t, m, usr)
	ctrl := RuntimeController{}
	ctrl.cmds(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if stepId, ok := result["stepId"].(string); ok {
		if stepId != "nonexistent-step-id" {
			t.Errorf("stepId = %q, want %q", stepId, "nonexistent-step-id")
		}
	} else {
		t.Error("stepId field missing or not a string")
	}

	// cmds field should exist, may be null or empty array depending on JSON encoding
	if cmds, ok := result["cmds"].([]interface{}); ok {
		if len(cmds) != 0 {
			t.Errorf("expected 0 cmds, got %d", len(cmds))
		}
	} else if cmdsVal, exists := result["cmds"]; exists {
		// null is also acceptable for empty slices in JSON
		if cmdsVal != nil {
			t.Errorf("cmds should be null or empty array, got %T", cmdsVal)
		}
	} else {
		t.Error("cmds field missing")
	}
}

func TestRuntimeController_cmds_WithCommands(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)

	// Insert test data
	_, _ = comm.Db.Exec(`INSERT INTO t_cmd_line (id, step_id, num, code, content, status) 
		VALUES ('cmd1', 'step1', 1, 0, 'echo hello', 'pending')`)
	_, _ = comm.Db.Exec(`INSERT INTO t_cmd_line (id, step_id, num, code, content, status) 
		VALUES ('cmd2', 'step1', 2, 0, 'echo world', 'running')`)
	_, _ = comm.Db.Exec(`INSERT INTO t_cmd_line (id, step_id, num, code, content, status) 
		VALUES ('cmd3', 'step2', 1, 0, 'echo other', 'pending')`)

	usr := &model.TUser{Id: "testuser", Name: "test", Active: 1}
	m := &hbtp.Map{}
	m.Set("stepId", "step1")

	c, w := makeRuntimeGinContext(t, m, usr)
	ctrl := RuntimeController{}
	ctrl.cmds(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if cmds, ok := result["cmds"].([]interface{}); ok {
		if len(cmds) != 2 {
			t.Errorf("expected 2 cmds for step1, got %d", len(cmds))
		}
	} else {
		t.Error("cmds field missing or not an array")
	}
}

func TestRuntimeController_build_MissingBuildId(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)

	usr := &model.TUser{Id: "testuser", Name: "test", Active: 1}
	m := &hbtp.Map{}

	c, w := makeRuntimeGinContext(t, m, usr)
	ctrl := RuntimeController{}
	ctrl.build(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestRuntimeController_build_NotFound(t *testing.T) {
	// Skip this test because it requires engine.Mgr.BuildEgn() to be initialized
	// which needs full engine startup. The build handler directly calls engine.Mgr
	// without nil checks, causing panic in unit test environment.
	t.Skip("Requires engine.Mgr initialization - integration test only")
}

func TestRuntimeController_cancel_MissingBuildId(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)

	usr := &model.TUser{Id: "testuser", Name: "test", Active: 1}
	m := &hbtp.Map{}

	c, w := makeRuntimeGinContext(t, m, usr)
	ctrl := RuntimeController{}
	ctrl.cancel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestRuntimeController_cancel_NotFound(t *testing.T) {
	// Skip this test because it requires engine.Mgr.BuildEgn() to be initialized
	// which needs full engine startup. The cancel handler directly calls engine.Mgr
	// without nil checks, causing panic in unit test environment.
	t.Skip("Requires engine.Mgr initialization - integration test only")
}

func TestRuntimeController_logs_MissingStepId(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)

	usr := &model.TUser{Id: "testuser", Name: "test", Active: 1}
	m := &hbtp.Map{}

	c, w := makeRuntimeGinContext(t, m, usr)
	ctrl := RuntimeController{}
	ctrl.logs(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestRuntimeController_logs_FileNotFound(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)

	usr := &model.TUser{Id: "testuser", Name: "test", Active: 1}
	m := &hbtp.Map{}
	m.Set("stepId", "nonexistent-step-id")
	m.Set("buildId", "nonexistent-build-id")

	c, w := makeRuntimeGinContext(t, m, usr)
	ctrl := RuntimeController{}
	ctrl.logs(c, m)

	// Should return 404 because file doesn't exist
	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}
