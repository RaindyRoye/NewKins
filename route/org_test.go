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

func setupOrgTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
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
			perm_org INT,
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
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_pipe (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_var (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
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
			t.Fatalf("create table: %v\nSQL: %s", err, sql)
		}
	}

	comm.Db = db
}

func makeOrgCtx(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest("POST", "/api/org/test", bytes.NewReader(bodyBytes))
	} else {
		req = httptest.NewRequest("POST", "/api/org/test", nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	if loggedInUser != nil {
		c.Set(service.LgUserKey, loggedInUser)
	}
	return c, w
}

func insertOrg(t *testing.T, id, uid, name string, public int) *model.TOrg {
	t.Helper()
	org := &model.TOrg{
		Id:      id,
		Uid:     uid,
		Name:    name,
		Created: time.Now(),
		Updated: time.Now(),
		Public:  public,
	}
	_, err := comm.Db.InsertOne(org)
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}
	return org
}

func insertUserOrg(t *testing.T, uid, _ string, _, _, permExec, permDown int) {
	t.Helper()
	orgId := "org1" // always use org1 in tests
	uo := &model.TUserOrg{
		Uid:      uid,
		OrgId:    orgId,
		Created:  time.Now(),
		PermAdm:  1, // always admin in tests
		PermRw:   1, // always read-write in tests
		PermExec: permExec,
		PermDown: permDown,
	}
	if _, err := comm.Db.InsertOne(uo); err != nil {
		t.Fatalf("failed to insert user org: %v", err)
	}
}

// ==================== list tests ====================

func TestOrgController_list_EmptyDB(t *testing.T) {
	setupOrgTestDB(t)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))

	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeOrgCtx(t, m, admin)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_AdminSeesAll(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", "admin", "Org One", 0)
	insertOrg(t, "org2", "user1", "Org Two", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgCtx(t, m, admin)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_NonAdminSeesPublicAndOwn(t *testing.T) {
	setupOrgTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	insertOrg(t, "org1", "user1", "User Org", 0)
	insertOrg(t, "org2", "other", "Public Org", 1)
	insertOrg(t, "org3", "other", "Private Org", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgCtx(t, m, user)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_WithSearch(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", "admin", "Alpha Org", 0)
	insertOrg(t, "org2", "admin", "Beta Org", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "Alpha")
	m.Set("page", int64(1))
	c, w := makeOrgCtx(t, m, admin)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ==================== new tests ====================

func TestOrgController_new_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("desc", "test")
	m.Set("public", false)
	c, w := makeOrgCtx(t, m, admin)
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_new_AdminSuccess(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "New Org")
	m.Set("desc", "description")
	m.Set("public", true)
	c, w := makeOrgCtx(t, m, admin)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify org was created
	org := &model.TOrg{}
	ok, err := comm.Db.Where("name=?", "New Org").Get(org)
	if err != nil {
		t.Fatalf("query org: %v", err)
	}
	if !ok {
		t.Fatal("org not found after creation")
	}
	if org.Public != 1 {
		t.Errorf("org.Public = %d, want 1", org.Public)
	}
	if org.Uid != admin.Id {
		t.Errorf("org.Uid = %q, want %q", org.Uid, admin.Id)
	}
}

func TestOrgController_new_NonAdminWithoutPerm(t *testing.T) {
	setupOrgTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	// No TUserInfo with PermOrg=1, so non-admin should be denied

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "New Org")
	m.Set("desc", "desc")
	m.Set("public", false)
	c, w := makeOrgCtx(t, m, user)
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestOrgController_new_NonAdminWithPerm(t *testing.T) {
	setupOrgTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	// Create user info with org permission
	uinfo := &model.TUserInfo{
		Id:      user.Id,
		PermOrg: 1,
	}
	_, err := comm.Db.InsertOne(uinfo)
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "User Org")
	m.Set("desc", "created by user")
	m.Set("public", false)
	c, w := makeOrgCtx(t, m, user)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ==================== info tests ====================

func TestOrgController_info_MissingID(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_info_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_DeletedOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	org := insertOrg(t, "org1", "admin", "Deleted Org", 0)
	org.Deleted = 1
	org.DeletedTime = time.Now()
	_, _ = comm.Db.Where("id=?", org.Id).Update(org)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	owner := &model.TUser{Id: "owner1", Name: "owner1", Active: 1}
	outside := &model.TUser{Id: "outside1", Name: "outside1", Active: 1}
	insertOrg(t, "org1", owner.Id, "Private Org", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	c, w := makeOrgCtx(t, m, outside)
	ctrl.info(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestOrgController_info_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1, Created: time.Now(), LoginTime: time.Now()}
	if _, err := comm.Db.InsertOne(admin); err != nil {
		t.Fatalf("failed to insert admin: %v", err)
	}
	insertOrg(t, "org1", admin.Id, "Admin Org", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ==================== save tests ====================

func TestOrgController_save_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Original", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("name", "")
	m.Set("desc", "updated")
	m.Set("public", false)
	c, w := makeOrgCtx(t, m, admin)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_save_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("name", "Updated")
	m.Set("desc", "desc")
	m.Set("public", false)
	c, w := makeOrgCtx(t, m, admin)
	ctrl.save(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_save_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	owner := &model.TUser{Id: "owner1", Name: "owner1", Active: 1}
	outside := &model.TUser{Id: "outside1", Name: "outside1", Active: 1}
	insertOrg(t, "org1", owner.Id, "Owner Org", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("name", "Hacked")
	m.Set("desc", "desc")
	m.Set("public", false)
	c, w := makeOrgCtx(t, m, outside)
	ctrl.save(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestOrgController_save_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Original", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("name", "Updated Name")
	m.Set("desc", "updated desc")
	m.Set("public", true)
	c, w := makeOrgCtx(t, m, admin)
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify update
	org := &model.TOrg{}
	ok, _ := comm.Db.Where("id=?", "org1").Get(org)
	if !ok {
		t.Fatal("org not found after update")
	}
	if org.Name != "Updated Name" {
		t.Errorf("org.Name = %q, want %q", org.Name, "Updated Name")
	}
	if org.Public != 1 {
		t.Errorf("org.Public = %d, want 1", org.Public)
	}
}

// ==================== rm tests ====================

func TestOrgController_rm_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_rm_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	owner := &model.TUser{Id: "owner1", Name: "owner1", Active: 1}
	outside := &model.TUser{Id: "outside1", Name: "outside1", Active: 1}
	insertOrg(t, "org1", owner.Id, "Owner Org", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	c, w := makeOrgCtx(t, m, outside)
	ctrl.rm(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestOrgController_rm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "To Delete", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify soft delete
	org := &model.TOrg{}
	ok, _ := comm.Db.Where("id=?", "org1").Get(org)
	if !ok {
		t.Fatal("org not found")
	}
	if org.Deleted != 1 {
		t.Errorf("org.Deleted = %d, want 1", org.Deleted)
	}
}

// ==================== users tests ====================

func TestOrgController_users_MissingID(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.users(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_users_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Nick: "Admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Org One", 1)
	member := &model.TUser{Id: "member1", Name: "member1", Nick: "Member", Active: 1, Created: time.Now(), LoginTime: time.Now()}
	if _, err := comm.Db.InsertOne(member); err != nil {
		t.Fatalf("failed to insert member: %v", err)
	}
	insertUserOrg(t, "member1", "org1", 1, 1, 0, 0)
	insertUserOrg(t, admin.Id, "org1", 1, 1, 1, 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.users(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ==================== userEdit tests ====================

func TestOrgController_userEdit_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", "user1")
	m.Set("adm", false)
	m.Set("rw", false)
	m.Set("ex", false)
	m.Set("dw", false)
	m.Set("add", false)
	c, w := makeOrgCtx(t, m, admin)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_userEdit_UserNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Org", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("uid", "nonexistent")
	m.Set("adm", false)
	m.Set("rw", false)
	m.Set("ex", false)
	m.Set("dw", false)
	m.Set("add", false)
	c, w := makeOrgCtx(t, m, admin)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_userEdit_SelfEdit(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1, Created: time.Now(), LoginTime: time.Now()}
	if _, err := comm.Db.InsertOne(admin); err != nil {
		t.Fatalf("failed to insert admin: %v", err)
	}
	insertOrg(t, "org1", admin.Id, "Org", 0)
	// Create membership for admin
	insertUserOrg(t, admin.Id, "org1", 1, 1, 1, 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("uid", admin.Id) // editing self
	m.Set("adm", false)
	m.Set("rw", false)
	m.Set("ex", false)
	m.Set("dw", false)
	m.Set("add", false)
	c, w := makeOrgCtx(t, m, admin)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrgController_userEdit_AddMember(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	member := &model.TUser{Id: "member1", Name: "member1", Nick: "Member", Active: 1, Created: time.Now(), LoginTime: time.Now()}
	if _, err := comm.Db.InsertOne(member); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	insertOrg(t, "org1", admin.Id, "Org", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("uid", member.Id)
	m.Set("adm", false)
	m.Set("rw", true)
	m.Set("ex", true)
	m.Set("dw", false)
	m.Set("add", true)
	c, w := makeOrgCtx(t, m, admin)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify membership created
	uo := &model.TUserOrg{}
	ok, _ := comm.Db.Where("uid=? and org_id=?", member.Id, "org1").Get(uo)
	if !ok {
		t.Fatal("user org membership not found")
	}
}

// ==================== userRm tests ====================

func TestOrgController_userRm_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", "user1")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.userRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_userRm_SelfRemove(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1, Created: time.Now(), LoginTime: time.Now()}
	if _, err := comm.Db.InsertOne(admin); err != nil {
		t.Fatalf("failed to insert admin: %v", err)
	}
	insertOrg(t, "org1", admin.Id, "Org", 0)
	insertUserOrg(t, admin.Id, "org1", 1, 1, 1, 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("uid", admin.Id) // removing self
	c, w := makeOrgCtx(t, m, admin)
	ctrl.userRm(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrgController_userRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	member := &model.TUser{Id: "member1", Name: "member1", Active: 1, Created: time.Now(), LoginTime: time.Now()}
	if _, err := comm.Db.InsertOne(member); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	insertOrg(t, "org1", admin.Id, "Org", 0)

	// Insert membership and get the record back to get the aid
	uo := &model.TUserOrg{
		Uid:     member.Id,
		OrgId:   "org1",
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(uo)
	if err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	// Query to get the aid
	uoCheck := &model.TUserOrg{}
	ok, _ := comm.Db.Where("uid=? and org_id=?", member.Id, "org1").Get(uoCheck)
	if !ok {
		t.Fatal("user org membership not found after insert")
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("uid", member.Id)
	c, w := makeOrgCtx(t, m, admin)
	ctrl.userRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify removed
	uoFinal := &model.TUserOrg{}
	ok, _ = comm.Db.Where("uid=? and org_id=?", member.Id, "org1").Get(uoFinal)
	if ok {
		t.Error("user org membership should have been deleted")
	}
}

// ==================== pipeAdd tests ====================

func TestOrgController_pipeAdd_OrgNotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pipeId", "pipe1")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_pipeAdd_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Org", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("pipeId", "pipe123")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify pipeline added
	op := &model.TOrgPipe{}
	ok, _ := comm.Db.Where("org_id=? and pipe_id=?", "org1", "pipe123").Get(op)
	if !ok {
		t.Error("org pipe not found after add")
	}
}

func TestOrgController_pipeAdd_Duplicate(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Org", 0)
	// Add pipeline first
	if _, err := comm.Db.InsertOne(&model.TOrgPipe{OrgId: "org1", PipeId: "pipe123", Created: time.Now()}); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("pipeId", "pipe123")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// ==================== pipeRm tests ====================

func TestOrgController_pipeRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Org", 0)
	if _, err := comm.Db.InsertOne(&model.TOrgPipe{OrgId: "org1", PipeId: "pipe123", Created: time.Now()}); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("pipeId", "pipe123")
	c, w := makeOrgCtx(t, m, admin)
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify removed
	op := &model.TOrgPipe{}
	ok, _ := comm.Db.Where("org_id=? and pipe_id=?", "org1", "pipe123").Get(op)
	if ok {
		t.Error("org pipe should have been deleted")
	}
}

func TestOrgController_pipeRm_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	owner := &model.TUser{Id: "owner1", Name: "owner1", Active: 1}
	outside := &model.TUser{Id: "outside1", Name: "outside1", Active: 1}
	insertOrg(t, "org1", owner.Id, "Org", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "org1")
	m.Set("pipeId", "pipe123")
	c, w := makeOrgCtx(t, m, outside)
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// ==================== vars tests ====================

func TestOrgController_vars_MissingOrgId(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgCtx(t, m, admin)
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_vars_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Org", 1)

	// Insert some vars
	if _, err := comm.Db.InsertOne(&model.TOrgVar{OrgId: "org1", Name: "VAR1", Value: "val1", Public: 1}); err != nil {
		t.Fatalf("insert org var: %v", err)
	}
	if _, err := comm.Db.InsertOne(&model.TOrgVar{OrgId: "org1", Name: "VAR2", Value: "val2", Public: 0}); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org1")
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgCtx(t, m, admin)
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_vars_WithSearch(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Org", 1)
	if _, err := comm.Db.InsertOne(&model.TOrgVar{OrgId: "org1", Name: "DATABASE_URL", Value: "postgres://...", Public: 0}); err != nil {
		t.Fatalf("insert org var: %v", err)
	}
	if _, err := comm.Db.InsertOne(&model.TOrgVar{OrgId: "org1", Name: "API_KEY", Value: "secret", Public: 0}); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org1")
	m.Set("q", "DATABASE")
	m.Set("page", int64(1))
	c, w := makeOrgCtx(t, m, admin)
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// ==================== varSave tests ====================

func TestOrgController_varSave_MissingParams(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	tests := []struct {
		name string
		pv   *bean.OrgVar
	}{
		{"empty name", &bean.OrgVar{OrgId: "org1", Name: "", Value: "val"}},
		{"empty value", &bean.OrgVar{OrgId: "org1", Name: "VAR", Value: ""}},
		{"empty orgId", &bean.OrgVar{OrgId: "", Name: "VAR", Value: "val"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := OrgController{}
			c, w := makeOrgCtx(t, tt.pv, admin)
			ctrl.varSave(c, tt.pv)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestOrgController_varSave_CreateNew(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Org", 0)

	ctrl := OrgController{}
	pv := &bean.OrgVar{OrgId: "org1", Name: "NEW_VAR", Value: "new_value", Public: true}
	c, w := makeOrgCtx(t, pv, admin)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify created
	v := &model.TOrgVar{}
	ok, _ := comm.Db.Where("org_id=? and name=?", "org1", "NEW_VAR").Get(v)
	if !ok {
		t.Fatal("org var not found after save")
	}
	if v.Value != "new_value" {
		t.Errorf("var.Value = %q, want %q", v.Value, "new_value")
	}
	if v.Public != 1 {
		t.Errorf("var.Public = %d, want 1", v.Public)
	}
}

func TestOrgController_varSave_DuplicateName(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Org", 0)
	if _, err := comm.Db.InsertOne(&model.TOrgVar{OrgId: "org1", Name: "EXISTING", Value: "val"}); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	ctrl := OrgController{}
	pv := &bean.OrgVar{OrgId: "org1", Name: "EXISTING", Value: "new_val"}
	c, w := makeOrgCtx(t, pv, admin)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// ==================== varDel tests ====================

func TestOrgController_varDel_InvalidAid(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	c, w := makeOrgCtx(t, m, admin)
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varDel_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(999))
	c, w := makeOrgCtx(t, m, admin)
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_varDel_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertOrg(t, "org1", admin.Id, "Org", 0)
	v := &model.TOrgVar{OrgId: "org1", Name: "DEL_VAR", Value: "val"}
	if _, err := comm.Db.InsertOne(v); err != nil {
		t.Fatalf("insert org var: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", v.Aid)
	c, w := makeOrgCtx(t, m, admin)
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify deleted
	check := &model.TOrgVar{}
	ok, _ := comm.Db.Where("aid=?", v.Aid).Get(check)
	if ok {
		t.Error("org var should have been deleted")
	}
}
