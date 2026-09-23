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

func setupOrgTestDB(t *testing.T) {
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
			t.Fatalf("exec %q: %v", sql[:40], err)
		}
	}

	comm.Db = db
}

func makeOrgGinCtx(t *testing.T, body interface{}, lgUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

func createOrgTestUser(t *testing.T, name, nick string, _ int) *model.TUser {
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

func createOrgTestOrg(t *testing.T, owner *model.TUser, name string, public bool) *model.TOrg {
	t.Helper()
	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     owner.Id,
		Name:    name,
		Public:  b2i(public),
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}
	return org
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ===== list =====

func TestOrgController_list_Admin(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	createOrgTestOrg(t, admin, "org1", true)
	createOrgTestOrg(t, admin, "org2", false)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.list(c, m)

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

func TestOrgController_list_NonAdmin_PublicOnly(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	user := createOrgTestUser(t, "regular", "Regular", 1)
	createOrgTestOrg(t, admin, "public-org", true)
	createOrgTestOrg(t, admin, "private-org", false)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, user)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Non-admin should only see public orgs
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1 (public only)", resp.Total)
	}
}

func TestOrgController_list_WithSearch(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	createOrgTestOrg(t, admin, "alpha-team", true)
	createOrgTestOrg(t, admin, "beta-team", true)
	createOrgTestOrg(t, admin, "gamma", true)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "team")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2 (alpha-team + beta-team)", resp.Total)
	}
}

// ===== new =====

func TestOrgController_new_Admin(t *testing.T) {
	setupOrgTestDB(t)
	// IsAdmin checks usr.Id == "admin", so set Id explicitly
	admin := &model.TUser{
		Id:      "admin",
		Name:    "admin",
		Nick:    "Admin",
		Active:  1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "test-org")
	m.Set("desc", "A test organization")
	m.Set("public", true)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.IdsRes
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Id == "" {
		t.Error("expected non-empty org ID")
	}
}

func TestOrgController_new_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "user1", "User", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "")
	c, w := makeOrgGinCtx(t, m, user)
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_new_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "noperm", "NoPerm", 1)
	// Create user_info with perm_org=0
	ui := &model.TUserInfo{
		Id:      user.Id,
		PermOrg: 0,
	}
	if _, err := comm.Db.InsertOne(ui); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "fail-org")
	m.Set("desc", "Should fail")
	m.Set("public", false)
	c, w := makeOrgGinCtx(t, m, user)
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestOrgController_new_WithPermission(t *testing.T) {
	setupOrgTestDB(t)
	user := createOrgTestUser(t, "hasperm", "HasPerm", 1)
	ui := &model.TUserInfo{
		Id:      user.Id,
		PermOrg: 1,
	}
	if _, err := comm.Db.InsertOne(ui); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "perm-org")
	m.Set("desc", "Created by regular user")
	m.Set("public", true)
	c, w := makeOrgGinCtx(t, m, user)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

// ===== info =====

func TestOrgController_info_Found(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "my-org", true)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	orgResp, ok := resp["org"].(map[string]interface{})
	if !ok {
		t.Fatal("expected org object in response")
	}
	if orgResp["name"] != "my-org" {
		t.Errorf("org name = %v, want my-org", orgResp["name"])
	}
}

func TestOrgController_info_EmptyId(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_info_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_DeletedOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "deleted-org", true)
	// Soft delete
	_, err := comm.Db.Where("id=?", org.Id).Cols("deleted").Update(&model.TOrg{Deleted: 1})
	if err != nil {
		t.Fatalf("soft delete org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d for deleted org", w.Code, http.StatusNotFound)
	}
}

// ===== users =====

func TestOrgController_users_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "org-with-users", true)

	// Add a member
	uo := &model.TUserOrg{
		Uid:     admin.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(uo); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.users(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["adms"] == nil {
		t.Error("expected adms array in response")
	}
}

func TestOrgController_users_EmptyId(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.users(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_users_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "private-org", false) // not public
	other := createOrgTestUser(t, "outsider", "Outsider", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, other)
	ctrl.users(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// ===== save =====

func TestOrgController_save_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "save-org", false)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("name", "updated-name")
	m.Set("desc", "updated description")
	m.Set("public", true)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify
	saved := &model.TOrg{}
	ok, _ := comm.Db.Where("id=?", org.Id).Get(saved)
	if !ok {
		t.Fatal("org not found after save")
	}
	if saved.Name != "updated-name" {
		t.Errorf("name = %q, want updated-name", saved.Name)
	}
}

func TestOrgController_save_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "save-org", false)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("name", "")
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_save_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("name", "whatever")
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.save(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_save_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "restricted-org", false)
	other := createOrgTestUser(t, "other", "Other", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("name", "hacked-name")
	c, w := makeOrgGinCtx(t, m, other)
	ctrl.save(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// ===== rm =====

func TestOrgController_rm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "rm-org", true)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	// Verify soft-deleted
	saved := &model.TOrg{}
	ok, _ := comm.Db.Where("id=?", org.Id).Get(saved)
	if !ok {
		t.Fatal("org not found")
	}
	if saved.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", saved.Deleted)
	}
}

func TestOrgController_rm_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_rm_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "protected-org", false)
	other := createOrgTestUser(t, "other", "Other", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, other)
	ctrl.rm(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// ===== userEdit =====

func TestOrgController_userEdit_AddMember(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "edit-org", true)
	member := createOrgTestUser(t, "member", "Member", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", member.Id)
	m.Set("add", false) // not an add-only operation; set full permissions
	m.Set("adm", false)
	m.Set("rw", true)
	m.Set("ex", false)
	m.Set("dw", false)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify membership
	uo := &model.TUserOrg{}
	ok, _ := comm.Db.Where("uid=? and org_id=?", member.Id, org.Id).Get(uo)
	if !ok {
		t.Error("user org membership not found")
	}
	if uo.PermRw != 1 {
		t.Errorf("perm_rw = %d, want 1", uo.PermRw)
	}
}

func TestOrgController_userEdit_SelfEdit(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "self-edit-org", true)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", admin.Id) // editing self
	m.Set("add", false)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (can't edit yourself)", w.Code, http.StatusConflict)
	}
}

func TestOrgController_userEdit_UserNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "edit-org", true)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", "nonexistent-user")
	m.Set("add", false)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_userEdit_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-org")
	m.Set("uid", admin.Id)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ===== userRm =====

func TestOrgController_userRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "rm-user-org", true)
	member := createOrgTestUser(t, "member", "Member", 1)

	// Add member first
	uo := &model.TUserOrg{
		Uid:     member.Id,
		OrgId:   org.Id,
		PermAdm: 0,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(uo); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", member.Id)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.userRm(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestOrgController_userRm_SelfRemove(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "self-rm-org", true)

	// Add admin to the org first
	uo := &model.TUserOrg{
		Uid:     admin.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(uo); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", admin.Id)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.userRm(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (can't remove yourself)", w.Code, http.StatusConflict)
	}
}

func TestOrgController_userRm_NotInOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "rm-org", true)
	outside := createOrgTestUser(t, "outside", "Outside", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", outside.Id)
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.userRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (not a member)", w.Code, http.StatusNotFound)
	}
}

// ===== pipeAdd =====

func TestOrgController_pipeAdd_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "pipe-org", true)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "pipe-123")
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify
	op := &model.TOrgPipe{}
	ok, _ := comm.Db.Where("org_id=? and pipe_id=?", org.Id, "pipe-123").Get(op)
	if !ok {
		t.Error("org pipe not found after add")
	}
}

func TestOrgController_pipeAdd_Duplicate(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "pipe-dup-org", true)

	// Add first time
	op := &model.TOrgPipe{OrgId: org.Id, PipeId: "pipe-dup", Created: time.Now()}
	if _, err := comm.Db.InsertOne(op); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "pipe-dup")
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (duplicate pipeline)", w.Code, http.StatusConflict)
	}
}

func TestOrgController_pipeAdd_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "pipe-noperm-org", false)
	other := createOrgTestUser(t, "other", "Other", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "pipe-new")
	c, w := makeOrgGinCtx(t, m, other)
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// ===== pipeRm =====

func TestOrgController_pipeRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "rm-pipe-org", true)
	op := &model.TOrgPipe{OrgId: org.Id, PipeId: "pipe-to-rm", Created: time.Now()}
	if _, err := comm.Db.InsertOne(op); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "pipe-to-rm")
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

// ===== vars =====

func TestOrgController_vars_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "vars-org", true)

	// Insert some vars
	for i := 0; i < 3; i++ {
		v := &model.TOrgVar{
			OrgId:  org.Id,
			Name:   utils.NewXid(),
			Value:  "val",
			Public: 0,
		}
		if _, err := comm.Db.InsertOne(v); err != nil {
			t.Fatalf("insert org var: %v", err)
		}
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", org.Id)
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.vars(c, m)

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

func TestOrgController_vars_EmptyOrgId(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	m.Set("q", "")
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_vars_WithSearch(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "search-vars-org", true)

	v1 := &model.TOrgVar{OrgId: org.Id, Name: "DB_HOST", Value: "localhost", Public: 0}
	v2 := &model.TOrgVar{OrgId: org.Id, Name: "API_KEY", Value: "secret", Public: 0}
	if _, err := comm.Db.InsertOne(v1); err != nil {
		t.Fatalf("insert v1: %v", err)
	}
	if _, err := comm.Db.InsertOne(v2); err != nil {
		t.Fatalf("insert v2: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", org.Id)
	m.Set("q", "DB")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, admin)
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

// ===== varSave =====

func TestOrgController_varSave_CreateNew(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "varsave-org", true)

	ctrl := OrgController{}
	pv := &bean.OrgVar{
		OrgId:  org.Id,
		Name:   "MY_VAR",
		Value:  "my_value",
		Public: true,
	}
	c, w := makeOrgGinCtx(t, pv, admin)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify
	v := &model.TOrgVar{}
	ok, _ := comm.Db.Where("org_id=? and name=?", org.Id, "MY_VAR").Get(v)
	if !ok {
		t.Error("org var not found after save")
	}
	if v.Value != "my_value" {
		t.Errorf("value = %q, want my_value", v.Value)
	}
}

func TestOrgController_varSave_Update(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "varupdate-org", true)

	// Insert with explicit aid so we can reference it for update
	ev := &model.TOrgVar{Aid: 100, OrgId: org.Id, Name: "UPD_VAR", Value: "old", Public: 0}
	if _, err := comm.Db.InsertOne(ev); err != nil {
		t.Fatalf("insert: %v", err)
	}

	ctrl := OrgController{}
	pv := &bean.OrgVar{
		Aid:    100,
		OrgId:  org.Id,
		Name:   "UPD_VAR",
		Value:  "new_value",
		Public: false,
	}
	c, w := makeOrgGinCtx(t, pv, admin)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	updated := &model.TOrgVar{}
	ok, _ := comm.Db.Where("aid=?", int64(100)).Get(updated)
	if !ok {
		t.Fatal("var not found after update")
	}
	if updated.Value != "new_value" {
		t.Errorf("value = %q, want new_value", updated.Value)
	}
}

func TestOrgController_varSave_DuplicateName(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "dup-var-org", true)

	ev := &model.TOrgVar{OrgId: org.Id, Name: "DUP_VAR", Value: "first", Public: 0}
	if _, err := comm.Db.InsertOne(ev); err != nil {
		t.Fatalf("insert: %v", err)
	}

	ctrl := OrgController{}
	pv := &bean.OrgVar{
		OrgId: org.Id,
		Name:  "DUP_VAR",
		Value: "second",
	}
	c, w := makeOrgGinCtx(t, pv, admin)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (duplicate name)", w.Code, http.StatusConflict)
	}
}

func TestOrgController_varSave_EmptyFields(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)

	ctrl := OrgController{}
	pv := &bean.OrgVar{
		OrgId: "",
		Name:  "",
		Value: "",
	}
	c, w := makeOrgGinCtx(t, pv, admin)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varSave_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "noperm-var-org", false)
	other := createOrgTestUser(t, "other", "Other", 1)

	ctrl := OrgController{}
	pv := &bean.OrgVar{
		OrgId: org.Id,
		Name:  "VAR",
		Value: "val",
	}
	c, w := makeOrgGinCtx(t, pv, other)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// ===== varDel =====

func TestOrgController_varDel_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)
	org := createOrgTestOrg(t, admin, "vardel-org", true)

	// Insert with explicit aid so we can reference it
	v := &model.TOrgVar{Aid: 200, OrgId: org.Id, Name: "DEL_VAR", Value: "del_val", Public: 0}
	if _, err := comm.Db.InsertOne(v); err != nil {
		t.Fatalf("insert: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(200))
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	del := &model.TOrgVar{}
	ok, _ := comm.Db.Where("aid=?", int64(200)).Get(del)
	if ok {
		t.Error("var should have been deleted")
	}
}

func TestOrgController_varDel_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(99999))
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_varDel_InvalidAid(t *testing.T) {
	setupOrgTestDB(t)
	admin := createOrgTestUser(t, "admin", "Admin", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	c, w := makeOrgGinCtx(t, m, admin)
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ===== Routes registration =====

func TestOrgController_GetPath_Route(t *testing.T) {
	ctrl := OrgController{}
	if ctrl.GetPath() != "/api/org" {
		t.Errorf("GetPath() = %q, want /api/org", ctrl.GetPath())
	}
}
