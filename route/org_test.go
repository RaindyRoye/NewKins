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

func setupOrgTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

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
		t.Fatalf("create t_org table: %v", err)
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
		t.Fatalf("create t_user table: %v", err)
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
		t.Fatalf("create t_user_info table: %v", err)
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
		t.Fatalf("create t_user_org table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org_pipe (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		created DATETIME,
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_org_pipe table: %v", err)
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
		t.Fatalf("create t_org_var table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(255),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		create_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline table: %v", err)
	}

	comm.Db = db
	return db
}

func makeOrgGinContext(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

// --- OrgController.new tests ---

func TestOrgController_new_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"name": "", "desc": "test"}, adminUser)
	ctrl := OrgController{}
	ctrl.new(c, &hbtp.Map{"name": "", "desc": "test"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_new_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"name": "test-org", "desc": "test desc", "public": true}, adminUser)
	ctrl := OrgController{}
	ctrl.new(c, &hbtp.Map{"name": "test-org", "desc": "test desc", "public": true})

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify org was created
	org := &model.TOrg{}
	ok, err := db.Where("name=?", "test-org").Get(org)
	if err != nil {
		t.Fatalf("query org: %v", err)
	}
	if !ok {
		t.Fatal("org was not created")
	}
	if org.Uid != "admin" {
		t.Errorf("org uid = %q, want %q", org.Uid, "admin")
	}
	if org.Public != 1 {
		t.Errorf("org public = %d, want 1", org.Public)
	}
}

func TestOrgController_new_NonAdminNoPerm(t *testing.T) {
	setupOrgTestDB(t)
	// Non-admin user without PermOrg
	_, err := comm.Db.InsertOne(&model.TUser{
		Id: "regular", Name: "regular", Active: 1,
		Created: time.Now(), LoginTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, err = comm.Db.InsertOne(&model.TUserInfo{Id: "regular", PermOrg: 0})
	if err != nil {
		t.Fatalf("create user info: %v", err)
	}
	regularUser := &model.TUser{Id: "regular", Name: "regular", Active: 1}

	c, w := makeOrgGinContext(t, hbtp.Map{"name": "test-org"}, regularUser)
	ctrl := OrgController{}
	ctrl.new(c, &hbtp.Map{"name": "test-org"})

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- OrgController.info tests ---

func TestOrgController_info_EmptyId(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"id": ""}, nil)
	ctrl := OrgController{}
	ctrl.info(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_info_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "nonexistent"}, nil)
	ctrl := OrgController{}
	ctrl.info(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_Deleted(t *testing.T) {
	setupOrgTestDB(t)
	_, err := comm.Db.Exec(`INSERT INTO t_org (id, aid, uid, name, deleted) VALUES ('org-del', 1, 'user1', 'Deleted Org', 1)`)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}

	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-del"}, nil)
	ctrl := OrgController{}
	ctrl.info(c, &hbtp.Map{"id": "org-del"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	// Create owner user
	_, err := db.InsertOne(&model.TUser{
		Id: "owner1", Name: "owner", Nick: "Owner Nick", Active: 1,
		Created: time.Now(), LoginTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	// Create org owned by owner1
	_, err = db.Exec(`INSERT INTO t_org (id, aid, uid, name, "desc", public, created, updated, deleted) VALUES ('org-1', 1, 'owner1', 'Test Org', 'desc', 1, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Public org should be readable
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-1"}, &model.TUser{Id: "other", Name: "other"})
	ctrl := OrgController{}
	ctrl.info(c, &hbtp.Map{"id": "org-1"})

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["org"] == nil {
		t.Error("expected 'org' in response")
	}
	if resp["user"] == nil {
		t.Error("expected 'user' in response")
	}
	if resp["perm"] == nil {
		t.Error("expected 'perm' in response")
	}
}

// --- OrgController.rm tests ---

func TestOrgController_rm_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "nonexistent"}, nil)
	ctrl := OrgController{}
	ctrl.rm(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_rm_NoPerm(t *testing.T) {
	db := setupOrgTestDB(t)
	_, err := db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-rm', 1, 'owner1', 'Test', 0, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	regularUser := &model.TUser{Id: "regular", Name: "regular", Active: 1}

	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-rm"}, regularUser)
	ctrl := OrgController{}
	ctrl.rm(c, &hbtp.Map{"id": "org-rm"})

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOrgController_rm_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	_, err := db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-del', 1, 'admin', 'To Delete', 1, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-del"}, adminUser)
	ctrl := OrgController{}
	ctrl.rm(c, &hbtp.Map{"id": "org-del"})

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify soft delete
	org := &model.TOrg{}
	ok, err := db.Where("id=?", "org-del").Get(org)
	if err != nil {
		t.Fatalf("query org: %v", err)
	}
	if !ok {
		t.Fatal("org should still exist (soft delete)")
	}
	if org.Deleted != 1 {
		t.Errorf("org deleted = %d, want 1", org.Deleted)
	}
}

// --- OrgController.vars tests ---

func TestOrgController_vars_EmptyOrgId(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"orgId": ""}, nil)
	ctrl := OrgController{}
	ctrl.vars(c, &hbtp.Map{"orgId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_vars_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	// Create public org
	_, err := db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-vars', 1, 'admin', 'Org Vars', 1, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	// Insert org vars
	_, err = db.Exec(`INSERT INTO t_org_var (aid, uid, org_id, name, value, remarks, public) VALUES (1, 'admin', 'org-vars', 'VAR1', 'val1', 'remark1', 0)`)
	if err != nil {
		t.Fatalf("insert org var: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_org_var (aid, uid, org_id, name, value, remarks, public) VALUES (2, 'admin', 'org-vars', 'VAR2', 'secret-val', 'remark2', 1)`)
	if err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"orgId": "org-vars", "q": "", "page": int64(1)}, adminUser)
	ctrl := OrgController{}
	ctrl.vars(c, &hbtp.Map{"orgId": "org-vars", "q": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- OrgController.varDel tests ---

func TestOrgController_varDel_InvalidAid(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"aid": int64(0)}, nil)
	ctrl := OrgController{}
	ctrl.varDel(c, &hbtp.Map{"aid": int64(0)})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varDel_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"aid": int64(999)}, nil)
	ctrl := OrgController{}
	ctrl.varDel(c, &hbtp.Map{"aid": int64(999)})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_varDel_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	// Create org owned by admin
	_, err := db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-vd', 1, 'admin', 'Org', 1, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_org_var (aid, uid, org_id, name, value, public) VALUES (1, 'admin', 'org-vd', 'VAR1', 'val1', 0)`)
	if err != nil {
		t.Fatalf("insert org var: %v", err)
	}
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeOrgGinContext(t, hbtp.Map{"aid": int64(1)}, adminUser)
	ctrl := OrgController{}
	ctrl.varDel(c, &hbtp.Map{"aid": int64(1)})

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- OrgController.pipeAdd tests ---

func TestOrgController_pipeAdd_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "nonexistent", "pipeId": "pipe-1"}, nil)
	ctrl := OrgController{}
	ctrl.pipeAdd(c, &hbtp.Map{"id": "nonexistent", "pipeId": "pipe-1"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_pipeAdd_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	_, err := db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-pa', 1, 'admin', 'Org', 1, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-pa", "pipeId": "pipe-1"}, adminUser)
	ctrl := OrgController{}
	ctrl.pipeAdd(c, &hbtp.Map{"id": "org-pa", "pipeId": "pipe-1"})

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_pipeAdd_Duplicate(t *testing.T) {
	db := setupOrgTestDB(t)
	_, err := db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-dup', 1, 'admin', 'Org', 1, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_org_pipe (aid, org_id, pipe_id, created, public) VALUES (1, 'org-dup', 'pipe-1', ?, 0)`, time.Now())
	if err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-dup", "pipeId": "pipe-1"}, adminUser)
	ctrl := OrgController{}
	ctrl.pipeAdd(c, &hbtp.Map{"id": "org-dup", "pipeId": "pipe-1"})

	if w.Code != http.StatusConflict {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusConflict)
	}
}

// --- OrgController.pipeRm tests ---

func TestOrgController_pipeRm_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "nonexistent", "pipeId": "pipe-1"}, nil)
	ctrl := OrgController{}
	ctrl.pipeRm(c, &hbtp.Map{"id": "nonexistent", "pipeId": "pipe-1"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- OrgController.users tests ---

func TestOrgController_users_EmptyId(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"id": ""}, nil)
	ctrl := OrgController{}
	ctrl.users(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_users_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	// 用户未登录，无法读取权限，返回 405
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "nonexistent"}, nil)
	ctrl := OrgController{}
	ctrl.users(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOrgController_users_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	// Create public org
	_, err := db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-users', 1, 'admin', 'Org', 1, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	// Create user in org
	_, err = db.InsertOne(&model.TUser{
		Id: "member1", Name: "member", Nick: "Member", Active: 1,
		Created: time.Now(), LoginTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_user_org (aid, uid, org_id, created, perm_adm, perm_rw, perm_exec) VALUES (1, 'member1', 'org-users', ?, 1, 0, 0)`, time.Now())
	if err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-users"}, &model.TUser{Id: "member1", Name: "member", Active: 1})
	ctrl := OrgController{}
	ctrl.users(c, &hbtp.Map{"id": "org-users"})

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["adms"] == nil {
		t.Error("expected 'adms' in response")
	}
}

// --- OrgController.userRm tests ---

func TestOrgController_userRm_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "nonexistent", "uid": "user-1"}, nil)
	ctrl := OrgController{}
	ctrl.userRm(c, &hbtp.Map{"id": "nonexistent", "uid": "user-1"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- OrgController.save tests ---

func TestOrgController_save_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-1", "name": ""}, nil)
	ctrl := OrgController{}
	ctrl.save(c, &hbtp.Map{"id": "org-1", "name": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_save_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "nonexistent", "name": "new name"}, nil)
	ctrl := OrgController{}
	ctrl.save(c, &hbtp.Map{"id": "nonexistent", "name": "new name"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- OrgController.list tests ---

func TestOrgController_list_EmptyDB(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"q": "", "page": int64(1)}, adminUser)
	ctrl := OrgController{}
	ctrl.list(c, &hbtp.Map{"q": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_WithSearch(t *testing.T) {
	db := setupOrgTestDB(t)
	_, err := db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-s1', 1, 'admin', 'Alpha Org', 1, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-s2', 2, 'admin', 'Beta Org', 1, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"q": "Alpha", "page": int64(1)}, adminUser)
	ctrl := OrgController{}
	ctrl.list(c, &hbtp.Map{"q": "Alpha", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- OrgController.userEdit tests ---

func TestOrgController_userEdit_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "nonexistent", "uid": "user-1", "adm": true}, nil)
	ctrl := OrgController{}
	ctrl.userEdit(c, &hbtp.Map{"id": "nonexistent", "uid": "user-1", "adm": true})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_userEdit_UserNotFound(t *testing.T) {
	db := setupOrgTestDB(t)
	_, err := db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-ue', 1, 'admin', 'Org', 1, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-ue", "uid": "nonexistent", "adm": true}, adminUser)
	ctrl := OrgController{}
	ctrl.userEdit(c, &hbtp.Map{"id": "org-ue", "uid": "nonexistent", "adm": true})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- OrgController.varSave tests ---

func TestOrgController_varSave_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, nil, nil)
	ctrl := OrgController{}

	pv := &bean.OrgVar{
		OrgId: "org-1",
		Name:  "",
		Value: "val",
	}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varSave_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	c, w := makeOrgGinContext(t, nil, nil)
	ctrl := OrgController{}

	pv := &bean.OrgVar{
		OrgId: "nonexistent",
		Name:  "VAR1",
		Value: "val1",
	}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_varSave_CreateSuccess(t *testing.T) {
	db := setupOrgTestDB(t)
	_, err := db.Exec(`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-vs', 1, 'admin', 'Org', 1, ?, ?, 0)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, nil, adminUser)
	ctrl := OrgController{}

	pv := &bean.OrgVar{
		OrgId: "org-vs",
		Name:  "NEW_VAR",
		Value: "new-value",
		Public: true,
	}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
