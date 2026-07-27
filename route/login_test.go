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
	"github.com/gokins/gokins/util"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupLoginTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create the t_user table
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

	// Create the t_user_info table
	_, err = db.Exec(`CREATE TABLE t_user_info (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		phone VARCHAR(100),
		email VARCHAR(200),
		birthday DATETIME,
		remark TEXT,
		perm_user INT,
		perm_org INT,
		perm_pipe INT
	)`)
	if err != nil {
		t.Fatalf("failed to create t_user_info table: %v", err)
	}

	comm.Db = db
}

func createLoginTestUser(t *testing.T, name, nick, pass string, active int) *model.TUser {
	t.Helper()
	user := &model.TUser{
		Id:        time.Now().Format("20060102150405.000000"),
		Name:      name,
		Nick:      nick,
		Pass:      utils.Md5String(pass),
		Active:    active,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := comm.Db.InsertOne(user)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func makeLoginGinContext(t *testing.T, body interface{}, token string) (*gin.Context, *httptest.ResponseRecorder) {
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
	if token != "" {
		req.Header.Set("Authorization", "TOKEN "+token)
	}
	c.Request = req
	return c, w
}

func generateTestToken(userId string, loginKey string) string {
	token, _ := util.CreateToken(
		map[string]interface{}{"uid": userId},
		loginKey,
		time.Hour*24,
	)
	return token
}

func TestLoginController_GetPathMethod(t *testing.T) {
	ctrl := LoginController{}
	if got := ctrl.GetPath(); got != "/api/lg" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/lg")
	}
}

func TestLoginController_info_NoAuth(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	// Set up login key
	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key-123"
	t.Cleanup(func() { comm.Cfg = origCfg })

	c, w := makeLoginGinContext(t, nil, "")
	ctrl.info(c)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}

	var resp hbtp.Map
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if login, ok := resp["login"].(bool); !ok || login {
		t.Errorf("login = %v, want false", resp["login"])
	}
}

func TestLoginController_info_WithValidToken(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	// Set up login key
	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key-123"
	t.Cleanup(func() { comm.Cfg = origCfg })

	// Create a test user
	user := createLoginTestUser(t, "testuser", "Test User", "password123", 1)

	// Generate token
	token := generateTestToken(user.Id, "test-key-123")

	c, w := makeLoginGinContext(t, nil, token)
	ctrl.info(c)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp hbtp.Map
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if login, ok := resp["login"].(bool); !ok || !login {
		t.Errorf("login = %v, want true", resp["login"])
	}
}

func TestLoginController_info_AdminUser(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	// Set up login key
	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key-123"
	t.Cleanup(func() { comm.Cfg = origCfg })

	// Create admin user
	admin := &model.TUser{
		Id:        "admin",
		Name:      "admin",
		Nick:      "Administrator",
		Pass:      utils.Md5String("admin123"),
		Active:    0, // Inactive
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := comm.Db.InsertOne(admin)
	if err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}

	// Create admin user info with full permissions
	adminInfo := &model.TUserInfo{
		Id:       "admin",
		PermUser: 1,
		PermOrg:  1,
		PermPipe: 1,
	}
	_, err = comm.Db.InsertOne(adminInfo)
	if err != nil {
		t.Fatalf("failed to create admin user info: %v", err)
	}

	// Generate token
	token := generateTestToken("admin", "test-key-123")

	c, w := makeLoginGinContext(t, nil, token)
	ctrl.info(c)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify response contains login=true
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if login, ok := resp["login"].(bool); !ok || !login {
		t.Errorf("login = %v, want true", resp["login"])
	}

	// Verify user is present
	if _, ok := resp["user"]; !ok {
		t.Error("expected user field in response")
	}
}

func TestLoginController_login_Success(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	// Set up login key
	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key-123"
	t.Cleanup(func() { comm.Cfg = origCfg })

	// Create a test user
	user := createLoginTestUser(t, "testuser", "Test User", "password123", 1)

	// Login request
	loginReq := &bean.LoginReq{
		Name: "testuser",
		Pass: "password123",
	}

	c, w := makeLoginGinContext(t, loginReq, "")
	ctrl.login(c, loginReq)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp bean.LoginRes
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.Id != user.Id {
		t.Errorf("response id = %q, want %q", resp.Id, user.Id)
	}
	if resp.Name != user.Name {
		t.Errorf("response name = %q, want %q", resp.Name, user.Name)
	}
}

func TestLoginController_login_EmptyUsername(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	loginReq := &bean.LoginReq{
		Name: "",
		Pass: "password123",
	}

	c, w := makeLoginGinContext(t, loginReq, "")
	ctrl.login(c, loginReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginController_login_EmptyPassword(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	loginReq := &bean.LoginReq{
		Name: "testuser",
		Pass: "",
	}

	c, w := makeLoginGinContext(t, loginReq, "")
	ctrl.login(c, loginReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginController_login_UserNotFound(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	loginReq := &bean.LoginReq{
		Name: "nonexistent",
		Pass: "password123",
	}

	c, w := makeLoginGinContext(t, loginReq, "")
	ctrl.login(c, loginReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestLoginController_login_WrongPassword(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	// Create a test user
	createLoginTestUser(t, "testuser", "Test User", "password123", 1)

	// Login with wrong password
	loginReq := &bean.LoginReq{
		Name: "testuser",
		Pass: "wrongpassword",
	}

	c, w := makeLoginGinContext(t, loginReq, "")
	ctrl.login(c, loginReq)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLoginController_login_InactiveUser(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	// Create an inactive user
	createLoginTestUser(t, "inactiveuser", "Inactive User", "password123", 0)

	// Login attempt
	loginReq := &bean.LoginReq{
		Name: "inactiveuser",
		Pass: "password123",
	}

	c, w := makeLoginGinContext(t, loginReq, "")
	ctrl.login(c, loginReq)

	if w.Code != http.StatusForbidden {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestLoginController_login_AdminInactiveUser(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	// Create inactive admin user
	admin := &model.TUser{
		Id:        "admin",
		Name:      "admin",
		Nick:      "Administrator",
		Pass:      utils.Md5String("admin123"),
		Active:    0, // Inactive
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := comm.Db.InsertOne(admin)
	if err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}

	// Admin should be able to login even when inactive
	loginReq := &bean.LoginReq{
		Name: "admin",
		Pass: "admin123",
	}

	// Set up login key
	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key-123"
	t.Cleanup(func() { comm.Cfg = origCfg })

	c, w := makeLoginGinContext(t, loginReq, "")
	ctrl.login(c, loginReq)

	// Admin bypasses active check
	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestLoginController_login_NoLoginKey(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	// Create a test user
	createLoginTestUser(t, "testuser", "Test User", "password123", 1)

	// Clear login key
	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = ""
	t.Cleanup(func() { comm.Cfg = origCfg })

	// Login request
	loginReq := &bean.LoginReq{
		Name: "testuser",
		Pass: "password123",
	}

	c, w := makeLoginGinContext(t, loginReq, "")
	ctrl.login(c, loginReq)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestLoginController_login_WhitespaceUsername(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	// Create a test user
	createLoginTestUser(t, "testuser", "Test User", "password123", 1)

	// Login with whitespace in username
	loginReq := &bean.LoginReq{
		Name: "  testuser  ",
		Pass: "password123",
	}

	// Set up login key
	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-key-123"
	t.Cleanup(func() { comm.Cfg = origCfg })

	c, w := makeLoginGinContext(t, loginReq, "")
	ctrl.login(c, loginReq)

	// Should trim whitespace and succeed
	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
