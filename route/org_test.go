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
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/util"
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

	// Create required tables
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
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)`,
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	comm.Db = db
}

func createOrgTestUser(t *testing.T, name string, isAdmin bool) *model.TUser {
	t.Helper()
	userId := utils.NewXid()
	if isAdmin {
		userId = "admin"
	}
	user := &model.TUser{
		Id:        userId,
		Name:      name,
		Nick:      name + " Nick",
		Pass:      utils.Md5String("password"),
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := comm.Db.InsertOne(user)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestOrg(t *testing.T, name, uid string, public int) *model.TOrg {
	t.Helper()
	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     uid,
		Name:    name,
		Desc:    name + " description",
		Public:  public,
		Created: time.Now(),
		Updated: time.Now(),
		Deleted: 0,
	}
	_, err := comm.Db.InsertOne(org)
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}
	return org
}

func addUserToOrg(t *testing.T, uid, orgId string, permAdm, permRw, permExec int) {
	t.Helper()
	userOrg := &model.TUserOrg{
		Uid:      uid,
		OrgId:    orgId,
		Created:  time.Now(),
		PermAdm:  permAdm,
		PermRw:   permRw,
		PermExec: permExec,
	}
	_, err := comm.Db.InsertOne(userOrg)
	if err != nil {
		t.Fatalf("failed to add user to org: %v", err)
	}
}

func makeOrgRequest(t *testing.T, body interface{}, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

	// Generate token for user
	token, _ := util.CreateToken(
		map[string]interface{}{"uid": user.Id},
		"test-key",
		time.Hour*24,
	)
	req.Header.Set("Authorization", "TOKEN "+token)

	c.Request = req
	// Set logged-in user in context
	c.Set("lguser", user)
	return c, w
}

func TestOrgController_GetPathRoute(t *testing.T) {
	ctrl := OrgController{}
	if got := ctrl.GetPath(); got != "/api/org" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/org")
	}
}

func TestOrgController_new_Success(t *testing.T) {
	setupOrgTestDB(t)

	// Set up config
	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	user := createOrgTestUser(t, "testuser", false)

	// Give user org creation permission
	userInfo := &model.TUserInfo{
		Id:       user.Id,
		PermOrg:  1,
	}
	_, err := comm.Db.InsertOne(userInfo)
	if err != nil {
		t.Fatalf("failed to create user info: %v", err)
	}

	body := &hbtp.Map{
		"name":   "Test Org",
		"desc":   "Test organization",
		"public": true,
	}

	c, w := makeOrgRequest(t, body, user)
	ctrl.new(c, body)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if _, ok := resp["id"]; !ok {
		t.Error("expected id in response")
	}
}

func TestOrgController_new_EmptyName(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	user := createOrgTestUser(t, "testuser", false)

	body := &hbtp.Map{
		"name": "",
		"desc": "Test organization",
	}

	c, w := makeOrgRequest(t, body, user)
	ctrl.new(c, body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_new_NoPermission(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	user := createOrgTestUser(t, "testuser", false)
	// Don't give user org permission

	body := &hbtp.Map{
		"name": "Test Org",
		"desc": "Test organization",
	}

	c, w := makeOrgRequest(t, body, user)
	ctrl.new(c, body)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOrgController_list_Success(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	user := createOrgTestUser(t, "testuser", true) // admin user

	// Create some orgs
	createTestOrg(t, "Org 1", user.Id, 1)
	createTestOrg(t, "Org 2", user.Id, 0)

	body := &hbtp.Map{
		"page": int64(1),
		"size": int64(10),
	}

	c, w := makeOrgRequest(t, body, user)
	ctrl.list(c, body)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if _, ok := resp["list"]; !ok {
		if _, ok2 := resp["data"]; !ok2 {
			if _, ok3 := resp["total"]; !ok3 {
				t.Logf("response keys: %v", resp)
			}
		}
	}
}

func TestOrgController_list_WithSearch(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	user := createOrgTestUser(t, "testuser", true)

	createTestOrg(t, "Alpha Org", user.Id, 1)
	createTestOrg(t, "Beta Org", user.Id, 1)

	body := &hbtp.Map{
		"page":    int64(1),
		"size":    int64(10),
		"keyword": "Alpha",
	}

	c, w := makeOrgRequest(t, body, user)
	ctrl.list(c, body)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOrgController_info_Success(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	user := createOrgTestUser(t, "testuser", false)
	org := createTestOrg(t, "Test Org", user.Id, 1)

	body := &hbtp.Map{
		"id": org.Id,
	}

	c, w := makeOrgRequest(t, body, user)
	ctrl.info(c, body)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if respOrg, ok := resp["org"].(map[string]interface{}); ok {
		if respOrg["name"] != "Test Org" {
			t.Errorf("org name = %v, want 'Test Org'", respOrg["name"])
		}
	} else {
		t.Error("expected org in response")
	}
}

func TestOrgController_info_NotFound(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	user := createOrgTestUser(t, "testuser", false)

	body := &hbtp.Map{
		"id": "nonexistent",
	}

	c, w := makeOrgRequest(t, body, user)
	ctrl.info(c, body)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_NoPermission(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	owner := createOrgTestUser(t, "owner", false)
	otherUser := createOrgTestUser(t, "other", false)
	org := createTestOrg(t, "Private Org", owner.Id, 0) // not public

	body := &hbtp.Map{
		"id": org.Id,
	}

	c, w := makeOrgRequest(t, body, otherUser)
	ctrl.info(c, body)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOrgController_save_Success(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	user := createOrgTestUser(t, "testuser", false)
	org := createTestOrg(t, "Test Org", user.Id, 0)

	body := &hbtp.Map{
		"id":     org.Id,
		"name":   "Updated Org",
		"desc":   "Updated description",
		"public": true,
	}

	c, w := makeOrgRequest(t, body, user)
	ctrl.save(c, body)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify update
	updatedOrg := &model.TOrg{}
	_, err := comm.Db.Where("id = ?", org.Id).Get(updatedOrg)
	if err != nil {
		t.Fatalf("failed to query updated org: %v", err)
	}
	if updatedOrg.Name != "Updated Org" {
		t.Errorf("org name = %q, want 'Updated Org'", updatedOrg.Name)
	}
	if updatedOrg.Public != 1 {
		t.Errorf("org public = %d, want 1", updatedOrg.Public)
	}
}

func TestOrgController_save_NoPermission(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	owner := createOrgTestUser(t, "owner", false)
	otherUser := createOrgTestUser(t, "other", false)
	org := createTestOrg(t, "Test Org", owner.Id, 0)

	body := &hbtp.Map{
		"id":   org.Id,
		"name": "Hacked Org",
	}

	c, w := makeOrgRequest(t, body, otherUser)
	ctrl.save(c, body)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOrgController_rm_Success(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	user := createOrgTestUser(t, "testuser", false)
	org := createTestOrg(t, "Test Org", user.Id, 0)

	body := &hbtp.Map{
		"id": org.Id,
	}

	c, w := makeOrgRequest(t, body, user)
	ctrl.rm(c, body)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify soft delete
	deletedOrg := &model.TOrg{}
	_, err := comm.Db.Where("id = ?", org.Id).Get(deletedOrg)
	if err != nil {
		t.Fatalf("failed to query deleted org: %v", err)
	}
	if deletedOrg.Deleted != 1 {
		t.Errorf("org deleted = %d, want 1", deletedOrg.Deleted)
	}
}

func TestOrgController_rm_NotFound(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	user := createOrgTestUser(t, "testuser", false)

	body := &hbtp.Map{
		"id": "nonexistent",
	}

	c, w := makeOrgRequest(t, body, user)
	ctrl.rm(c, body)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_users_Success(t *testing.T) {
	setupOrgTestDB(t)

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key"
	t.Cleanup(func() { comm.Cfg = origCfg })

	ctrl := OrgController{}
	user := createOrgTestUser(t, "testuser", false)
	org := createTestOrg(t, "Test Org", user.Id, 1)

	// Add user to org
	addUserToOrg(t, user.Id, org.Id, 1, 1, 1)

	body := &hbtp.Map{
		"id": org.Id,
	}

	c, w := makeOrgRequest(t, body, user)
	ctrl.users(c, body)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if _, ok := resp["usrs"]; !ok {
		t.Error("expected usrs in response")
	}
}
