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
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupOrgTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	origLoginKey := comm.Cfg.Server.LoginKey
	t.Cleanup(func() {
		comm.Db = origDb
		comm.Cfg.Server.LoginKey = origLoginKey
	})

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
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
		t.Fatalf("failed to create t_user table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_info (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		phone VARCHAR(100),
		email VARCHAR(200),
		birthday DATETIME,
		remark TEXT,
		perm_user INT DEFAULT 0,
		perm_org INT DEFAULT 0,
		perm_pipe INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("failed to create t_user_info table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org (
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
	)`)
	if err != nil {
		t.Fatalf("failed to create t_org table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_org (
		aid BIGINT NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		perm_adm INT DEFAULT 0,
		perm_rw INT DEFAULT 0,
		perm_exec INT DEFAULT 0,
		perm_down INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("failed to create t_user_org table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org_var (
		aid BIGINT NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		name VARCHAR(255),
		value TEXT,
		remarks VARCHAR(255),
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("failed to create t_org_var table: %v", err)
	}

	comm.Db = db
	comm.Cfg.Server.LoginKey = "test-secret-key-for-org-tests-32bytes"
	return db
}

func createOrgTestUser(name, nick string, active int) *model.TUser { //nolint:unparam // active varies in future tests
	return &model.TUser{
		Id:      utils.NewXid(),
		Name:    name,
		Nick:    nick,
		Pass:    utils.Md5String("password123"),
		Active:  active,
		Created: time.Now(),
	}
}

func setupOrgRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	oc := OrgController{}
	oc.Routes(r.Group("/api/org"))
	return r
}

func getOrgLoginToken(t *testing.T, db *xorm.Engine, username string) string {
	t.Helper()
	// Generate token directly to avoid hitting the login rate limiter
	// when running the full test suite (login tests + org tests > 10 req/min).
	user := &model.TUser{}
	ok, err := db.Where("name=?", username).Get(user)
	if err != nil || !ok {
		t.Fatalf("getOrgLoginToken: user %q not found: %v", username, err)
	}
	if user.Pass != utils.Md5String("password123") {
		t.Fatalf("getOrgLoginToken: password mismatch for %q", username)
	}
	token, err := util.CreateToken(jwt.MapClaims{
		"uid": user.Id,
	}, comm.Cfg.Server.LoginKey, time.Hour*24*5)
	if err != nil {
		t.Fatalf("getOrgLoginToken: create token: %v", err)
	}
	return token
}

func TestOrg_New_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	r := setupOrgRouter()

	user := createOrgTestUser("alice", "Alice", 1)
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	// Grant org creation permission
	if _, err := db.InsertOne(&model.TUserInfo{Id: user.Id, PermOrg: 1}); err != nil {
		t.Fatalf("insert user_info: %v", err)
	}

	token := getOrgLoginToken(t, db, "alice")

	m := map[string]interface{}{
		"name":   "Test Org",
		"desc":   "A test organization",
		"public": true,
	}

	body, _ := json.Marshal(m)
	req := httptest.NewRequest(http.MethodPost, "/api/org/new", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected org id in response")
	}

	// Verify org exists in DB
	org := &model.TOrg{}
	ok, err := db.Where("name=?", "Test Org").Get(org)
	if err != nil {
		t.Fatalf("query org: %v", err)
	}
	if !ok {
		t.Error("org not found in database")
	}
	if org.Uid != user.Id {
		t.Errorf("uid = %s, want %s", org.Uid, user.Id)
	}
	if org.Public != 1 {
		t.Errorf("public = %d, want 1", org.Public)
	}
}

func TestOrg_New_NoAuth(t *testing.T) {
	_ = setupOrgTestDB(t)
	r := setupOrgRouter()

	m := &hbtp.Map{}
	m.Set("name", "Test Org")
	m.Set("desc", "A test organization")

	body, _ := json.Marshal(m)
	req := httptest.NewRequest(http.MethodPost, "/api/org/new", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestOrg_New_EmptyName(t *testing.T) {
	db := setupOrgTestDB(t)
	r := setupOrgRouter()

	user := createOrgTestUser("bob", "Bob", 1)
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	token := getOrgLoginToken(t, db, "bob")

	m := map[string]interface{}{
		"name": "",
		"desc": "A test organization",
	}

	body, _ := json.Marshal(m)
	req := httptest.NewRequest(http.MethodPost, "/api/org/new", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestOrg_Info_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	r := setupOrgRouter()

	user := createOrgTestUser("charlie", "Charlie", 1)
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	token := getOrgLoginToken(t, db, "charlie")

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Name:    "Test Org",
		Desc:    "Test Description",
		Uid:     user.Id,
		Created: time.Now(),
		Updated: time.Now(),
		Deleted: 0,
	}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	m := map[string]interface{}{
		"id": org.Id,
	}

	body, _ := json.Marshal(m)
	req := httptest.NewRequest(http.MethodPost, "/api/org/info", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// info response wraps data in "org"/"user"/"perm" keys
	orgResp, ok := resp["org"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected org object in response, got: %+v", resp)
	}
	if orgResp["id"] != org.Id {
		t.Errorf("id = %v, want %s", orgResp["id"], org.Id)
	}
	if orgResp["name"] != "Test Org" {
		t.Errorf("name = %v, want 'Test Org'", orgResp["name"])
	}
}

func TestOrg_Info_NotFound(t *testing.T) {
	db := setupOrgTestDB(t)
	r := setupOrgRouter()

	user := createOrgTestUser("david", "David", 1)
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	token := getOrgLoginToken(t, db, "david")

	m := map[string]interface{}{
		"id": "nonexistent",
	}

	body, _ := json.Marshal(m)
	req := httptest.NewRequest(http.MethodPost, "/api/org/info", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestOrg_Rm_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	r := setupOrgRouter()

	user := createOrgTestUser("eve", "Eve", 1)
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	token := getOrgLoginToken(t, db, "eve")

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Name:    "Delete Me",
		Desc:    "This will be deleted",
		Uid:     user.Id,
		Created: time.Now(),
		Updated: time.Now(),
		Deleted: 0,
	}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	m := map[string]interface{}{
		"id": org.Id,
	}

	body, _ := json.Marshal(m)
	req := httptest.NewRequest(http.MethodPost, "/api/org/rm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify soft delete in DB
	deleted := &model.TOrg{}
	ok, err := db.Where("id=?", org.Id).Get(deleted)
	if err != nil {
		t.Fatalf("query deleted org: %v", err)
	}
	if !ok {
		t.Fatal("org not found")
	}
	if deleted.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted.Deleted)
	}
}

func TestOrg_Rm_NotOwner(t *testing.T) {
	db := setupOrgTestDB(t)
	r := setupOrgRouter()

	owner := createOrgTestUser("frank", "Frank", 1)
	if _, err := db.InsertOne(owner); err != nil {
		t.Fatalf("insert owner: %v", err)
	}

	other := createOrgTestUser("grace", "Grace", 1)
	if _, err := db.InsertOne(other); err != nil {
		t.Fatalf("insert other user: %v", err)
	}

	token := getOrgLoginToken(t, db, "grace")

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Name:    "Frank's Org",
		Desc:    "Owned by Frank",
		Uid:     owner.Id,
		Created: time.Now(),
		Updated: time.Now(),
		Deleted: 0,
	}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	m := map[string]interface{}{
		"id": org.Id,
	}

	body, _ := json.Marshal(m)
	req := httptest.NewRequest(http.MethodPost, "/api/org/rm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestOrg_Save_Success(t *testing.T) {
	db := setupOrgTestDB(t)
	r := setupOrgRouter()

	user := createOrgTestUser("harry", "Harry", 1)
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	token := getOrgLoginToken(t, db, "harry")

	org := &model.TOrg{
		Id:      utils.NewXid(),
		Name:    "Old Name",
		Desc:    "Old Description",
		Uid:     user.Id,
		Created: time.Now(),
		Updated: time.Now(),
		Deleted: 0,
	}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	m := map[string]interface{}{
		"id":   org.Id,
		"name": "New Name",
		"desc": "New Description",
	}

	body, _ := json.Marshal(m)
	req := httptest.NewRequest(http.MethodPost, "/api/org/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify update in DB
	updated := &model.TOrg{}
	ok, err := db.Where("id=?", org.Id).Get(updated)
	if err != nil {
		t.Fatalf("query updated org: %v", err)
	}
	if !ok {
		t.Fatal("org not found")
	}
	if updated.Name != "New Name" {
		t.Errorf("name = %s, want 'New Name'", updated.Name)
	}
	if updated.Desc != "New Description" {
		t.Errorf("desc = %s, want 'New Description'", updated.Desc)
	}
}
