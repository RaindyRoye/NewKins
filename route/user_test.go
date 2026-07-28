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

func setupUserTestDB(t *testing.T) {
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

func generateTestID() string {
	return time.Now().Format("20060102150405.000000")
}

func createUserForTest(t *testing.T, name, nick, pass string, active int) *model.TUser {
	t.Helper()
	user := &model.TUser{
		Id:        generateTestID(),
		Name:      name,
		Nick:      nick,
		Pass:      pass,
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

func makeGinContext(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestUserController_page_EmptyDB(t *testing.T) {
	setupUserTestDB(t)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeGinContext(t, m, nil)
	ctrl.page(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUserController_page_WithUsers(t *testing.T) {
	setupUserTestDB(t)
	createUserForTest(t, "alice", "Alice", "hash1", 1)
	createUserForTest(t, "bob", "Bob", "hash2", 1)

	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makeGinContext(t, m, nil)
	ctrl := UserController{}
	ctrl.page(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUserController_page_WithSearch(t *testing.T) {
	setupUserTestDB(t)
	createUserForTest(t, "alice", "Alice Smith", "hash1", 1)
	createUserForTest(t, "bob", "Bob Jones", "hash2", 1)

	m := &hbtp.Map{}
	m.Set("q", "alice")
	m.Set("page", int64(1))
	c, w := makeGinContext(t, m, nil)
	ctrl := UserController{}
	ctrl.page(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestUserController_info_MissingID(t *testing.T) {
	setupUserTestDB(t)
	c, w := makeGinContext(t, hbtp.Map{"id": ""}, nil)
	ctrl := UserController{}
	ctrl.info(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_info_UserNotFound(t *testing.T) {
	setupUserTestDB(t)
	c, w := makeGinContext(t, hbtp.Map{"id": "nonexistent"}, nil)
	ctrl := UserController{}
	ctrl.info(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUserController_info_Success(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "testuser", "Test User", "hash", 1)

	c, w := makeGinContext(t, hbtp.Map{"id": user.Id}, nil)
	ctrl := UserController{}
	ctrl.info(c, &hbtp.Map{"id": user.Id})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUserController_upinfo_MissingParams(t *testing.T) {
	setupUserTestDB(t)
	c, w := makeGinContext(t, hbtp.Map{"id": "", "nick": "Updated"}, nil)
	ctrl := UserController{}
	ctrl.upinfo(c, &hbtp.Map{"id": "", "nick": "Updated"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_upinfo_UserNotFound(t *testing.T) {
	setupUserTestDB(t)
	c, w := makeGinContext(t, hbtp.Map{
		"id": "nonexistent", "nick": "Updated",
		"phone": "123", "email": "test@example.com",
	}, nil)
	ctrl := UserController{}
	ctrl.upinfo(c, &hbtp.Map{
		"id": "nonexistent", "nick": "Updated",
		"phone": "123", "email": "test@example.com",
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUserController_upass_MissingParams(t *testing.T) {
	setupUserTestDB(t)
	c, w := makeGinContext(t, hbtp.Map{"id": "", "pass": ""}, nil)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": "", "pass": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_upass_UserNotFound(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, hbtp.Map{"id": "nonexistent", "pass": "newpass"}, adminUser)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": "nonexistent", "pass": "newpass"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUserController_upass_NotAdminChangingOtherUser(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "testuser", "Test User", "hash", 1)
	regularUser := &model.TUser{Id: "regular", Name: "regular", Active: 1}

	c, w := makeGinContext(t, hbtp.Map{"id": user.Id, "pass": "newpass"}, regularUser)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": user.Id, "pass": "newpass"})

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUserController_upass_SelfUpdateMissingOldPass(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "testuser", "Test User", "hash", 1)

	c, w := makeGinContext(t, hbtp.Map{"id": user.Id, "pass": "newpass"}, user)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": user.Id, "pass": "newpass"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_upass_SelfUpdateWrongOldPass(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "testuser", "Test User", "hash", 1)

	c, w := makeGinContext(t, hbtp.Map{"id": user.Id, "pass": "newpass", "olds": "wrongold"}, user)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": user.Id, "pass": "newpass", "olds": "wrongold"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestUserController_upass_SelfUpdateSuccess(t *testing.T) {
	setupUserTestDB(t)
	// Create user with known MD5-hashed password
	oldPass := "oldpass123"
	user := createUserForTest(t, "testuser", "Test User", utils.Md5String(oldPass), 1)

	c, w := makeGinContext(t, hbtp.Map{"id": user.Id, "pass": "newpass", "olds": oldPass}, user)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": user.Id, "pass": "newpass", "olds": oldPass})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify password was updated
	updated := &model.TUser{}
	ok, err := comm.Db.Where("id=?", user.Id).Get(updated)
	if err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after update")
	}
	// The new password should be hashed (different from original)
	if updated.Pass == utils.Md5String(oldPass) {
		t.Error("password was not updated")
	}
}

func TestUserController_upass_AdminUpdateOtherUser(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "testuser", "Test User", "hash", 1)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeGinContext(t, hbtp.Map{"id": user.Id, "pass": "newpass"}, adminUser)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": user.Id, "pass": "newpass"})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify password was updated
	updated := &model.TUser{}
	ok, err := comm.Db.Where("id=?", user.Id).Get(updated)
	if err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after update")
	}
	// The new password should be hashed
	if updated.Pass == "hash" {
		t.Error("password was not updated")
	}
}

func TestUserController_active_MissingParams(t *testing.T) {
	setupUserTestDB(t)
	c, w := makeGinContext(t, hbtp.Map{"id": "", "act": ""}, nil)
	ctrl := UserController{}
	ctrl.active(c, &hbtp.Map{"id": "", "act": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_perm_MissingID(t *testing.T) {
	setupUserTestDB(t)
	c, w := makeGinContext(t, hbtp.Map{"id": ""}, nil)
	ctrl := UserController{}
	ctrl.perm(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_perm_UserNotFound(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, hbtp.Map{"id": "nonexistent", "permUser": true}, adminUser)
	ctrl := UserController{}
	ctrl.perm(c, &hbtp.Map{"id": "nonexistent", "permUser": true})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUserController_active_Success(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "actuser", "Active User", "hash", 0)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeGinContext(t, hbtp.Map{"id": user.Id, "act": "1"}, adminUser)
	ctrl := UserController{}
	ctrl.active(c, &hbtp.Map{"id": user.Id, "act": "1"})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify the user is now active
	updated := &model.TUser{}
	ok, err := comm.Db.Where("id=?", user.Id).Get(updated)
	if err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after update")
	}
	if updated.Active != 1 {
		t.Errorf("user active = %d, want 1", updated.Active)
	}
}

func TestUserController_perm_Success(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "permuser", "Perm User", "hash", 1)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeGinContext(t, hbtp.Map{
		"id": user.Id, "permUser": true, "permOrg": false, "permPipe": true,
	}, adminUser)
	ctrl := UserController{}
	ctrl.perm(c, &hbtp.Map{
		"id": user.Id, "permUser": true, "permOrg": false, "permPipe": true,
	})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify user info was created with correct permissions
	uinfo := &model.TUserInfo{}
	ok, err := comm.Db.Where("id=?", user.Id).Get(uinfo)
	if err != nil {
		t.Fatalf("query user info: %v", err)
	}
	if !ok {
		t.Fatal("user info not found after perm update")
	}
	if uinfo.PermUser != 1 {
		t.Errorf("PermUser = %d, want 1", uinfo.PermUser)
	}
	if uinfo.PermOrg != 0 {
		t.Errorf("PermOrg = %d, want 0", uinfo.PermOrg)
	}
	if uinfo.PermPipe != 1 {
		t.Errorf("PermPipe = %d, want 1", uinfo.PermPipe)
	}
}

func TestUserController_upinfo_Success(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "upuser", "Old Nick", "hash", 1)
	loggedInUser := &model.TUser{Id: user.Id, Name: user.Name, Active: 1}

	c, w := makeGinContext(t, hbtp.Map{
		"id": user.Id, "nick": "New Nick",
		"phone": "555-1234", "email": "test@example.com", "remark": "A remark",
	}, loggedInUser)
	ctrl := UserController{}
	ctrl.upinfo(c, &hbtp.Map{
		"id": user.Id, "nick": "New Nick",
		"phone": "555-1234", "email": "test@example.com", "remark": "A remark",
	})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify nick was updated
	updated := &model.TUser{}
	ok, err := comm.Db.Where("id=?", user.Id).Get(updated)
	if err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after update")
	}
	if updated.Nick != "New Nick" {
		t.Errorf("user nick = %q, want %q", updated.Nick, "New Nick")
	}
}
