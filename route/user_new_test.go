package route

import (
	"net/http"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

// TestUserController_new_EmptyParams tests that user.new returns 400 when parameters are missing.
func TestUserController_new_EmptyParams(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	tests := []struct {
		name string
		m    *hbtp.Map
	}{
		{
			name: "empty name",
			m:    &hbtp.Map{"name": "", "nick": "Nick", "pass": "pass123"},
		},
		{
			name: "empty nick",
			m:    &hbtp.Map{"name": "newuser", "nick": "", "pass": "pass123"},
		},
		{
			name: "empty pass",
			m:    &hbtp.Map{"name": "newuser", "nick": "Nick", "pass": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := makeGinContext(t, tt.m, adminUser)
			ctrl := UserController{}
			ctrl.new(c, tt.m)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}
}

// TestUserController_new_NoPermission tests that non-admin users without perm_user get 405.
func TestUserController_new_NoPermission(t *testing.T) {
	setupUserTestDB(t)
	regularUser := &model.TUser{Id: "regular", Name: "regular", Active: 1}

	// Create user_info with perm_user=0
	uinfo := &model.TUserInfo{Id: regularUser.Id, PermUser: 0}
	if _, err := comm.Db.InsertOne(uinfo); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	m := &hbtp.Map{"name": "newuser", "nick": "New User", "pass": "pass123"}
	c, w := makeGinContext(t, m, regularUser)
	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// TestUserController_new_UserAlreadyExists tests that creating a duplicate user returns 409.
func TestUserController_new_UserAlreadyExists(t *testing.T) {
	setupUserTestDB(t)
	createUserForTest(t, "existinguser", "Existing User", "hash", 1)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{"name": "existinguser", "nick": "New User", "pass": "pass123"}
	c, w := makeGinContext(t, m, adminUser)
	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

// TestUserController_new_SuccessAsAdmin tests that admin can create a new user.
func TestUserController_new_SuccessAsAdmin(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{"name": "newuser", "nick": "New User", "pass": "pass123"}
	c, w := makeGinContext(t, m, adminUser)
	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify user was created
	user := &model.TUser{}
	ok, err := comm.Db.Where("name = ?", "newuser").Get(user)
	if err != nil {
		t.Fatalf("query user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after creation")
	}
	if user.Nick != "New User" {
		t.Errorf("user nick = %q, want %q", user.Nick, "New User")
	}
	if user.Active != 1 {
		t.Errorf("user active = %d, want 1", user.Active)
	}
}

// TestUserController_new_SuccessWithPermission tests that non-admin with perm_user can create users.
func TestUserController_new_SuccessWithPermission(t *testing.T) {
	setupUserTestDB(t)
	regularUser := &model.TUser{Id: "regular", Name: "regular", Active: 1}

	// Create user_info with perm_user=1
	uinfo := &model.TUserInfo{Id: regularUser.Id, PermUser: 1}
	if _, err := comm.Db.InsertOne(uinfo); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	m := &hbtp.Map{"name": "newuser2", "nick": "New User 2", "pass": "pass456"}
	c, w := makeGinContext(t, m, regularUser)
	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify user was created
	user := &model.TUser{}
	ok, err := comm.Db.Where("name = ?", "newuser2").Get(user)
	if err != nil {
		t.Fatalf("query user: %v", err)
	}
	if !ok {
		t.Fatal("user not found after creation")
	}
	if user.Nick != "New User 2" {
		t.Errorf("user nick = %q, want %q", user.Nick, "New User 2")
	}
}

// TestUserController_new_TrimWhitespace tests that leading/trailing whitespace is trimmed from name and nick.
func TestUserController_new_TrimWhitespace(t *testing.T) {
	setupUserTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{"name": "  trimmed  ", "nick": "  Trimmed Nick  ", "pass": "pass123"}
	c, w := makeGinContext(t, m, adminUser)
	ctrl := UserController{}
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify user was created with trimmed name
	user := &model.TUser{}
	ok, err := comm.Db.Where("name = ?", "trimmed").Get(user)
	if err != nil {
		t.Fatalf("query user: %v", err)
	}
	if !ok {
		t.Fatal("user not found - name was not trimmed")
	}
	if user.Nick != "Trimmed Nick" {
		t.Errorf("user nick = %q, want %q", user.Nick, "Trimmed Nick")
	}
}
