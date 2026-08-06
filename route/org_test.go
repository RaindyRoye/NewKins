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

func setupOrgTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	comm.Db = db

	_, err = db.Exec(`CREATE TABLE t_org (
		id VARCHAR(64) NOT NULL,
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		name VARCHAR(200),
		"desc" TEXT,
		public INT DEFAULT 0,
		created DATETIME,
		updated DATETIME,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create org table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		name VARCHAR(100),
		pass VARCHAR(255),
		nick VARCHAR(100),
		avatar VARCHAR(500),
		created DATETIME,
		login_time DATETIME,
		active INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create user table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_info (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		phone VARCHAR(100),
		email VARCHAR(200),
		birthday DATETIME,
		remark TEXT,
		perm_user INT,
		perm_org INT,
		perm_pipe INT
	)`)
	if err != nil {
		t.Fatalf("create user_info table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_org (
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		created DATETIME,
		perm_adm INT DEFAULT 0,
		perm_rw INT DEFAULT 0,
		perm_exec INT DEFAULT 0,
		perm_down INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create user_org table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org_pipe (
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		created DATETIME,
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create org_pipe table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(255),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		create_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create pipeline table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org_var (
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		name VARCHAR(200),
		value TEXT,
		remarks VARCHAR(500),
		public INT DEFAULT 0,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create org_var table: %v", err)
	}

	return db
}

func makeOrgGinCtx(t *testing.T, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestOrg_New_MissingName(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("desc", "test org")
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", w.Code)
	}
}

func TestOrg_New_AdminSuccess(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("name", "test-org")
	m.Set("desc", "A test org")
	m.Set("public", true)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected id in response")
	}

	// Verify in database
	org := &model.TOrg{}
	ok, err := db.Where("name = ?", "test-org").Get(org)
	if err != nil {
		t.Fatalf("query org: %v", err)
	}
	if !ok {
		t.Error("expected org to be created")
	}
	if org.Public != 1 {
		t.Errorf("expected public=1, got %d", org.Public)
	}
}

func TestOrg_New_NonAdminNoPermission(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "regular", Active: 1}
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	// Create user_info with perm_org = 0
	ui := &model.TUserInfo{Id: "user-1", PermOrg: 0}
	if _, err := db.InsertOne(ui); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	c, w := makeOrgGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("name", "new-org")
	m.Set("desc", "test")
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for non-admin without perm_org, got %d", w.Code)
	}
}

func TestOrg_Info_MissingId(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing id, got %d", w.Code)
	}
}

func TestOrg_Info_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent org, got %d", w.Code)
	}
}

func TestOrg_Info_Deleted(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	org := &model.TOrg{Id: "org-1", Uid: "admin", Name: "deleted-org", Deleted: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	c, w := makeOrgGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted org, got %d", w.Code)
	}
}

func TestOrg_Info_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	org := &model.TOrg{Id: "org-1", Uid: "admin", Name: "my-org", Public: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	c, w := makeOrgGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["org"] == nil {
		t.Error("expected org in response")
	}
	if resp["user"] == nil {
		t.Error("expected user in response")
	}
	if resp["perm"] == nil {
		t.Error("expected perm in response")
	}
}

func TestOrg_Save_MissingName(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("name", "")
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrg_Save_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("name", "updated")
	ctrl.save(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrg_Rm_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrg_Rm_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	org := &model.TOrg{Id: "org-1", Uid: "admin", Name: "to-delete", Created: time.Now(), Updated: time.Now()}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	c, w := makeOrgGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify soft delete
	updatedOrg := &model.TOrg{}
	ok, _ := db.Where("id = ?", "org-1").Get(updatedOrg)
	if !ok {
		t.Fatal("org should still exist")
	}
	if updatedOrg.Deleted != 1 {
		t.Errorf("expected deleted=1, got %d", updatedOrg.Deleted)
	}
}

func TestOrg_Users_NoPermission(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.users(c, m)

	// empty id returns 400 (bad request)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrg_Users_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.users(c, m)

	// nonexistent org returns not found or method not allowed
	if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 404 or 405, got %d", w.Code)
	}
}

func TestOrg_PipeAdd_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pipeId", "pipe-1")
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrg_PipeRm_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pipeId", "pipe-1")
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrg_Vars_MissingOrgId(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "")
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrg_Vars_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrg_VarSave_MissingFields(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Test empty name
	c, w := makeOrgGinCtx(t, user)
	pv := &bean.OrgVar{OrgId: "org-1", Name: "", Value: "val"}
	ctrl.varSave(c, pv)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrg_VarSave_NotFoundOrg(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeOrgGinCtx(t, user)
	pv := &bean.OrgVar{OrgId: "nonexistent", Name: "var1", Value: "val1"}
	ctrl.varSave(c, pv)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent org, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrg_VarDel_InvalidAid(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid aid, got %d", w.Code)
	}
}

func TestOrg_VarDel_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("aid", int64(999))
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrg_UserRm_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", "some-user")
	ctrl.userRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrg_UserEdit_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", "some-user")
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrg_List_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Insert test orgs
	orgs := []*model.TOrg{
		{Id: "org-1", Uid: "admin", Name: "org-alpha", Public: 1, Created: time.Now(), Updated: time.Now()},
		{Id: "org-2", Uid: "admin", Name: "org-beta", Public: 0, Created: time.Now(), Updated: time.Now()},
	}
	for _, o := range orgs {
		if _, err := db.InsertOne(o); err != nil {
			t.Fatalf("insert org: %v", err)
		}
	}

	c, w := makeOrgGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("page", int64(1))
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrg_List_WithSearch(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	orgs := []*model.TOrg{
		{Id: "org-1", Uid: "admin", Name: "frontend-team", Public: 1, Created: time.Now(), Updated: time.Now()},
		{Id: "org-2", Uid: "admin", Name: "backend-team", Public: 1, Created: time.Now(), Updated: time.Now()},
	}
	for _, o := range orgs {
		if _, err := db.InsertOne(o); err != nil {
			t.Fatalf("insert org: %v", err)
		}
	}

	c, w := makeOrgGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("q", "frontend")
	m.Set("page", int64(1))
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrg_PipeAdd_DuplicatePipeline(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	org := &model.TOrg{Id: "org-1", Uid: "admin", Name: "test-org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Insert pipeline
	pipe := &model.TPipeline{Id: "pipe-1", Name: "test-pipe", Uid: "admin"}
	if _, err := db.InsertOne(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Add pipeline to org first
	op := &model.TOrgPipe{OrgId: "org-1", PipeId: "pipe-1", Created: time.Now()}
	if _, err := db.InsertOne(op); err != nil {
		t.Fatalf("insert org_pipe: %v", err)
	}

	c, w := makeOrgGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("pipeId", "pipe-1")
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate pipeline, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrg_PipeAdd_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	org := &model.TOrg{Id: "org-1", Uid: "admin", Name: "test-org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	c, w := makeOrgGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("pipeId", "pipe-new")
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrg_PipeRm_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	org := &model.TOrg{Id: "org-1", Uid: "admin", Name: "test-org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	op := &model.TOrgPipe{OrgId: "org-1", PipeId: "pipe-1", Created: time.Now()}
	if _, err := db.InsertOne(op); err != nil {
		t.Fatalf("insert org_pipe: %v", err)
	}

	c, w := makeOrgGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("pipeId", "pipe-1")
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrg_Save_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	org := &model.TOrg{Id: "org-1", Uid: "admin", Name: "old-name", Created: time.Now(), Updated: time.Now()}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	c, w := makeOrgGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("name", "new-name")
	m.Set("desc", "updated description")
	m.Set("public", true)
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	updated := &model.TOrg{}
	ok, _ := db.Where("id = ?", "org-1").Get(updated)
	if !ok {
		t.Fatal("org should exist")
	}
	if updated.Name != "new-name" {
		t.Errorf("expected name 'new-name', got %q", updated.Name)
	}
}
