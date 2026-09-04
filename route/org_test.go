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

	// Create required tables
	tables := []string{
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
		`CREATE TABLE t_user_org (
			aid BIGINT NOT NULL PRIMARY KEY,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_pipe (
			aid BIGINT NOT NULL PRIMARY KEY,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_var (
			aid BIGINT NOT NULL PRIMARY KEY,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			name VARCHAR(255),
			value TEXT,
			remarks VARCHAR(255),
			public INT DEFAULT 0
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
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	comm.Db = db
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

func TestOrgController_list_EmptyDB(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	ctrl := OrgController{}

	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_WithOrgs(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create test orgs
	for i := 0; i < 3; i++ {
		org := &model.TOrg{
			Id:      time.Now().Format("20060102150405.000000"),
			Uid:     "admin",
			Name:    "org" + string(rune('0'+i)),
			Desc:    "Test org",
			Public:  1,
			Created: time.Now(),
			Updated: time.Now(),
		}
		if _, err := comm.Db.InsertOne(org); err != nil {
			t.Fatalf("create org: %v", err)
		}
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_WithSearch(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	org := &model.TOrg{
		Id:      time.Now().Format("20060102150405.000000"),
		Uid:     "admin",
		Name:    "searchable-org",
		Desc:    "Test org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("q", "searchable")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOrgController_new_MissingName(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	ctrl := OrgController{}

	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("desc", "desc")
	m.Set("public", false)
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_new_Success(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	ctrl := OrgController{}

	m := &hbtp.Map{}
	m.Set("name", "new-org")
	m.Set("desc", "A new organization")
	m.Set("public", true)
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp bean.IdsRes
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Id == "" {
		t.Error("expected non-empty ID in response")
	}
}

func TestOrgController_info_MissingID(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	ctrl := OrgController{}

	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_info_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	ctrl := OrgController{}

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_Success(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1, Created: time.Now(), LoginTime: time.Now()}

	// Insert the admin user into the DB so the info endpoint can look them up
	if _, err := comm.Db.InsertOne(adminUser); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	org := &model.TOrg{
		Id:      time.Now().Format("20060102150405.000000"),
		Uid:     "admin",
		Name:    "test-org",
		Desc:    "Test org",
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_users_MissingID(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	ctrl := OrgController{}

	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.users(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_save_MissingName(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	org := &model.TOrg{
		Id:      time.Now().Format("20060102150405.000000"),
		Uid:     "admin",
		Name:    "test-org",
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("name", "")
	m.Set("desc", "updated")
	m.Set("public", false)
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_save_Success(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	org := &model.TOrg{
		Id:      time.Now().Format("20060102150405.000000"),
		Uid:     "admin",
		Name:    "test-org",
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("name", "updated-org")
	m.Set("desc", "updated desc")
	m.Set("public", true)
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_rm_NotFound(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	ctrl := OrgController{}

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_rm_Success(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	org := &model.TOrg{
		Id:      time.Now().Format("20060102150405.000000"),
		Uid:     "admin",
		Name:    "test-org",
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_userEdit_MissingParams(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	org := &model.TOrg{
		Id:      time.Now().Format("20060102150405.000000"),
		Uid:     "admin",
		Name:    "test-org",
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("uid", "")
	m.Set("adm", false)
	m.Set("rw", false)
	m.Set("ex", false)
	m.Set("dw", false)
	m.Set("add", false)
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.userEdit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_pipeAdd_MissingPipeId(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	org := &model.TOrg{
		Id:      time.Now().Format("20060102150405.000000"),
		Uid:     "admin",
		Name:    "test-org",
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	ctrl := OrgController{}
	m := &hbtp.Map{}
	m.Set("id", org.Id)
	m.Set("pipeId", "")
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.pipeAdd(c, m)

	// Empty pipeId should still attempt insertion (or fail on query)
	if w.Code == http.StatusInternalServerError {
		t.Log("expected internal error with empty pipeId")
	}
}

func TestOrgController_vars_MissingOrgId(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	ctrl := OrgController{}

	m := &hbtp.Map{}
	m.Set("orgId", "")
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varSave_MissingParams(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	ctrl := OrgController{}

	pv := &bean.OrgVar{
		OrgId: "",
		Name:  "var1",
		Value: "value1",
	}
	c, w := makeOrgGinCtx(t, pv, adminUser)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varDel_InvalidAid(t *testing.T) {
	setupOrgTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	ctrl := OrgController{}

	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	c, w := makeOrgGinCtx(t, m, adminUser)
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
