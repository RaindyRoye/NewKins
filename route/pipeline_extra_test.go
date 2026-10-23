package route

import (
	"bytes"
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

func setupPipelineExtraTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	tables := []string{
		`CREATE TABLE t_pipeline (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			uid VARCHAR(64),
			name VARCHAR(255),
			display_name VARCHAR(255),
			pipeline_type VARCHAR(255),
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			create_time DATETIME
		)`,
		`CREATE TABLE t_pipeline_conf (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id VARCHAR(64) NOT NULL,
			url VARCHAR(255),
			access_token VARCHAR(255),
			yml_content TEXT,
			username VARCHAR(255)
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
			perm_pipe INT
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
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
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
			aid BIGINT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_pipe (
			aid BIGINT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INT DEFAULT 0
		)`,
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("exec %q: %v", sql[:40], err)
		}
	}

	comm.Db = db
	return db
}

func makePipelineExtraGinCtx(t *testing.T, user *model.TUser, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bts, _ := json.Marshal(body)
		req = httptest.NewRequest("POST", "/test", bytes.NewReader(bts))
	} else {
		req = httptest.NewRequest("POST", "/test", nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if user != nil {
		c.Set(service.LgUserKey, user)
	}
	return c, w
}

// ===================== save tests =====================

func TestPipelineSave_EmptyPipelineId(t *testing.T) {
	setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineExtraGinCtx(t, user, m)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pipelineId, got %d", w.Code)
	}
}

func TestPipelineSave_NoAuth(t *testing.T) {
	db := setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipe", Uid: "other-user"}
	if _, err := db.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("name", "new-name")
	m.Set("content", "pipeline:\n  name: test\n  stages: []\n")
	c, w := makePipelineExtraGinCtx(t, user, m)
	ctrl.save(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 no auth, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineSave_InvalidYaml(t *testing.T) {
	db := setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	// Use admin user so IsAdmin returns true and we bypass permission check
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipe", Uid: "admin"}
	if _, err := db.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("name", "new-name")
	m.Set("content", "{{not valid yaml}}")
	c, w := makePipelineExtraGinCtx(t, admin, m)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid yaml, got %d, body: %s", w.Code, w.Body.String())
	}
}

// ===================== new tests =====================

func TestPipelineNew_InvalidParams(t *testing.T) {
	setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	// Name is empty => Check() returns false
	np := &bean.NewPipeline{
		Name:    "",
		Content: "pipeline:\n  name: test\n  stages: []\n",
	}
	c, w := makePipelineExtraGinCtx(t, user, np)
	ctrl.new(c, np)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineNew_InvalidYaml(t *testing.T) {
	setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	np := &bean.NewPipeline{
		Name:    "test-pipe",
		Content: "{{bad yaml}}",
	}
	c, w := makePipelineExtraGinCtx(t, user, np)
	ctrl.new(c, np)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid yaml, got %d, body: %s", w.Code, w.Body.String())
	}
}

func validPipelineYAML() string {
	return `pipeline:
  name: test
stages:
  - name: build
    steps:
      - step: shell
        name: echo
        commands:
          - echo hello
`
}

func TestPipelineNew_OrgNotFound(t *testing.T) {
	db := setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	// admin bypasses IsAdmin
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	np := &bean.NewPipeline{
		Name:    "test-pipe",
		Content: validPipelineYAML(),
		OrgId:   "nonexistent-org",
	}
	c, w := makePipelineExtraGinCtx(t, admin, np)
	ctrl.new(c, np)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent org, got %d, body: %s", w.Code, w.Body.String())
	}
}

// ===================== pipelineVersions tests =====================

func TestPipelineVersions_NonexistentPipeline(t *testing.T) {
	setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	m.Set("page", int64(1))
	c, w := makePipelineExtraGinCtx(t, user, m)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineVersions_NoAuth(t *testing.T) {
	db := setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipe", Uid: "other-user"}
	if _, err := db.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("page", int64(1))
	c, w := makePipelineExtraGinCtx(t, user, m)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 no auth, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVersions_Success_AdminEmptyPipelineId(t *testing.T) {
	db := setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	pv := &model.TPipelineVersion{
		Id: "pv-1", PipelineId: "pipe-1", Number: 1,
		PipelineName: "pipe", Created: time.Now(),
	}
	if _, err := db.Insert(pv); err != nil {
		t.Fatalf("insert pv: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", int64(1))
	c, w := makePipelineExtraGinCtx(t, admin, m)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for admin empty pipelineId, got %d, body: %s", w.Code, w.Body.String())
	}
}

// ===================== varSave tests =====================

func TestPipelineVarSave_MissingFields(t *testing.T) {
	setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	c, w := makePipelineExtraGinCtx(t, user, nil)
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "",
		Name:       "v",
		Value:      "x",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pipelineId, got %d", w.Code)
	}

	c, w = makePipelineExtraGinCtx(t, user, nil)
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "p",
		Name:       "",
		Value:      "x",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", w.Code)
	}

	c, w = makePipelineExtraGinCtx(t, user, nil)
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "p",
		Name:       "v",
		Value:      "",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty value, got %d", w.Code)
	}
}

func TestPipelineVarSave_PipelineNotFound(t *testing.T) {
	setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	c, w := makePipelineExtraGinCtx(t, user, nil)
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "nonexistent",
		Name:       "v",
		Value:      "x",
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineVarSave_NoAuth(t *testing.T) {
	db := setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipe := &model.TPipeline{Id: "pipe-1", Name: "p", Uid: "other-user"}
	if _, err := db.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	c, w := makePipelineExtraGinCtx(t, user, nil)
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "pipe-1",
		Name:       "v",
		Value:      "x",
	})

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVarSave_CreateNew(t *testing.T) {
	db := setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	pipe := &model.TPipeline{Id: "pipe-1", Name: "p", Uid: "admin"}
	if _, err := db.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	c, w := makePipelineExtraGinCtx(t, admin, nil)
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "pipe-1",
		Name:       "MY_VAR",
		Value:      "myval",
		Public:     true,
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	count, err := db.Where("pipeline_id = ? AND name = ?", "pipe-1", "MY_VAR").Count(&model.TPipelineVar{})
	if err != nil {
		t.Fatalf("count vars: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 var, got %d", count)
	}
}

func TestPipelineVarSave_DuplicateName(t *testing.T) {
	db := setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	pipe := &model.TPipeline{Id: "pipe-1", Name: "p", Uid: "admin"}
	if _, err := db.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	existing := &model.TPipelineVar{
		PipelineId: "pipe-1",
		Name:       "MY_VAR",
		Value:      "old",
	}
	if _, err := db.Insert(existing); err != nil {
		t.Fatalf("insert existing var: %v", err)
	}

	c, w := makePipelineExtraGinCtx(t, admin, nil)
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "pipe-1",
		Name:       "MY_VAR",
		Value:      "newval",
	})

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 duplicate name, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVarSave_UpdateExisting(t *testing.T) {
	db := setupPipelineExtraTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	pipe := &model.TPipeline{Id: "pipe-1", Name: "p", Uid: "admin"}
	if _, err := db.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	existing := &model.TPipelineVar{
		PipelineId: "pipe-1",
		Name:       "MY_VAR",
		Value:      "old",
	}
	if _, err := db.Insert(existing); err != nil {
		t.Fatalf("insert existing var: %v", err)
	}

	// Look up to get aid
	var pv model.TPipelineVar
	if _, err := db.Where("pipeline_id = ? AND name = ?", "pipe-1", "MY_VAR").Get(&pv); err != nil {
		t.Fatalf("get var: %v", err)
	}

	c, w := makePipelineExtraGinCtx(t, admin, nil)
	ctrl.varSave(c, &bean.PipelineVar{
		Aid:        pv.Aid,
		PipelineId: "pipe-1",
		Name:       "MY_VAR",
		Value:      "updated",
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}
