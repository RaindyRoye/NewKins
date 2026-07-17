package route

import (
	"bytes"
	"context"
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
	"xorm.io/xorm"
)

func setupLoginTestRouter(t *testing.T, db *xorm.Engine) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	origKey := comm.Cfg.Server.LoginKey
	comm.Cfg.Server.LoginKey = "test-login-key-12345678"
	t.Cleanup(func() { comm.Cfg.Server.LoginKey = origKey })

	r := gin.New()
	lc := &LoginController{}
	lc.Routes(r.Group("/api/lg"))
	return r
}

func createLoginTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
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
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestLogin_EmptyName(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	body := bytes.NewBufferString(`{"name":"","pass":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", w.Code)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

func TestLogin_EmptyPass(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	body := bytes.NewBufferString(`{"name":"alice","pass":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pass, got %d", w.Code)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	body := bytes.NewBufferString(`{"name":"nonexistent","pass":"whatever"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent user, got %d", w.Code)
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	// Create a non-admin user with active=0
	_, err := db.Exec(`INSERT INTO t_user (id, name, pass, nick, active, created, login_time)
		VALUES ('user1', 'alice', 'hash', 'Alice', 0, ?, ?)`, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	body := bytes.NewBufferString(`{"name":"alice","pass":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for inactive non-admin user, got %d", w.Code)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	passHash := utils.Md5String("correct-password")
	_, err := db.Exec(`INSERT INTO t_user (id, name, pass, nick, active, created, login_time)
		VALUES ('user1', 'alice', ?, 'Alice', 1, ?, ?)`, passHash, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	body := bytes.NewBufferString(`{"name":"alice","pass":"wrong-password"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", w.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	passHash := utils.Md5String("my-password")
	_, err := db.Exec(`INSERT INTO t_user (id, name, pass, nick, active, created, login_time)
		VALUES ('user1', 'alice', ?, 'Alice', 1, ?, ?)`, passHash, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	body := bytes.NewBufferString(`{"name":"alice","pass":"my-password"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for successful login, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected token in response")
	}
	if resp["id"] != "user1" {
		t.Errorf("id = %v, want 'user1'", resp["id"])
	}
	if resp["name"] != "alice" {
		t.Errorf("name = %v, want 'alice'", resp["name"])
	}
	if resp["nick"] != "Alice" {
		t.Errorf("nick = %v, want 'Alice'", resp["nick"])
	}
}

func TestLogin_AdminInactiveStillAllowed(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	// Admin user (id='admin') with active=0 should still be allowed to login
	passHash := utils.Md5String("admin-pass")
	_, err := db.Exec(`INSERT INTO t_user (id, name, pass, nick, active, created, login_time)
		VALUES ('admin', 'admin', ?, 'Admin', 0, ?, ?)`, passHash, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	body := bytes.NewBufferString(`{"name":"admin","pass":"admin-pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Admin bypasses the active check (service.IsAdmin returns true for id="admin")
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for admin login (even inactive), got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestLogin_WhitespaceOnlyName(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	body := bytes.NewBufferString(`{"name":"   ","pass":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Name gets trimmed to empty, so it should be param err
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for whitespace-only name, got %d", w.Code)
	}
}

func TestInfo_NotLoggedIn(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/lg/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["login"] != false {
		t.Errorf("login = %v, want false", resp["login"])
	}
}

func TestInfo_LoggedInUser(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	// Create a user and generate a token
	passHash := utils.Md5String("test")
	_, err := db.Exec(`INSERT INTO t_user (id, name, pass, nick, active, created, login_time)
		VALUES ('user1', 'alice', ?, 'Alice', 1, ?, ?)`, passHash, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Create a valid token
	token, err := util.CreateToken(map[string]any{"uid": "user1"}, "test-login-key-12345678", time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/lg/info", nil)
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["login"] != true {
		t.Errorf("login = %v, want true", resp["login"])
	}
	if resp["user"] == nil {
		t.Error("expected user in response")
	}
}

func TestLogin_UpdatesLastLoginTime(t *testing.T) {
	db := createLoginTestDb(t)
	r := setupLoginTestRouter(t, db)

	oldTime := time.Now().Add(-24 * time.Hour)
	passHash := utils.Md5String("pw")
	_, err := db.Exec(`INSERT INTO t_user (id, name, pass, nick, active, created, login_time)
		VALUES ('user1', 'alice', ?, 'Alice', 1, ?, ?)`, passHash, time.Now(), oldTime)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	body := bytes.NewBufferString(`{"name":"alice","pass":"pw"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify login_time was updated
	usr := &model.TUser{}
	ok, err := db.Where("id=?", "user1").Get(usr)
	if err != nil || !ok {
		t.Fatalf("query user: %v, found=%v", err, ok)
	}
	if usr.LoginTime.Before(oldTime.Add(time.Second)) {
		t.Errorf("login_time was not updated: got %v, should be after %v", usr.LoginTime, oldTime)
	}
}

// Ensure context.Background() is used so the compiler doesn't complain about unused imports
var _ = context.Background
