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
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupOrgTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create tables
	_, err = db.Exec(`CREATE TABLE t_org (
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
	)`)
	if err != nil {
		t.Fatalf("create org table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL,
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
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

	_, err = db.Exec(`CREATE TABLE t_user_org (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
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

	_, err = db.Exec(`CREATE TABLE t_org_var (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		name VARCHAR(255),
		value TEXT,
		remarks VARCHAR(255),
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create org_var table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org_pipe (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		created DATETIME,
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create org_pipe table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) NOT NULL,
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		name VARCHAR(200),
		display_name VARCHAR(200),
		pipeline_type VARCHAR(50),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create pipeline table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`INSERT INTO t_user (id, aid, name, nick, avatar, active) 
		VALUES ('admin', 1, 'admin', 'Admin User', '', 1)`)
	if err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	_, err = db.Exec(`INSERT INTO t_user (id, aid, name, nick, avatar, active) 
		VALUES ('user-1', 2, 'alice', 'Alice', 'avatar.png', 1)`)
	if err != nil {
		t.Fatalf("insert user-1: %v", err)
	}

	_, err = db.Exec(`INSERT INTO t_org (id, aid, uid, name, "desc", public, created, updated) 
		VALUES ('org-1', 1, 'user-1', 'Test Org', 'Description', 1, ?, ?)`,
		time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org-1: %v", err)
	}

	_, err = db.Exec(`INSERT INTO t_org (id, aid, uid, name, "desc", public, created, updated, deleted) 
		VALUES ('org-deleted', 2, 'user-1', 'Deleted Org', '', 0, ?, ?, 1)`,
		time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert deleted org: %v", err)
	}

	comm.Db = db
}

func makeOrgTestContext(t *testing.T, body interface{}, userId string) (*gin.Context, *httptest.ResponseRecorder) {
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
	req = req.WithContext(context.Background())
	c.Request = req

	// Set test user
	c.Set(service.LgUserKey, &model.TUser{
		Id:     userId,
		Name:   userId,
		Active: 1,
	})
	return c, w
}

func TestOrgController_GetPathFromOrg(t *testing.T) {
	c := &OrgController{}
	if got := c.GetPath(); got != "/api/org" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/org")
	}
}

func TestOrgController_Routes(t *testing.T) {
	setupOrgTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	oc := &OrgController{}
	oc.Routes(r.Group("/api/org"))
	// Routes registered successfully
}

func TestOrgList_Admin(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	// Should have data field (Page struct uses 'data' not 'list')
	if _, ok := resp["data"]; !ok {
		t.Error("response should contain 'data' field")
	}
}

func TestOrgList_RegularUser(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgTestContext(t, m, "user-1")
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOrgList_WithSearch(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "Test")
	m.Set("page", int64(1))
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestOrgNew_Admin_Success(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "New Org")
	m.Set("desc", "New description")
	m.Set("public", true)
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if _, ok := resp["id"]; !ok {
		t.Error("response should contain 'id' field")
	}
}

func TestOrgNew_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgInfo_Success(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if _, ok := resp["org"]; !ok {
		t.Error("response should contain 'org' field")
	}
}

func TestOrgInfo_EmptyId(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgInfo_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgInfo_Deleted(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-deleted")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted org, got %d", w.Code)
	}
}

func TestOrgUsers_Success(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.users(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgUsers_EmptyId(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.users(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgSave_Success(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("name", "Updated Org")
	m.Set("desc", "Updated description")
	m.Set("public", true)
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgSave_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("name", "")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgRm_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgVars_Success(t *testing.T) {
	setupOrgTestDB(t)
	// Insert test var
	_, err := comm.Db.Exec(`INSERT INTO t_org_var (aid, org_id, name, value, public) 
		VALUES (1, 'org-1', 'TEST_VAR', 'test_value', 1)`)
	if err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgVars_EmptyOrgId(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgVarSave_Success(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	
	// Create var
	pv := &bean.OrgVar{
		OrgId: "org-1",
		Name:  "NEW_VAR",
		Value: "new_value",
	}
	c, w := makeOrgTestContext(t, pv, "admin")
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgVarSave_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	pv := &bean.OrgVar{
		OrgId: "org-1",
		Name:  "",
		Value: "value",
	}
	c, w := makeOrgTestContext(t, pv, "admin")
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgVarDel_Success(t *testing.T) {
	setupOrgTestDB(t)
	// Insert test var
	_, err := comm.Db.Exec(`INSERT INTO t_org_var (aid, org_id, name, value, public) 
		VALUES (100, 'org-1', 'DEL_VAR', 'value', 1)`)
	if err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(100))
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgVarDel_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(999))
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgVarDel_InvalidAid(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgUserEdit_Success(t *testing.T) {
	setupOrgTestDB(t)
	// Insert a user to edit
	_, err := comm.Db.Exec(`INSERT INTO t_user (id, name, nick) VALUES ('user-2', 'user-2', 'User 2')`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("uid", "user-2")
	m.Set("adm", true)
	m.Set("rw", true)
	m.Set("ex", true)
	m.Set("dw", true)
	m.Set("add", false)
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.userEdit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgUserEdit_NotFoundOrg(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", "user-1")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgUserEdit_NotFoundUser(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("uid", "nonexistent")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgUserEdit_CantEditSelf(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("uid", "user-1") // Same as org owner
	c, w := makeOrgTestContext(t, m, "user-1")
	ctrl.userEdit(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestOrgUserRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	// Add user to org first
	_, err := comm.Db.Exec(`INSERT INTO t_user (id, name, nick) VALUES ('user-2', 'user-2', 'User 2')`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	_, err = comm.Db.Exec(`INSERT INTO t_user_org (uid, org_id) VALUES ('user-2', 'org-1')`)
	if err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("uid", "user-2")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.userRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgUserRm_NotFoundOrg(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", "user-1")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.userRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgUserRm_CantRemoveSelf(t *testing.T) {
	setupOrgTestDB(t)
	// Insert user_org record for self-removal test
	_, err := comm.Db.Exec(`INSERT INTO t_user_org (uid, org_id) VALUES ('user-1', 'org-1')`)
	if err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("uid", "user-1") // Same as requester
	c, w := makeOrgTestContext(t, m, "user-1")
	ctrl.userRm(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestOrgPipeAdd_Success(t *testing.T) {
	setupOrgTestDB(t)
	// Insert a pipeline
	_, err := comm.Db.Exec(`INSERT INTO t_pipeline (id, aid, name, uid) VALUES ('pipe-1', 1, 'Test Pipe', 'user-1')`)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("pipeId", "pipe-1")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgPipeAdd_NotFoundOrg(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pipeId", "pipe-1")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgPipeAdd_AlreadyExists(t *testing.T) {
	setupOrgTestDB(t)
	// Insert pipeline and org_pipe relationship
	_, err := comm.Db.Exec(`INSERT INTO t_pipeline (id, aid, name, uid) VALUES ('pipe-2', 2, 'Test Pipe 2', 'user-1')`)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	_, err = comm.Db.Exec(`INSERT INTO t_org_pipe (aid, org_id, pipe_id) VALUES (1, 'org-1', 'pipe-2')`)
	if err != nil {
		t.Fatalf("insert org_pipe: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("pipeId", "pipe-2")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestOrgPipeRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	// Insert org_pipe to remove
	_, err := comm.Db.Exec(`INSERT INTO t_org_pipe (aid, org_id, pipe_id) VALUES (10, 'org-1', 'pipe-10')`)
	if err != nil {
		t.Fatalf("insert org_pipe: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("pipeId", "pipe-10")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgPipeRm_NotFoundOrg(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pipeId", "pipe-1")
	c, w := makeOrgTestContext(t, m, "admin")
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
