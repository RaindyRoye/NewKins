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
	"xorm.io/xorm"
)

func setupLoginTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL,
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(100),
		nick VARCHAR(100),
		pass VARCHAR(255),
		avatar VARCHAR(500),
		active INT DEFAULT 1,
		login_time DATETIME,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_user: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_info (
		id VARCHAR(64) PRIMARY KEY,
		phone VARCHAR(50),
		email VARCHAR(100),
		remark TEXT,
		perm_user INT,
		perm_org INT,
		perm_pipe INT
	)`)
	if err != nil {
		t.Fatalf("create t_user_info: %v", err)
	}

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	// Set a login key for testing
	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-secret-key-for-jwt-signing"
	t.Cleanup(func() { comm.Cfg = origCfg })

	return db
}

func setupLoginRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	lc := &LoginController{}
	group := r.Group("/api/lg")
	lc.Routes(group)
	return r
}

func makeLoginTestToken(t *testing.T, uid string) string {
	t.Helper()
	token, err := util.CreateToken(jwt.MapClaims{
		"uid": uid,
	}, comm.Cfg.Server.LoginKey, time.Hour*24)
	if err != nil {
		t.Fatalf("create test token: %v", err)
	}
	return token
}

func TestLoginInfo_NoUser(t *testing.T) {
	setupLoginTestDB(t)
	r := setupLoginRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/lg/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result["login"] != false {
		t.Errorf("login = %v, want false", result["login"])
	}
}

func TestLoginInfo_WithValidToken(t *testing.T) {
	db := setupLoginTestDB(t)

	user := &model.TUser{
		Id:     "user-1",
		Name:   "alice",
		Nick:   "Alice",
		Active: 1,
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	r := setupLoginRouter(t)
	token := makeLoginTestToken(t, user.Id)

	req := httptest.NewRequest(http.MethodPost, "/api/lg/info", nil)
	req.Header.Set("Authorization", "TOKEN "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result["login"] != true {
		t.Errorf("login = %v, want true", result["login"])
	}
}

func TestLoginInfo_InvalidToken(t *testing.T) {
	setupLoginTestDB(t)
	r := setupLoginRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/lg/info", nil)
	req.Header.Set("Authorization", "TOKEN invalid-token-value")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result["login"] != false {
		t.Errorf("login = %v, want false", result["login"])
	}
}

func TestLogin_EmptyCredentials(t *testing.T) {
	setupLoginTestDB(t)
	r := setupLoginRouter(t)

	body := map[string]string{"name": "", "pass": ""}
	bts, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(bts))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	setupLoginTestDB(t)
	r := setupLoginRouter(t)

	body := map[string]string{"name": "nonexistent", "pass": "password"}
	bts, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(bts))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	db := setupLoginTestDB(t)

	user := &model.TUser{
		Id:     "user-inactive",
		Name:   "inactive",
		Nick:   "Inactive",
		Pass:   "somepasshash",
		Active: 0,
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	r := setupLoginRouter(t)

	body := map[string]string{"name": "inactive", "pass": "anypass"}
	bts, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(bts))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	db := setupLoginTestDB(t)

	user := &model.TUser{
		Id:     "user-wrongpass",
		Name:   "testuser",
		Nick:   "Test",
		Pass:   utils.Md5String("correctpass"),
		Active: 1,
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	r := setupLoginRouter(t)

	body := map[string]string{"name": "testuser", "pass": "wrongpass"}
	bts, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(bts))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogin_NoLoginKey(t *testing.T) {
	db := setupLoginTestDB(t)

	// Clear the login key
	comm.Cfg.Server.LoginKey = ""

	user := &model.TUser{
		Id:     "user-nokey",
		Name:   "nokey",
		Nick:   "NoKey",
		Pass:   utils.Md5String("hello"),
		Active: 1,
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	r := setupLoginRouter(t)

	body := map[string]string{"name": "nokey", "pass": "hello"}
	bts, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(bts))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestLogin_Success(t *testing.T) {
	db := setupLoginTestDB(t)

	user := &model.TUser{
		Id:        "user-success",
		Name:      "success",
		Nick:      "Success",
		Pass:      utils.Md5String("hello"),
		Active:    1,
		LoginTime: time.Now(),
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	r := setupLoginRouter(t)

	body := map[string]string{"name": "success", "pass": "hello"}
	bts, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(bts))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	// Verify token is present
	if token, ok := result["token"].(string); !ok || token == "" {
		t.Errorf("token missing or empty")
	}
	if result["id"] != "user-success" {
		t.Errorf("id = %v, want user-success", result["id"])
	}
	if result["name"] != "success" {
		t.Errorf("name = %v, want success", result["name"])
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	setupLoginTestDB(t)
	r := setupLoginRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLogin_AdminBypassesActiveCheck(t *testing.T) {
	db := setupLoginTestDB(t)

	// Admin user (id = "admin") should bypass active check
	user := &model.TUser{
		Id:     "admin",
		Name:   "admin",
		Nick:   "Admin",
		Pass:   utils.Md5String("adminpass"),
		Active: 0, // even though inactive
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	r := setupLoginRouter(t)

	body := map[string]string{"name": "admin", "pass": "adminpass"}
	bts, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(bts))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Admin should get 200 even when Active = 0
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestLogin_WhitespaceTrimmed(t *testing.T) {
	setupLoginTestDB(t)
	r := setupLoginRouter(t)

	// Name with whitespace should be trimmed
	body := map[string]string{"name": "   ", "pass": "password"}
	bts, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(bts))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
