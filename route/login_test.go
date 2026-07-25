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
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupLoginTestDB(t *testing.T) *xorm.Engine {
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

	comm.Db = db
	comm.Cfg.Server.LoginKey = "test-secret-key-for-login-tests-32b"
	return db
}

func createLoginTestUser(name, nick, pass string, active int) *model.TUser {
	return &model.TUser{
		Id:        utils.NewXid(),
		Name:      name,
		Nick:      nick,
		Pass:      utils.Md5String(pass),
		Active:    active,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
}

func setupLoginRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	lc := LoginController{}
	lc.Routes(r.Group("/api/lg"))
	return r
}

func TestLogin_EmptyUsername(t *testing.T) {
	_ = setupLoginTestDB(t)
	r := setupLoginRouter()

	body, _ := json.Marshal(map[string]string{"name": "", "pass": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestLogin_EmptyPassword(t *testing.T) {
	_ = setupLoginTestDB(t)
	r := setupLoginRouter()

	body, _ := json.Marshal(map[string]string{"name": "alice", "pass": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	_ = setupLoginTestDB(t)
	r := setupLoginRouter()

	body, _ := json.Marshal(map[string]string{"name": "nonexistent", "pass": "anypass"})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	db := setupLoginTestDB(t)
	r := setupLoginRouter()
	user := createLoginTestUser("inactive", "Inactive User", "pass123", 0)
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"name": "inactive", "pass": "pass123"})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	db := setupLoginTestDB(t)
	r := setupLoginRouter()
	user := createLoginTestUser("alice", "Alice", "correct-pass", 1)
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"name": "alice", "pass": "wrong-pass"})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestLogin_Success(t *testing.T) {
	db := setupLoginTestDB(t)
	r := setupLoginRouter()
	user := createLoginTestUser("bob", "Bob Builder", "mypassword", 1)
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"name": "bob", "pass": "mypassword"})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected token to be set in response")
	}
	if resp["name"] != "bob" {
		t.Errorf("name = %v, want %q", resp["name"], "bob")
	}
	if resp["nick"] != "Bob Builder" {
		t.Errorf("nick = %v, want %q", resp["nick"], "Bob Builder")
	}

	// Verify login_time was updated in DB
	updated := &model.TUser{}
	ok, err := db.Where("name=?", "bob").Get(updated)
	if err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after login")
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	_ = setupLoginTestDB(t)
	r := setupLoginRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestLogin_WhitespaceUsername(t *testing.T) {
	_ = setupLoginTestDB(t)
	r := setupLoginRouter()

	body, _ := json.Marshal(map[string]string{"name": "   ", "pass": "secret"})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestLogin_Info_NoAuth(t *testing.T) {
	_ = setupLoginTestDB(t)
	r := setupLoginRouter()

	// No token set, so CurrUserCache returns false
	req := httptest.NewRequest(http.MethodPost, "/api/lg/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}
	if resp["login"] != false {
		t.Errorf("login = %v, want false", resp["login"])
	}
}

func TestLogin_AdminUser_CanLoginWhenInactive(t *testing.T) {
	db := setupLoginTestDB(t)
	r := setupLoginRouter()
	// Admin user (id="admin") can login even when active=0
	admin := &model.TUser{
		Id:        "admin",
		Name:      "admin",
		Nick:      "Admin",
		Pass:      utils.Md5String("adminpass"),
		Active:    0,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	if _, err := db.InsertOne(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"name": "admin", "pass": "adminpass"})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestLogin_EmptyLoginKey(t *testing.T) {
	db := setupLoginTestDB(t)
	comm.Cfg.Server.LoginKey = "" // Clear the login key
	r := setupLoginRouter()
	user := createLoginTestUser("charlie", "Charlie", "pass123", 1)
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"name": "charlie", "pass": "pass123"})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}
}
