package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

// Helper to create a gin context with a logged-in user
func makeUserGinCtx(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestUserController_new_EmptyParams(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"name": "", "nick": "", "pass": ""}, adminUser)
	ctrl := UserController{}
	ctrl.new(c, &hbtp.Map{"name": "", "nick": "", "pass": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

func TestUserController_new_UserAlreadyExistsExtra(t *testing.T) {
	setupUserTestDB(t)
	// Create an existing user
	createUserForTest(t, "existing", "Existing", "pass123", 1)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"name": "existing", "nick": "New", "pass": "pass"}, adminUser)
	ctrl := UserController{}
	ctrl.new(c, &hbtp.Map{"name": "existing", "nick": "New", "pass": "pass"})

	if w.Code != http.StatusConflict {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestUserController_new_SuccessExtra(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"name": "newuser", "nick": "New User", "pass": "secret"}, adminUser)
	ctrl := UserController{}
	ctrl.new(c, &hbtp.Map{"name": "newuser", "nick": "New User", "pass": "secret"})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	// Should return the new user ID
	if w.Body.String() == "" {
		t.Error("expected non-empty user ID in response")
	}
}

func TestUserController_new_NonAdminNoPermission(t *testing.T) {
	setupUserTestDB(t)
	// Create a non-admin user without user management permission
	normalUser := &model.TUser{Id: "normal", Name: "normal", Active: 1}
	_, err := comm.Db.InsertOne(normalUser)
	if err != nil {
		t.Fatalf("insert normal user: %v", err)
	}
	// Create user_info with no permissions
	userInfo := &model.TUserInfo{Id: normalUser.Id, PermUser: 0}
	_, err = comm.Db.InsertOne(userInfo)
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	c, w := makeUserGinCtx(t, hbtp.Map{"name": "test", "nick": "Test", "pass": "pass"}, normalUser)
	ctrl := UserController{}
	ctrl.new(c, &hbtp.Map{"name": "test", "nick": "Test", "pass": "pass"})

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUserController_page_EmptyQuery(t *testing.T) {
	setupUserTestDB(t)
	// Create some test users
	for i := 0; i < 3; i++ {
		createUserForTest(t, "user"+string(rune('a'+i)), "User "+string(rune('A'+i)), "pass", 1)
	}

	c, w := makeUserGinCtx(t, hbtp.Map{"q": "", "page": int64(1)}, nil)
	ctrl := UserController{}
	ctrl.page(c, &hbtp.Map{"q": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["data"] == nil {
		t.Error("expected 'data' in response")
	}
}

func TestUserController_page_WithQuery(t *testing.T) {
	setupUserTestDB(t)
	createUserForTest(t, "alice", "Alice", "pass", 1)
	createUserForTest(t, "bob", "Bob", "pass", 1)

	c, w := makeUserGinCtx(t, hbtp.Map{"q": "alice", "page": int64(1)}, nil)
	ctrl := UserController{}
	ctrl.page(c, &hbtp.Map{"q": "alice", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUserController_upass_EmptyParams(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"id": "", "pass": ""}, adminUser)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": "", "pass": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_upass_UserNotFoundExtra(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"id": "nonexistent", "pass": "newpass"}, adminUser)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": "nonexistent", "pass": "newpass"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUserController_upass_SelfChangeWithoutOldPassword(t *testing.T) {
	setupUserTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Pass: utils.Md5String("oldpass"), Active: 1}
	_, err := comm.Db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, w := makeUserGinCtx(t, hbtp.Map{"id": "user1", "olds": "", "pass": "newpass"}, user)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": "user1", "olds": "", "pass": "newpass"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUserController_upass_SelfChangeWrongOldPassword(t *testing.T) {
	setupUserTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Pass: utils.Md5String("correctold"), Active: 1}
	_, err := comm.Db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, w := makeUserGinCtx(t, hbtp.Map{"id": "user1", "olds": "wrongold", "pass": "newpass"}, user)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": "user1", "olds": "wrongold", "pass": "newpass"})

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestUserController_upass_SelfChangeSuccessExtra(t *testing.T) {
	setupUserTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Pass: utils.Md5String("oldpass"), Active: 1}
	_, err := comm.Db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, w := makeUserGinCtx(t, hbtp.Map{"id": "user1", "olds": "oldpass", "pass": "newpass"}, user)
	ctrl := UserController{}
	ctrl.upass(c, &hbtp.Map{"id": "user1", "olds": "oldpass", "pass": "newpass"})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUserController_active_EmptyParams(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"id": "", "act": ""}, adminUser)
	ctrl := UserController{}
	ctrl.active(c, &hbtp.Map{"id": "", "act": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_active_NonAdmin(t *testing.T) {
	setupUserTestDB(t)
	normalUser := &model.TUser{Id: "normal", Name: "normal", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"id": "user1", "act": "1"}, normalUser)
	ctrl := UserController{}
	ctrl.active(c, &hbtp.Map{"id": "user1", "act": "1"})

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestUserController_active_UserNotFoundExtra(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"id": "nonexistent", "act": "1"}, adminUser)
	ctrl := UserController{}
	ctrl.active(c, &hbtp.Map{"id": "nonexistent", "act": "1"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUserController_active_ActivateUser(t *testing.T) {
	setupUserTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 0}
	_, err := comm.Db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"id": "user1", "act": "1"}, adminUser)
	ctrl := UserController{}
	ctrl.active(c, &hbtp.Map{"id": "user1", "act": "1"})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify user is now active
	updatedUser := &model.TUser{}
	ok, err := comm.Db.Where("id=?", "user1").Get(updatedUser)
	if err != nil || !ok {
		t.Fatalf("query user: %v", err)
	}
	if updatedUser.Active != 1 {
		t.Errorf("user.Active = %d, want 1", updatedUser.Active)
	}
}

func TestUserController_active_DeactivateUser(t *testing.T) {
	setupUserTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	_, err := comm.Db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"id": "user1", "act": "0"}, adminUser)
	ctrl := UserController{}
	ctrl.active(c, &hbtp.Map{"id": "user1", "act": "0"})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify user is now inactive
	updatedUser := &model.TUser{}
	ok, err := comm.Db.Where("id=?", "user1").Get(updatedUser)
	if err != nil || !ok {
		t.Fatalf("query user: %v", err)
	}
	if updatedUser.Active != 0 {
		t.Errorf("user.Active = %d, want 0", updatedUser.Active)
	}
}

func TestUserController_perm_EmptyId(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"id": ""}, adminUser)
	ctrl := UserController{}
	ctrl.perm(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_perm_UserNotFoundExtra(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"id": "nonexistent", "permUser": true}, adminUser)
	ctrl := UserController{}
	ctrl.perm(c, &hbtp.Map{"id": "nonexistent", "permUser": true})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUserController_perm_UpdatePermissionsExtra(t *testing.T) {
	setupUserTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	_, err := comm.Db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	// Create existing user_info
	userInfo := &model.TUserInfo{Id: "user1", PermUser: 0, PermOrg: 0, PermPipe: 0}
	_, err = comm.Db.InsertOne(userInfo)
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"id": "user1", "permUser": true, "permOrg": true, "permPipe": false}, adminUser)
	ctrl := UserController{}
	ctrl.perm(c, &hbtp.Map{"id": "user1", "permUser": true, "permOrg": true, "permPipe": false})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify permissions updated
	updatedInfo := &model.TUserInfo{}
	ok, err := comm.Db.Where("id=?", "user1").Get(updatedInfo)
	if err != nil || !ok {
		t.Fatalf("query user info: %v", err)
	}
	if updatedInfo.PermUser != 1 {
		t.Errorf("PermUser = %d, want 1", updatedInfo.PermUser)
	}
	if updatedInfo.PermOrg != 1 {
		t.Errorf("PermOrg = %d, want 1", updatedInfo.PermOrg)
	}
	if updatedInfo.PermPipe != 0 {
		t.Errorf("PermPipe = %d, want 0", updatedInfo.PermPipe)
	}
}

func TestUserController_perm_CreateNewUserInfoExtra(t *testing.T) {
	setupUserTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	_, err := comm.Db.InsertOne(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	// Don't create user_info - it should be created

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeUserGinCtx(t, hbtp.Map{"id": "user1", "permUser": true}, adminUser)
	ctrl := UserController{}
	ctrl.perm(c, &hbtp.Map{"id": "user1", "permUser": true})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify user_info was created
	newInfo := &model.TUserInfo{}
	ok, err := comm.Db.Where("id=?", "user1").Get(newInfo)
	if err != nil || !ok {
		t.Fatalf("query user info: %v", err)
	}
	if newInfo.PermUser != 1 {
		t.Errorf("PermUser = %d, want 1", newInfo.PermUser)
	}
}
