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

// TestUserController_upass_SelfCorrectOldPass tests changing own password
// with the correct old password.
func TestUserController_upass_SelfCorrectOldPass(t *testing.T) {
	setupUserTestDB(t)
	oldPass := "oldpass123"
	user := createUserForTest(t, "selfuser", "Self User", utils.Md5String(oldPass), 1)
	loggedIn := &model.TUser{Id: user.Id, Name: user.Name, Pass: user.Pass, Active: 1}

	c, w := makeGinContext(t, nil, loggedIn)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", oldPass)
	m.Set("pass", "newpass456")
	ctrl.upass(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify password was updated in DB
	updated := &model.TUser{}
	ok, err := comm.Db.Where("id=?", user.Id).Get(updated)
	if err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after password update")
	}
	expectedHash := utils.Md5String("newpass456")
	if updated.Pass != expectedHash {
		t.Errorf("password hash = %q, want %q", updated.Pass, expectedHash)
	}
}

// TestUserController_upass_SelfWrongOldPass tests that providing the wrong
// old password returns 401 Unauthorized.
func TestUserController_upass_SelfWrongOldPass(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "wrongold", "Wrong Old", utils.Md5String("correct"), 1)
	loggedIn := &model.TUser{Id: user.Id, Name: user.Name, Pass: user.Pass, Active: 1}

	c, w := makeGinContext(t, nil, loggedIn)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "incorrect")
	m.Set("pass", "newpass456")
	ctrl.upass(c, m)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

// TestUserController_upass_SelfMissingOldPass tests that a self-password-change
// without providing the old password returns 400 Bad Request.
func TestUserController_upass_SelfMissingOldPass(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "noolpass", "No Old Pass", utils.Md5String("pass"), 1)
	loggedIn := &model.TUser{Id: user.Id, Name: user.Name, Pass: user.Pass, Active: 1}

	c, w := makeGinContext(t, nil, loggedIn)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "")
	m.Set("pass", "newpass456")
	ctrl.upass(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
	if w.Body.String() != "param err1" {
		t.Errorf("body = %q, want 'param err1'", w.Body.String())
	}
}

// TestUserController_upass_NonAdminChangingOtherUser tests that a non-admin
// user cannot change another user's password.
func TestUserController_upass_NonAdminChangingOtherUser(t *testing.T) {
	setupUserTestDB(t)
	target := createUserForTest(t, "target", "Target User", utils.Md5String("targetpass"), 1)
	nonAdmin := &model.TUser{Id: "other-id", Name: "nonadmin", Active: 1}

	c, w := makeGinContext(t, nil, nonAdmin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", target.Id)
	m.Set("pass", "hacked")
	ctrl.upass(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
	if w.Body.String() != "is not admin" {
		t.Errorf("body = %q, want 'is not admin'", w.Body.String())
	}
}

// TestUserController_upass_AdminChangesOtherPass tests that an admin can
// change another user's password without needing the old password.
func TestUserController_upass_AdminChangesOtherPass(t *testing.T) {
	setupUserTestDB(t)
	target := createUserForTest(t, "admintarget", "Admin Target", utils.Md5String("orig"), 1)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeGinContext(t, nil, admin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", target.Id)
	m.Set("pass", "newpass789")
	ctrl.upass(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify password was updated
	updated := &model.TUser{}
	ok, err := comm.Db.Where("id=?", target.Id).Get(updated)
	if err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after admin password change")
	}
	expectedHash := utils.Md5String("newpass789")
	if updated.Pass != expectedHash {
		t.Errorf("password hash = %q, want %q", updated.Pass, expectedHash)
	}
}

// TestUserController_upass_NotUpPassFlag tests that when comm.NotUpPass is
// set, non-admin users are blocked from changing passwords.
func TestUserController_upass_NotUpPassFlag(t *testing.T) {
	setupUserTestDB(t)
	origNotUpPass := comm.NotUpPass
	comm.NotUpPass = true
	t.Cleanup(func() { comm.NotUpPass = origNotUpPass })

	user := createUserForTest(t, "noupuser", "No Up Pass", utils.Md5String("pass"), 1)
	loggedIn := &model.TUser{Id: user.Id, Name: user.Name, Pass: user.Pass, Active: 1}

	c, w := makeGinContext(t, nil, loggedIn)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "pass")
	m.Set("pass", "newpass")
	ctrl.upass(c, m)

	if w.Code != http.StatusForbidden {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusForbidden, w.Body.String())
	}
	if w.Body.String() != "can't update" {
		t.Errorf("body = %q, want 'can't update'", w.Body.String())
	}
}

// TestUserController_upass_AdminNotBlockedByNotUpPass tests that admins can
// still change passwords even when comm.NotUpPass is set.
func TestUserController_upass_AdminNotBlockedByNotUpPass(t *testing.T) {
	setupUserTestDB(t)
	origNotUpPass := comm.NotUpPass
	comm.NotUpPass = true
	t.Cleanup(func() { comm.NotUpPass = origNotUpPass })

	target := createUserForTest(t, "admintarget2", "Admin Target 2", utils.Md5String("orig"), 1)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	c, w := makeGinContext(t, nil, admin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", target.Id)
	m.Set("pass", "adminnewpass")
	ctrl.upass(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// TestUserController_upass_TargetUserNotFound tests that changing the password
// of a nonexistent user returns 404.
func TestUserController_upass_TargetUserNotFound(t *testing.T) {
	setupUserTestDB(t)
	nonAdmin := &model.TUser{Id: "other-999", Name: "other", Active: 1}

	c, w := makeGinContext(t, nil, nonAdmin)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-id")
	m.Set("pass", "anything")
	ctrl.upass(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

// TestUserController_upass_EmptyNewPass tests that an empty new password
// returns 400 Bad Request.
func TestUserController_upass_EmptyNewPass(t *testing.T) {
	setupUserTestDB(t)
	user := &model.TUser{Id: "u1", Name: "u1", Active: 1}

	c, w := makeGinContext(t, nil, user)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", "u1")
	m.Set("pass", "")
	ctrl.upass(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// TestUserController_upass_EmptyID tests that an empty ID returns 400 Bad Request.
func TestUserController_upass_EmptyID(t *testing.T) {
	setupUserTestDB(t)
	user := &model.TUser{Id: "u1", Name: "u1", Active: 1}

	c, w := makeGinContext(t, nil, user)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	m.Set("pass", "newpass")
	ctrl.upass(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// TestUserController_upass_ResponseContainsUserID verifies the response body
// contains the user ID on success.
func TestUserController_upass_ResponseContainsUserID(t *testing.T) {
	setupUserTestDB(t)
	user := createUserForTest(t, "respuser", "Resp User", utils.Md5String("old"), 1)
	loggedIn := &model.TUser{Id: user.Id, Name: user.Name, Pass: user.Pass, Active: 1}

	c, w := makeGinContext(t, nil, loggedIn)
	ctrl := UserController{}
	m := &hbtp.Map{}
	m.Set("id", user.Id)
	m.Set("olds", "old")
	m.Set("pass", "newpass")
	ctrl.upass(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var respBody string
	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		// Try direct string comparison as the response might be plain text
		if w.Body.String() != user.Id {
			t.Errorf("response body = %q, want user ID %q", w.Body.String(), user.Id)
		}
	} else if respBody != user.Id {
		t.Errorf("response body = %q, want user ID %q", respBody, user.Id)
	}
}
