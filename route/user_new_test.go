package route

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

// --- UserController.new tests ---

func TestUserController_new_EmptyName(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("nick", "Nick")
	m.Set("pass", "secret")

	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUserController_new_EmptyNick(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("name", "newuser")
	m.Set("nick", "")
	m.Set("pass", "secret")

	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_new_EmptyPass(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("name", "newuser")
	m.Set("nick", "Nick")
	m.Set("pass", "")

	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_new_NoPermission(t *testing.T) {
	setupUserTestDB(t)
	// Non-admin user without PermUser permission
	regularUser := &model.TUser{Id: "user1", Name: "regular", Active: 1}
	c, w := makeGinContext(t, nil, regularUser)

	m := &hbtp.Map{}
	m.Set("name", "newuser")
	m.Set("nick", "Nick")
	m.Set("pass", "secret")

	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestUserController_new_UserAlreadyExists(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	createUserForTest(t, "existing", "Existing", "pass", 1)

	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("name", "existing")
	m.Set("nick", "New Nick")
	m.Set("pass", "secret")

	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestUserController_new_Success_Admin(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("name", "brandnew")
	m.Set("nick", "Brand New User")
	m.Set("pass", "securepass")

	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify the user was created in DB
	newUser := &model.TUser{}
	ok, err := comm.Db.Where("name = ?", "brandnew").Get(newUser)
	if err != nil {
		t.Fatalf("query new user: %v", err)
	}
	if !ok {
		t.Fatal("new user not found in DB")
	}
	if newUser.Nick != "Brand New User" {
		t.Errorf("nick = %q, want %q", newUser.Nick, "Brand New User")
	}
	if newUser.Pass != utils.Md5String("securepass") {
		t.Error("password was not MD5-hashed correctly")
	}
	if newUser.Active != 1 {
		t.Errorf("active = %d, want 1", newUser.Active)
	}
}

func TestUserController_new_Success_WithPermUser(t *testing.T) {
	setupUserTestDB(t)
	// Non-admin user with PermUser=1
	permUser := &model.TUser{Id: "permuser1", Name: "permuser", Active: 1}
	_, err := comm.Db.InsertOne(&model.TUserInfo{
		Id:       "permuser1",
		PermUser: 1,
	})
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	c, w := makeGinContext(t, nil, permUser)

	m := &hbtp.Map{}
	m.Set("name", "newbyperm")
	m.Set("nick", "Created By PermUser")
	m.Set("pass", "pass123")

	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUserController_new_WhitespaceOnlyName(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("name", "   ")
	m.Set("nick", "Nick")
	m.Set("pass", "secret")

	ctrl := UserController{}
	ctrl.new(c, m)

	// After trimming, name is empty → 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- UserController.upass tests ---

func TestUserController_upass_EmptyId(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", "")
	m.Set("pass", "newpass")

	ctrl := UserController{}
	ctrl.upass(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_upass_EmptyPass(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", "admin1")
	m.Set("pass", "")

	ctrl := UserController{}
	ctrl.upass(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_upass_UserNotFound(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("pass", "newpass")

	ctrl := UserController{}
	ctrl.upass(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestUserController_upass_NotUpPass_NonAdmin(t *testing.T) {
	setupUserTestDB(t)
	// Enable NotUpPass (password updates disabled for non-admins)
	origNotUpPass := comm.NotUpPass
	comm.NotUpPass = true
	t.Cleanup(func() { comm.NotUpPass = origNotUpPass })

	user := createUserForTest(t, "testuser", "Test", "hash", 1)
	// logged-in user is the same user trying to change their own password
	c, w := makeGinContext(t, nil, user)

	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "someoldpass")
	m.Set("pass", "newpass")

	ctrl := UserController{}
	ctrl.upass(c, m)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestUserController_upass_ChangeOwnPassword_WrongOld(t *testing.T) {
	setupUserTestDB(t)
	comm.NotUpPass = false

	user := createUserForTest(t, "testuser", "Test", utils.Md5String("correctpass"), 1)
	c, w := makeGinContext(t, nil, user)

	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "wrongpass")
	m.Set("pass", "newpass")

	ctrl := UserController{}
	ctrl.upass(c, m)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestUserController_upass_ChangeOwnPassword_MissingOld(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "testuser", "Test", utils.Md5String("pass"), 1)
	c, w := makeGinContext(t, nil, user)

	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "")
	m.Set("pass", "newpass")

	ctrl := UserController{}
	ctrl.upass(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUserController_upass_ChangeOwnPassword_Success(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "testuser", "Test", utils.Md5String("oldpass"), 1)
	c, w := makeGinContext(t, nil, user)

	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "oldpass")
	m.Set("pass", "newpass123")

	ctrl := UserController{}
	ctrl.upass(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify password was updated in DB
	updated := &model.TUser{}
	ok, err := comm.Db.Where("id = ?", user.Id).Get(updated)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("user not found after password update")
	}
	if updated.Pass != utils.Md5String("newpass123") {
		t.Error("password was not updated correctly")
	}
}

func TestUserController_upass_AdminChangeOtherPassword(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	target := createUserForTest(t, "target", "Target", utils.Md5String("old"), 1)

	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", target.Id)
	m.Set("pass", "adminreset")

	ctrl := UserController{}
	ctrl.upass(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify password was updated
	updated := &model.TUser{}
	ok, err := comm.Db.Where("id = ?", target.Id).Get(updated)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("target user not found")
	}
	if updated.Pass != utils.Md5String("adminreset") {
		t.Error("admin password reset failed")
	}
}

func TestUserController_upass_NonAdminChangeOtherPassword(t *testing.T) {
	setupUserTestDB(t)
	nonAdmin := createUserForTest(t, "nonadmin", "NonAdmin", "hash", 1)
	target := createUserForTest(t, "target2", "Target2", "hash", 1)

	c, w := makeGinContext(t, nil, nonAdmin)

	m := &hbtp.Map{}
	m.Set("id", target.Id)
	m.Set("pass", "hacked")

	ctrl := UserController{}
	ctrl.upass(c, m)

	// Non-admin trying to change another user's password → 405
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// --- UserController.active tests (additional coverage) ---

func TestUserController_active_DeactivateUser(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	user := createUserForTest(t, "activeuser", "Active", "hash", 1)

	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("act", "0")

	ctrl := UserController{}
	ctrl.active(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify deactivated
	updated := &model.TUser{}
	ok, err := comm.Db.Where("id = ?", user.Id).Get(updated)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("user not found")
	}
	if updated.Active != 0 {
		t.Errorf("active = %d, want 0", updated.Active)
	}
}

func TestUserController_active_NonAdminDenied(t *testing.T) {
	setupUserTestDB(t)
	regularUser := &model.TUser{Id: "user1", Name: "regular", Active: 1}

	c, w := makeGinContext(t, nil, regularUser)

	m := &hbtp.Map{}
	m.Set("id", "user1")
	m.Set("act", "1")

	ctrl := UserController{}
	ctrl.active(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- UserController.page tests (search filtering) ---

func TestUserController_page_SearchByName(t *testing.T) {
	setupUserTestDB(t)
	createUserForTest(t, "alice", "Alice Smith", "hash1", 1)
	createUserForTest(t, "bob", "Bob Jones", "hash2", 1)

	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("q", "alice")
	m.Set("page", int64(1))
	c, w := makeGinContext(t, nil, nil)
	ctrl.page(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

// --- UserController.upinfo tests (update existing info) ---

func TestUserController_upinfo_UpdateExistingInfo(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "infouser", "Old Nick", "hash", 1)
	// Pre-create user info
	_, err := comm.Db.InsertOne(&model.TUserInfo{
		Id:    user.Id,
		Phone: "111",
		Email: "old@example.com",
	})
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	loggedInUser := &model.TUser{Id: user.Id, Name: user.Name, Active: 1}
	c, w := makeGinContext(t, nil, loggedInUser)

	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("nick", "New Nick")
	m.Set("phone", "999")
	m.Set("email", "new@example.com")
	m.Set("remark", "updated remark")

	ctrl := UserController{}
	ctrl.upinfo(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify updated info
	uinfo := &model.TUserInfo{}
	ok, err := comm.Db.Where("id = ?", user.Id).Get(uinfo)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("user info not found")
	}
	if uinfo.Phone != "999" {
		t.Errorf("phone = %q, want %q", uinfo.Phone, "999")
	}
	if uinfo.Email != "new@example.com" {
		t.Errorf("email = %q, want %q", uinfo.Email, "new@example.com")
	}
	if uinfo.Remark != "updated remark" {
		t.Errorf("remark = %q, want %q", uinfo.Remark, "updated remark")
	}
}

func TestUserController_upinfo_CannotUpdateOtherUser(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "other", "Other", "hash", 1)
	loggedInUser := &model.TUser{Id: "different-user", Name: "different", Active: 1}

	c, w := makeGinContext(t, nil, loggedInUser)

	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("nick", "Hacked")
	m.Set("phone", "")
	m.Set("email", "")

	ctrl := UserController{}
	ctrl.upinfo(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// --- UserController.perm tests (update existing perm) ---

func TestUserController_perm_UpdateExisting(t *testing.T) {
	setupUserTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	user := createUserForTest(t, "permusr", "Perm User", "hash", 1)
	// Pre-create user info
	_, err := comm.Db.InsertOne(&model.TUserInfo{
		Id:       user.Id,
		PermUser: 0,
		PermOrg:  0,
		PermPipe: 0,
	})
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	c, w := makeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("permUser", true)
	m.Set("permOrg", true)
	m.Set("permPipe", false)

	ctrl := UserController{}
	ctrl.perm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	uinfo := &model.TUserInfo{}
	ok, err := comm.Db.Where("id = ?", user.Id).Get(uinfo)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("user info not found")
	}
	if uinfo.PermUser != 1 {
		t.Errorf("PermUser = %d, want 1", uinfo.PermUser)
	}
	if uinfo.PermOrg != 1 {
		t.Errorf("PermOrg = %d, want 1", uinfo.PermOrg)
	}
	if uinfo.PermPipe != 0 {
		t.Errorf("PermPipe = %d, want 0", uinfo.PermPipe)
	}
}
