package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupPipelineTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	tables := []string{
		`CREATE TABLE t_user (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			name VARCHAR(100),
			pass VARCHAR(255),
			nick VARCHAR(100),
			avatar VARCHAR(500),
			created DATETIME,
			login_time DATETIME,
			active INT DEFAULT 0
		)`,
		`CREATE TABLE t_user_info (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			phone VARCHAR(100),
			email VARCHAR(200),
			birthday DATETIME,
			remark TEXT,
			perm_user INT,
			perm_org INT DEFAULT 0,
			perm_pipe INT DEFAULT 0
		)`,
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			name VARCHAR(200),
			"desc" VARCHAR(500),
			public INT DEFAULT 0,
			created DATETIME,
			updated DATETIME,
			deleted INT DEFAULT 0,
			deleted_time DATETIME
		)`,
		`CREATE TABLE t_user_org (
			aid BIGINT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)`,
		`CREATE TABLE t_pipeline (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			uid VARCHAR(64),
			name VARCHAR(255),
			display_name VARCHAR(255),
			pipeline_type VARCHAR(255),
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			created DATETIME,
			create_time DATETIME
		)`,
		`CREATE TABLE t_pipeline_conf (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			pipeline_id VARCHAR(64),
			url VARCHAR(255),
			access_token VARCHAR(255),
			yml_content TEXT,
			username VARCHAR(255)
		)`,
		`CREATE TABLE t_pipeline_version (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			uid VARCHAR(64),
			number BIGINT,
			events VARCHAR(100),
			sha VARCHAR(255),
			pipeline_name VARCHAR(255),
			pipeline_display_name VARCHAR(255),
			pipeline_id VARCHAR(64),
			version VARCHAR(255),
			content TEXT,
			created DATETIME,
			deleted INT DEFAULT 0,
			pr_number BIGINT,
			repo_clone_url VARCHAR(255)
		)`,
		`CREATE TABLE t_pipeline_var (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			pipeline_id VARCHAR(64),
			name VARCHAR(255),
			value TEXT,
			remarks VARCHAR(255),
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_pipe (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_run_build (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			uid VARCHAR(64),
			pipeline_id VARCHAR(64),
			pipeline_version_id VARCHAR(64),
			sha VARCHAR(255),
			events VARCHAR(100),
			status INT DEFAULT 0,
			errs TEXT,
			started DATETIME,
			stopped DATETIME,
			created DATETIME
		)`,
		`CREATE TABLE t_build (
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
		)`,
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("exec %q: %v", sql[:min(50, len(sql))], err)
		}
	}

	// Insert test data
	insertTestData(t, db)
	comm.Db = db
}

func insertTestData(t *testing.T, db *xorm.Engine) {
	t.Helper()

	// Test users
	_, err := db.Exec(`INSERT INTO t_user (id, name, nick, avatar, active, created)
		VALUES ('admin', 'admin', 'Admin User', '/avatar/admin.png', 1, datetime('now'))`)
	if err != nil {
		t.Fatalf("insert admin user: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_user (id, name, nick, avatar, active, created)
		VALUES ('user-1', 'user1', 'User One', '/avatar/user1.png', 1, datetime('now'))`)
	if err != nil {
		t.Fatalf("insert user1: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_user (id, name, nick, avatar, active, created)
		VALUES ('user-2', 'user2', 'User Two', '/avatar/user2.png', 1, datetime('now'))`)
	if err != nil {
		t.Fatalf("insert user2: %v", err)
	}

	// User info (admin has perm_user=2)
	_, err = db.Exec(`INSERT INTO t_user_info (id, perm_user, perm_pipe) VALUES ('admin', 2, 1)`)
	if err != nil {
		t.Fatalf("insert admin user_info: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_user_info (id, perm_user, perm_pipe) VALUES ('user-1', 0, 1)`)
	if err != nil {
		t.Fatalf("insert user1 user_info: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_user_info (id, perm_user, perm_pipe) VALUES ('user-2', 0, 0)`)
	if err != nil {
		t.Fatalf("insert user2 user_info: %v", err)
	}

	// Test org
	_, err = db.Exec(`INSERT INTO t_org (id, uid, name, public, deleted)
		VALUES ('org-1', 'admin', 'Test Org', 1, 0)`)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Org membership
	_, err = db.Exec(`INSERT INTO t_user_org (uid, org_id, perm_adm, perm_rw, perm_exec, perm_down)
		VALUES ('user-1', 'org-1', 0, 1, 1, 1)`)
	if err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	// Pipelines
	_, err = db.Exec(`INSERT INTO t_pipeline (id, uid, name, display_name, deleted, create_time)
		VALUES ('pipe-1', 'user-1', 'test-pipeline', 'Test Pipeline', 0, datetime('now'))`)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_pipeline (id, uid, name, display_name, deleted, create_time)
		VALUES ('pipe-deleted', 'user-1', 'deleted-pipe', 'Deleted', 1, datetime('now'))`)
	if err != nil {
		t.Fatalf("insert deleted pipeline: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_pipeline (id, uid, name, display_name, deleted, create_time)
		VALUES ('pipe-org-1', 'user-1', 'org-pipeline', 'Org Pipeline', 0, datetime('now'))`)
	if err != nil {
		t.Fatalf("insert org pipeline: %v", err)
	}

	// Pipeline config
	_, err = db.Exec(`INSERT INTO t_pipeline_conf (pipeline_id, url, username, access_token, yml_content)
		VALUES ('pipe-1', 'https://github.com/test/repo', 'bot', 'secret-token', 'stages:\n  - name: build\n    steps:\n      - sh: echo hello')`)
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	// Org-pipe association
	_, err = db.Exec(`INSERT INTO t_org_pipe (org_id, pipe_id, created, public)
		VALUES ('org-1', 'pipe-org-1', datetime('now'), 1)`)
	if err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	// Pipeline versions
	_, err = db.Exec(`INSERT INTO t_pipeline_version (id, uid, number, events, sha, pipeline_name, pipeline_id, created, deleted)
		VALUES ('pv-1', 'user-1', 1, 'push', 'abc123', 'test-pipeline', 'pipe-1', datetime('now'), 0)`)
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_pipeline_version (id, uid, number, events, sha, pipeline_name, pipeline_id, created, deleted)
		VALUES ('pv-2', 'user-1', 2, 'push', 'def456', 'test-pipeline', 'pipe-1', datetime('now'), 0)`)
	if err != nil {
		t.Fatalf("insert pipeline version 2: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_pipeline_version (id, uid, number, events, sha, pipeline_name, pipeline_id, created, deleted)
		VALUES ('pv-deleted', 'user-1', 3, 'push', 'ghi789', 'test-pipeline', 'pipe-1', datetime('now'), 1)`)
	if err != nil {
		t.Fatalf("insert deleted pipeline version: %v", err)
	}

	// Pipeline vars
	_, err = db.Exec(`INSERT INTO t_pipeline_var (uid, pipeline_id, name, value, remarks, public)
		VALUES ('user-1', 'pipe-1', 'PUBLIC_VAR', 'public-value', 'A public var', 1)`)
	if err != nil {
		t.Fatalf("insert pipeline var: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_pipeline_var (uid, pipeline_id, name, value, remarks, public)
		VALUES ('user-1', 'pipe-1', 'SECRET_VAR', 'secret-value', 'A secret var', 0)`)
	if err != nil {
		t.Fatalf("insert secret pipeline var: %v", err)
	}
}

func makePipelineTestContext(t *testing.T, user *model.TUser, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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
	c.Set(service.LgUserKey, user)
	return c, w
}

func adminUser() *model.TUser {
	return &model.TUser{Id: "admin", Name: "admin", Nick: "Admin User", Active: 1}
}

func regularUser() *model.TUser {
	return &model.TUser{Id: "user-1", Name: "user1", Nick: "User One", Active: 1}
}

func user2() *model.TUser {
	return &model.TUser{Id: "user-2", Name: "user2", Nick: "User Two", Active: 1}
}

func TestPipelineAdminCheck(t *testing.T) {
	// Verify IsAdmin checks for Id == "admin"
	if !service.IsAdmin(adminUser()) {
		t.Error("adminUser() should be admin")
	}
	if service.IsAdmin(regularUser()) {
		t.Error("regularUser() should not be admin")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- GetPath and Routes ---

func TestPipelineController_GetPath_Route(t *testing.T) {
	c := &PipelineController{}
	if got := c.GetPath(); got != "/api/pipeline" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/pipeline")
	}
}

func TestPipelineController_Routes(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	pc := &PipelineController{}
	pc.Routes(r.Group("/api/pipeline"))
	// Routes registered successfully
}

// --- info handler ---

func TestPipelineInfo_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineInfo_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineInfo_Deleted(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-deleted")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted pipeline, got %d", w.Code)
	}
}

func TestPipelineInfo_Success_Owner(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	pipe, ok := resp["pipe"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing 'pipe' field")
	}
	if pipe["name"] != "test-pipeline" {
		t.Errorf("pipe name = %v, want 'test-pipeline'", pipe["name"])
	}
	// Owner should see real credentials
	if pipe["username"] == comm.MaskedValue {
		t.Error("owner should see real username, got masked")
	}
	if pipe["accessToken"] == comm.MaskedValue {
		t.Error("owner should see real accessToken, got masked")
	}
	// Check permissions
	perm, ok := resp["perm"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing 'perm' field")
	}
	if perm["read"] != true {
		t.Error("owner should have read permission")
	}
	if perm["write"] != true {
		t.Error("owner should have write permission")
	}
}

func TestPipelineInfo_Success_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	c, w := makePipelineTestContext(t, adminUser(), m)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	perm := resp["perm"].(map[string]interface{})
	if perm["exec"] != true {
		t.Error("admin should have exec permission")
	}
}

// --- delete handler ---

func TestPipelineDelete_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineDelete_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineDelete_AlreadyDeleted(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-deleted")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for already-deleted pipeline, got %d", w.Code)
	}
}

func TestPipelineDelete_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	// user-2 doesn't own pipe-1 and isn't admin
	c, w := makePipelineTestContext(t, user2(), m)
	ctrl.delete(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestPipelineDelete_Success_Owner(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify pipeline is soft-deleted
	p := &model.TPipeline{}
	ok, _ := comm.Db.Where("id = ?", "pipe-1").Get(p)
	if !ok || p.Deleted != 1 {
		t.Error("pipeline should be soft-deleted")
	}
}

func TestPipelineDelete_Success_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-org-1")
	c, w := makePipelineTestContext(t, adminUser(), m)
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- searchSha handler ---

func TestPipelineSearchSha_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineSearchSha_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	c, w := makePipelineTestContext(t, user2(), m)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestPipelineSearchSha_Success(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	// Should return distinct SHA values (excluding empty ones)
	if len(resp) < 2 {
		t.Errorf("expected at least 2 sha results, got %d", len(resp))
	}
}

func TestPipelineSearchSha_WithQuery(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	m.Set("q", "abc")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 sha result with query 'abc', got %d", len(resp))
	}
}

func TestPipelineSearchSha_NoResults(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	m.Set("q", "zzzzz")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected 0 sha results, got %d", len(resp))
	}
}

// --- vars handler ---

func TestPipelineVars_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineVars_PipeNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineVars_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	c, w := makePipelineTestContext(t, user2(), m)
	ctrl.vars(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestPipelineVars_Success_Owner(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	// Owner should see real values (no masking)
	if resp.Data == nil {
		t.Fatal("expected non-nil data")
	}
	listBytes, _ := json.Marshal(resp.Data)
	var vars []model.TPipelineVar
	if err := json.Unmarshal(listBytes, &vars); err != nil {
		t.Fatalf("parse vars: %v", err)
	}
	if len(vars) != 2 {
		t.Errorf("expected 2 vars, got %d", len(vars))
	}
}

func TestPipelineVars_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("q", "PUBLIC")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- varSave handler ---

func TestPipelineVarSave_EmptyFields(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}

	// Empty name
	pv := &bean.PipelineVar{PipelineId: "pipe-1", Value: "val"}
	c, w := makePipelineTestContext(t, regularUser(), pv)
	ctrl.varSave(c, pv)
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty name: expected 400, got %d", w.Code)
	}

	// Empty value
	pv2 := &bean.PipelineVar{PipelineId: "pipe-1", Name: "VAR"}
	c2, w2 := makePipelineTestContext(t, regularUser(), pv2)
	ctrl.varSave(c2, pv2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("empty value: expected 400, got %d", w2.Code)
	}

	// Empty pipelineId
	pv3 := &bean.PipelineVar{Name: "VAR", Value: "val"}
	c3, w3 := makePipelineTestContext(t, regularUser(), pv3)
	ctrl.varSave(c3, pv3)
	if w3.Code != http.StatusBadRequest {
		t.Errorf("empty pipelineId: expected 400, got %d", w3.Code)
	}
}

func TestPipelineVarSave_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	pv := &bean.PipelineVar{PipelineId: "pipe-1", Name: "NEW_VAR", Value: "val"}
	c, w := makePipelineTestContext(t, user2(), pv)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestPipelineVarSave_CreateNew(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		PipelineId: "pipe-1",
		Name:       "NEW_VAR",
		Value:      "new-value",
		Remarks:    "new variable",
		Public:     true,
	}
	c, w := makePipelineTestContext(t, regularUser(), pv)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify variable was created
	v := &model.TPipelineVar{}
	ok, _ := comm.Db.Where("pipeline_id = ? AND name = ?", "pipe-1", "NEW_VAR").Get(v)
	if !ok {
		t.Error("new variable not found in DB")
	}
	if v.Value != "new-value" {
		t.Errorf("variable value = %q, want 'new-value'", v.Value)
	}
	if v.Public != 1 {
		t.Errorf("variable public = %d, want 1", v.Public)
	}
}

func TestPipelineVarSave_DuplicateName(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		PipelineId: "pipe-1",
		Name:       "PUBLIC_VAR", // Already exists
		Value:      "dup-value",
	}
	c, w := makePipelineTestContext(t, regularUser(), pv)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

// --- varDel handler ---

func TestPipelineVarDel_InvalidAid(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", "0")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineVarDel_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", "99999")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineVarDel_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	// Get the aid of an existing var
	v := &model.TPipelineVar{}
	ok, _ := comm.Db.Where("pipeline_id = ? AND name = ?", "pipe-1", "PUBLIC_VAR").Get(v)
	if !ok {
		t.Fatal("test var not found")
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", v.Aid)
	c, w := makePipelineTestContext(t, user2(), m)
	ctrl.varDel(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestPipelineVarDel_Success(t *testing.T) {
	setupPipelineTestDB(t)
	v := &model.TPipelineVar{}
	ok, _ := comm.Db.Where("pipeline_id = ? AND name = ?", "pipe-1", "SECRET_VAR").Get(v)
	if !ok {
		t.Fatal("test var not found")
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", v.Aid)
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	v2 := &model.TPipelineVar{}
	ok2, _ := comm.Db.Where("aid = ?", v.Aid).Get(v2)
	if ok2 {
		t.Error("variable should have been deleted")
	}
}

// --- pipelineVersion handler ---

func TestPipelineVersion_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.pipelineVersion(c, m)

	// Empty ID returns 400 (Bad Request) not 404
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineVersion_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- pipelineVersions handler ---

func TestPipelineVersions_EmptyPipelineId_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", "1")
	c, w := makePipelineTestContext(t, adminUser(), m)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVersions_WithPipelineId_Success(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("page", "1")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVersions_WithPipelineId_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	c, w := makePipelineTestContext(t, user2(), m)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestPipelineVersions_PipelineDeleted(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-deleted")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted pipeline, got %d", w.Code)
	}
}

func TestPipelineVersions_EmptyPipelineId_NonAdmin(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", "1")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVersions_EmptyPipelineId_UserWithNoPipelines(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", "1")
	// user-2 has no pipelines
	c, w := makePipelineTestContext(t, user2(), m)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- copy handler ---

func TestPipelineCopy_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.copy(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineCopy_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.copy(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineCopy_DeletedPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-deleted")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.copy(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted pipeline, got %d", w.Code)
	}
}

func TestPipelineCopy_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	// user-2 has no read access to pipe-1
	c, w := makePipelineTestContext(t, user2(), m)
	ctrl.copy(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestPipelineCopy_Success_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	c, w := makePipelineTestContext(t, adminUser(), m)
	ctrl.copy(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var pipe model.TPipeline
	if err := json.Unmarshal(w.Body.Bytes(), &pipe); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if pipe.Id == "" {
		t.Error("copied pipeline should have an id")
	}
	if pipe.Name != "test-pipeline_copy" {
		t.Errorf("copied pipeline name = %q, want 'test-pipeline_copy'", pipe.Name)
	}

	// Verify in DB
	p := &model.TPipeline{}
	ok, _ := comm.Db.Where("id = ?", pipe.Id).Get(p)
	if !ok {
		t.Error("copied pipeline not found in DB")
	}
	// Verify config was copied
	conf := &model.TPipelineConf{}
	ok2, _ := comm.Db.Where("pipeline_id = ?", pipe.Id).Get(conf)
	if !ok2 {
		t.Error("copied pipeline config not found in DB")
	}
}

// --- run handler ---

func TestPipelineRun_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.run(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineRun_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.run(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineRun_DeletedPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-deleted")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.run(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted pipeline, got %d", w.Code)
	}
}

// --- rebuild handler ---

func TestPipelineRebuild_EmptyVersionId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.rebuild(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineRebuild_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "nonexistent")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.rebuild(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- orgPipelines handler ---

func TestOrgPipelines_EmptyOrgId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgPipelines_OrgNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent-org")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgPipelines_Success(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("page", "1")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOrgPipelines_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("q", "org")
	m.Set("page", "1")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- getPipelines handler ---

func TestGetPipelines_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("page", "1")
	c, w := makePipelineTestContext(t, adminUser(), m)
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPipelines_NonAdmin(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("page", "1")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPipelines_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("q", "test")
	m.Set("page", "1")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- save handler ---

func TestPipelineSave_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestContext(t, regularUser(), m)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineSave_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("name", "updated")
	m.Set("content", "stages:\n  - name: build\n    steps:\n      - sh: echo hello")
	c, w := makePipelineTestContext(t, user2(), m)
	ctrl.save(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// Ensure test compiles
var _ = time.Now
var _ = utils.NewXid
