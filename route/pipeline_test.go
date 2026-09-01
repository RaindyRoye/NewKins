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
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

// setupPipelineTestDB creates an in-memory SQLite database with all tables
// needed by the PipelineController handlers, and seeds test data.
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
			id VARCHAR(64) NOT NULL,
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(100),
			pass VARCHAR(255),
			nick VARCHAR(100),
			avatar VARCHAR(500),
			created DATETIME,
			login_time DATETIME,
			active INT DEFAULT 0
		)`,
		`CREATE TABLE t_user_info (
			id VARCHAR(64) NOT NULL,
			phone VARCHAR(100),
			email VARCHAR(200),
			birthday DATETIME,
			remark TEXT,
			perm_user INT DEFAULT 0,
			perm_org INT DEFAULT 0,
			perm_pipe INT DEFAULT 0
		)`,
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL,
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			name VARCHAR(200),
			"desc" TEXT,
			public INT DEFAULT 0,
			created DATETIME,
			updated DATETIME,
			deleted INT DEFAULT 0,
			deleted_time DATETIME
		)`,
		`CREATE TABLE t_user_org (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_pipe (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_pipeline (
			id VARCHAR(64) NOT NULL,
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			name VARCHAR(255),
			display_name VARCHAR(255),
			pipeline_type VARCHAR(255),
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			create_time DATETIME,
			created DATETIME
		)`,
		`CREATE TABLE t_pipeline_conf (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id VARCHAR(64),
			url VARCHAR(255),
			access_token VARCHAR(255),
			yml_content TEXT,
			username VARCHAR(255)
		)`,
		`CREATE TABLE t_pipeline_version (
			id VARCHAR(64) NOT NULL,
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
			pr_number BIGINT DEFAULT 0,
			repo_clone_url VARCHAR(255)
		)`,
		`CREATE TABLE t_pipeline_var (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			pipeline_id VARCHAR(64),
			name VARCHAR(255),
			value TEXT,
			remarks VARCHAR(255),
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_build (
			id VARCHAR(64) NOT NULL,
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
			t.Fatalf("exec table SQL: %v", err)
		}
	}

	// Seed admin user — IsAdmin() checks usr.Id == "admin"
	_, _ = db.Exec(`INSERT INTO t_user (id, aid, name, nick, active) VALUES ('admin', 1, 'admin', 'Admin', 1)`)
	// Seed regular user (owner of test pipelines)
	_, _ = db.Exec(`INSERT INTO t_user (id, aid, name, nick, active) VALUES ('user-uid', 2, 'testuser', 'Test User', 1)`)
	// Seed another user
	_, _ = db.Exec(`INSERT INTO t_user (id, aid, name, nick, active) VALUES ('other-uid', 3, 'other', 'Other User', 1)`)

	comm.Db = db
}

// pipeTestUser returns the "testuser" model (pipeline owner).
func pipeTestUser() *model.TUser {
	return &model.TUser{Id: "user-uid", Name: "testuser", Active: 1}
}

// pipeAdminUser returns the "admin" model (super-admin).
// Note: IsAdmin() checks usr.Id == "admin".
func pipeAdminUser() *model.TUser {
	return &model.TUser{Id: "admin", Name: "admin", Active: 1}
}

// pipeOtherUser returns the "other" model (no permissions).
func pipeOtherUser() *model.TUser {
	return &model.TUser{Id: "other-uid", Name: "other", Active: 1}
}

// makePipeGinContext creates a gin test context with the given body and logged-in user.
func makePipeGinContext(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

// seedPipeline inserts a pipeline and its conf row for testing.
func seedPipeline(t *testing.T, id, uid, name string) {
	t.Helper()
	p := &model.TPipeline{
		Id:   id,
		Uid:  uid,
		Name: name,
	}
	_, err := comm.Db.InsertOne(p)
	if err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	conf := &model.TPipelineConf{
		PipelineId:  id,
		YmlContent:  "stages:\n  - name: build\n    steps:\n      - step: shell\n        name: echo\n",
		Url:         "https://example.com/repo.git",
		Username:    "gituser",
		AccessToken: "secret-token",
	}
	_, err = comm.Db.InsertOne(conf)
	if err != nil {
		t.Fatalf("seed pipeline conf: %v", err)
	}
}

// seedPipelineVersion inserts a pipeline version row for testing.
func seedPipelineVersion(t *testing.T, id, pipeId, uid string) {
	t.Helper()
	pv := &model.TPipelineVersion{
		Id:         id,
		PipelineId: pipeId,
		Uid:        uid,
		Number:     1,
		Sha:        "abc123def",
		Created:    time.Now(),
	}
	_, err := comm.Db.InsertOne(pv)
	if err != nil {
		t.Fatalf("seed pipeline version: %v", err)
	}
}

// validYAML returns a minimal valid pipeline YAML string.
func validYAML() string {
	return `stages:
  - name: build
    steps:
      - step: shell
        name: echo
`
}

// ── GetPath ───────────────────────────────────────────────────────────────
// (TestPipelineController_GetPath is already declared in api_test.go)

// ── orgPipelines ──────────────────────────────────────────────────────────

func TestPipelineController_orgPipelines_EmptyOrgId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.orgPipelines(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_orgPipelines_OrgNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent-org")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.orgPipelines(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_orgPipelines_Success_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	// Create an org owned by the admin
	_, err := comm.Db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated) VALUES ('org-1', 1, 'admin-uid', 'TestOrg', 1, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeAdminUser())
	ctrl.orgPipelines(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_orgPipelines_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	_, _ = comm.Db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated) VALUES ('org-2', 2, 'admin-uid', 'SearchOrg', 1, datetime('now'), datetime('now'))`)
	seedPipeline(t, "pipe-search", "admin-uid", "search-pipe")
	_, _ = comm.Db.Exec(`INSERT INTO t_org_pipe (org_id, pipe_id, created) VALUES ('org-2', 'pipe-search', datetime('now'))`)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-2")
	m.Set("q", "search")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeAdminUser())
	ctrl.orgPipelines(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ── getPipelines ──────────────────────────────────────────────────────────

func TestPipelineController_getPipelines_Admin_EmptyDB(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeAdminUser())
	ctrl.getPipelines(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_getPipelines_Admin_WithPipelines(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-admin-1", "admin-uid", "admin-pipe-1")
	seedPipeline(t, "pipe-admin-2", "user-uid", "user-pipe-1")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeAdminUser())
	ctrl.getPipelines(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_getPipelines_NonAdmin(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-user-1", "user-uid", "user-pipe-1")
	seedPipeline(t, "pipe-admin-only", "admin-uid", "admin-only-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.getPipelines(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_getPipelines_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-findme", "admin-uid", "findme-pipe")
	seedPipeline(t, "pipe-hide", "admin-uid", "hidden-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("q", "findme")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeAdminUser())
	ctrl.getPipelines(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ── save ──────────────────────────────────────────────────────────────────

func TestPipelineController_save_MissingPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.save(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_save_InvalidYAML(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-save-1", "user-uid", "save-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-save-1")
	m.Set("name", "updated-name")
	m.Set("content", "{{invalid yaml")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.save(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestPipelineController_save_EmptyStagesYAML(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-save-2", "user-uid", "save-pipe-2")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-save-2")
	m.Set("name", "updated-name")
	m.Set("content", "version: '1'")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.save(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (empty stages should fail validation), body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestPipelineController_save_Success(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-save-3", "user-uid", "save-pipe-3")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-save-3")
	m.Set("name", "updated-name")
	m.Set("displayName", "Updated Display")
	m.Set("content", validYAML())
	m.Set("url", "https://example.com/new-repo.git")
	m.Set("username", "newuser")
	m.Set("accessToken", "newtoken")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.save(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_save_NoAuth(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-save-other", "admin-uid", "admin-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-save-other")
	m.Set("name", "hacked")
	m.Set("content", validYAML())
	c, w := makePipeGinContext(t, m, pipeOtherUser())
	ctrl.save(c, m)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d (non-owner, non-admin should be denied)", w.Code, http.StatusMethodNotAllowed)
	}
}

// ── delete ────────────────────────────────────────────────────────────────

func TestPipelineController_delete_MissingId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.delete(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_delete_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-pipe")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.delete(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_delete_Success(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-del-1", "user-uid", "del-pipe")
	seedPipelineVersion(t, "pv-del-1", "pipe-del-1", "user-uid")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-del-1")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.delete(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	// Verify soft-deleted
	p := &model.TPipeline{}
	ok, _ := comm.Db.Where("id=?", "pipe-del-1").Get(p)
	if !ok {
		t.Fatal("pipeline should still exist (soft-deleted)")
	}
	if p.Deleted != 1 {
		t.Errorf("pipeline.Deleted = %d, want 1", p.Deleted)
	}
}

func TestPipelineController_delete_NoAuth(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-del-noauth", "admin-uid", "admin-only-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-del-noauth")
	c, w := makePipeGinContext(t, m, pipeOtherUser())
	ctrl.delete(c, m)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ── info ──────────────────────────────────────────────────────────────────

func TestPipelineController_info_MissingId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.info(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_info_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-pipe")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.info(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_info_Success_Owner(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-info-1", "user-uid", "info-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-info-1")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.info(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	// Owner should see unmasked credentials
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	pipe, ok := resp["pipe"].(map[string]interface{})
	if ok {
		if un, _ := pipe["username"].(string); un == comm.MaskedValue {
			t.Error("owner should see unmasked username")
		}
	}
}

func TestPipelineController_info_Success_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-info-adm", "user-uid", "user-info-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-info-adm")
	c, w := makePipeGinContext(t, m, pipeAdminUser())
	ctrl.info(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ── new ───────────────────────────────────────────────────────────────────

func TestPipelineController_new_MissingName(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:    "",
		Content: validYAML(),
	}
	c, w := makePipeGinContext(t, npipe, pipeAdminUser())
	ctrl.new(c, npipe)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_new_MissingContent(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:    "my-pipe",
		Content: "",
	}
	c, w := makePipeGinContext(t, npipe, pipeAdminUser())
	ctrl.new(c, npipe)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_new_InvalidYAML(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:    "my-pipe",
		Content: "{{bad yaml",
	}
	c, w := makePipeGinContext(t, npipe, pipeAdminUser())
	ctrl.new(c, npipe)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_new_InvalidStages(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:    "my-pipe",
		Content: "version: '1'",
	}
	c, w := makePipeGinContext(t, npipe, pipeAdminUser())
	ctrl.new(c, npipe)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (no stages = invalid), body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestPipelineController_new_Success_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:        "admin-new-pipe",
		DisplayName: "Admin New Pipe",
		Content:     validYAML(),
		Url:         "https://example.com/repo.git",
		Username:    "gituser",
		AccessToken: "token123",
	}
	c, w := makePipeGinContext(t, npipe, pipeAdminUser())
	ctrl.new(c, npipe)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_new_WithOrg(t *testing.T) {
	setupPipelineTestDB(t)
	_, _ = comm.Db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated) VALUES ('org-new', 10, 'admin-uid', 'NewOrg', 1, datetime('now'), datetime('now'))`)

	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:    "org-pipe",
		Content: validYAML(),
		OrgId:   "org-new",
	}
	c, w := makePipeGinContext(t, npipe, pipeAdminUser())
	ctrl.new(c, npipe)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_new_WithVars(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:    "var-pipe",
		Content: validYAML(),
		Vars: []*bean.NewPipelineVar{
			{Name: "KEY1", Value: "val1", Remarks: "first var", Public: true},
			{Name: "KEY2", Value: "val2", Remarks: "second var", Public: false},
		},
	}
	c, w := makePipeGinContext(t, npipe, pipeAdminUser())
	ctrl.new(c, npipe)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_new_NonAdmin_NoPerm(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:    "no-perm-pipe",
		Content: validYAML(),
	}
	c, w := makePipeGinContext(t, npipe, pipeOtherUser())
	ctrl.new(c, npipe)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d (non-admin without perm_pipe)", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPipelineController_new_OrgNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:    "org-missing-pipe",
		Content: validYAML(),
		OrgId:   "nonexistent-org-999",
	}
	c, w := makePipeGinContext(t, npipe, pipeAdminUser())
	ctrl.new(c, npipe)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (org not found)", w.Code, http.StatusNotFound)
	}
}

// ── copy ──────────────────────────────────────────────────────────────────

func TestPipelineController_copy_MissingPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.copy(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_copy_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.copy(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_copy_Success_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-copy-1", "user-uid", "copy-src")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-copy-1")
	c, w := makePipeGinContext(t, m, pipeAdminUser())
	ctrl.copy(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_copy_Success_Owner(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-copy-own", "admin-uid", "admin-copy-src")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-copy-own")
	c, w := makePipeGinContext(t, m, pipeAdminUser())
	ctrl.copy(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ── pipelineVersions ──────────────────────────────────────────────────────

func TestPipelineController_pipelineVersions_WithPipelineId_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent-pipe")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.pipelineVersions(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_pipelineVersions_Success_Owner(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-pv-1", "user-uid", "pv-pipe")
	seedPipelineVersion(t, "pv-1", "pipe-pv-1", "user-uid")
	seedPipelineVersion(t, "pv-2", "pipe-pv-1", "user-uid")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-pv-1")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.pipelineVersions(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_pipelineVersions_EmptyPipeId_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-pv-adm", "admin-uid", "adm-pv-pipe")
	seedPipelineVersion(t, "pv-adm-1", "pipe-pv-adm", "admin-uid")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeAdminUser())
	ctrl.pipelineVersions(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_pipelineVersions_EmptyPipeId_NonAdmin(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-pv-user", "user-uid", "user-pv-pipe")
	seedPipelineVersion(t, "pv-user-1", "pipe-pv-user", "user-uid")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.pipelineVersions(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_pipelineVersions_EmptyPipeId_NonAdmin_NoPipelines(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeOtherUser())
	ctrl.pipelineVersions(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ── pipelineVersion (single) ──────────────────────────────────────────────

func TestPipelineController_pipelineVersion_MissingId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.pipelineVersion(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_pipelineVersion_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-pv")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.pipelineVersion(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_pipelineVersion_NoBuild(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-pvnb", "user-uid", "pvnb-pipe")
	seedPipelineVersion(t, "pv-nobuild", "pipe-pvnb", "user-uid")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pv-nobuild")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.pipelineVersion(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (no build found), body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

// ── searchSha ─────────────────────────────────────────────────────────────

func TestPipelineController_searchSha_MissingId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.searchSha(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_searchSha_PipeNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-pipe")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.searchSha(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_searchSha_Success(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-sha-1", "user-uid", "sha-pipe")
	seedPipelineVersion(t, "pv-sha-1", "pipe-sha-1", "user-uid")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-sha-1")
	m.Set("q", "")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.searchSha(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_searchSha_WithFilter(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-sha-2", "user-uid", "sha-pipe-2")
	seedPipelineVersion(t, "pv-sha-2", "pipe-sha-2", "user-uid")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-sha-2")
	m.Set("q", "abc")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.searchSha(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ── vars ──────────────────────────────────────────────────────────────────

func TestPipelineController_vars_MissingPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.vars(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_vars_PipeNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent-pipe")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.vars(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_vars_Success_Owner(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-vars-1", "user-uid", "vars-pipe")
	// Seed a pipeline variable
	_, _ = comm.Db.Exec(`INSERT INTO t_pipeline_var (uid, pipeline_id, name, value, public) VALUES ('user-uid', 'pipe-vars-1', 'MY_VAR', 'my_value', 1)`)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-vars-1")
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.vars(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_vars_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-vars-2", "user-uid", "vars-pipe-2")
	_, _ = comm.Db.Exec(`INSERT INTO t_pipeline_var (uid, pipeline_id, name, value, public) VALUES ('user-uid', 'pipe-vars-2', 'SEARCH_VAR', 'search_value', 0)`)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-vars-2")
	m.Set("q", "SEARCH")
	m.Set("page", int64(1))
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.vars(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ── varSave ───────────────────────────────────────────────────────────────

func TestPipelineController_varSave_MissingParams(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	c, w := makePipeGinContext(t, nil, pipeTestUser())
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "",
		Name:       "KEY",
		Value:      "val",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varSave_PipeNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	c, w := makePipeGinContext(t, nil, pipeTestUser())
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "nonexistent-pipe",
		Name:       "KEY",
		Value:      "val",
	})
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_varSave_NewVar(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-vs-1", "user-uid", "vs-pipe")

	ctrl := PipelineController{}
	c, w := makePipeGinContext(t, nil, pipeTestUser())
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "pipe-vs-1",
		Name:       "NEW_VAR",
		Value:      "new_value",
		Public:     true,
	})
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_varSave_UpdateExisting(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-vs-2", "user-uid", "vs-pipe-2")
	_, _ = comm.Db.Exec(`INSERT INTO t_pipeline_var (uid, pipeline_id, name, value, public) VALUES ('user-uid', 'pipe-vs-2', 'EXIST_VAR', 'old_val', 0)`)

	// Get the aid of the inserted variable
	pv := &model.TPipelineVar{}
	ok, _ := comm.Db.Where("pipeline_id=? and name=?", "pipe-vs-2", "EXIST_VAR").Get(pv)
	if !ok {
		t.Fatal("seeded var not found")
	}

	ctrl := PipelineController{}
	c, w := makePipeGinContext(t, nil, pipeTestUser())
	ctrl.varSave(c, &bean.PipelineVar{
		Aid:        pv.Aid,
		PipelineId: "pipe-vs-2",
		Name:       "EXIST_VAR",
		Value:      "updated_value",
	})
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_varSave_DuplicateName(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-vs-3", "user-uid", "vs-pipe-3")
	_, _ = comm.Db.Exec(`INSERT INTO t_pipeline_var (uid, pipeline_id, name, value, public) VALUES ('user-uid', 'pipe-vs-3', 'DUP_VAR', 'val1', 0)`)

	ctrl := PipelineController{}
	c, w := makePipeGinContext(t, nil, pipeTestUser())
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "pipe-vs-3",
		Name:       "DUP_VAR",
		Value:      "val2",
	})
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (duplicate name)", w.Code, http.StatusConflict)
	}
}

// ── varDel ────────────────────────────────────────────────────────────────

func TestPipelineController_varDel_InvalidAid(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.varDel(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varDel_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(99999))
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.varDel(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_varDel_Success(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-vd-1", "user-uid", "vd-pipe")
	_, _ = comm.Db.Exec(`INSERT INTO t_pipeline_var (uid, pipeline_id, name, value, public) VALUES ('user-uid', 'pipe-vd-1', 'DEL_VAR', 'del_val', 0)`)

	pv := &model.TPipelineVar{}
	ok, _ := comm.Db.Where("pipeline_id=? and name=?", "pipe-vd-1", "DEL_VAR").Get(pv)
	if !ok {
		t.Fatal("seeded var not found")
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", pv.Aid)
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.varDel(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ── rebuild ───────────────────────────────────────────────────────────────

func TestPipelineController_rebuild_MissingPipelineVersionId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.rebuild(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_rebuild_VersionNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "nonexistent-pv")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.rebuild(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_rebuild_PipelineDeleted(t *testing.T) {
	setupPipelineTestDB(t)
	// Create a pipeline, mark it deleted, and create a version for it
	seedPipeline(t, "pipe-rb-del", "user-uid", "rb-deleted-pipe")
	seedPipelineVersion(t, "pv-rb-del", "pipe-rb-del", "user-uid")
	// Soft-delete the pipeline
	_, _ = comm.Db.Exec(`UPDATE t_pipeline SET deleted = 1 WHERE id = 'pipe-rb-del'`)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "pv-rb-del")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.rebuild(c, m)
	// Pipeline is deleted, so perm check should fail with 404
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (deleted pipeline), body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

// ── run ───────────────────────────────────────────────────────────────────

func TestPipelineController_run_MissingPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.run(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_run_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent-pipe")
	c, w := makePipeGinContext(t, m, pipeTestUser())
	ctrl.run(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_run_NoExecPermission(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-run-noauth", "admin-uid", "admin-run-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-run-noauth")
	c, w := makePipeGinContext(t, m, pipeOtherUser())
	ctrl.run(c, m)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d (no exec permission)", w.Code, http.StatusMethodNotAllowed)
	}
}

// ── fillPipelineListBuildInfo ─────────────────────────────────────────────

func TestFillPipelineListBuildInfo_Empty(t *testing.T) {
	err := fillPipelineListBuildInfo(context.Background(), nil)
	if err != nil {
		t.Errorf("fillPipelineListBuildInfo(nil) error = %v", err)
	}

	err = fillPipelineListBuildInfo(context.Background(), []*model.TPipeline{})
	if err != nil {
		t.Errorf("fillPipelineListBuildInfo(empty) error = %v", err)
	}
}

func TestFillPipelineListBuildInfo_WithData(t *testing.T) {
	setupPipelineTestDB(t)
	seedPipeline(t, "pipe-fill-1", "user-uid", "fill-pipe-1")

	pipes := []*model.TPipeline{
		{Id: "pipe-fill-1", Uid: "user-uid", Name: "fill-pipe-1"},
	}
	err := fillPipelineListBuildInfo(context.Background(), pipes)
	if err != nil {
		t.Errorf("fillPipelineListBuildInfo error = %v", err)
	}
}
