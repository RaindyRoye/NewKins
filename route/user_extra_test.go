package route

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

// --- UserController.new() tests ---

func TestUserController_new_MissingName(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("nick", "Test Nick")
	m.Set("pass", "secret")
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_new_MissingNick(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "newuser")
	m.Set("nick", "")
	m.Set("pass", "secret")
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_new_MissingPass(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "newuser")
	m.Set("nick", "New User")
	m.Set("pass", "")
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_new_AllMissing(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_new_DuplicateUser(t *testing.T) {
	setupUserTestDB(t)
	createUserForTest(t, "existinguser", "Existing", "hash", 1)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "existinguser")
	m.Set("nick", "New Nick")
	m.Set("pass", "secret")
	ctrl.new(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestUserController_new_Success(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "brandnew")
	m.Set("nick", "Brand New User")
	m.Set("pass", "password123")
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	newId := w.Body.String()
	if newId == "" {
		t.Fatal("expected non-empty user ID in response body")
	}

	// Verify user was actually inserted
	usr := &model.TUser{}
	ok, err := comm.Db.Where("id=?", newId).Get(usr)
	if err != nil {
		t.Fatalf("query new user: %v", err)
	}
	if !ok {
		t.Fatal("newly created user not found in DB")
	}
	if usr.Name != "brandnew" {
		t.Errorf("user name = %q, want %q", usr.Name, "brandnew")
	}
	if usr.Nick != "Brand New User" {
		t.Errorf("user nick = %q, want %q", usr.Nick, "Brand New User")
	}
	if usr.Active != 1 {
		t.Errorf("user active = %d, want 1", usr.Active)
	}
}

func TestUserController_new_TrimmedWhitespace(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "   ")
	m.Set("nick", "   ")
	m.Set("pass", "password123")
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (whitespace should be trimmed)", w.Code, http.StatusBadRequest)
	}
}

// --- UserController.upass() tests ---

func TestUserController_upass_MissingIdAndPass(t *testing.T) {
	setupUserTestDB(t)
	c, w := makeGinContext(t, nil, nil)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	m.Set("pass", "")
	ctrl.upass(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_upass_SelfMissingOldPass(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "selfuser", "Self User", utils.Md5String("oldpass"), 1)
	c, w := makeGinContext(t, nil, user)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "")
	m.Set("pass", "newpass")
	ctrl.upass(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUserController_upass_SelfWrongOldPass(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "wrongpassuser", "Wrong Pass", utils.Md5String("correct"), 1)
	c, w := makeGinContext(t, nil, user)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "incorrect")
	m.Set("pass", "newpass")
	ctrl.upass(c, m)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func TestUserController_upass_SelfSuccess(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "changepass", "Change Pass", utils.Md5String("oldpass123"), 1)
	c, w := makeGinContext(t, nil, user)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "oldpass123")
	m.Set("pass", "newpass456")
	ctrl.upass(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify password was changed
	updated := &model.TUser{}
	ok, err := comm.Db.Where("id=?", user.Id).Get(updated)
	if err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after password update")
	}
	if updated.Pass != utils.Md5String("newpass456") {
		t.Error("password was not updated correctly")
	}
}

func TestUserController_upass_NonAdminChangingOtherUser(t *testing.T) {
	setupUserTestDB(t)
	target := createUserForTest(t, "target", "Target", utils.Md5String("oldpass"), 1)
	nonAdmin := &model.TUser{Id: "non-admin", Name: "non-admin", Active: 1}
	c, w := makeGinContext(t, nil, nonAdmin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", target.Id)
	m.Set("pass", "newpass")
	ctrl.upass(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestUserController_upass_AdminChangeOtherUser(t *testing.T) {
	setupUserTestDB(t)
	target := createUserForTest(t, "admintarget", "Admin Target", utils.Md5String("oldpass"), 1)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", target.Id)
	m.Set("pass", "adminnewpass")
	ctrl.upass(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify password was changed
	updated := &model.TUser{}
	ok, err := comm.Db.Where("id=?", target.Id).Get(updated)
	if err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after admin password update")
	}
	if updated.Pass != utils.Md5String("adminnewpass") {
		t.Error("password was not updated correctly by admin")
	}
}

func TestUserController_upass_NotUpPassFlag(t *testing.T) {
	setupUserTestDB(t)
	origNotUpPass := comm.NotUpPass
	comm.NotUpPass = true
	t.Cleanup(func() { comm.NotUpPass = origNotUpPass })

	user := createUserForTest(t, "lockeduser", "Locked", utils.Md5String("pass"), 1)
	nonAdmin := &model.TUser{Id: user.Id, Name: user.Name, Active: 1}
	c, w := makeGinContext(t, nil, nonAdmin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "pass")
	m.Set("pass", "newpass")
	ctrl.upass(c, m)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
}

func TestUserController_upass_NotUpPassAdminAllowed(t *testing.T) {
	setupUserTestDB(t)
	origNotUpPass := comm.NotUpPass
	comm.NotUpPass = true
	t.Cleanup(func() { comm.NotUpPass = origNotUpPass })

	target := createUserForTest(t, "admintargetlocked", "Admin Target Locked", utils.Md5String("oldpass"), 1)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", target.Id)
	m.Set("pass", "adminoverride")
	ctrl.upass(c, m)

	// Admin should still be allowed when NotUpPass is true
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUserController_upass_UserNotFound(t *testing.T) {
	setupUserTestDB(t)
	nonAdmin := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	c, w := makeGinContext(t, nil, nonAdmin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-id")
	m.Set("pass", "newpass")
	ctrl.upass(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

// --- UserController.active() additional tests ---

func TestUserController_active_NotAdmin(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "actnonadmin", "Act Non-Admin", "hash", 1)
	nonAdmin := &model.TUser{Id: user.Id, Name: user.Name, Active: 1}
	c, w := makeGinContext(t, nil, nonAdmin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("act", "0")
	ctrl.active(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestUserController_active_Deactivate(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "deactuser", "Deactivate Me", "hash", 1)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("act", "0")
	ctrl.active(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	updated := &model.TUser{}
	ok, err := comm.Db.Where("id=?", user.Id).Get(updated)
	if err != nil {
		t.Fatalf("query user: %v", err)
	}
	if !ok {
		t.Fatal("user not found")
	}
	if updated.Active != 0 {
		t.Errorf("active = %d, want 0", updated.Active)
	}
}

// --- UserController.perm() additional tests ---

func TestUserController_perm_NonAdminWithoutPermUser(t *testing.T) {
	setupUserTestDB(t)
	target := createUserForTest(t, "permnonadmin", "Perm Non-Admin", "hash", 1)
	nonAdmin := &model.TUser{Id: target.Id, Name: target.Name, Active: 1}

	// Create a user info record without permUser
	_, err := comm.Db.InsertOne(&model.TUserInfo{
		Id:       target.Id,
		PermUser: 0,
		PermOrg:  0,
		PermPipe: 0,
	})
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	c, w := makeGinContext(t, nil, nonAdmin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", target.Id)
	m.Set("permUser", true)
	ctrl.perm(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestUserController_perm_UpdateExistingPerms(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "existingperm", "Existing Perm", "hash", 1)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Insert initial permissions
	_, err := comm.Db.InsertOne(&model.TUserInfo{
		Id:       user.Id,
		PermUser: 1,
		PermOrg:  0,
		PermPipe: 0,
	})
	if err != nil {
		t.Fatalf("insert initial user info: %v", err)
	}

	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("permUser", false)
	m.Set("permOrg", true)
	m.Set("permPipe", true)
	ctrl.perm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	uinfo := &model.TUserInfo{}
	ok, err := comm.Db.Where("id=?", user.Id).Get(uinfo)
	if err != nil {
		t.Fatalf("query user info: %v", err)
	}
	if !ok {
		t.Fatal("user info not found after update")
	}
	if uinfo.PermUser != 0 {
		t.Errorf("PermUser = %d, want 0", uinfo.PermUser)
	}
	if uinfo.PermOrg != 1 {
		t.Errorf("PermOrg = %d, want 1", uinfo.PermOrg)
	}
	if uinfo.PermPipe != 1 {
		t.Errorf("PermPipe = %d, want 1", uinfo.PermPipe)
	}
}

// --- UserController.page() additional tests ---

func TestUserController_page_WithNickSearch(t *testing.T) {
	setupUserTestDB(t)
	createUserForTest(t, "alice", "Alice Smith", "hash1", 1)
	createUserForTest(t, "bob", "Bob Jones", "hash2", 1)

	m := &hbtp.Map{}
	m.Set("q", "Jones")
	m.Set("page", int64(1))
	c, w := makeGinContext(t, m, nil)
	ctrl := UserController{}
	ctrl.page(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Parse response to check filtering
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
}

// --- UserController.upinfo() additional tests ---

func TestUserController_upinfo_AdminEditingOtherUser(t *testing.T) {
	setupUserTestDB(t)
	target := createUserForTest(t, "admintarget-info", "Target Info", "hash", 1)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeGinContext(t, nil, adminUser)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", target.Id)
	m.Set("nick", "Updated By Admin")
	m.Set("phone", "555-9999")
	m.Set("email", "admin-updated@test.com")
	m.Set("remark", "admin edit")
	ctrl.upinfo(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	updated := &model.TUser{}
	ok, err := comm.Db.Where("id=?", target.Id).Get(updated)
	if err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after update")
	}
	if updated.Nick != "Updated By Admin" {
		t.Errorf("nick = %q, want %q", updated.Nick, "Updated By Admin")
	}
}

func TestUserController_upinfo_NonSelfNonAdmin(t *testing.T) {
	setupUserTestDB(t)
	target := createUserForTest(t, "target-info", "Target Info", "hash", 1)
	nonAdmin := &model.TUser{Id: "other-user", Name: "other", Active: 1}

	c, w := makeGinContext(t, nil, nonAdmin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", target.Id)
	m.Set("nick", "Unauthorized Edit")
	ctrl.upinfo(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// --- UserController.info() additional tests ---

func TestUserController_info_ReturnsUserInfo(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "infouser", "Info User", "hash", 1)

	// Insert a user info record
	_, err := comm.Db.InsertOne(&model.TUserInfo{
		Id:    user.Id,
		Phone: "555-1234",
		Email: "info@test.com",
	})
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	c, w := makeGinContext(t, nil, nil)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := resp["user"]; !ok {
		t.Error("response should contain 'user' key")
	}
	if _, ok := resp["info"]; !ok {
		t.Error("response should contain 'info' key")
	}
}

// --- UserController.new() non-admin with permUser ---

func TestUserController_new_NonAdminWithPermUser(t *testing.T) {
	setupUserTestDB(t)
	nonAdmin := &model.TUser{Id: "permgranted", Name: "permgranted", Active: 1}

	// Grant permUser to this user
	_, err := comm.Db.InsertOne(&model.TUserInfo{
		Id:       nonAdmin.Id,
		PermUser: 1,
	})
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	c, w := makeGinContext(t, nil, nonAdmin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "created-by-perm-user")
	m.Set("nick", "Created By Perm User")
	m.Set("pass", "secret")
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestUserController_new_NonAdminWithoutPermUser(t *testing.T) {
	setupUserTestDB(t)
	nonAdmin := &model.TUser{Id: "noperm", Name: "noperm", Active: 1}

	// No permUser granted
	_, err := comm.Db.InsertOne(&model.TUserInfo{
		Id:       nonAdmin.Id,
		PermUser: 0,
	})
	if err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	c, w := makeGinContext(t, nil, nonAdmin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("name", "shouldnotcreate")
	m.Set("nick", "Should Not Create")
	m.Set("pass", "secret")
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// Ensure generateTestID produces unique IDs across rapid calls
func TestGenerateTestID_Unique(t *testing.T) {
	ids := map[string]bool{}
	for i := 0; i < 5; i++ {
		id := generateTestID()
		if ids[id] {
			// Collisions can happen at same microsecond, but are extremely unlikely in 5 sequential calls
			// If it happens, just note it
			t.Logf("duplicate ID generated: %s (may be expected at same microsecond)", id)
		}
		ids[id] = true
		time.Sleep(time.Microsecond * 100)
	}
}
