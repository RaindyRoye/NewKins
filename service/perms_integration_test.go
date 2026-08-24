package service

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/gokins/model"
)

// --- CheckPermissionCtx ---

func TestCheckPermissionCtx_AdminUser(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "admin",
		Aid:     1,
		Name:    "admin",
		Nick:    "Administrator",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if !CheckPermissionCtx(ctx, "admin", "admin") {
		t.Error("admin user should have admin permission")
	}
	if !CheckPermissionCtx(ctx, "admin", "common") {
		t.Error("admin user should have common permission")
	}
}

func TestCheckPermissionCtx_RegularUser(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "user1",
		Aid:     2,
		Name:    "alice",
		Nick:    "Alice",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if !CheckPermissionCtx(ctx, "user1", "common") {
		t.Error("regular user should have common permission")
	}
	if CheckPermissionCtx(ctx, "user1", "admin") {
		t.Error("regular user should not have admin permission")
	}
}

func TestCheckPermissionCtx_NonexistentUser(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	if CheckPermissionCtx(ctx, "nonexistent", "common") {
		t.Error("nonexistent user should not have any permission")
	}
}

func TestCheckPermission_DelegatesToCtx(t *testing.T) {
	eng := setupUserTestDB(t)

	user := &model.TUser{
		Id:      "admin",
		Aid:     3,
		Name:    "admin",
		Nick:    "Administrator",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if !CheckPermission("admin", "admin") {
		t.Error("admin user should have admin permission via global context")
	}
}

func TestCheckPermissionCtx_EmptyID(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	if CheckPermissionCtx(ctx, "", "common") {
		t.Error("empty ID should not have any permission")
	}
}

// --- GetUserOrgCtx ---

func TestGetUserOrgCtx_Found(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org1", Aid: 1, Name: "TestOrg", Uid: "creator"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Uid:      "user1",
		OrgId:    "org1",
		PermRw:   1,
		PermExec: 0,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	got, ok := GetUserOrgCtx(ctx, "user1", "org1")
	if !ok {
		t.Fatal("GetUserOrgCtx should find user org")
	}
	if got.PermRw != 1 {
		t.Errorf("PermRw = %d, want 1", got.PermRw)
	}
	if got.PermExec != 0 {
		t.Errorf("PermExec = %d, want 0", got.PermExec)
	}
}

func TestGetUserOrgCtx_NotMember(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org2", Aid: 2, Name: "TestOrg2", Uid: "creator"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	_, ok := GetUserOrgCtx(ctx, "nonmember", "org2")
	if ok {
		t.Error("GetUserOrgCtx should return false for non-member")
	}
}

func TestGetUserOrgCtx_OrgNotFound(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	_, ok := GetUserOrgCtx(ctx, "user1", "nonexistent-org")
	if ok {
		t.Error("GetUserOrgCtx should return false for nonexistent org")
	}
}

// --- CheckPermissionCtx with canceled context ---

func TestCheckPermissionCtx_CancelledContext(t *testing.T) {
	eng := setupUserTestDB(t)

	user := &model.TUser{
		Id:      "user1",
		Aid:     10,
		Name:    "alice",
		Nick:    "Alice",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should not panic; result is indeterminate with canceled context
	_ = CheckPermissionCtx(ctx, "user1", "common")
}
