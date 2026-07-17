package route

import (
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
	"xorm.io/xorm"
)

func setupLoginTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	origCtx := comm.Ctx
	t.Cleanup(func() {
		comm.Db = origDb
		comm.Ctx = origCtx
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

	comm.Db = db
	comm.Ctx = t.Context()
}

func makeLoginGinContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, w
}

func TestLoginController_info_NoLogin(t *testing.T) {
	setupLoginTestDB(t)
	origCfg := comm.Cfg
	t.Cleanup(func() { comm.Cfg = origCfg })
	comm.Cfg.Server.LoginKey = "test-login-key"

	c, w := makeLoginGinContext(t)
	ctrl := LoginController{}
	ctrl.info(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["login"] != false {
		t.Errorf("login = %v, want false", resp["login"])
	}
}

func TestLoginController_login_EmptyName(t *testing.T) {
	setupLoginTestDB(t)
	c, w := makeLoginGinContext(t)
	ctrl := LoginController{}
	ctrl.login(c, &bean.LoginReq{Name: "", Pass: "secret"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginController_login_EmptyPassword(t *testing.T) {
	setupLoginTestDB(t)
	c, w := makeLoginGinContext(t)
	ctrl := LoginController{}
	ctrl.login(c, &bean.LoginReq{Name: "alice", Pass: ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginController_login_BothEmpty(t *testing.T) {
	setupLoginTestDB(t)
	c, w := makeLoginGinContext(t)
	ctrl := LoginController{}
	ctrl.login(c, &bean.LoginReq{Name: "", Pass: ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginController_login_WhitespaceName(t *testing.T) {
	setupLoginTestDB(t)
	c, w := makeLoginGinContext(t)
	ctrl := LoginController{}
	ctrl.login(c, &bean.LoginReq{Name: "   ", Pass: "secret"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginController_login_UserNotFound(t *testing.T) {
	setupLoginTestDB(t)
	c, w := makeLoginGinContext(t)
	ctrl := LoginController{}
	ctrl.login(c, &bean.LoginReq{Name: "nonexistent", Pass: "password"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestLoginController_login_InactiveUser(t *testing.T) {
	setupLoginTestDB(t)
	_, err := comm.Db.Insert(&model.TUser{
		Id:        "inactive-user",
		Name:      "inactive",
		Pass:      "somehash",
		Nick:      "Inactive User",
		Active:    0,
		Created:   time.Now(),
		LoginTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, w := makeLoginGinContext(t)
	ctrl := LoginController{}
	ctrl.login(c, &bean.LoginReq{Name: "inactive", Pass: "password"})

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestLoginController_login_WrongPassword(t *testing.T) {
	setupLoginTestDB(t)
	_, err := comm.Db.Insert(&model.TUser{
		Id:        "active-user",
		Name:      "activeuser",
		Pass:      "wrong_hash_value",
		Nick:      "Active User",
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, w := makeLoginGinContext(t)
	ctrl := LoginController{}
	ctrl.login(c, &bean.LoginReq{Name: "activeuser", Pass: "password"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestLoginController_login_NoLoginKey(t *testing.T) {
	setupLoginTestDB(t)
	origCfg := comm.Cfg
	t.Cleanup(func() { comm.Cfg = origCfg })
	comm.Cfg.Server.LoginKey = ""

	// utils.Md5String("password") = "5f4dcc3b5aa765d61d8327deb882cf99"
	_, err := comm.Db.Insert(&model.TUser{
		Id:        "good-user",
		Name:      "gooduser",
		Pass:      "5f4dcc3b5aa765d61d8327deb882cf99",
		Nick:      "Good User",
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, w := makeLoginGinContext(t)
	ctrl := LoginController{}
	ctrl.login(c, &bean.LoginReq{Name: "gooduser", Pass: "password"})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusInternalServerError, w.Body.String())
	}
}

func TestLoginController_login_Success(t *testing.T) {
	setupLoginTestDB(t)
	origCfg := comm.Cfg
	t.Cleanup(func() { comm.Cfg = origCfg })
	comm.Cfg.Server.LoginKey = "test-secret-key-for-jwt"

	_, err := comm.Db.Insert(&model.TUser{
		Id:        "success-user",
		Name:      "successuser",
		Pass:      "5f4dcc3b5aa765d61d8327deb882cf99",
		Nick:      "Success User",
		Avatar:    "avatar.png",
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, w := makeLoginGinContext(t)
	ctrl := LoginController{}
	ctrl.login(c, &bean.LoginReq{Name: "successuser", Pass: "password"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp bean.LoginRes
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Token == "" {
		t.Error("response should contain a token")
	}
	if resp.Name != "successuser" {
		t.Errorf("name = %q, want %q", resp.Name, "successuser")
	}
	if resp.Nick != "Success User" {
		t.Errorf("nick = %q, want %q", resp.Nick, "Success User")
	}
	if resp.Id != "success-user" {
		t.Errorf("id = %q, want %q", resp.Id, "success-user")
	}
}

func TestLoginController_login_AdminUser(t *testing.T) {
	setupLoginTestDB(t)
	origCfg := comm.Cfg
	t.Cleanup(func() { comm.Cfg = origCfg })
	comm.Cfg.Server.LoginKey = "admin-jwt-key"

	_, err := comm.Db.Insert(&model.TUser{
		Id:        "admin",
		Name:      "admin",
		Pass:      "5f4dcc3b5aa765d61d8327deb882cf99",
		Nick:      "Admin",
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	c, w := makeLoginGinContext(t)
	ctrl := LoginController{}
	ctrl.login(c, &bean.LoginReq{Name: "admin", Pass: "password"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestGetMidLgUser_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	c.Request = req

	user := service.GetMidLgUser(c)
	if user != nil {
		t.Error("GetMidLgUser should return nil when not set")
	}
}

func TestGetMidLgUser_Set(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	c.Request = req

	expected := &model.TUser{Id: "test-user"}
	c.Set(service.LgUserKey, expected)

	user := service.GetMidLgUser(c)
	if user == nil {
		t.Fatal("GetMidLgUser should return user")
	}
	if user.Id != "test-user" {
		t.Errorf("user.Id = %q, want %q", user.Id, "test-user")
	}
}

func TestGetMidLgUser_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "/", nil)
	c.Request = req

	c.Set(service.LgUserKey, "not a user")

	user := service.GetMidLgUser(c)
	if user != nil {
		t.Error("GetMidLgUser should return nil for wrong type")
	}
}
