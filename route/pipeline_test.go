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
	origIsMySQL := comm.IsMySQL
	t.Cleanup(func() {
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	})

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	comm.IsMySQL = false

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
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id VARCHAR(64),
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

func createPipelineTestUser(t *testing.T, name, nick string, active int) *model.TUser {
	t.Helper()
	usr := &model.TUser{
		Id:        utils.NewXid(),
		Name:      name,
		Nick:      nick,
		Active:    active,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	if _, err := comm.Db.InsertOne(usr); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return usr
}

func createPipelineTestPipeline(t *testing.T, uid, name string) *model.TPipeline {
	t.Helper()
	p := &model.TPipeline{
		Id:         utils.NewXid(),
		Uid:        uid,
		Name:       name,
		CreateTime: time.Now(),
	}
	if _, err := comm.Db.InsertOne(p); err != nil {
		t.Fatalf("create pipeline: %v", err)
	}
	return p
}

func createPipelineConf(t *testing.T, pipelineId, content, url string) {
	t.Helper()
	tpc := &model.TPipelineConf{
		PipelineId: pipelineId,
		YmlContent: content,
		Url:        url,
	}
	if _, err := comm.Db.InsertOne(tpc); err != nil {
		t.Fatalf("create pipeline conf: %v", err)
	}
}

// ===== GetPath =====

func TestPipelineController_GetPathValue(t *testing.T) {
	ctrl := PipelineController{}
	if got := ctrl.GetPath(); got != "/api/pipeline" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/pipeline")
	}
}

// ===== info =====

func TestPipelineController_info_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestPipelineController_info_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestPipelineController_info_Deleted(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	p := createPipelineTestPipeline(t, usr.Id, "deleted-pipe")
	// Soft delete
	_, err := comm.Db.Where("id=?", p.Id).Cols("deleted").Update(&model.TPipeline{Deleted: 1})
	if err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", p.Id)
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d for deleted pipeline", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_info_OwnerCanRead(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	p := createPipelineTestPipeline(t, usr.Id, "my-pipe")
	createPipelineConf(t, p.Id, "version: 1\nstages: []", "https://github.com/test/repo")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", p.Id)
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	pipe, ok := resp["pipe"].(map[string]interface{})
	if !ok {
		t.Fatal("expected pipe object in response")
	}
	if pipe["name"] != "my-pipe" {
		t.Errorf("name = %v, want my-pipe", pipe["name"])
	}
	perm, ok := resp["perm"].(map[string]interface{})
	if !ok {
		t.Fatal("expected perm object in response")
	}
	if perm["read"] != true {
		t.Error("expected read=true for owner")
	}
	if perm["write"] != true {
		t.Error("expected write=true for owner")
	}
}

func TestPipelineController_info_NonOwnerNoAccess(t *testing.T) {
	setupPipelineTestDB(t)
	owner := createPipelineTestUser(t, "owner", "Owner", 1)
	other := createPipelineTestUser(t, "other", "Other", 1)
	p := createPipelineTestPipeline(t, owner.Id, "private-pipe")
	createPipelineConf(t, p.Id, "version: 1\nstages: []", "")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", p.Id)
	c, w := makePipelineGinCtx(t, m, other)
	ctrl.info(c, m)

	// Non-owner, non-admin, no org membership -> can't read
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// ===== delete =====

func TestPipelineController_delete_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

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
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestPipelineController_delete_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	owner := createPipelineTestUser(t, "owner", "Owner", 1)
	other := createPipelineTestUser(t, "other", "Other", 1)
	p := createPipelineTestPipeline(t, owner.Id, "protected-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", p.Id)
	c, w := makePipelineGinCtx(t, m, other)
	ctrl.delete(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestPipelineController_delete_OwnerSuccess(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	p := createPipelineTestPipeline(t, usr.Id, "to-delete")
	// Also create a pipeline version to test cascade soft-delete
	pv := &model.TPipelineVersion{
		Id:         utils.NewXid(),
		PipelineId: p.Id,
		Number:     1,
		Created:    time.Now(),
	}
	if _, err := comm.Db.InsertOne(pv); err != nil {
		t.Fatalf("insert pv: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", p.Id)
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify pipeline is soft-deleted
	tp := &model.TPipeline{}
	ok, _ := comm.Db.Where("id=?", p.Id).Get(tp)
	if !ok || tp.Deleted != 1 {
		t.Error("pipeline should be soft-deleted")
	}

	// Verify pipeline version is also soft-deleted
	tpv := &model.TPipelineVersion{}
	ok, _ = comm.Db.Where("id=?", pv.Id).Get(tpv)
	if !ok || tpv.Deleted != 1 {
		t.Error("pipeline version should be soft-deleted")
	}
}

// ===== save =====

func TestPipelineController_save_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_save_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	owner := createPipelineTestUser(t, "owner", "Owner", 1)
	other := createPipelineTestUser(t, "other", "Other", 1)
	p := createPipelineTestPipeline(t, owner.Id, "protected-pipe")
	createPipelineConf(t, p.Id, "", "")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", p.Id)
	m.Set("name", "new-name")
	m.Set("content", "stages: []")
	c, w := makePipelineGinCtx(t, m, other)
	ctrl.save(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestPipelineController_save_InvalidYaml(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	p := createPipelineTestPipeline(t, usr.Id, "yaml-test")
	createPipelineConf(t, p.Id, "", "")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", p.Id)
	m.Set("name", "updated")
	m.Set("content", "{{invalid yaml")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d for invalid yaml, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// ===== vars =====

func TestPipelineController_vars_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_vars_PipeNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

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
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	p := createPipelineTestPipeline(t, usr.Id, "vars-pipe")

	// Insert some vars
	for i, name := range []string{"VAR_A", "VAR_B", "VAR_C"} {
		v := &model.TPipelineVar{
			Uid:        usr.Id,
			PipelineId: p.Id,
			Name:       name,
			Value:      "value_" + name,
			Public:     i, // VAR_A not public, VAR_B and VAR_C public
		}
		if _, err := comm.Db.InsertOne(v); err != nil {
			t.Fatalf("insert var: %v", err)
		}
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", p.Id)
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineController_vars_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	p := createPipelineTestPipeline(t, usr.Id, "search-pipe")

	for _, name := range []string{"DB_HOST", "DB_PORT", "API_KEY"} {
		v := &model.TPipelineVar{
			Uid:        usr.Id,
			PipelineId: p.Id,
			Name:       name,
			Value:      "val_" + name,
		}
		if _, err := comm.Db.InsertOne(v); err != nil {
			t.Fatalf("insert var: %v", err)
		}
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", p.Id)
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
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2 (DB_HOST + DB_PORT)", resp.Total)
	}
}

func TestPipelineController_vars_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	owner := createPipelineTestUser(t, "owner", "Owner", 1)
	other := createPipelineTestUser(t, "other", "Other", 1)
	p := createPipelineTestPipeline(t, owner.Id, "private-vars-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", p.Id)
	c, w := makePipelineGinCtx(t, m, other)
	ctrl.vars(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// ===== varSave =====

func TestPipelineController_varSave_EmptyParams(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}

	// Empty name
	pv := &bean.PipelineVar{
		PipelineId: "some-id",
		Name:       "",
		Value:      "val",
	}
	c, w := makePipelineGinCtx(t, nil, usr)
	ctrl.varSave(c, pv)
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty name: status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	// Empty pipelineId
	pv2 := &bean.PipelineVar{
		PipelineId: "",
		Name:       "VAR",
		Value:      "val",
	}
	c2, w2 := makePipelineGinCtx(t, nil, usr)
	ctrl.varSave(c2, pv2)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("empty pipelineId: status = %d, want %d", w2.Code, http.StatusBadRequest)
	}

	// Empty value
	pv3 := &bean.PipelineVar{
		PipelineId: "some-id",
		Name:       "VAR",
		Value:      "",
	}
	c3, w3 := makePipelineGinCtx(t, nil, usr)
	ctrl.varSave(c3, pv3)
	if w3.Code != http.StatusBadRequest {
		t.Errorf("empty value: status = %d, want %d", w3.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varSave_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	owner := createPipelineTestUser(t, "owner", "Owner", 1)
	other := createPipelineTestUser(t, "other", "Other", 1)
	p := createPipelineTestPipeline(t, owner.Id, "protected-pipe")

	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		PipelineId: p.Id,
		Name:       "NEW_VAR",
		Value:      "value",
	}
	c, w := makePipelineGinCtx(t, nil, other)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestPipelineController_varSave_InsertNew(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	p := createPipelineTestPipeline(t, usr.Id, "var-pipe")

	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		PipelineId: p.Id,
		Name:       "MY_VAR",
		Value:      "hello",
		Remarks:    "test var",
		Public:     true,
	}
	c, w := makePipelineGinCtx(t, nil, usr)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify var was inserted
	tpv := &model.TPipelineVar{}
	ok, _ := comm.Db.Where("pipeline_id = ? and name = ?", p.Id, "MY_VAR").Get(tpv)
	if !ok {
		t.Fatal("expected var to be inserted")
	}
	if tpv.Value != "hello" {
		t.Errorf("value = %q, want %q", tpv.Value, "hello")
	}
	if tpv.Public != 1 {
		t.Errorf("public = %d, want 1", tpv.Public)
	}
}

func TestPipelineController_varSave_DuplicateName(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	p := createPipelineTestPipeline(t, usr.Id, "dup-pipe")

	// Insert existing var
	v := &model.TPipelineVar{
		Uid:        usr.Id,
		PipelineId: p.Id,
		Name:       "EXISTING",
		Value:      "old",
	}
	if _, err := comm.Db.InsertOne(v); err != nil {
		t.Fatalf("insert var: %v", err)
	}

	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		PipelineId: p.Id,
		Name:       "EXISTING",
		Value:      "new",
	}
	c, w := makePipelineGinCtx(t, nil, usr)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d for duplicate, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

// ===== varDel =====

func TestPipelineController_varDel_BadParam(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

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
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(99999))
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestPipelineController_varDel_Success(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	p := createPipelineTestPipeline(t, usr.Id, "delpipe")

	v := &model.TPipelineVar{
		Uid:        usr.Id,
		PipelineId: p.Id,
		Name:       "TO_DELETE",
		Value:      "val",
	}
	if _, err := comm.Db.InsertOne(v); err != nil {
		t.Fatalf("insert var: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", v.Aid)
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify deletion
	tpv := &model.TPipelineVar{}
	ok, _ := comm.Db.Where("aid=?", v.Aid).Get(tpv)
	if ok {
		t.Error("var should have been deleted")
	}
}

// ===== searchSha =====

func TestPipelineController_searchSha_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_searchSha_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_searchSha_NoPermission(t *testing.T) {
	setupPipelineTestDB(t)
	owner := createPipelineTestUser(t, "owner", "Owner", 1)
	other := createPipelineTestUser(t, "other", "Other", 1)
	p := createPipelineTestPipeline(t, owner.Id, "private-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", p.Id)
	c, w := makePipelineGinCtx(t, m, other)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPipelineController_searchSha_Success(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	p := createPipelineTestPipeline(t, usr.Id, "sha-pipe")

	// Create some versions with SHAs
	for _, sha := range []string{"abc123", "def456", "abc789"} {
		pv := &model.TPipelineVersion{
			Id:         utils.NewXid(),
			PipelineId: p.Id,
			Sha:        sha,
			Number:     1,
			Created:    time.Now(),
		}
		if _, err := comm.Db.InsertOne(pv); err != nil {
			t.Fatalf("insert pv: %v", err)
		}
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", p.Id)
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
		t.Errorf("got %d results, want 2 (abc123, abc789)", len(resp))
	}
}

// ===== getPipelines =====

func TestPipelineController_getPipelines_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{
		Id:     "admin",
		Name:   "admin",
		Nick:   "Admin",
		Active: 1,
	}
	if _, err := comm.Db.InsertOne(admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	// Create pipelines from different users
	createPipelineTestPipeline(t, "user1", "pipe-1")
	createPipelineTestPipeline(t, "user2", "pipe-2")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Admin sees all pipelines
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
}

func TestPipelineController_getPipelines_NonAdminOwnOnly(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	other := createPipelineTestUser(t, "user2", "User2", 1)
	createPipelineTestPipeline(t, usr.Id, "my-pipe")
	createPipelineTestPipeline(t, other.Id, "other-pipe")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("q", "")
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
	// Non-admin sees only own pipelines
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1 (own only)", resp.Total)
	}
}

func TestPipelineController_getPipelines_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	createPipelineTestPipeline(t, usr.Id, "alpha-build")
	createPipelineTestPipeline(t, usr.Id, "beta-deploy")
	createPipelineTestPipeline(t, usr.Id, "alpha-test")

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
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2 (alpha-build + alpha-test)", resp.Total)
	}
}

// ===== orgPipelines =====

func TestPipelineController_orgPipelines_EmptyOrgId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_orgPipelines_OrgNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

// ===== pipelineVersions =====

func TestPipelineController_pipelineVersions_WithPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	p := createPipelineTestPipeline(t, usr.Id, "versioned-pipe")

	// Create versions
	for i := 1; i <= 3; i++ {
		pv := &model.TPipelineVersion{
			Id:         utils.NewXid(),
			PipelineId: p.Id,
			Number:     int64(i),
			Created:    time.Now(),
		}
		if _, err := comm.Db.InsertOne(pv); err != nil {
			t.Fatalf("insert pv: %v", err)
		}
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", p.Id)
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("total = %d, want 3", resp.Total)
	}
}

func TestPipelineController_pipelineVersions_PipeNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_pipelineVersions_AdminAllVersions(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{
		Id:     "admin",
		Name:   "admin",
		Nick:   "Admin",
		Active: 1,
	}
	if _, err := comm.Db.InsertOne(admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	p := createPipelineTestPipeline(t, "someuser", "test-pipe")
	pv := &model.TPipelineVersion{
		Id:         utils.NewXid(),
		PipelineId: p.Id,
		Number:     1,
		Created:    time.Now(),
	}
	if _, err := comm.Db.InsertOne(pv); err != nil {
		t.Fatalf("insert pv: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "") // empty = all versions for admin
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1 (admin sees all)", resp.Total)
	}
}

func TestPipelineController_pipelineVersions_NonAdminEmpty(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)
	// Non-admin with no pipelines -> empty result

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "") // empty = own pipelines for non-admin
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

// ===== run =====

func TestPipelineController_run_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.run(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_run_PipeNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.run(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

// ===== rebuild =====

func TestPipelineController_rebuild_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.rebuild(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_rebuild_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "nonexistent")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.rebuild(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

// ===== copy =====

func TestPipelineController_copy_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.copy(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_copy_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.copy(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ===== pipelineVersion =====

func TestPipelineController_pipelineVersion_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_pipelineVersion_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	usr := createPipelineTestUser(t, "user1", "User1", 1)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineGinCtx(t, m, usr)
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
