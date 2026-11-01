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
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupUserHandlerTestDB(t *testing.T) {
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

func createTestUser(t *testing.T, name, nick, pass string, active int) *model.TUser { //nolint:unparam // pass is kept for API flexibility
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

func makeUserHandlerGinCtx(t *testing.T, body interface{}, lguser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

	// Set login user
	c.Set(service.LgUserKey, lguser)
	return c, w
}

func TestUser_new_EmptyParams(t *testing.T) {
	setupUserHandlerTestDB(t)
	ctrl := UserController{}

	adminUser := createTestUser(t, "admin", "Admin", "pass123", 1)
	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("nick", "Test")
	m.Set("pass", "pass123")

	c, w := makeUserHandlerGinCtx(t, m, adminUser)
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUser_new_DuplicateUser(t *testing.T) {
	setupUserHandlerTestDB(t)
	ctrl := UserController{}

	// Create admin user with perm_user=1
	adminUser := createTestUser(t, "admin", "Admin", "pass123", 1)
	uinfo := &model.TUserInfo{
		Id:       adminUser.Id,
		PermUser: 1,
	}
	_, err := comm.Db.InsertOne(uinfo)
	if err != nil {
		t.Fatalf("create user info: %v", err)
	}

	// Create existing user
	createTestUser(t, "existing", "Existing", "pass123", 1)

	m := &hbtp.Map{}
	m.Set("name", "existing")
	m.Set("nick", "New User")
	m.Set("pass", "pass123")

	c, w := makeUserHandlerGinCtx(t, m, adminUser)
	ctrl.new(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestUser_new_Success(t *testing.T) {
	setupUserHandlerTestDB(t)
	ctrl := UserController{}

	// Create admin user with perm_user=1
	adminUser := createTestUser(t, "admin", "Admin", "pass123", 1)
	uinfo := &model.TUserInfo{
		Id:       adminUser.Id,
		PermUser: 1,
	}
	_, err := comm.Db.InsertOne(uinfo)
	if err != nil {
		t.Fatalf("create user info: %v", err)
	}

	m := &hbtp.Map{}
	m.Set("name", "newuser")
	m.Set("nick", "New User")
	m.Set("pass", "pass123")

	c, w := makeUserHandlerGinCtx(t, m, adminUser)
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUser_info_EmptyId(t *testing.T) {
	setupUserHandlerTestDB(t)
	ctrl := UserController{}

	adminUser := createTestUser(t, "admin", "Admin", "pass123", 1)
	m := &hbtp.Map{}
	m.Set("id", "")

	c, w := makeUserHandlerGinCtx(t, m, adminUser)
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUser_info_NotFound(t *testing.T) {
	setupUserHandlerTestDB(t)
	ctrl := UserController{}

	adminUser := createTestUser(t, "admin", "Admin", "pass123", 1)
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")

	c, w := makeUserHandlerGinCtx(t, m, adminUser)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUser_info_Success(t *testing.T) {
	setupUserHandlerTestDB(t)
	ctrl := UserController{}

	targetUser := createTestUser(t, "target", "Target User", "pass123", 1)
	adminUser := createTestUser(t, "admin", "Admin", "pass123", 1)

	m := &hbtp.Map{}
	m.Set("id", targetUser.Id)

	c, w := makeUserHandlerGinCtx(t, m, adminUser)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
}
