package route

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func setupOrgTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Use plain BIGINT for aid (not PRIMARY KEY) so SQLite auto-increments via rowid.
	tables := []string{
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL,
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
			perm_user INT DEFAULT 0,
			perm_org INT DEFAULT 0,
			perm_pipe INT DEFAULT 0
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
		`CREATE TABLE t_org_var (
			aid BIGINT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			name VARCHAR(255),
			value TEXT,
			remarks VARCHAR(255),
			public INT DEFAULT 0
		)`,
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("failed to create table: %v\nSQL: %s", err, sql)
		}
	}

	comm.Db = db
}

// createOrgTestUser creates a user; if isAdmin is true, the Id is set to "admin"
// so that service.IsAdmin returns true.
func createOrgTestUser(t *testing.T, name, nick string, isAdmin bool) *model.TUser {
	t.Helper()
	id := utils.NewXid()
	if isAdmin {
		id = "admin"
	}
	user := &model.TUser{
		Id:        id,
		Aid:       time.Now().UnixNano() % 1000000,
		Name:      name,
		Nick:      nick,
		Pass:      utils.Md5String("testpass"),
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	if _, err := comm.Db.InsertOne(user); err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return user
}

func createTestOrg(t *testing.T, uid, name, desc string, public int) *model.TOrg {
	t.Helper()
	org := &model.TOrg{
		Id:      utils.NewXid(),
		Aid:     time.Now().UnixNano() % 1000000,
		Uid:     uid,
		Name:    name,
		Desc:    desc,
		Public:  public,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create test org: %v", err)
	}
	return org
}

func makeOrgGinCtx(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

// --- list ---

func TestOrgController_list_EmptyDB(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_WithOrgs(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	createTestOrg(t, admin.Id, "org1", "First org", 1)
	createTestOrg(t, admin.Id, "org2", "Second org", 0)

	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if total, ok := resp["total"].(float64); ok && total < 2 {
		t.Errorf("expected total >= 2, got %v", total)
	}
}

func TestOrgController_list_WithSearchQuery(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	createTestOrg(t, admin.Id, "searchme", "Searchable", 1)
	createTestOrg(t, admin.Id, "other", "Other org", 1)

	m := &hbtp.Map{}
	m.Set("q", "search")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_NonAdminFiltersByUser(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "normaluser", "Normal", false)

	// Create org owned by this user (public=0 so others can't see)
	createTestOrg(t, user.Id, "myorg", "My org", 0)
	// Create org owned by someone else
	createTestOrg(t, "someone-else", "otherorg", "Not mine", 0)

	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, user)

	ctrl := OrgController{}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Non-admin should only see their own org + public orgs
	if total, ok := resp["total"].(float64); ok && total > 1 {
		t.Errorf("non-admin expected total <= 1, got %v", total)
	}
}

func TestOrgController_list_PublicOrgsVisibleToAll(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "stranger", "Stranger", false)

	createTestOrg(t, "someone-else", "publicorg", "Public org", 1)

	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, user)

	ctrl := OrgController{}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if total, ok := resp["total"].(float64); ok && total < 1 {
		t.Errorf("expected total >= 1 (public org), got %v", total)
	}
}

// --- new ---

func TestOrgController_new_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("name", "")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_new_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("name", "New Org")
	m.Set("desc", "A new org")
	m.Set("public", true)
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if id, ok := resp["id"].(string); !ok || id == "" {
		t.Error("expected non-empty id in response")
	}

	// Verify org exists
	org := &model.TOrg{}
	ok, err := comm.Db.Where("name=?", "New Org").Get(org)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("org not created")
	}
	if org.Public != 1 {
		t.Errorf("public = %d, want 1", org.Public)
	}
}

func TestOrgController_new_NonAdminNoPermission(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "normal", "Normal", false)

	m := &hbtp.Map{}
	m.Set("name", "Denied Org")
	c, w := makeOrgGinCtx(t, m, user)

	ctrl := OrgController{}
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// --- info ---

func TestOrgController_info_EmptyID(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_info_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent-id")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	org := createTestOrg(t, admin.Id, "testorg", "Test org", 1)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["org"] == nil {
		t.Error("expected 'org' key in response")
	}
	if resp["perm"] == nil {
		t.Error("expected 'perm' key in response")
	}
}

func TestOrgController_info_DeletedOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	org := createTestOrg(t, admin.Id, "deleted-org", "Deleted", 1)
	org.Deleted = 1
	org.DeletedTime = time.Now()
	comm.Db.Cols("deleted", "deleted_time").Where("id=?", org.Id).Update(org)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d for deleted org", w.Code, http.StatusNotFound)
	}
}

// --- rm ---

func TestOrgController_rm_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "normal", "Normal", false)
	// Org doesn't exist
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeOrgGinCtx(t, m, user)

	ctrl := OrgController{}
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_rm_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "normal", "Normal", false)
	org := createTestOrg(t, "other", "Org", "Not mine", 0)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, user)

	ctrl := OrgController{}
	ctrl.rm(c, m)

	// Non-admin, non-owner → either NotFound (no org access) or MethodNotAllowed
	if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d or %d", w.Code, http.StatusNotFound, http.StatusMethodNotAllowed)
	}
}

func TestOrgController_rm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	org := createTestOrg(t, admin.Id, "todelete", "Delete me", 1)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify soft delete
	updated := &model.TOrg{}
	ok, err := comm.Db.Where("id=?", org.Id).Get(updated)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("org not found")
	}
	if updated.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", updated.Deleted)
	}
}

// --- users ---

func TestOrgController_users_EmptyID(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.users(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_users_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "normal", "Normal", false)
	org := createTestOrg(t, "other", "Private Org", "Desc", 0)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, user)

	ctrl := OrgController{}
	ctrl.users(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOrgController_users_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	org := createTestOrg(t, admin.Id, "testorg", "Test", 1)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.users(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- save ---

func TestOrgController_save_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("name", "")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_save_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("name", "Updated")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.save(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_save_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	org := createTestOrg(t, admin.Id, "oldname", "old desc", 0)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("name", "newname")
	m.Set("desc", "new desc")
	m.Set("public", true)
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify update
	updated := &model.TOrg{}
	ok, err := comm.Db.Where("id=?", org.Id).Get(updated)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("org not found")
	}
	if updated.Name != "newname" {
		t.Errorf("name = %q, want %q", updated.Name, "newname")
	}
	if updated.Public != 1 {
		t.Errorf("public = %d, want 1", updated.Public)
	}
}

// --- pipeAdd ---

func TestOrgController_pipeAdd_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "normal", "Normal", false)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pipeId", "some-pipe-id")
	c, w := makeOrgGinCtx(t, m, user)

	ctrl := OrgController{}
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- pipeRm ---

func TestOrgController_pipeRm_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "normal", "Normal", false)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pipeId", "some-pipe-id")
	c, w := makeOrgGinCtx(t, m, user)

	ctrl := OrgController{}
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- vars ---

func TestOrgController_vars_EmptyOrgID(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("orgId", "")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_vars_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- varDel ---

func TestOrgController_varDel_InvalidAid(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varDel_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("aid", int64(99999))
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- userEdit ---

func TestOrgController_userEdit_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", "some-uid")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_userEdit_SelfEdit(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	// Create user_org record for admin
	userOrg := &model.TUserOrg{
		Uid:      admin.Id,
		OrgId:    org.Id,
		Created:  time.Now(),
		PermAdm:  1,
		PermRw:   1,
		PermExec: 1,
	}
	if _, err := comm.Db.InsertOne(userOrg); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", admin.Id)
	m.Set("adm", true)
	m.Set("add", true)
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.userEdit(c, m)

	// Should be blocked with "can't edit yourself"
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrgController_userEdit_UserNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)
	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", "nonexistent-user-id")
	m.Set("adm", false)
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_userEdit_AddNewMember(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)
	newUser := createOrgTestUser(t, "newbie", "Newbie", false)

	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	// When add=true, only adm is set; rw/ex/dw are ignored
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", newUser.Id)
	m.Set("adm", false)
	m.Set("rw", true)
	m.Set("ex", true)
	m.Set("dw", false)
	m.Set("add", true)
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.userEdit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify user_org was created
	uo := &model.TUserOrg{}
	ok, err := comm.Db.Where("uid=? and org_id=?", newUser.Id, org.Id).Get(uo)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("user_org not created")
	}
	// When add=true, only adm is set; rw/ex/dw remain at default 0
	if uo.PermAdm != 0 {
		t.Errorf("PermAdm = %d, want 0", uo.PermAdm)
	}
}

// --- userRm ---

func TestOrgController_userRm_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", "some-uid")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.userRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_userRm_SelfRemoval(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	// Add admin to user_org
	userOrg := &model.TUserOrg{
		Uid:      admin.Id,
		OrgId:    org.Id,
		Created:  time.Now(),
		PermAdm:  1,
		PermRw:   1,
		PermExec: 1,
	}
	if _, err := comm.Db.InsertOne(userOrg); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", admin.Id)
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.userRm(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrgController_userRm_UserNotInOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)
	otherUser := createOrgTestUser(t, "other", "Other", false)

	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", otherUser.Id) // not in org
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.userRm(c, m)

	// User is not in the org
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestOrgController_userRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)
	member := createOrgTestUser(t, "member", "Member", false)

	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	// Add member to user_org with pre-assigned aid (required for xorm composite PK)
	userOrg := &model.TUserOrg{
		Aid:      time.Now().UnixNano() % 1000000,
		Uid:      member.Id,
		OrgId:    org.Id,
		Created:  time.Now(),
		PermAdm:  0,
		PermRw:   1,
		PermExec: 1,
	}
	if _, err := comm.Db.InsertOne(userOrg); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", member.Id)
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.userRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify user_org was deleted
	uo := &model.TUserOrg{}
	ok, err := comm.Db.Where("uid=? and org_id=?", member.Id, org.Id).Get(uo)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if ok {
		t.Error("user_org should have been deleted")
	}
}

// --- pipeAdd success ---

func TestOrgController_pipeAdd_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "pipe-123")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify org_pipe was created
	op := &model.TOrgPipe{}
	ok, err := comm.Db.Where("org_id=? and pipe_id=?", org.Id, "pipe-123").Get(op)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("org_pipe not created")
	}
}

func TestOrgController_pipeAdd_DuplicatePipeline(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	// Insert existing pipe
	existing := &model.TOrgPipe{
		OrgId:   org.Id,
		PipeId:  "pipe-456",
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(existing); err != nil {
		t.Fatalf("insert: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "pipe-456")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

// --- pipeRm success ---

func TestOrgController_pipeRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	existing := &model.TOrgPipe{
		OrgId:   org.Id,
		PipeId:  "pipe-789",
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(existing); err != nil {
		t.Fatalf("insert: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "pipe-789")
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- varSave ---

func TestOrgController_varSave_EmptyParams(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	pv := &bean.OrgVar{
		Name:  "",
		Value: "val",
		OrgId: "some-org",
	}
	c, w := makeOrgGinCtx(t, pv, admin)

	ctrl := OrgController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varSave_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)

	pv := &bean.OrgVar{
		Name:  "varname",
		Value: "varvalue",
		OrgId: "nonexistent",
	}
	c, w := makeOrgGinCtx(t, pv, admin)

	ctrl := OrgController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_varSave_CreateNew(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)
	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	pv := &bean.OrgVar{
		Name:   "MY_VAR",
		Value:  "my_value",
		OrgId:  org.Id,
		Public: false,
		Aid:    0,
	}
	c, w := makeOrgGinCtx(t, pv, admin)

	ctrl := OrgController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify var was created
	v := &model.TOrgVar{}
	ok, err := comm.Db.Where("org_id=? and name=?", org.Id, "MY_VAR").Get(v)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("org_var not created")
	}
}

func TestOrgController_varSave_DuplicateName(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)
	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	// Insert existing var
	existing := &model.TOrgVar{
		Aid:   time.Now().UnixNano() % 1000000,
		OrgId: org.Id,
		Name:  "EXISTING_VAR",
		Value: "val",
	}
	if _, err := comm.Db.InsertOne(existing); err != nil {
		t.Fatalf("insert: %v", err)
	}

	pv := &bean.OrgVar{
		Name:  "EXISTING_VAR",
		Value: "new_value",
		OrgId: org.Id,
		Aid:   0, // new insert
	}
	c, w := makeOrgGinCtx(t, pv, admin)

	ctrl := OrgController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrgController_varSave_UpdateExisting(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)
	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	// Manually insert a var with a known aid value
	existing := &model.TOrgVar{
		Aid:     100,
		OrgId:   org.Id,
		Name:    "UPD_VAR",
		Value:   "old_value",
		Public:  0,
	}
	if _, err := comm.Db.InsertOne(existing); err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Now update it
	pv := &bean.OrgVar{
		Name:  "UPD_VAR",
		Value: "new_value",
		OrgId: org.Id,
		Aid:   100,
	}
	c, w := makeOrgGinCtx(t, pv, admin)
	ctrl := OrgController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Errorf("update: status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify update
	updated := &model.TOrgVar{}
	ok, err := comm.Db.Where("aid=?", 100).Get(updated)
	if err != nil {
		t.Fatalf("query updated: %v", err)
	}
	if !ok {
		t.Fatal("var not found after update")
	}
	if updated.Value != "new_value" {
		t.Errorf("value = %q, want %q", updated.Value, "new_value")
	}
}

// --- vars with data ---

func TestOrgController_vars_WithData(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)
	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	// Insert some vars
	for i := 0; i < 3; i++ {
		v := &model.TOrgVar{
			Aid:    int64(1000 + i),
			OrgId:  org.Id,
			Name:   fmt.Sprintf("VAR_%d", i),
			Value:  fmt.Sprintf("val_%d", i),
			Public: 0,
		}
		if _, err := comm.Db.InsertOne(v); err != nil {
			t.Fatalf("insert var: %v", err)
		}
	}

	m := &hbtp.Map{}
	m.Set("orgId", org.Id)
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_vars_WithSearchQuery(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)
	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	v := &model.TOrgVar{
		Aid:   2000,
		OrgId: org.Id,
		Name:  "SEARCHABLE",
		Value: "findme",
	}
	if _, err := comm.Db.InsertOne(v); err != nil {
		t.Fatalf("insert: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("orgId", org.Id)
	m.Set("q", "SEARCH")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- varDel success ---

func TestOrgController_varDel_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", true)
	org := createTestOrg(t, admin.Id, "myorg", "desc", 1)

	v := &model.TOrgVar{
		Aid:   3000,
		OrgId: org.Id,
		Name:  "TO_DELETE",
		Value: "val",
	}
	if _, err := comm.Db.InsertOne(v); err != nil {
		t.Fatalf("insert: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("aid", int64(3000))
	c, w := makeOrgGinCtx(t, m, admin)

	ctrl := OrgController{}
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
