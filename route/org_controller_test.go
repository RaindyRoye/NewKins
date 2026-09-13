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

// setupOrgTestDb creates an in-memory SQLite database with all tables
// needed for org controller tests.
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

	tables := []string{
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid INTEGER,
			uid VARCHAR(64),
			name VARCHAR(200),
			"desc" TEXT,
			public INT DEFAULT 0,
			created DATETIME,
			updated DATETIME,
			deleted INT DEFAULT 0,
			deleted_time DATETIME
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
		`CREATE TABLE t_org_pipe (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_var (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
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
			create_time DATETIME,
			created DATETIME
		)`,
	}
	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("exec table DDL: %v\nSQL: %s", err, sql)
		}
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

// --- list endpoint tests ---

func TestOrgList_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	
	// Create some orgs
	orgs := []*model.TOrg{
		{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Alpha Org", Public: 1, Created: time.Now(), Updated: time.Now()},
		{Id: "org-2", Aid: 2, Uid: "user-2", Name: "Beta Org", Public: 0, Created: time.Now(), Updated: time.Now()},
	}
	for _, org := range orgs {
		if _, err := db.Insert(org); err != nil {
			t.Fatalf("insert org: %v", err)
		}
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "admin-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("page", int64(1))
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	// Response contains "data" not "list"
	if resp["data"] == nil && resp["list"] == nil {
		t.Error("expected data or list in response")
	}
}

func TestOrgList_WithSearch(t *testing.T) {
	db := setupOrgTestDb(t)
	
	orgs := []*model.TOrg{
		{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Alpha Org", Public: 1, Created: time.Now(), Updated: time.Now()},
		{Id: "org-2", Aid: 2, Uid: "user-2", Name: "Beta Org", Public: 1, Created: time.Now(), Updated: time.Now()},
	}
	for _, org := range orgs {
		if _, err := db.Insert(org); err != nil {
			t.Fatalf("insert org: %v", err)
		}
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "admin-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("q", "Alpha")
	m.Set("page", int64(1))
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgList_NonAdmin(t *testing.T) {
	db := setupOrgTestDb(t)
	
	// Create public org visible to non-admin
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Public Org", Public: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	ctrl := OrgController{}
	user := &model.TUser{Id: "user-2", Name: "regular", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("page", int64(1))
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

// --- new endpoint tests ---

func TestOrgNew_MissingName(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("name", "")
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgNew_NoPermission(t *testing.T) {
	db := setupOrgTestDb(t)
	
	// User without PermOrg
	uinfo := &model.TUserInfo{Id: "user-1", PermOrg: 0}
	if _, err := db.Insert(uinfo); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "regular", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("name", "New Org")
	m.Set("desc", "Description")
	m.Set("public", true)
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgNew_Success_Admin(t *testing.T) {
	db := setupOrgTestDb(t)
	ctrl := OrgController{}
	// IsAdmin checks usr.Id == "admin" (literal), so Id must be "admin"
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("name", "New Org")
	m.Set("desc", "A new organization")
	m.Set("public", true)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify org was created
	count, _ := db.Where("name=?", "New Org").Count(&model.TOrg{})
	if count != 1 {
		t.Errorf("expected 1 org, got %d", count)
	}
}

func TestOrgNew_Success_WithPermission(t *testing.T) {
	db := setupOrgTestDb(t)
	
	// User with PermOrg
	uinfo := &model.TUserInfo{Id: "user-1", PermOrg: 1}
	if _, err := db.Insert(uinfo); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "regular", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("name", "User Org")
	m.Set("desc", "User created org")
	m.Set("public", false)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

// --- info endpoint tests ---

func TestOrgInfo_MissingId(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgInfo_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgInfo_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Public: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Insert owner user
	owner := &model.TUser{Id: "user-1", Name: "owner", Nick: "Owner User", Active: 1}
	if _, err := db.Insert(owner); err != nil {
		t.Fatalf("insert owner: %v", err)
	}

	ctrl := OrgController{}
	user := &model.TUser{Id: "user-2", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
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

// --- users endpoint tests ---

func TestOrgUsers_MissingId(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.users(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgUsers_OrgNotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.users(c, m)

	// Non-existent org means perm can't be resolved, so CanRead() returns false -> 405
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestOrgUsers_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Public: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Add users to org
	users := []*model.TUser{
		{Id: "user-1", Name: "admin", Nick: "Admin User", Active: 1},
		{Id: "user-2", Name: "member", Nick: "Member User", Active: 1},
	}
	for _, u := range users {
		if _, err := db.Insert(u); err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}

	userOrgs := []*model.TUserOrg{
		{Uid: "user-1", OrgId: "org-1", PermAdm: 1, Created: time.Now()},
		{Uid: "user-2", OrgId: "org-1", PermAdm: 0, Created: time.Now()},
	}
	for _, uo := range userOrgs {
		if _, err := db.Insert(uo); err != nil {
			t.Fatalf("insert user org: %v", err)
		}
	}

	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	ctrl.users(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["adms"] == nil {
		t.Error("expected adms in response")
	}
	if resp["usrs"] == nil {
		t.Error("expected usrs in response")
	}
}

// --- save endpoint tests ---

func TestOrgSave_MissingName(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("name", "")
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgSave_OrgNotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("name", "Updated Name")
	ctrl.save(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgSave_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Old Name", Public: 0, Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("name", "Updated Name")
	m.Set("desc", "Updated description")
	m.Set("public", true)
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify update
	updated := &model.TOrg{}
	ok, _ := db.Where("id=?", "org-1").Get(updated)
	if !ok {
		t.Fatal("org not found")
	}
	if updated.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %q", updated.Name)
	}
	if updated.Public != 1 {
		t.Errorf("expected public=1, got %d", updated.Public)
	}
}

// --- rm endpoint tests ---

func TestOrgRm_OrgNotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgRm_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify soft-deleted
	updated := &model.TOrg{}
	ok, _ := db.Where("id=?", "org-1").Get(updated)
	if !ok {
		t.Fatal("org should still exist (soft delete)")
	}
	if updated.Deleted != 1 {
		t.Errorf("expected deleted=1, got %d", updated.Deleted)
	}
}

// --- userEdit endpoint tests ---

func TestOrgUserEdit_MissingFields(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	m.Set("uid", "")
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgUserEdit_OrgNotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", "user-2")
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgUserEdit_UserNotFound(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("uid", "nonexistent")
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgUserEdit_CantEditSelf(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	if _, err := db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	ctrl := OrgController{}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("uid", "user-1")
	m.Set("adm", true)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for self-edit, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgUserEdit_AddNewMember(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	users := []*model.TUser{
		{Id: "user-1", Name: "admin", Active: 1},
		{Id: "user-2", Name: "newmember", Active: 1},
	}
	for _, u := range users {
		if _, err := db.Insert(u); err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("uid", "user-2")
	m.Set("adm", false)
	m.Set("rw", true)
	m.Set("ex", true)
	m.Set("dw", false)
	m.Set("add", true)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify membership created
	count, _ := db.Where("uid=? AND org_id=?", "user-2", "org-1").Count(&model.TUserOrg{})
	if count != 1 {
		t.Errorf("expected 1 membership, got %d", count)
	}
}

// --- userRm endpoint tests ---

func TestOrgUserRm_OrgNotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", "user-2")
	ctrl.userRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgUserRm_UserNotFound(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("uid", "nonexistent")
	ctrl.userRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgUserRm_CantRemoveSelf(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	if _, err := db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	uo := &model.TUserOrg{Uid: "user-1", OrgId: "org-1", PermAdm: 1, Created: time.Now()}
	if _, err := db.Insert(uo); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	ctrl := OrgController{}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("uid", "user-1")
	ctrl.userRm(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for self-removal, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgUserRm_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	users := []*model.TUser{
		{Id: "user-1", Name: "admin", Active: 1},
		{Id: "user-2", Name: "member", Active: 1},
	}
	for _, u := range users {
		if _, err := db.Insert(u); err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}

	uo := &model.TUserOrg{Uid: "user-2", OrgId: "org-1", Created: time.Now()}
	if _, err := db.Insert(uo); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("uid", "user-2")
	ctrl.userRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify membership deleted
	count, _ := db.Where("uid=? AND org_id=?", "user-2", "org-1").Count(&model.TUserOrg{})
	if count != 0 {
		t.Errorf("expected 0 memberships, got %d", count)
	}
}

// --- pipeAdd endpoint tests ---

func TestOrgPipeAdd_OrgNotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pipeId", "pipe-1")
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgPipeAdd_PipelineAlreadyAdded(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	pipe := &model.TPipeline{Id: "pipe-1", Uid: "user-1", Name: "Test Pipe"}
	if _, err := db.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	op := &model.TOrgPipe{OrgId: "org-1", PipeId: "pipe-1", Created: time.Now()}
	if _, err := db.Insert(op); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("pipeId", "pipe-1")
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgPipeAdd_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	pipe := &model.TPipeline{Id: "pipe-1", Uid: "user-1", Name: "Test Pipe"}
	if _, err := db.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("pipeId", "pipe-1")
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify org pipe created
	count, _ := db.Where("org_id=? AND pipe_id=?", "org-1", "pipe-1").Count(&model.TOrgPipe{})
	if count != 1 {
		t.Errorf("expected 1 org pipe, got %d", count)
	}
}

// --- pipeRm endpoint tests ---

func TestOrgPipeRm_OrgNotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pipeId", "pipe-1")
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgPipeRm_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	op := &model.TOrgPipe{OrgId: "org-1", PipeId: "pipe-1", Created: time.Now()}
	if _, err := db.Insert(op); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("id", "org-1")
	m.Set("pipeId", "pipe-1")
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	count, _ := db.Where("org_id=? AND pipe_id=?", "org-1", "pipe-1").Count(&model.TOrgPipe{})
	if count != 0 {
		t.Errorf("expected 0 org pipes, got %d", count)
	}
}

// --- vars endpoint tests ---

func TestOrgVars_MissingOrgId(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "")
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgVars_OrgNotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgVars_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Public: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Add a var
	ov := &model.TOrgVar{OrgId: "org-1", Name: "VAR1", Value: "value1", Public: 1}
	if _, err := db.Insert(ov); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("page", int64(1))
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

// --- varSave endpoint tests ---

func TestOrgVarSave_MissingFields(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	ov := &bean.OrgVar{OrgId: "", Name: "", Value: ""}
	ctrl.varSave(c, ov)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgVarSave_OrgNotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	ov := &bean.OrgVar{OrgId: "nonexistent", Name: "VAR1", Value: "val1"}
	ctrl.varSave(c, ov)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgVarSave_CreateNew(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	ov := &bean.OrgVar{
		OrgId: "org-1", Name: "NEW_VAR", Value: "new-value", Public: true,
	}
	ctrl.varSave(c, ov)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify var created
	count, _ := db.Where("org_id=? AND name=?", "org-1", "NEW_VAR").Count(&model.TOrgVar{})
	if count != 1 {
		t.Errorf("expected 1 var, got %d", count)
	}
}

func TestOrgVarSave_DuplicateName(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	ov := &model.TOrgVar{OrgId: "org-1", Name: "EXISTING", Value: "old"}
	if _, err := db.Insert(ov); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	bov := &bean.OrgVar{OrgId: "org-1", Name: "EXISTING", Value: "new"}
	ctrl.varSave(c, bov)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate, got %d, body: %s", w.Code, w.Body.String())
	}
}

// --- varDel endpoint tests ---

func TestOrgVarDel_InvalidAid(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgVarDel_NotFound(t *testing.T) {
	setupOrgTestDb(t)
	ctrl := OrgController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeOrgGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("aid", int64(999))
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgVarDel_Success(t *testing.T) {
	db := setupOrgTestDb(t)
	
	org := &model.TOrg{Id: "org-1", Aid: 1, Uid: "user-1", Name: "Test Org", Created: time.Now(), Updated: time.Now()}
	if _, err := db.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	ov := &model.TOrgVar{OrgId: "org-1", Name: "DEL_VAR", Value: "val"}
	if _, err := db.Insert(ov); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	stored := &model.TOrgVar{}
	ok, _ := db.Where("org_id=? AND name=?", "org-1", "DEL_VAR").Get(stored)
	if !ok {
		t.Fatal("org var not found")
	}

	ctrl := OrgController{}
	admin := &model.TUser{Id: "user-1", Name: "admin", Active: 1}
	c, w := makeOrgGinCtx(t, admin)

	m := &hbtp.Map{}
	m.Set("aid", stored.Aid)
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	count, _ := db.Where("aid=?", stored.Aid).Count(&model.TOrgVar{})
	if count != 0 {
		t.Errorf("expected 0 vars, got %d", count)
	}
}
