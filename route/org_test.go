package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

// setupOrgTestDB creates an in-memory SQLite database with all tables needed
// by the OrgController tests.
func setupOrgTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		comm.Db = origDb
	})

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
			perm_org INT,
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
			aid BIGINT PRIMARY KEY,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_pipe (
			aid BIGINT PRIMARY KEY,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_var (
			aid BIGINT PRIMARY KEY,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			name VARCHAR(255),
			value TEXT,
			remarks VARCHAR(255),
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
			create_time DATETIME
		)`,
	}

	for _, ddl := range tables {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("exec DDL: %v", err)
		}
	}

	comm.Db = db
	return db
}

func makeOrgGinCtx(t *testing.T, body interface{}, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest("POST", "/test", bytes.NewReader(b))
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

// --- OrgController.new ---

func TestOrgController_new_MissingName(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "u1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"name": ""}, admin)
	ctrl := OrgController{}
	ctrl.new(c, &hbtp.Map{"name": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_new_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	regular := &model.TUser{Id: "u1", Name: "regular", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"name": "test-org"}, regular)
	ctrl := OrgController{}
	ctrl.new(c, &hbtp.Map{"name": "test-org"})

	// User has no PermOrg and is not admin, should get 405
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestOrgController_new_SuccessAdmin(t *testing.T) {
	setupOrgTestDB(t)
	// Create super admin user (Id must be "admin" for IsAdmin to return true)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(admin)
	
	c, w := makeOrgGinCtx(t, hbtp.Map{"name": "my-org", "desc": "desc", "public": true}, admin)
	ctrl := OrgController{}
	ctrl.new(c, &hbtp.Map{"name": "my-org", "desc": "desc", "public": true})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["id"]; !ok {
		t.Error("response should contain 'id'")
	}
}

func TestOrgController_new_SuccessWithPerm(t *testing.T) {
	setupOrgTestDB(t)

	user := &model.TUser{Id: "u1", Name: "regular", Active: 1}
	_, _ = comm.Db.InsertOne(user)
	uinfo := &model.TUserInfo{Id: "u1", PermOrg: 1}
	_, _ = comm.Db.InsertOne(uinfo)

	c, w := makeOrgGinCtx(t, hbtp.Map{"name": "perm-org"}, user)
	ctrl := OrgController{}
	ctrl.new(c, &hbtp.Map{"name": "perm-org"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- OrgController.list ---

func TestOrgController_list_EmptyDB(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(admin)
	c, w := makeOrgGinCtx(t, hbtp.Map{"q": "", "page": int64(1)}, admin)
	ctrl := OrgController{}
	ctrl.list(c, &hbtp.Map{"q": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_NonAdmin(t *testing.T) {
	setupOrgTestDB(t)
	user := &model.TUser{Id: "u1", Name: "regular", Active: 1}

	// Create a public org and a private org
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Name: "public-org", Public: 1, Created: time.Now(), Updated: time.Now()})
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org2", Uid: "u1", Name: "my-org", Created: time.Now(), Updated: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"q": "", "page": int64(1)}, user)
	ctrl := OrgController{}
	ctrl.list(c, &hbtp.Map{"q": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_WithSearch(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Name: "alpha-org", Created: time.Now(), Updated: time.Now()})
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org2", Name: "beta-org", Created: time.Now(), Updated: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"q": "alpha", "page": int64(1)}, admin)
	ctrl := OrgController{}
	ctrl.list(c, &hbtp.Map{"q": "alpha", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- OrgController.info ---

func TestOrgController_info_MissingID(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"id": ""}, admin)
	ctrl := OrgController{}
	ctrl.info(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_info_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "nonexistent"}, admin)
	ctrl := OrgController{}
	ctrl.info(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	orgOwner := &model.TUser{Id: "owner1", Name: "owner", Nick: "Owner", Active: 1}
	_, _ = comm.Db.InsertOne(orgOwner)
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Uid: "owner1", Name: "test-org", Public: 1, Created: time.Now(), Updated: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "org1"}, admin)
	ctrl := OrgController{}
	ctrl.info(c, &hbtp.Map{"id": "org1"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_info_DeletedOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Name: "del-org", Deleted: 1, Created: time.Now(), Updated: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "org1"}, admin)
	ctrl := OrgController{}
	ctrl.info(c, &hbtp.Map{"id": "org1"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- OrgController.users ---

func TestOrgController_users_MissingID(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"id": ""}, admin)
	ctrl := OrgController{}
	ctrl.users(c, &hbtp.Map{"id": ""})

	// Empty id should return 400 Bad Request per the handler
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_users_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "nonexistent"}, admin)
	ctrl := OrgController{}
	ctrl.users(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_users_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Name: "test-org", Public: 1, Created: time.Now(), Updated: time.Now()})
	member := &model.TUser{Id: "u2", Name: "member", Nick: "Member", Active: 1}
	_, _ = comm.Db.InsertOne(member)
	_, _ = comm.Db.InsertOne(&model.TUserOrg{Uid: "u2", OrgId: "org1", PermAdm: 1, Created: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "org1"}, admin)
	ctrl := OrgController{}
	ctrl.users(c, &hbtp.Map{"id": "org1"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- OrgController.save ---

func TestOrgController_save_MissingName(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "org1", "name": ""}, admin)
	ctrl := OrgController{}
	ctrl.save(c, &hbtp.Map{"id": "org1", "name": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_save_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "nonexistent", "name": "new-name"}, admin)
	ctrl := OrgController{}
	ctrl.save(c, &hbtp.Map{"id": "nonexistent", "name": "new-name"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_save_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Name: "old-name", Created: time.Now(), Updated: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "org1", "name": "new-name", "desc": "updated"}, admin)
	ctrl := OrgController{}
	ctrl.save(c, &hbtp.Map{"id": "org1", "name": "new-name", "desc": "updated"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- OrgController.rm ---

func TestOrgController_rm_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "nonexistent"}, admin)
	ctrl := OrgController{}
	ctrl.rm(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_rm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Name: "rm-org", Created: time.Now(), Updated: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "org1"}, admin)
	ctrl := OrgController{}
	ctrl.rm(c, &hbtp.Map{"id": "org1"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify soft-deleted
	org := &model.TOrg{}
	ok, _ := comm.Db.Where("id=?", "org1").Get(org)
	if !ok || org.Deleted != 1 {
		t.Error("org should be soft-deleted")
	}
}

// --- OrgController.userEdit ---

func TestOrgController_userEdit_SelfEdit(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(admin)
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Uid: "admin", Name: "test-org", Created: time.Now(), Updated: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "org1", "uid": "admin"}, admin)
	ctrl := OrgController{}
	ctrl.userEdit(c, &hbtp.Map{"id": "org1", "uid": "admin"})

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// --- OrgController.userRm ---

func TestOrgController_userRm_SelfRemoval(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(admin)
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Uid: "admin", Name: "test-org", Created: time.Now(), Updated: time.Now()})
	_, _ = comm.Db.InsertOne(&model.TUserOrg{Uid: "admin", OrgId: "org1", PermAdm: 1, Created: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "org1", "uid": "admin"}, admin)
	ctrl := OrgController{}
	ctrl.userRm(c, &hbtp.Map{"id": "org1", "uid": "admin"})

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// --- OrgController.pipeAdd ---

func TestOrgController_pipeAdd_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "nonexistent", "pipeId": "pipe1"}, admin)
	ctrl := OrgController{}
	ctrl.pipeAdd(c, &hbtp.Map{"id": "nonexistent", "pipeId": "pipe1"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_pipeAdd_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Name: "test-org", Created: time.Now(), Updated: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "org1", "pipeId": "pipe1"}, admin)
	ctrl := OrgController{}
	ctrl.pipeAdd(c, &hbtp.Map{"id": "org1", "pipeId": "pipe1"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_pipeAdd_Duplicate(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Name: "test-org", Created: time.Now(), Updated: time.Now()})
	_, _ = comm.Db.InsertOne(&model.TOrgPipe{OrgId: "org1", PipeId: "pipe1", Created: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "org1", "pipeId": "pipe1"}, admin)
	ctrl := OrgController{}
	ctrl.pipeAdd(c, &hbtp.Map{"id": "org1", "pipeId": "pipe1"})

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// --- OrgController.pipeRm ---

func TestOrgController_pipeRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Name: "test-org", Created: time.Now(), Updated: time.Now()})
	_, _ = comm.Db.InsertOne(&model.TOrgPipe{OrgId: "org1", PipeId: "pipe1", Created: time.Now()})

	c, w := makeOrgGinCtx(t, hbtp.Map{"id": "org1", "pipeId": "pipe1"}, admin)
	ctrl := OrgController{}
	ctrl.pipeRm(c, &hbtp.Map{"id": "org1", "pipeId": "pipe1"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- OrgController.vars ---

func TestOrgController_vars_MissingOrgId(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"orgId": ""}, admin)
	ctrl := OrgController{}
	ctrl.vars(c, &hbtp.Map{"orgId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_vars_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"orgId": "nonexistent"}, admin)
	ctrl := OrgController{}
	ctrl.vars(c, &hbtp.Map{"orgId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_vars_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TOrg{Id: "org1", Name: "test-org", Public: 1, Created: time.Now(), Updated: time.Now()})
	_, _ = comm.Db.InsertOne(&model.TOrgVar{OrgId: "org1", Name: "VAR1", Value: "val1", Public: 1})

	c, w := makeOrgGinCtx(t, hbtp.Map{"orgId": "org1", "q": "", "page": int64(1)}, admin)
	ctrl := OrgController{}
	ctrl.vars(c, &hbtp.Map{"orgId": "org1", "q": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- OrgController.varDel ---

func TestOrgController_varDel_InvalidAid(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"aid": int64(0)}, admin)
	ctrl := OrgController{}
	ctrl.varDel(c, &hbtp.Map{"aid": int64(0)})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varDel_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	c, w := makeOrgGinCtx(t, hbtp.Map{"aid": int64(999)}, admin)
	ctrl := OrgController{}
	ctrl.varDel(c, &hbtp.Map{"aid": int64(999)})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
