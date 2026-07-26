package route

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/engine"
	"github.com/gokins/gokins/util"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// Ensure hbtp package is available for handler signatures
var _ = hbtp.Map{}

func createRuntimeTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create tables matching actual model schemas
	_, err = db.Exec(`CREATE TABLE t_pipeline_version (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT NOT NULL,
		pipeline_id VARCHAR(64),
		uid VARCHAR(64),
		number BIGINT DEFAULT 0,
		status VARCHAR(50),
		pipeline_name VARCHAR(100),
		pipeline_display_name VARCHAR(200),
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_version table: %v", err)
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
		t.Fatalf("create t_build table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_stage (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		pipeline_version_id VARCHAR(64),
		build_id VARCHAR(64),
		status VARCHAR(100),
		error VARCHAR(500),
		name VARCHAR(255),
		display_name VARCHAR(255),
		stage VARCHAR(255),
		sort BIGINT DEFAULT 0,
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_stage table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_step (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		build_id VARCHAR(64),
		stage_id VARCHAR(100),
		display_name VARCHAR(255),
		pipeline_version_id VARCHAR(64),
		step VARCHAR(255),
		status VARCHAR(100),
		exit_code BIGINT DEFAULT 0,
		error VARCHAR(500),
		name VARCHAR(100),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		errignore INT DEFAULT 0,
		waits TEXT,
		sort BIGINT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_step table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_cmd_line (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		group_id VARCHAR(64),
		build_id VARCHAR(64),
		step_id VARCHAR(64),
		status VARCHAR(50),
		num INT DEFAULT 0,
		code INT DEFAULT 0,
		content TEXT,
		created DATETIME,
		started DATETIME,
		finished DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_cmd_line table: %v", err)
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

	// Use test-safe BuildEngine without background goroutines
	origBuildEgn := engine.Mgr.BuildEgn()
	testBuildEgn := engine.NewBuildEngineForTest()
	engine.Mgr.SetBuildEngine(testBuildEgn)
	t.Cleanup(func() {
		engine.Mgr.SetBuildEngine(origBuildEgn)
	})

	rc := &RuntimeController{}
	routeGroup := r.Group("/api/runtime")
	// Skip auth middleware in tests, use util.GinReqParseJson like production
	routeGroup.POST("/stages", util.GinReqParseJson(rc.stages))
	routeGroup.POST("/cmds", util.GinReqParseJson(rc.cmds))
	routeGroup.POST("/build", util.GinReqParseJson(rc.build))
	routeGroup.POST("/logs", util.GinReqParseJson(rc.logs))
	return r
}

func TestRuntimeStages_EmptyPvId(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/stages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pvId, got %d", w.Code)
	}
}

func TestRuntimeStages_NoData(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)

	body, _ := json.Marshal(map[string]any{"pvId": "nonexistent-pv"})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/stages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 200 with empty results
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if ids, ok := resp["ids"].([]any); ok && len(ids) != 0 {
		t.Errorf("expected empty ids, got %v", ids)
	}
}

func TestRuntimeStages_WithData(t *testing.T) {
	db := createRuntimeTestDb(t)

	// Insert test stage data
	_, err := db.Exec(`INSERT INTO t_stage (id, pipeline_version_id, name, status)
		VALUES ('stage-1', 'pv-1', 'build', 'ok')`)
	if err != nil {
		t.Fatalf("insert stage: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_stage (id, pipeline_version_id, name, status)
		VALUES ('stage-2', 'pv-1', 'test', 'running')`)
	if err != nil {
		t.Fatalf("insert stage: %v", err)
	}
	// Insert step for stage-1
	_, err = db.Exec(`INSERT INTO t_step (id, stage_id, build_id, name, sort, status, waits)
		VALUES ('step-1', 'stage-1', 'build-1', 'compile', 1, 'ok', '[]')`)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}

	r := setupRuntimeTestRouter(t, db)

	body, _ := json.Marshal(map[string]any{"pvId": "pv-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/stages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	ids, ok := resp["ids"].([]any)
	if !ok {
		t.Fatal("expected ids array in response")
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 stage ids, got %d", len(ids))
	}
}

func TestRuntimeCmds_EmptyStepId(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/cmds", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty stepId, got %d", w.Code)
	}
}

func TestRuntimeCmds_NoData(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)

	body, _ := json.Marshal(map[string]any{"stepId": "nonexistent-step"})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/cmds", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	cmds, ok := resp["cmds"].([]any)
	if !ok {
		t.Fatal("expected cmds array in response")
	}
	if len(cmds) != 0 {
		t.Errorf("expected empty cmds, got %d", len(cmds))
	}
}

func TestRuntimeCmds_WithData(t *testing.T) {
	db := createRuntimeTestDb(t)

	// Insert cmd lines
	_, err := db.Exec(`INSERT INTO t_cmd_line (id, step_id, num, status, code)
		VALUES ('cmd-1', 'step-1', 1, 'ok', 0)`)
	if err != nil {
		t.Fatalf("insert cmd: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_cmd_line (id, step_id, num, status, code)
		VALUES ('cmd-2', 'step-1', 2, 'running', 0)`)
	if err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	r := setupRuntimeTestRouter(t, db)

	body, _ := json.Marshal(map[string]any{"stepId": "step-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/cmds", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	cmds, ok := resp["cmds"].([]any)
	if !ok {
		t.Fatal("expected cmds array in response")
	}
	if len(cmds) != 2 {
		t.Errorf("expected 2 cmds, got %d", len(cmds))
	}
	if resp["stepId"] != "step-1" {
		t.Errorf("expected stepId 'step-1', got %v", resp["stepId"])
	}
}

func TestRuntimeBuild_EmptyBuildId(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/build", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty buildId, got %d", w.Code)
	}
}

func TestRuntimeBuild_NotFound(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)

	body, _ := json.Marshal(map[string]any{"buildId": "nonexistent"})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/build", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 since the build engine doesn't have this build
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent build, got %d", w.Code)
	}
}

func TestRuntimeLogs_EmptyStepId(t *testing.T) {
	db := createRuntimeTestDb(t)
	r := setupRuntimeTestRouter(t, db)

	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/logs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty stepId, got %d", w.Code)
	}
}

func TestRuntimeLogs_FileNotFound(t *testing.T) {
	db := createRuntimeTestDb(t)

	origWorkPath := comm.WorkPath
	comm.WorkPath = "/tmp/gokins-test-nonexistent"
	t.Cleanup(func() { comm.WorkPath = origWorkPath })

	r := setupRuntimeTestRouter(t, db)

	body, _ := json.Marshal(map[string]any{"stepId": "step-1", "buildId": "build-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/logs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing log file, got %d", w.Code)
	}
}

func TestBatchRunPipelineVersions_Empty(t *testing.T) {
	result, err := batchRunPipelineVersions(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestBatchRunPipelineVersions_WithData(t *testing.T) {
	db := createRuntimeTestDb(t)

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	// Insert test data
	_, err := db.Exec(`INSERT INTO t_pipeline_version (id, aid, pipeline_id, uid, number, status, pipeline_name, pipeline_display_name)
		VALUES ('pv-1', 1, 'pipe-1', 'user-1', 1, 'ok', 'test-pipe', 'Test Pipeline')`)
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_pipeline_version (id, aid, pipeline_id, uid, number, status, pipeline_name, pipeline_display_name)
		VALUES ('pv-2', 2, 'pipe-1', 'user-1', 2, 'running', 'test-pipe', 'Test Pipeline')`)
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	result, err := batchRunPipelineVersions(context.Background(), []string{"pv-1", "pv-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
	if _, ok := result["pv-1"]; !ok {
		t.Error("expected pv-1 in results")
	}
	if _, ok := result["pv-2"]; !ok {
		t.Error("expected pv-2 in results")
	}
}

func TestBatchRunPipelineVersions_NonexistentIds(t *testing.T) {
	db := createRuntimeTestDb(t)

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	result, err := batchRunPipelineVersions(context.Background(), []string{"nonexistent-1", "nonexistent-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty results for nonexistent ids, got %d", len(result))
	}
}
