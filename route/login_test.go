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
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupLoginTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
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
		t.Fatalf("create t_user table: %v", err)
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
		t.Fatalf("create t_user_info table: %v", err)
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
		t.Fatalf("create test user: %v", err)
	}
	return user
}

func makeLoginGinCtx(t *testing.T, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

	return c, w
}

func TestLogin_info_NoAuth(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)

	ctrl.info(c)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
	// Should return login=false
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if login, ok := resp["login"].(bool); !ok || login {
		t.Errorf("expected login=false, got %v", resp["login"])
	}
}

func TestLogin_info_WithValidToken(t *testing.T) {
	setupLoginTestDB(t)
	user := createLoginTestUser(t, "testuser", "Test User", "pass123", 1)

	// Generate a valid token
	comm.Cfg.Server.LoginKey = "test-secret-key-for-unit-tests"
	token, err := util.CreateToken(jwt.MapClaims{
		"uid": user.Id,
	}, comm.Cfg.Server.LoginKey, time.Hour*24)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	ctrl := LoginController{}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "TOKEN "+token)
	c.Request = req

	ctrl.info(c)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if login, ok := resp["login"].(bool); !ok || !login {
		t.Errorf("expected login=true, got %v", resp["login"])
	}
}

func TestLogin_login_EmptyCredentials(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	c, w := makeLoginGinCtx(t, &bean.LoginReq{Name: "", Pass: ""})
	ctrl.login(c, &bean.LoginReq{Name: "", Pass: ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLogin_login_UserNotFound(t *testing.T) {
	setupLoginTestDB(t)
	ctrl := LoginController{}

	c, w := makeLoginGinCtx(t, &bean.LoginReq{Name: "nonexistent", Pass: "pass123"})
	ctrl.login(c, &bean.LoginReq{Name: "nonexistent", Pass: "pass123"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestLogin_login_InactiveUser(t *testing.T) {
	setupLoginTestDB(t)
	createLoginTestUser(t, "inactive", "Inactive User", "pass123", 0)
	ctrl := LoginController{}

	c, w := makeLoginGinCtx(t, &bean.LoginReq{Name: "inactive", Pass: "pass123"})
	ctrl.login(c, &bean.LoginReq{Name: "inactive", Pass: "pass123"})

	if w.Code != http.StatusForbidden {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestLogin_login_WrongPassword(t *testing.T) {
	setupLoginTestDB(t)
	createLoginTestUser(t, "activeuser", "Active User", "correctpass", 1)
	ctrl := LoginController{}

	c, w := makeLoginGinCtx(t, &bean.LoginReq{Name: "activeuser", Pass: "wrongpass"})
	ctrl.login(c, &bean.LoginReq{Name: "activeuser", Pass: "wrongpass"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogin_login_Success(t *testing.T) {
	setupLoginTestDB(t)
	comm.Cfg.Server.LoginKey = "test-secret-key-for-unit-tests"
	createLoginTestUser(t, "validuser", "Valid User", "correctpass", 1)
	ctrl := LoginController{}

	c, w := makeLoginGinCtx(t, &bean.LoginReq{Name: "validuser", Pass: "correctpass"})
	ctrl.login(c, &bean.LoginReq{Name: "validuser", Pass: "correctpass"})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp bean.LoginRes
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token in response")
	}
	if resp.Name != "validuser" {
		t.Errorf("expected name=validuser, got %s", resp.Name)
	}
}
