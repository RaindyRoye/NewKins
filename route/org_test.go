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
			aid INTEGER,
			uid VARCHAR(64),
			name VARCHAR(200),
			desc TEXT,
			public INTEGER DEFAULT 0,
			created DATETIME,
			updated DATETIME,
			deleted INTEGER DEFAULT 0,
			deleted_time DATETIME
		)`,
		`CREATE TABLE t_user_org (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INTEGER DEFAULT 0,
			perm_rw INTEGER DEFAULT 0,
			perm_exec INTEGER DEFAULT 0,
			perm_down INTEGER DEFAULT 0
		)`,
		`CREATE TABLE t_org_pipe (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INTEGER DEFAULT 0
		)`,
		`CREATE TABLE t_org_var (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			name VARCHAR(255),
			value TEXT,
			remarks VARCHAR(255),
			public INTEGER DEFAULT 0
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
			t.Fatalf("failed to create table: %v\nDDL: %s", err, ddl)
		}
	}

	comm.Db = db
}

func orgTestID() string {
	return time.Now().Format("20060102150405.000000")
}

func orgTestUser(t *testing.T, name string) *model.TUser {
	t.Helper()
	u := &model.TUser{
		Id:      utils.NewXid(),
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
		Id:      utils.NewXid(),
		Uid:     uid,
		Name:    name,
		Desc:    "test org",
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

func orgGinCtx(t *testing.T, body interface{}, lgusr *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

	if lgusr != nil {
		c.Set(service.LgUserKey, lgusr)
	}
	return c, w
}

// --- list ---

func TestOrgController_list_Empty(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")
	// Make admin: IsAdmin checks id == "admin"
	admin.Id = "admin"
	_, _ = comm.Db.Where("name=?", "admin").Delete(&model.TUser{})
	_, _ = comm.Db.InsertOne(admin)

	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := orgGinCtx(t, m, admin)
	ctrl := OrgController{}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_WithOrgs(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")
	admin.Id = "admin"
	_, _ = comm.Db.Where("name=?", "admin").Delete(&model.TUser{})
	_, _ = comm.Db.InsertOne(admin)

	orgTestOrg(t, admin.Id, "org1", 1)
	orgTestOrg(t, admin.Id, "org2", 0)

	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := orgGinCtx(t, m, admin)
	ctrl := OrgController{}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_WithSearch(t *testing.T) {
	setupOrgTestDB(t)
	admin := orgTestUser(t, "admin")
	admin.Id = "admin"
	_, _ = comm.Db.Where("name=?", "admin").Delete(&model.TUser{})
	_, _ = comm.Db.InsertOne(admin)

	orgTestOrg(t, admin.Id, "alpha-org", 1)
	orgTestOrg(t, admin.Id, "beta-org", 1)

	m := &hbtp.Map{}
	m.Set("q", "alpha")
	m.Set("page", int64(1))
	c, w := orgGinCtx(t, m, admin)
	ctrl := OrgController{}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOrgController_list_NonAdmin(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "regular")

	// Create a public org and a private org owned by this user
	orgTestOrg(t, usr.Id, "my-org", 0)
	orgTestOrg(t, "other", "public-org", 1)

	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- new ---

func TestOrgController_new_MissingName(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("desc", "desc")
	m.Set("public", false)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_new_Success(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	// Give user PermOrg=1 via user_info
	_, _ = comm.Db.InsertOne(&model.TUserInfo{Id: usr.Id, PermOrg: 1})

	m := &hbtp.Map{}
	m.Set("name", "new-org")
	m.Set("desc", "a new org")
	m.Set("public", true)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify org was created
	var count int64
	count, _ = comm.Db.Where("name=?", "new-org").Count(&model.TOrg{})
	if count != 1 {
		t.Errorf("org count = %d, want 1", count)
	}
}

func TestOrgController_new_NoPermission(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	// No PermOrg in user_info
	_, _ = comm.Db.InsertOne(&model.TUserInfo{Id: usr.Id, PermOrg: 0})

	m := &hbtp.Map{}
	m.Set("name", "forbidden-org")
	m.Set("desc", "should fail")
	m.Set("public", false)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- info ---

func TestOrgController_info_MissingID(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	m := &hbtp.Map{}
	m.Set("id", "")

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_info_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_Success(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	m := &hbtp.Map{}
	m.Set("id", org.Id)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- save ---

func TestOrgController_save_MissingName(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	m := &hbtp.Map{}
	m.Set("id", "some-id")
	m.Set("name", "")

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_save_Success(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "old-name", 0)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("name", "new-name")
	m.Set("desc", "updated")
	m.Set("public", true)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	updated := &model.TOrg{}
	_, _ = comm.Db.Where("id=?", org.Id).Get(updated)
	if updated.Name != "new-name" {
		t.Errorf("name = %q, want %q", updated.Name, "new-name")
	}
}

// --- rm ---

func TestOrgController_rm_Success(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "delete-me", 0)

	m := &hbtp.Map{}
	m.Set("id", org.Id)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	updated := &model.TOrg{}
	_, _ = comm.Db.Where("id=?", org.Id).Get(updated)
	if updated.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", updated.Deleted)
	}
}

func TestOrgController_rm_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- users ---

func TestOrgController_users_Success(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	// Add user to org
	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid:     usr.Id,
		OrgId:   org.Id,
		Created: time.Now(),
		PermAdm: 1,
	})

	m := &hbtp.Map{}
	m.Set("id", org.Id)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.users(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_users_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.users(c, m)

	// perm.Org() is nil => "no permission" since CanRead() on nil org is false
	if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 405 or 404", w.Code)
	}
}

// --- pipeAdd / pipeRm ---

func TestOrgController_pipeAdd_Success(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	// Create a pipeline
	pipe := &model.TPipeline{
		Id:   utils.NewXid(),
		Uid:  usr.Id,
		Name: "test-pipe",
	}
	_, _ = comm.Db.InsertOne(pipe)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", pipe.Id)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_pipeAdd_Duplicate(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	pipe := &model.TPipeline{
		Id:   utils.NewXid(),
		Uid:  usr.Id,
		Name: "test-pipe",
	}
	_, _ = comm.Db.InsertOne(pipe)

	// Already linked
	_, _ = comm.Db.InsertOne(&model.TOrgPipe{
		OrgId:   org.Id,
		PipeId:  pipe.Id,
		Created: time.Now(),
	})

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", pipe.Id)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestOrgController_pipeRm_Success(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	pipe := &model.TPipeline{
		Id:   utils.NewXid(),
		Uid:  usr.Id,
		Name: "test-pipe",
	}
	_, _ = comm.Db.InsertOne(pipe)
	_, _ = comm.Db.InsertOne(&model.TOrgPipe{
		OrgId:   org.Id,
		PipeId:  pipe.Id,
		Created: time.Now(),
	})

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", pipe.Id)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- vars / varSave / varDel ---

func TestOrgController_vars_Empty(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	m := &hbtp.Map{}
	m.Set("orgId", org.Id)
	m.Set("q", "")
	m.Set("page", int64(1))

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_vars_MissingOrgId(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")

	m := &hbtp.Map{}
	m.Set("orgId", "")

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varSave_MissingParams(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	// varSave requires *bean.OrgVar; test missing name
	pv := &bean.OrgVar{
		OrgId: org.Id,
		Name:  "",
		Value: "hello",
	}
	c, w := orgGinCtx(t, nil, usr)
	ctrl := OrgController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestOrgController_varSave_Success(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	pv := &bean.OrgVar{
		OrgId: org.Id,
		Name:  "MY_VAR",
		Value: "hello",
	}
	c, w := orgGinCtx(t, nil, usr)
	ctrl := OrgController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify var was created
	count, _ := comm.Db.Where("org_id=? and name=?", org.Id, "MY_VAR").Count(&model.TOrgVar{})
	if count != 1 {
		t.Errorf("org var count = %d, want 1", count)
	}
}

func TestOrgController_varSave_Duplicate(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	// Insert an existing var
	_, _ = comm.Db.InsertOne(&model.TOrgVar{
		OrgId: org.Id,
		Name:  "EXISTING",
		Value: "val",
	})

	pv := &bean.OrgVar{
		OrgId: org.Id,
		Name:  "EXISTING",
		Value: "new-val",
	}
	c, w := orgGinCtx(t, nil, usr)
	ctrl := OrgController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrgController_varDel_InvalidParam(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")

	m := &hbtp.Map{}
	m.Set("aid", int64(0))

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varDel_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")

	m := &hbtp.Map{}
	m.Set("aid", int64(9999))

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- userEdit / userRm ---

func TestOrgController_userRm_CantRemoveSelf(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid:     usr.Id,
		OrgId:   org.Id,
		Created: time.Now(),
		PermAdm: 1,
	})

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", usr.Id)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.userRm(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestOrgController_userRm_NotFoundOrg(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("uid", "someuser")

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.userRm(c, m)

	// perm.Org() is nil -> 404
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_userEdit_CantEditSelf(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid:     usr.Id,
		OrgId:   org.Id,
		Created: time.Now(),
		PermAdm: 1,
	})

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", usr.Id)
	m.Set("adm", true)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.userEdit(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestOrgController_userEdit_UserNotFound(t *testing.T) {
	setupOrgTestDB(t)
	usr := orgTestUser(t, "user1")
	org := orgTestOrg(t, usr.Id, "test-org", 1)

	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", "nonexistent-user")
	m.Set("adm", false)

	c, w := orgGinCtx(t, m, usr)
	ctrl := OrgController{}
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- GetPath ---

func TestOrgController_GetPath_Org(t *testing.T) {
	ctrl := OrgController{}
	if p := ctrl.GetPath(); p != "/api/org" {
		t.Errorf("GetPath() = %q, want %q", p, "/api/org")
	}
}
