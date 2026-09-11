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
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create t_org table
	_, err = db.Exec(`CREATE TABLE t_org (
		id VARCHAR(64) PRIMARY KEY,
		aid BIGINT,
		uid VARCHAR(64),
		name VARCHAR(100),
		"desc" VARCHAR(500),
		avatar VARCHAR(500),
		public INT DEFAULT 0,
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_org table: %v", err)
	}

	// Create t_user table
	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) PRIMARY KEY,
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

	// Create t_user_info table
	_, err = db.Exec(`CREATE TABLE t_user_info (
		id VARCHAR(64) PRIMARY KEY,
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

	// Create t_user_org table
	_, err = db.Exec(`CREATE TABLE t_user_org (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		perm_adm INT DEFAULT 0,
		perm_rw INT DEFAULT 0,
		perm_exec INT DEFAULT 0,
		perm_down INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_user_org table: %v", err)
	}

	// Create t_pipeline table
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

	// Create t_org_pipe table
	_, err = db.Exec(`CREATE TABLE t_org_pipe (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		public INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_org_pipe table: %v", err)
	}

	// Create t_org_var table
	_, err = db.Exec(`CREATE TABLE t_org_var (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		name VARCHAR(100),
		value TEXT,
		remarks TEXT,
		public INT DEFAULT 0,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_org_var table: %v", err)
	}

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

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

func TestOrgController_list_EmptyDB(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"q": "", "page": int64(1)}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_WithOrgs(t *testing.T) {
	db := setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create test orgs
	orgs := []*model.TOrg{
		{Id: "org-1", Aid: 1, Uid: "admin", Name: "Org One", Public: 1, Created: time.Now(), Updated: time.Now()},
		{Id: "org-2", Aid: 2, Uid: "admin", Name: "Org Two", Public: 1, Created: time.Now(), Updated: time.Now()},
	}
	for _, org := range orgs {
		if _, err := db.Insert(org); err != nil {
			t.Fatalf("insert org: %v", err)
		}
	}

	c, w := makeOrgGinContext(t, hbtp.Map{"q": "", "page": int64(1)}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_new_MissingName(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"name": "", "desc": "test"}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("desc", "test")
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_new_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"name": "New Org", "desc": "A new org", "public": true}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "New Org")
	m.Set("desc", "A new org")
	m.Set("public", true)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify org was created
	count, err := db.Where("name = ?", "New Org").Count(&model.TOrg{})
	if err != nil {
		t.Fatalf("count orgs: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 org created, got %d", count)
	}
}

func TestOrgController_info_MissingId(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"id": ""}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_info_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Insert the admin user into t_user table (org owner lookup)
	if _, err := db.Insert(adminUser); err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	// Create org
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "admin", Name: "Test Org", Public: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-1"}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_users_MissingId(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"id": ""}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.users(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_users_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.users(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_save_MissingName(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-1", "name": ""}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("name", "")
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_save_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create org
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "admin", Name: "Old Name", Public: 0, Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Add admin as org admin
	userOrg := &model.TUserOrg{Uid: "admin", OrgId: "org-1", PermAdm: 1, Created: time.Now()}
	if _, err := db.Insert(userOrg); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-1", "name": "New Name", "desc": "Updated", "public": true}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("name", "New Name")
	m.Set("desc", "Updated")
	m.Set("public", true)
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify org was updated
	updated := &model.TOrg{}
	ok, err := db.Where("id = ?", "org-1").Get(updated)
	if err != nil {
		t.Fatalf("get org: %v", err)
	}
	if !ok {
		t.Fatal("org not found")
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", updated.Name)
	}
}

func TestOrgController_rm_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_rm_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create org
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "admin", Name: "Test Org", Public: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Add admin as org admin
	userOrg := &model.TUserOrg{Uid: "admin", OrgId: "org-1", PermAdm: 1, Created: time.Now()}
	if _, err := db.Insert(userOrg); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	c, w := makeOrgGinContext(t, hbtp.Map{"id": "org-1"}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org-1")
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify org was marked as deleted
	deleted := &model.TOrg{}
	ok, err := db.Where("id = ?", "org-1").Get(deleted)
	if err != nil {
		t.Fatalf("get org: %v", err)
	}
	if !ok {
		t.Fatal("org not found")
	}
	if deleted.Deleted != 1 {
		t.Errorf("expected deleted = 1, got %d", deleted.Deleted)
	}
}

func TestOrgController_vars_MissingOrgId(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"orgId": "", "q": "", "page": int64(1)}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	m.Set("q", "")
	m.Set("page", int64(1))
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_vars_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"orgId": "nonexistent", "q": "", "page": int64(1)}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	m.Set("q", "")
	m.Set("page", int64(1))
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_varSave_MissingFields(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, nil, adminUser)
	ctrl := OrgController{}

	// varSave takes *bean.OrgVar — empty Name should trigger param err
	pv := &bean.OrgVar{OrgId: "org-1", Name: "", Value: "test"}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varSave_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create org
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "admin", Name: "Test Org", Public: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	// Add admin as org admin
	userOrg := &model.TUserOrg{Uid: "admin", OrgId: "org-1", PermAdm: 1, PermRw: 1, Created: time.Now()}
	if _, err := db.Insert(userOrg); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	c, w := makeOrgGinContext(t, nil, adminUser)
	ctrl := OrgController{}
	pv := &bean.OrgVar{OrgId: "org-1", Name: "MY_VAR", Value: "my-value", Remarks: "test var", Public: true}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify variable was created
	var count int64
	count, err := db.Where("org_id = ? AND name = ?", "org-1", "MY_VAR").Count(&model.TOrgVar{})
	if err != nil {
		t.Fatalf("count vars: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 var, got %d", count)
	}
}

func TestOrgController_varDel_InvalidAid(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"aid": int64(0)}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varDel_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinContext(t, hbtp.Map{"aid": int64(999)}, adminUser)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(999))
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}
