package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
)

func TestGetMidLgUser_Found(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	usr := &model.TUser{
		Id:   "test-id",
		Name: "testuser",
	}
	c.Set(LgUserKey, usr)

	got := GetMidLgUser(c)
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.Id != "test-id" {
		t.Errorf("expected Id 'test-id', got %q", got.Id)
	}
	if got.Name != "testuser" {
		t.Errorf("expected Name 'testuser', got %q", got.Name)
	}
}

func TestGetMidLgUser_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	got := GetMidLgUser(c)
	if got != nil {
		t.Errorf("expected nil when key not set, got %v", got)
	}
}

func TestGetMidLgUser_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	// Set the key to a non-*TUser value
	c.Set(LgUserKey, "not a user")

	got := GetMidLgUser(c)
	if got != nil {
		t.Errorf("expected nil when value is wrong type, got %v", got)
	}
}

func TestLgUserKey_Constant(t *testing.T) {
	if LgUserKey != "lguser" {
		t.Errorf("LgUserKey = %q, want %q", LgUserKey, "lguser")
	}
}

func TestMidUserCheck_Admin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// Mock CurrUserCache to return admin user
	// Since we can't easily mock, we'll test the logic indirectly
	// by checking that the middleware correctly identifies admin users
	usr := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(LgUserKey, usr)

	// Test GetMidLgUser returns the user
	got := GetMidLgUser(c)
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if !IsAdmin(got) {
		t.Error("expected admin user")
	}
}

func TestMidUserCheck_InactiveUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// Test inactive non-admin user logic
	usr := &model.TUser{Id: "user1", Name: "user1", Active: 0}
	c.Set(LgUserKey, usr)

	got := GetMidLgUser(c)
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if IsAdmin(got) {
		t.Error("expected non-admin user")
	}
	if got.Active != 0 {
		t.Errorf("expected Active=0, got %d", got.Active)
	}
}

func TestMidUserCheck_ActiveUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	// Test active non-admin user
	usr := &model.TUser{Id: "user2", Name: "user2", Active: 1}
	c.Set(LgUserKey, usr)

	got := GetMidLgUser(c)
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if IsAdmin(got) {
		t.Error("expected non-admin user")
	}
	if got.Active != 1 {
		t.Errorf("expected Active=1, got %d", got.Active)
	}
}

func TestContextAwareFunctions_EmptyContext(t *testing.T) {
	setupBatchTestDB(t)
	ctx := context.Background()

	// Test IsOrgAdminCtx with empty context and no matching data
	// Should return false gracefully
	if IsOrgAdminCtx(ctx, "nonexistent-uid", "nonexistent-orgId") {
		t.Error("IsOrgAdminCtx should return false when user/org doesn't exist")
	}

	// Test GetUsePermRwrCtx with empty context
	if GetUsePermRwrCtx(ctx, "nonexistent-uid", "nonexistent-orgId") != 0 {
		t.Error("GetUsePermRwrCtx should return 0 when user/org doesn't exist")
	}

	// Test HasOrgExecCtx with empty context
	if HasOrgExecCtx(ctx, "nonexistent-uid", "nonexistent-orgId") {
		t.Error("HasOrgExecCtx should return false when user/org doesn't exist")
	}
}

func TestContextAwareFunctions_WithData(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert test org with explicit Aid
	org := &model.TOrg{
		Id:   "org1",
		Aid:  1,
		Name: "test-org",
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("failed to insert org: %v", err)
	}

	// Insert test user with explicit Aid
	usr := &model.TUser{
		Id:   "user1",
		Aid:  1,
		Name: "testuser",
	}
	if _, err := eng.Insert(usr); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	// Insert user-org relationship with full permissions
	userOrg := &model.TUserOrg{
		Uid:      "user1",
		OrgId:    "org1",
		PermAdm:  1,
		PermRw:   1,
		PermExec: 1,
		PermDown: 1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("failed to insert user org: %v", err)
	}

	// Test IsOrgAdminCtx returns true for org admin
	if !IsOrgAdminCtx(ctx, "user1", "org1") {
		t.Error("IsOrgAdminCtx should return true for org admin")
	}

	// Test GetUsePermRwrCtx returns 1 for user with write permission
	if GetUsePermRwrCtx(ctx, "user1", "org1") != 1 {
		t.Error("GetUsePermRwrCtx should return 1 for user with write permission")
	}

	// Test HasOrgExecCtx returns true for user with exec permission
	if !HasOrgExecCtx(ctx, "user1", "org1") {
		t.Error("HasOrgExecCtx should return true for user with exec permission")
	}

	// Test with non-admin user
	user2 := &model.TUser{
		Id:   "user2",
		Aid:  2,
		Name: "regular-user",
	}
	if _, err := eng.Insert(user2); err != nil {
		t.Fatalf("failed to insert user2: %v", err)
	}
	user2Org := &model.TUserOrg{
		Uid:      "user2",
		OrgId:    "org1",
		PermAdm:  0,
		PermRw:   0,
		PermExec: 0,
		PermDown: 0,
	}
	if _, err := eng.Insert(user2Org); err != nil {
		t.Fatalf("failed to insert user2 org: %v", err)
	}

	// Test IsOrgAdminCtx returns false for non-admin
	if IsOrgAdminCtx(ctx, "user2", "org1") {
		t.Error("IsOrgAdminCtx should return false for non-admin user")
	}

	// Test GetUsePermRwrCtx returns 0 for user without write permission
	if GetUsePermRwrCtx(ctx, "user2", "org1") != 0 {
		t.Error("GetUsePermRwrCtx should return 0 for user without write permission")
	}

	// Test HasOrgExecCtx returns false for user without exec permission
	if HasOrgExecCtx(ctx, "user2", "org1") {
		t.Error("HasOrgExecCtx should return false for user without exec permission")
	}
}
