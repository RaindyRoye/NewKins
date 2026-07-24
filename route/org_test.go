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
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupOrgTestDB(t *testing.T) *xorm.Engine {
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
			active INT DEFAULT 1
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
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			name VARCHAR(100),
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
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0,
			created DATETIME
		)`,
		`CREATE TABLE t_pipeline (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			name VARCHAR(100),
			display_name VARCHAR(100),
			pipeline_type VARCHAR(50),
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			create_time DATETIME
		)`,
		`CREATE TABLE t_org_pipe (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			public INT DEFAULT 0,
			created DATETIME
		)`,
		`CREATE TABLE t_org_var (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			name VARCHAR(100),
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
	return db
}

func createOrgTestUser(t *testing.T, db *xorm.Engine, name, nick string) *model.TUser {
	t.Helper()
	user := &model.TUser{
		Id:        utils.NewXid(),
		Name:      name,
		Nick:      nick,
		Pass:      utils.Md5String("password"),
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createTestOrg(t *testing.T, db *xorm.Engine, uid, name string, public int) *model.TOrg {
	t.Helper()
	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     uid,
		Name:    name,
		Public:  public,
		Created: time.Now(),
		Updated: time.Now(),
		Deleted: 0,
	}
	_, err := db.InsertOne(org)
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}
	return org
}

func orgCtx(t *testing.T, body interface{}, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

	if user != nil {
		c.Set(service.LgUserKey, user)
	}
	return c, w
}

func makeAdmin(db *xorm.Engine, user *model.TUser) {
	db.Where("id=?", user.Id).Delete(&model.TUser{})
	user.Id = "admin"
	user.Name = "admin"
	db.InsertOne(user)
}

func mapToHbtpMap(data []byte) *hbtp.Map {
	m := &hbtp.Map{}
	if err := json.Unmarshal(data, m); err != nil {
		return &hbtp.Map{}
	}
	return m
}

func parseOrgVarReq(data []byte) *bean.OrgVar {
	pv := &bean.OrgVar{}
	if err := json.Unmarshal(data, pv); err != nil {
		return &bean.OrgVar{}
	}
	return pv
}

func TestOrg_New_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	user := createOrgTestUser(t, db, "alice", "Alice")
	makeAdmin(db, user)

	c, w := orgCtx(t, nil, user)
	ctrl := OrgController{}
	m := map[string]interface{}{
		"name":   "Test Org",
		"desc":   "Test organization",
		"public": 1,
	}
	bm, _ := json.Marshal(m)
	ctrl.new(c, mapToHbtpMap(bm))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected org id in response")
	}
}

func TestOrg_New_EmptyName(t *testing.T) {
	db := setupOrgTestDB(t)
	user := createOrgTestUser(t, db, "bob", "Bob")
	makeAdmin(db, user)

	c, w := orgCtx(t, nil, user)
	ctrl := OrgController{}
	m := map[string]interface{}{
		"name":   "",
		"desc":   "Test",
		"public": 0,
	}
	bm, _ := json.Marshal(m)
	ctrl.new(c, mapToHbtpMap(bm))

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestOrg_Info_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	user := createOrgTestUser(t, db, "alice", "Alice")
	makeAdmin(db, user)

	org := createTestOrg(t, db, user.Id, "Test Org", 1)

	c, w := orgCtx(t, nil, user)
	ctrl := OrgController{}
	m := map[string]interface{}{"id": org.Id}
	bm, _ := json.Marshal(m)
	ctrl.info(c, mapToHbtpMap(bm))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrg_Info_NotFound(t *testing.T) {
	db := setupOrgTestDB(t)
	user := createOrgTestUser(t, db, "alice", "Alice")
	makeAdmin(db, user)

	c, w := orgCtx(t, nil, user)
	ctrl := OrgController{}
	m := map[string]interface{}{"id": "nonexistent"}
	bm, _ := json.Marshal(m)
	ctrl.info(c, mapToHbtpMap(bm))

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestOrg_Rm_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	user := createOrgTestUser(t, db, "alice", "Alice")
	makeAdmin(db, user)

	org := createTestOrg(t, db, user.Id, "Test Org", 1)

	c, w := orgCtx(t, nil, user)
	ctrl := OrgController{}
	m := map[string]interface{}{"id": org.Id}
	bm, _ := json.Marshal(m)
	ctrl.rm(c, mapToHbtpMap(bm))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	updated := &model.TOrg{}
	ok, err := db.Where("id=?", org.Id).Get(updated)
	if err != nil {
		t.Fatalf("query org: %v", err)
	}
	if !ok {
		t.Fatal("org not found")
	}
	if updated.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", updated.Deleted)
	}
}

func TestOrg_UserEdit_AddUser(t *testing.T) {
	db := setupOrgTestDB(t)
	user := createOrgTestUser(t, db, "admin", "Admin")
	makeAdmin(db, user)

	newUser := createOrgTestUser(t, db, "newuser", "New User")
	org := createTestOrg(t, db, user.Id, "Test Org", 1)

	c, w := orgCtx(t, nil, user)
	ctrl := OrgController{}
	m := map[string]interface{}{
		"id":  org.Id,
		"uid": newUser.Id,
		"adm": true,
		"rw":  true,
		"ex":  true,
		"dw":  true,
		"add": true,
	}
	bm, _ := json.Marshal(m)
	ctrl.userEdit(c, mapToHbtpMap(bm))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	userOrg := &model.TUserOrg{}
	ok, err := db.Where("uid=? AND org_id=?", newUser.Id, org.Id).Get(userOrg)
	if err != nil {
		t.Fatalf("query user_org: %v", err)
	}
	if !ok {
		t.Fatal("user_org not found")
	}
	if userOrg.PermAdm != 1 {
		t.Errorf("perm_adm = %d, want 1", userOrg.PermAdm)
	}
}

func TestOrg_VarSave_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	user := createOrgTestUser(t, db, "admin", "Admin")
	makeAdmin(db, user)

	org := createTestOrg(t, db, user.Id, "Test Org", 1)

	c, w := orgCtx(t, nil, user)
	ctrl := OrgController{}

	pv := &struct {
		OrgId  string `json:"orgId"`
		Name   string `json:"name"`
		Value  string `json:"value"`
		Public bool   `json:"public"`
		Aid    int64  `json:"aid"`
	}{
		OrgId:  org.Id,
		Name:   "TEST_VAR",
		Value:  "test-value",
		Public: true,
	}
	bm, _ := json.Marshal(pv)
	ctrl.varSave(c, parseOrgVarReq(bm))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	orgVar := &model.TOrgVar{}
	ok, err := db.Where("org_id=? AND name=?", org.Id, "TEST_VAR").Get(orgVar)
	if err != nil {
		t.Fatalf("query org_var: %v", err)
	}
	if !ok {
		t.Fatal("org_var not found")
	}
	if orgVar.Value != "test-value" {
		t.Errorf("value = %q, want %q", orgVar.Value, "test-value")
	}
}
