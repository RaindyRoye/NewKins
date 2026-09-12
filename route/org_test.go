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
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ddls := []string{
		`CREATE TABLE t_user (
			id VARCHAR(64) NOT NULL,
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(100),
			pass VARCHAR(255),
			nick VARCHAR(100),
			avatar VARCHAR(500),
			created DATETIME,
			login_time DATETIME,
			active INT DEFAULT 0
		)`,
		`CREATE UNIQUE INDEX idx_user_id ON t_user(id)`,
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
		)`,
		`CREATE UNIQUE INDEX idx_org_id ON t_org(id)`,
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
	for _, ddl := range ddls {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("failed to create table: %v\nDDL: %s", err, ddl)
		}
	}

	comm.Db = db
}

func makeOrgTestContext(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

func orgTestUser(t *testing.T, name string) *model.TUser {
	t.Helper()
	u := &model.TUser{
		Id:      name,
		Name:    name,
		Nick:    name + " Nick",
		Pass:    "hash",
		Active:  1,
		Created: time.Now(),
	}
	_, err := comm.Db.InsertOne(u)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func orgTestOrg(t *testing.T, uid, name string, public int) *model.TOrg {
	t.Helper()
	o := &model.TOrg{
		Id:      "org-" + name,
		Uid:     uid,
		Name:    name,
		Desc:    name + " description",
		Public:  public,
		Created: time.Now(),
		Updated: time.Now(),
	}
	_, err := comm.Db.InsertOne(o)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	return o
}

func TestOrgCtrl_GetPath(t *testing.T) {
	c := &OrgController{}
	if got := c.GetPath(); got != "/api/org" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/org")
	}
}

func TestOrgCtrl_Routes(t *testing.T) {
	setupOrgTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	oc := &OrgController{}
	oc.Routes(r.Group("/api/org"))
	// Routes registered without panic — success
}

// === list ===

func TestOrgList_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")
	orgTestOrg(t, admin.Id, "testorg", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgList_WithSearch(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")
	orgTestOrg(t, admin.Id, "alphaorg", 1)
	orgTestOrg(t, admin.Id, "betaorg", 0)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "alpha")
	m.Set("page", int64(1))
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgList_NonAdmin(t *testing.T) {
	setupOrgTestDB(t)
	regularUser := orgTestUser(t, "regularuser")
	orgTestOrg(t, regularUser.Id, "userorg", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgTestContext(t, m, regularUser)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// === new ===

func TestOrgNew_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "")
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgNew_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "neworg")
	m.Set("desc", "a new org")
	m.Set("public", true)
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify org was created
	var count int64
	count, _ = comm.Db.Where("name=?", "neworg").Count(&model.TOrg{})
	if count != 1 {
		t.Errorf("expected 1 org named neworg, got %d", count)
	}
}

func TestOrgNew_NonAdminNoPerm(t *testing.T) {
	setupOrgTestDB(t)
	regularUser := orgTestUser(t, "regularuser")
	// No t_user_info row, so PermOrg = 0

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "restrictedorg")
	c, w := makeOrgTestContext(t, m, regularUser)
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestOrgNew_NonAdminWithPerm(t *testing.T) {
	setupOrgTestDB(t)
	regularUser := orgTestUser(t, "regularuser")
	// Give user PermOrg = 1
	_, err := comm.Db.InsertOne(&model.TUserInfo{
		Id:      regularUser.Id,
		PermOrg: 1,
	})
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("name", "userorg")
	m.Set("desc", "user's org")
	m.Set("public", true)
	c, w := makeOrgTestContext(t, m, regularUser)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// === info ===

func TestOrgInfo_EmptyId(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgInfo_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgInfo_DeletedOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")
	org := orgTestOrg(t, admin.Id, "deletedorg", 1)
	org.Deleted = 1
	_, _ = comm.Db.Where("id=?", org.Id).Cols("deleted").Update(org)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgInfo_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")
	org := orgTestOrg(t, admin.Id, "infoorg", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgInfo_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "privateorg", 0) // not public

	other := orgTestUser(t, "other")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgTestContext(t, m, other)
	ctrl.info(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// === users ===

func TestOrgUsers_EmptyId(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.users(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgUsers_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")
	org := orgTestOrg(t, admin.Id, "usersorg", 1)

	// Add a user to the org
	_, err := comm.Db.InsertOne(&model.TUserOrg{
		Uid:      admin.Id,
		OrgId:    org.Id,
		Created:  time.Now(),
		PermAdm:  1,
		PermRw:   1,
		PermExec: 1,
	})
	if err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.users(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgUsers_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "restrictedorg", 0)

	other := orgTestUser(t, "other")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgTestContext(t, m, other)
	ctrl.users(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// === save ===

func TestOrgSave_EmptyName(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")
	org := orgTestOrg(t, admin.Id, "saveorg", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("name", "")
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgSave_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("name", "updated")
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.save(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgSave_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")
	org := orgTestOrg(t, admin.Id, "saveorg", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("name", "updated-name")
	m.Set("desc", "updated desc")
	m.Set("public", false)
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify update
	updated := &model.TOrg{}
	ok, _ := comm.Db.Where("id=?", org.Id).Get(updated)
	if !ok {
		t.Fatal("org not found after save")
	}
	if updated.Name != "updated-name" {
		t.Errorf("name = %q, want %q", updated.Name, "updated-name")
	}
}

func TestOrgSave_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "nopermorg", 0)

	other := orgTestUser(t, "other")
	// Give other user membership but no admin
	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid:     other.Id,
		OrgId:   org.Id,
		Created: time.Now(),
	})

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("name", "hacked")
	c, w := makeOrgTestContext(t, m, other)
	ctrl.save(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// === rm ===

func TestOrgRm_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")
	org := orgTestOrg(t, admin.Id, "rmorg", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify soft delete
	updated := &model.TOrg{}
	_, _ = comm.Db.Where("id=?", org.Id).Get(updated)
	if updated.Deleted != 1 {
		t.Errorf("expected deleted=1, got %d", updated.Deleted)
	}
}

// === userEdit ===

func TestOrgUserEdit_NotFoundOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", admin.Id)
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgUserEdit_EditSelf(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "selforg", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", owner.Id) // editing self
	c, w := makeOrgTestContext(t, m, owner)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrgUserEdit_AddNew(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "addorg", 1)
	member := orgTestUser(t, "member")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", member.Id)
	m.Set("adm", false)
	m.Set("rw", true)
	m.Set("ex", true)
	m.Set("dw", true)
	m.Set("add", true)
	c, w := makeOrgTestContext(t, m, owner)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// === userRm ===

func TestOrgUserRm_NotFoundOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", admin.Id)
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.userRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgUserRm_RemoveSelf(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "rmselforg", 1)

	// Add owner to the org so the membership check passes
	_, err := comm.Db.InsertOne(&model.TUserOrg{
		Uid:     owner.Id,
		OrgId:   org.Id,
		Created: time.Now(),
		PermAdm: 1,
	})
	if err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", owner.Id)
	c, w := makeOrgTestContext(t, m, owner)
	ctrl.userRm(c, m)

	// userRm checks self-removal after membership lookup
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrgUserRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "rmuserorg", 1)
	member := orgTestUser(t, "member")

	// Add member to org
	_, err := comm.Db.InsertOne(&model.TUserOrg{
		Uid:     member.Id,
		OrgId:   org.Id,
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", member.Id)
	c, w := makeOrgTestContext(t, m, owner)
	ctrl.userRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// === pipeAdd ===

func TestOrgPipeAdd_NotFoundOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pipeId", "pipe-1")
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgPipeAdd_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "pipeorg", 0)
	other := orgTestUser(t, "other")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "pipe-1")
	c, w := makeOrgTestContext(t, m, other)
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOrgPipeAdd_Duplicate(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "duppipeorg", 1)

	// Pre-add pipeline
	_, err := comm.Db.InsertOne(&model.TOrgPipe{
		OrgId:   org.Id,
		PipeId:  "pipe-1",
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "pipe-1")
	c, w := makeOrgTestContext(t, m, owner)
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrgPipeAdd_Success(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "addpipeorg", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "pipe-new")
	c, w := makeOrgTestContext(t, m, owner)
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// === pipeRm ===

func TestOrgPipeRm_NotFoundOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pipeId", "pipe-1")
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgPipeRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "rmpipeorg", 1)

	_, err := comm.Db.InsertOne(&model.TOrgPipe{
		OrgId:   org.Id,
		PipeId:  "pipe-rm",
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "pipe-rm")
	c, w := makeOrgTestContext(t, m, owner)
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// === vars ===

func TestOrgVars_EmptyOrgId(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgVars_NotFoundOrg(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgVars_Success(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "varsorg", 1)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("orgId", org.Id)
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgTestContext(t, m, owner)
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// === varSave ===

func TestOrgVarSave_EmptyParams(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	pv := &bean.OrgVar{
		OrgId: "",
		Name:  "key",
		Value: "val",
	}
	c, w := makeOrgTestContext(t, pv, admin)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgVarSave_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	owner := orgTestUser(t, "owner")
	org := orgTestOrg(t, owner.Id, "varpermorg", 0)
	other := orgTestUser(t, "other")

	ctrl := OrgController{}
	pv := &bean.OrgVar{
		OrgId: org.Id,
		Name:  "key",
		Value: "val",
	}
	c, w := makeOrgTestContext(t, pv, other)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// === varDel ===

func TestOrgVarDel_InvalidAid(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgVarDel_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(9999))
	c, w := makeOrgTestContext(t, m, admin)
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
