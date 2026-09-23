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
		t.Fatalf("open sqlite: %v", err)
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
			perm_pipe INT
		)`,
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
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			name VARCHAR(200),
			desc TEXT,
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
			aid BIGINT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
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
			t.Fatalf("exec %q: %v", sql[:40], err)
		}
	}

	comm.Db = db
}

func makePipelineGinCtx(t *testing.T, body interface{}, lgUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

	if lgUser != nil {
		c.Set(service.LgUserKey, lgUser)
	}
	return c, w
}

func createPipelineTestUser(t *testing.T, name, nick string) *model.TUser {
	t.Helper()
	usr := &model.TUser{
		Id:        utils.NewXid(),
		Name:      name,
		Nick:      nick,
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	if _, err := comm.Db.InsertOne(usr); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return usr
}

func createTestPipeline(t *testing.T, owner *model.TUser, name, displayName string) (*model.TPipeline, *model.TPipelineConf) {
	t.Helper()
	pipe := &model.TPipeline{
		Id:          utils.NewXid(),
		Uid:         owner.Id,
		Name:        name,
		DisplayName: displayName,
	}
	if _, err := comm.Db.InsertOne(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	conf := &model.TPipelineConf{
		PipelineId:  pipe.Id,
		YmlContent:  "stages:\n  - name: build\n    steps:\n      - name: test\n        step: shell\n        commands: echo ok",
		Url:         "https://github.com/example/repo",
		Username:    owner.Name,
		AccessToken: "test-token",
	}
	if _, err := comm.Db.InsertOne(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}
	return pipe, conf
}

// === info ===

func TestPipelineController_info_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "admin", "Admin")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_info_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "admin", "Admin")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-id")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_info_Success(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, conf := createTestPipeline(t, usr, "test-pipe", "Test Pipeline")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", pipe.Id)
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, ok := resp["pipe"].(map[string]interface{})
	if !ok {
		t.Fatal("resp missing pipe field")
	}
	if p["name"] != "test-pipe" {
		t.Errorf("pipe.name = %v, want %q", p["name"], "test-pipe")
	}
	// Owner should see username and access token (CanWrite)
	if p["username"] != conf.Username {
		t.Errorf("pipe.username = %v, want %q", p["username"], conf.Username)
	}
	perm, ok := resp["perm"].(map[string]interface{})
	if !ok {
		t.Fatal("resp missing perm field")
	}
	if perm["read"] != true {
		t.Errorf("perm.read = %v, want true", perm["read"])
	}
	if perm["write"] != true {
		t.Errorf("perm.write = %v, want true", perm["write"])
	}
}

// === delete ===

func TestPipelineController_delete_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "admin", "Admin")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_delete_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "admin", "Admin")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-id")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_delete_Success(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, _ := createTestPipeline(t, usr, "del-pipe", "Delete Pipeline")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", pipe.Id)
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify soft-deleted
	deleted := &model.TPipeline{}
	ok, _ := comm.Db.Where("id = ?", pipe.Id).Get(deleted)
	if !ok || deleted.Deleted != 1 {
		t.Errorf("pipeline not soft-deleted after delete")
	}
}

// === vars ===

func TestPipelineController_vars_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "admin", "Admin")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_vars_NotFoundPipe(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "admin", "Admin")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_vars_Success(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, _ := createTestPipeline(t, usr, "var-pipe", "Var Pipeline")

	// Insert test vars
	v1 := &model.TPipelineVar{Uid: usr.Id, PipelineId: pipe.Id, Name: "DB_HOST", Value: "localhost", Public: 0}
	v2 := &model.TPipelineVar{Uid: usr.Id, PipelineId: pipe.Id, Name: "API_KEY", Value: "secret", Public: 1}
	if _, err := comm.Db.InsertOne(v1); err != nil {
		t.Fatalf("insert v1: %v", err)
	}
	if _, err := comm.Db.InsertOne(v2); err != nil {
		t.Fatalf("insert v2: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", pipe.Id)
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
}

func TestPipelineController_vars_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, _ := createTestPipeline(t, usr, "search-pipe", "Search Pipeline")

	v1 := &model.TPipelineVar{Uid: usr.Id, PipelineId: pipe.Id, Name: "DB_HOST", Value: "localhost", Public: 0}
	v2 := &model.TPipelineVar{Uid: usr.Id, PipelineId: pipe.Id, Name: "API_KEY", Value: "secret", Public: 0}
	if _, err := comm.Db.InsertOne(v1); err != nil {
		t.Fatalf("insert v1: %v", err)
	}
	if _, err := comm.Db.InsertOne(v2); err != nil {
		t.Fatalf("insert v2: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", pipe.Id)
	m.Set("q", "DB")
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
}

// === varSave ===

func TestPipelineController_varSave_EmptyParams(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, _ := createTestPipeline(t, usr, "save-pipe", "Save Pipeline")

	ctrl := PipelineController{}

	tests := []struct {
		name string
		pv   *bean.PipelineVar
	}{
		{"empty name", &bean.PipelineVar{Name: "", Value: "val", PipelineId: pipe.Id}},
		{"empty value", &bean.PipelineVar{Name: "KEY", Value: "", PipelineId: pipe.Id}},
		{"empty pipelineId", &bean.PipelineVar{Name: "KEY", Value: "val", PipelineId: ""}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, w := makePipelineGinCtx(t, tc.pv, usr)
			ctrl.varSave(c, tc.pv)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}
}

func TestPipelineController_varSave_NewVar(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, _ := createTestPipeline(t, usr, "varsave-pipe", "VarSave Pipeline")

	ctrl := PipelineController{}
	pv := &bean.PipelineVar{Name: "NEW_VAR", Value: "new_val", PipelineId: pipe.Id}
	c, w := makePipelineGinCtx(t, pv, usr)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify inserted
	saved := &model.TPipelineVar{}
	ok, _ := comm.Db.Where("pipeline_id = ? and name = ?", pipe.Id, "NEW_VAR").Get(saved)
	if !ok {
		t.Fatal("var not found after save")
	}
	if saved.Value != "new_val" {
		t.Errorf("saved.Value = %q, want %q", saved.Value, "new_val")
	}
}

func TestPipelineController_varSave_DuplicateName(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, _ := createTestPipeline(t, usr, "dup-pipe", "Dup Pipeline")

	// Insert existing var
	v := &model.TPipelineVar{Uid: usr.Id, PipelineId: pipe.Id, Name: "EXISTING", Value: "old"}
	if _, err := comm.Db.InsertOne(v); err != nil {
		t.Fatalf("insert: %v", err)
	}

	ctrl := PipelineController{}
	pv := &bean.PipelineVar{Name: "EXISTING", Value: "new", PipelineId: pipe.Id}
	c, w := makePipelineGinCtx(t, pv, usr)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

// === varDel ===

func TestPipelineController_varDel_InvalidAid(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "admin", "Admin")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varDel_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "admin", "Admin")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(99999))
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_varDel_Success(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, _ := createTestPipeline(t, usr, "delvar-pipe", "DelVar Pipeline")

	v := &model.TPipelineVar{Uid: usr.Id, PipelineId: pipe.Id, Name: "TO_DELETE", Value: "val"}
	if _, err := comm.Db.InsertOne(v); err != nil {
		t.Fatalf("insert: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", v.Aid)
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	deleted := &model.TPipelineVar{}
	ok, _ := comm.Db.Where("aid = ?", v.Aid).Get(deleted)
	if ok {
		t.Error("var should be deleted")
	}
}

// === getPipelines ===

func TestPipelineController_getPipelines_Success(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	createTestPipeline(t, usr, "pipe1", "Pipeline 1")
	createTestPipeline(t, usr, "pipe2", "Pipeline 2")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
}

func TestPipelineController_getPipelines_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	createTestPipeline(t, usr, "alpha", "Alpha Pipeline")
	createTestPipeline(t, usr, "beta", "Beta Pipeline")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("q", "alpha")
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
}

// === save ===

func TestPipelineController_save_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_save_InvalidYaml(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, _ := createTestPipeline(t, usr, "yaml-pipe", "YAML Pipeline")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", pipe.Id)
	m.Set("content", "{{invalid yaml content")
	m.Set("name", "yaml-pipe")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestPipelineController_save_Success(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, _ := createTestPipeline(t, usr, "save-pipe", "Save Pipeline")

	yamlContent := "stages:\n  - name: build\n    steps:\n      - name: test\n        step: shell\n        commands: echo hello"

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", pipe.Id)
	m.Set("content", yamlContent)
	m.Set("name", "save-pipe-updated")
	m.Set("displayName", "Save Pipeline Updated")
	m.Set("url", "https://github.com/example/updated")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify updated pipeline name
	updated := &model.TPipeline{}
	ok, _ := comm.Db.Where("id = ?", pipe.Id).Get(updated)
	if !ok {
		t.Fatal("pipeline not found after save")
	}
	if updated.Name != "save-pipe-updated" {
		t.Errorf("pipeline.Name = %q, want %q", updated.Name, "save-pipe-updated")
	}

	// Verify updated conf
	conf := &model.TPipelineConf{}
	ok, _ = comm.Db.Where("pipeline_id = ?", pipe.Id).Get(conf)
	if !ok {
		t.Fatal("conf not found after save")
	}
	if conf.Url != "https://github.com/example/updated" {
		t.Errorf("conf.Url = %q, want %q", conf.Url, "https://github.com/example/updated")
	}
}

// === searchSha ===

func TestPipelineController_searchSha_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "admin", "Admin")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_searchSha_NotFoundPipe(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "admin", "Admin")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_searchSha_Success(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, _ := createTestPipeline(t, usr, "sha-pipe", "SHA Pipeline")

	// Insert some versions
	ver1 := &model.TPipelineVersion{
		Id: utils.NewXid(), PipelineId: pipe.Id, Sha: "abc123", Created: time.Now(),
	}
	ver2 := &model.TPipelineVersion{
		Id: utils.NewXid(), PipelineId: pipe.Id, Sha: "abc456", Created: time.Now(),
	}
	ver3 := &model.TPipelineVersion{
		Id: utils.NewXid(), PipelineId: pipe.Id, Sha: "def789", Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(ver1); err != nil {
		t.Fatalf("insert ver1: %v", err)
	}
	if _, err := comm.Db.InsertOne(ver2); err != nil {
		t.Fatalf("insert ver2: %v", err)
	}
	if _, err := comm.Db.InsertOne(ver3); err != nil {
		t.Fatalf("insert ver3: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", pipe.Id)
	m.Set("q", "abc")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("len = %d, want 2", len(resp))
	}
}

// === pipelineVersion ===

func TestPipelineController_pipelineVersion_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "admin", "Admin")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// === fillPipelineListBuildInfo ===

func TestFillPipelineListBuildInfo_Empty(t *testing.T) {
	setupPipelineTestDB(t)
	ctx := context.Background()
	if err := fillPipelineListBuildInfo(ctx, nil); err != nil {
		t.Errorf("fillPipelineListBuildInfo(nil) = %v", err)
	}
	if err := fillPipelineListBuildInfo(ctx, []*model.TPipeline{}); err != nil {
		t.Errorf("fillPipelineListBuildInfo(empty) = %v", err)
	}
}

func TestFillPipelineListBuildInfo_WithData(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "owner", "Owner")
	pipe, _ := createTestPipeline(t, usr, "fill-pipe", "Fill Pipeline")

	ctx := context.Background()
	ls := []*model.TPipeline{pipe}
	if err := fillPipelineListBuildInfo(ctx, ls); err != nil {
		t.Fatalf("fillPipelineListBuildInfo: %v", err)
	}
	// Verify nick is populated
	if pipe.Nick != "Owner" {
		t.Errorf("pipe.Nick = %q, want %q", pipe.Nick, "Owner")
	}
}
