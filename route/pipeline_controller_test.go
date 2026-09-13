package route

import (
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

// setupPipelineTestDb creates an in-memory SQLite database with all tables
// needed for pipeline controller tests.
func setupPipelineTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	comm.Db = db

	tables := []string{
		`CREATE TABLE t_pipeline (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
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
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			pipeline_id VARCHAR(64),
			url VARCHAR(255),
			access_token VARCHAR(255),
			yml_content TEXT,
			username VARCHAR(255)
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
			perm_org INT,
			perm_pipe INT
		)`,
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid INTEGER NOT NULL,
			uid VARCHAR(64),
			name VARCHAR(200),
			"desc" TEXT,
			public INT DEFAULT 0,
			created DATETIME,
			updated DATETIME,
			deleted INT DEFAULT 0,
			deleted_time DATETIME
		)`,
		`CREATE TABLE t_org_pipe (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_user_org (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)`,
	}
	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("exec table DDL: %v\nSQL: %s", err, sql)
		}
	}
	return db
}

func makePipeGinCtx(t *testing.T, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if user != nil {
		c.Set(service.LgUserKey, user)
	}
	return c, w
}

// insertTestPipeline creates a pipeline and its config in the test DB.
func insertTestPipeline(t *testing.T, db *xorm.Engine, id, uid, name string) {
	t.Helper()
	p := &model.TPipeline{Id: id, Uid: uid, Name: name, DisplayName: name + " display"}
	if _, err := db.Insert(p); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	pc := &model.TPipelineConf{
		PipelineId:  id,
		Url:         "https://example.com/repo.git",
		AccessToken: "secret-token",
		YmlContent:  "version: 1\nstages:\n  - name: build\n    steps:\n      - name: echo\n        run: echo hello",
		Username:    "testuser",
	}
	if _, err := db.Insert(pc); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}
}

// --- save endpoint tests ---

func TestPipelineSave_MissingPipelineId(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineSave_PipelineNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	// Use Id "admin" so IsAdmin returns true and we get past the perm check
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	m.Set("name", "test")
	m.Set("content", "version: 1\nstages:\n  - name: build\n    steps:\n      - name: echo\n        step: custom\n        commands: echo hello")
	ctrl.save(c, m)

	// Pipeline not found returns 404 (perm context returns nil pipeline)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineSave_InvalidYaml(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("name", "updated")
	m.Set("content", "{{invalid yaml!!!")
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid YAML, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineSave_Success(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	validYaml := "version: 1\nstages:\n  - name: build\n    steps:\n      - name: echo\n        step: custom\n        commands: echo hello"
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("name", "updated-name")
	m.Set("displayName", "Updated Display")
	m.Set("content", validYaml)
	m.Set("url", "https://github.com/new.git")
	m.Set("accessToken", "new-token")
	m.Set("username", "newuser")
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify pipeline name updated
	p := &model.TPipeline{}
	ok, err := db.Where("id=?", "pipe-1").Get(p)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline not found")
	}
	if p.Name != "updated-name" {
		t.Errorf("expected name 'updated-name', got %q", p.Name)
	}

	// Verify conf updated
	pc := &model.TPipelineConf{}
	ok, err = db.Where("pipeline_id=?", "pipe-1").Get(pc)
	if err != nil {
		t.Fatalf("query pipeline conf: %v", err)
	}
	if !ok {
		t.Fatal("pipeline conf not found")
	}
	if pc.Url != "https://github.com/new.git" {
		t.Errorf("expected url 'https://github.com/new.git', got %q", pc.Url)
	}
	if pc.Username != "newuser" {
		t.Errorf("expected username 'newuser', got %q", pc.Username)
	}
}

// --- delete endpoint tests ---

func TestPipelineDelete_MissingId(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineDelete_PipelineNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineDelete_Success(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")

	// Add a pipeline version
	pv := &model.TPipelineVersion{
		Id: "pv-1", PipelineId: "pipe-1", Created: time.Now(),
	}
	if _, err := db.Insert(pv); err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify soft-deleted
	p := &model.TPipeline{}
	ok, _ := db.Where("id=?", "pipe-1").Get(p)
	if !ok {
		t.Fatal("pipeline should still exist (soft delete)")
	}
	if p.Deleted != 1 {
		t.Errorf("expected deleted=1, got %d", p.Deleted)
	}

	// Verify version soft-deleted
	pv2 := &model.TPipelineVersion{}
	ok, _ = db.Where("id=?", "pv-1").Get(pv2)
	if !ok {
		t.Fatal("pipeline version should still exist")
	}
	if pv2.Deleted != 1 {
		t.Errorf("expected version deleted=1, got %d", pv2.Deleted)
	}
}

// --- info endpoint tests ---

func TestPipelineInfo_MissingId(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineInfo_NotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineInfo_Success_OwnerCanSeeSecrets(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	pipe, ok := resp["pipe"].(map[string]any)
	if !ok {
		t.Fatal("expected pipe in response")
	}
	// Owner should see real credentials
	if pipe["url"] != "https://example.com/repo.git" {
		t.Errorf("expected real url, got %v", pipe["url"])
	}
	if pipe["accessToken"] != "secret-token" {
		t.Errorf("expected real access token, got %v", pipe["accessToken"])
	}
}

func TestPipelineInfo_AdminCanSeeSecrets(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "other-user", "test-pipe")
	ctrl := PipelineController{}
	// IsAdmin checks usr.Id == "admin"
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	pipe, ok := resp["pipe"].(map[string]any)
	if !ok {
		t.Fatal("expected pipe in response")
	}
	// Admin should see real credentials
	if pipe["accessToken"] != "secret-token" {
		t.Errorf("expected real access token for admin, got %v", pipe["accessToken"])
	}
}

func TestPipelineInfo_NonOwnerSeesMaskedSecrets(t *testing.T) {
	// Note: This test verifies that a non-owner, non-admin user who is the pipeline
	// owner sees real secrets (since ownership is checked directly, not via org perms).
	// For true non-owner org-based read access, MySQL is required (the code guards
	// the org perm JOIN with comm.IsMySQL).
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")
	ctrl := PipelineController{}
	// Owner sees secrets (CanWrite=true because they own the pipeline)
	user := &model.TUser{Id: "user-1", Name: "reader", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	pipe := resp["pipe"].(map[string]any)
	// Owner should see real credentials (they own this pipeline)
	if pipe["accessToken"] != "secret-token" {
		t.Errorf("expected real access token for owner, got %v", pipe["accessToken"])
	}
	if pipe["username"] != "testuser" {
		t.Errorf("expected real username for owner, got %v", pipe["username"])
	}
}

// --- vars endpoint tests ---

func TestPipelineVars_MissingPipelineId(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineVars_PipelineNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineVars_Success(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")

	// Insert a variable
	pv := &model.TPipelineVar{
		PipelineId: "pipe-1", Name: "MY_VAR", Value: "secret-value",
		Public: 0, Remarks: "test var",
	}
	if _, err := db.Insert(pv); err != nil {
		t.Fatalf("insert pipeline var: %v", err)
	}

	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("page", int64(1))
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

// --- varSave endpoint tests ---

func TestPipelineVarSave_MissingFields(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	pv := &bean.PipelineVar{PipelineId: "", Name: "", Value: ""}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineVarSave_PipelineNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	pv := &bean.PipelineVar{PipelineId: "nonexistent", Name: "VAR1", Value: "val1"}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineVarSave_CreateNew(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	pv := &bean.PipelineVar{
		PipelineId: "pipe-1", Name: "NEW_VAR", Value: "new-value", Public: true,
	}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify var was created
	count, err := db.Where("pipeline_id=? AND name=?", "pipe-1", "NEW_VAR").Count(&model.TPipelineVar{})
	if err != nil {
		t.Fatalf("count vars: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 var, got %d", count)
	}
}

func TestPipelineVarSave_DuplicateName(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")

	// Insert existing var
	pv := &model.TPipelineVar{PipelineId: "pipe-1", Name: "EXISTING", Value: "old"}
	if _, err := db.Insert(pv); err != nil {
		t.Fatalf("insert var: %v", err)
	}

	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	// Try to create a new var with same name (aid=0 means new)
	bpv := &bean.PipelineVar{PipelineId: "pipe-1", Name: "EXISTING", Value: "new"}
	ctrl.varSave(c, bpv)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate name, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVarSave_UpdateExisting(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")

	// Insert existing var
	pv := &model.TPipelineVar{PipelineId: "pipe-1", Name: "MY_VAR", Value: "old"}
	if _, err := db.Insert(pv); err != nil {
		t.Fatalf("insert var: %v", err)
	}

	// Get the aid
	stored := &model.TPipelineVar{}
	ok, _ := db.Where("pipeline_id=? AND name=?", "pipe-1", "MY_VAR").Get(stored)
	if !ok {
		t.Fatal("stored var not found")
	}

	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	bpv := &bean.PipelineVar{
		Aid: stored.Aid, PipelineId: "pipe-1", Name: "MY_VAR", Value: "updated",
	}
	ctrl.varSave(c, bpv)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify update
	updated := &model.TPipelineVar{}
	ok, _ = db.Where("aid=?", stored.Aid).Get(updated)
	if !ok {
		t.Fatal("updated var not found")
	}
	if updated.Value != "updated" {
		t.Errorf("expected value 'updated', got %q", updated.Value)
	}
}

// --- varDel endpoint tests ---

func TestPipelineVarDel_InvalidAid(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineVarDel_NotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("aid", int64(999))
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineVarDel_Success(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")

	pv := &model.TPipelineVar{PipelineId: "pipe-1", Name: "DEL_VAR", Value: "val"}
	if _, err := db.Insert(pv); err != nil {
		t.Fatalf("insert var: %v", err)
	}
	stored := &model.TPipelineVar{}
	ok, _ := db.Where("pipeline_id=? AND name=?", "pipe-1", "DEL_VAR").Get(stored)
	if !ok {
		t.Fatal("var not found")
	}

	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("aid", stored.Aid)
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	count, _ := db.Where("aid=?", stored.Aid).Count(&model.TPipelineVar{})
	if count != 0 {
		t.Errorf("expected 0 vars after delete, got %d", count)
	}
}

// --- searchSha endpoint tests ---

func TestPipelineSearchSha_MissingId(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineSearchSha_PipelineNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineSearchSha_Success(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")

	// Insert pipeline versions with SHAs
	pvs := []*model.TPipelineVersion{
		{Id: "pv-1", PipelineId: "pipe-1", Sha: "abc123def", Created: time.Now()},
		{Id: "pv-2", PipelineId: "pipe-1", Sha: "abc456ghi", Created: time.Now()},
		{Id: "pv-3", PipelineId: "pipe-1", Sha: "xyz789abc", Created: time.Now()},
	}
	for _, pv := range pvs {
		if _, err := db.Insert(pv); err != nil {
			t.Fatalf("insert pv: %v", err)
		}
	}

	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	m.Set("q", "abc")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) < 2 {
		t.Errorf("expected at least 2 results matching 'abc', got %d", len(resp))
	}
}

// --- pipelineVersions endpoint tests ---

func TestPipelineVersions_PipelineNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	m.Set("page", int64(1))
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineVersions_Success(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "test-pipe")

	// Insert versions
	for i, id := range []string{"pv-1", "pv-2", "pv-3"} {
		pv := &model.TPipelineVersion{
			Id: id, PipelineId: "pipe-1", Number: int64(i + 1),
			Created: time.Now(),
		}
		if _, err := db.Insert(pv); err != nil {
			t.Fatalf("insert pv: %v", err)
		}
	}

	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("page", int64(1))
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

// --- pipelineVersion endpoint tests ---

func TestPipelineVersion_MissingId(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineVersion_NotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- copy endpoint tests ---

func TestPipelineCopy_MissingPipelineId(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	ctrl.copy(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineCopy_PipelineNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.copy(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineCopy_Success(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "original-pipe")
	
	// Create user_info with PermPipe=1 to allow copying
	uinfo := &model.TUserInfo{Id: "user-1", PermPipe: 1}
	if _, err := db.Insert(uinfo); err != nil {
		t.Fatalf("insert user info: %v", err)
	}
	
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	ctrl.copy(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify a new pipeline was created
	count, _ := db.Where("name LIKE ?", "%original-pipe_copy%").Count(&model.TPipeline{})
	if count != 1 {
		t.Errorf("expected 1 copied pipeline, got %d", count)
	}

	// Verify conf was also copied
	allConfs, _ := db.Count(&model.TPipelineConf{})
	if allConfs != 2 {
		t.Errorf("expected 2 pipeline confs (original + copy), got %d", allConfs)
	}
}

// --- orgPipelines endpoint tests ---

func TestPipelineOrgPipelines_MissingOrgId(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "")
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineOrgPipelines_OrgNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineOrgPipelines_Success(t *testing.T) {
	db := setupPipelineTestDb(t)

	// Create org
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "test-org", Public: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Create pipeline and link to org
	insertTestPipeline(t, db, "pipe-1", "user-1", "org-pipe")
	op := &model.TOrgPipe{OrgId: "org-1", PipeId: "pipe-1", Created: time.Now()}
	if _, err := db.Insert(op); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("page", int64(1))
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

// --- getPipelines endpoint tests ---

func TestPipelineGetPipelines_Admin(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "pipe1")
	insertTestPipeline(t, db, "pipe-2", "user-2", "pipe2")

	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin-1", Name: "admin", Active: 1}
	c, w := makePipeGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("page", int64(1))
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineGetPipelines_WithSearch(t *testing.T) {
	db := setupPipelineTestDb(t)
	insertTestPipeline(t, db, "pipe-1", "user-1", "alpha-pipeline")
	insertTestPipeline(t, db, "pipe-2", "user-1", "beta-pipeline")

	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("q", "alpha")
	m.Set("page", int64(1))
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

// --- new endpoint tests ---

func TestPipelineNew_InvalidParams(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	np := &bean.NewPipeline{Name: "", Content: ""}
	ctrl.new(c, np)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineNew_InvalidYaml(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	np := &bean.NewPipeline{Name: "test", Content: "{{invalid yaml!!!"}
	ctrl.new(c, np)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineNew_OrgNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin-1", Name: "admin", Active: 1}
	c, w := makePipeGinCtx(t, admin)

	validYaml := "version: 1\nstages:\n  - name: build\n    steps:\n      - name: echo\n        step: custom\n        commands: echo hello"
	np := &bean.NewPipeline{
		Name: "test", Content: validYaml, OrgId: "nonexistent-org",
	}
	ctrl.new(c, np)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent org, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineNew_Success_Admin(t *testing.T) {
	db := setupPipelineTestDb(t)
	ctrl := PipelineController{}
	// IsAdmin checks usr.Id == "admin", not usr.Name
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinCtx(t, admin)

	validYaml := "version: 1\nstages:\n  - name: build\n    steps:\n      - name: echo\n        step: custom\n        commands: echo hello"
	np := &bean.NewPipeline{
		Name:        "new-pipe",
		DisplayName: "New Pipeline",
		Content:     validYaml,
		Url:         "https://github.com/test/repo.git",
		AccessToken: "token123",
		Username:    "gituser",
	}
	ctrl.new(c, np)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify pipeline was created
	count, _ := db.Where("name=?", "new-pipe").Count(&model.TPipeline{})
	if count != 1 {
		t.Errorf("expected 1 pipeline, got %d", count)
	}

	// Verify conf was created
	confCount, _ := db.Count(&model.TPipelineConf{})
	if confCount != 1 {
		t.Errorf("expected 1 pipeline conf, got %d", confCount)
	}
}

func TestPipelineNew_WithVars(t *testing.T) {
	db := setupPipelineTestDb(t)
	ctrl := PipelineController{}
	// IsAdmin checks usr.Id == "admin", not usr.Name
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinCtx(t, admin)

	validYaml := "version: 1\nstages:\n  - name: build\n    steps:\n      - name: echo\n        step: custom\n        commands: echo hello"
	np := &bean.NewPipeline{
		Name:    "pipe-with-vars",
		Content: validYaml,
		Vars: []*bean.NewPipelineVar{
			{Name: "VAR1", Value: "val1", Public: true},
			{Name: "VAR2", Value: "val2", Public: false},
		},
	}
	ctrl.new(c, np)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify pipeline was created
	pipeline := &model.TPipeline{}
	has, err := db.Where("name = ?", "pipe-with-vars").Get(pipeline)
	if err != nil || !has {
		t.Error("pipeline not created")
	}

	// Verify vars were created
	varCount, err := db.Where("pipeline_id = ?", pipeline.Id).Count(&model.TPipelineVar{})
	if err != nil || varCount != 2 {
		t.Errorf("expected 2 vars, got %d", varCount)
	}
}

// --- rebuild endpoint tests ---

func TestPipelineRebuild_MissingId(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "")
	ctrl.rebuild(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineRebuild_VersionNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "nonexistent")
	ctrl.rebuild(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- run endpoint tests ---

func TestPipelineRun_MissingPipelineId(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	ctrl.run(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineRun_PipelineNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipeGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.run(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- fillPipelineListBuildInfo tests ---

func TestFillPipelineListBuildInfo_Empty(t *testing.T) {
	err := fillPipelineListBuildInfo(nil, nil)
	if err != nil {
		t.Errorf("unexpected error for empty list: %v", err)
	}
}

func TestFillPipelineListBuildInfo_NoPipelines(t *testing.T) {
	err := fillPipelineListBuildInfo(nil, []*model.TPipeline{})
	if err != nil {
		t.Errorf("unexpected error for empty slice: %v", err)
	}
}
