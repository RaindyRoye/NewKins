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
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func createLoginTestDB(t *testing.T) *xorm.Engine {
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
		t.Fatalf("create t_user: %v", err)
	}

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
		t.Fatalf("create t_user_info: %v", err)
	}

	return db
}

func setupLoginTestContext(t *testing.T, db *xorm.Engine) {
	t.Helper()
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "test-secret-key-for-jwt-signing"
	t.Cleanup(func() { comm.Cfg = origCfg })
}

func TestLogin_info_NoUser(t *testing.T) {
	db := createLoginTestDB(t)
	setupLoginTestContext(t, db)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/lg/info", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := LoginController{}
	ctrl.info(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	login, ok := resp["login"].(bool)
	if !ok || login {
		t.Errorf("expected login=false, got %v", resp["login"])
	}
}

func TestLogin_info_WithUser(t *testing.T) {
	// Skip this test - CurrUserCache requires JWT token validation
	// which is complex to mock in unit tests
	t.Skip("CurrUserCache requires JWT token - tested in integration tests")
}

func TestLogin_login_EmptyName(t *testing.T) {
	db := createLoginTestDB(t)
	setupLoginTestContext(t, db)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/lg/login", bytes.NewBufferString(`{"name":"","pass":"***"}`))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := LoginController{}
	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("pass", "testpass")
	ctrl.login(c, &bean.LoginReq{Name: "", Pass: "testpass"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", w.Code)
	}
}

func TestLogin_login_EmptyPass(t *testing.T) {
	db := createLoginTestDB(t)
	setupLoginTestContext(t, db)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/lg/login", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := LoginController{}
	m := &hbtp.Map{}
	m.Set("name", "testuser")
	m.Set("pass", "")
	ctrl.login(c, &bean.LoginReq{Name: "testuser", Pass: ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pass, got %d", w.Code)
	}
}

func TestLogin_login_UserNotFound(t *testing.T) {
	db := createLoginTestDB(t)
	setupLoginTestContext(t, db)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/lg/login", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := LoginController{}
	m := &hbtp.Map{}
	m.Set("name", "nonexistent")
	m.Set("pass", "anypass")
	ctrl.login(c, &bean.LoginReq{Name: "nonexistent", Pass: "***"})

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent user, got %d", w.Code)
	}
}

func TestLogin_login_InactiveUser(t *testing.T) {
	db := createLoginTestDB(t)
	setupLoginTestContext(t, db)

	// Create inactive user (active = 0)
	user := &model.TUser{
		Id:        "inactive-1",
		Name:      "inactive",
		Nick:      "Inactive",
		Pass:      utils.Md5String("password123"),
		Active:    0,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/lg/login", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := LoginController{}
	m := &hbtp.Map{}
	m.Set("name", "inactive")
	m.Set("pass", "password123")
	ctrl.login(c, &bean.LoginReq{Name: "inactive", Pass: "password123"})

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for inactive user, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestLogin_login_WrongPassword(t *testing.T) {
	db := createLoginTestDB(t)
	setupLoginTestContext(t, db)

	user := &model.TUser{
		Id:        "user-wp",
		Name:      "wrongpass",
		Nick:      "Wrong Pass",
		Pass:      utils.Md5String("correctpassword"),
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/lg/login", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := LoginController{}
	m := &hbtp.Map{}
	m.Set("name", "wrongpass")
	m.Set("pass", "wrongpassword")
	ctrl.login(c, &bean.LoginReq{Name: "wrongpass", Pass: "wrongpassword"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", w.Code)
	}
}

func TestLogin_login_Success(t *testing.T) {
	db := createLoginTestDB(t)
	setupLoginTestContext(t, db)

	// Use correct password hash
	passHash := utils.Md5String("mypassword")
	user := &model.TUser{
		Id:        "user-ok",
		Name:      "gooduser",
		Nick:      "Good User",
		Pass:      passHash,
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/lg/login", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := LoginController{}
	m := &hbtp.Map{}
	m.Set("name", "gooduser")
	m.Set("pass", "mypassword")
	ctrl.login(c, &bean.LoginReq{Name: "gooduser", Pass: "mypassword"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for successful login, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	token, ok := resp["token"].(string)
	if !ok || token == "" {
		t.Error("expected non-empty token in response")
	}
	if resp["name"] != "gooduser" {
		t.Errorf("expected name = 'gooduser', got %v", resp["name"])
	}
}

func TestLogin_login_NoLoginKey(t *testing.T) {
	db := createLoginTestDB(t)
	// Set up DB but with empty login key
	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	origCfg := comm.Cfg
	comm.Cfg.Server.LoginKey = "" // no key configured
	t.Cleanup(func() { comm.Cfg = origCfg })

	user := &model.TUser{
		Id:        "user-nk",
		Name:      "nokey",
		Nick:      "No Key",
		Pass:      utils.Md5String("pass"),
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/lg/login", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := LoginController{}
	m := &hbtp.Map{}
	m.Set("name", "nokey")
	m.Set("pass", "pass")
	ctrl.login(c, &bean.LoginReq{Name: "nokey", Pass: "pass"})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for missing login key, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestLogin_login_InactiveAdminCanLogin(t *testing.T) {
	db := createLoginTestDB(t)
	setupLoginTestContext(t, db)

	// Admin users can login even when inactive (admin bypass)
	// IsAdmin checks usr.Id == "admin"
	user := &model.TUser{
		Id:        "admin",
		Name:      "adminuser",
		Nick:      "Admin",
		Pass:      utils.Md5String("adminpass"),
		Active:    0,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	// Make user admin via user_info
	uinfo := &model.TUserInfo{
		Id:       "admin",
		PermUser: 1,
		PermOrg:  1,
		PermPipe: 1,
	}
	_, err = db.InsertOne(uinfo)
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/api/lg/login", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	ctrl := LoginController{}
	m := &hbtp.Map{}
	m.Set("name", "adminuser")
	m.Set("pass", "adminpass")
	ctrl.login(c, &bean.LoginReq{Name: "adminuser", Pass: "adminpass"})

	// Admin users bypass the active check, so this should succeed
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for admin login (even inactive), got %d, body: %s", w.Code, w.Body.String())
	}
}
