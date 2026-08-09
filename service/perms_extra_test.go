package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gokins/gokins/model"
)

func TestPerms_CheckPermission_NilDB(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic (Db is nil): %v", r)
		}
	}()
	_ = CheckPermission("user-id", PermCommon)
}

func TestPerms_CheckPermissionCtx_NilDB(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic (Db is nil): %v", r)
		}
	}()
	_ = CheckPermissionCtx(context.Background(), "user-id", PermCommon)
}

func TestPerms_CheckUPermission_NilUser(t *testing.T) {
	result := CheckUPermission(nil, PermCommon)
	if result {
		t.Error("CheckUPermission(nil, PermCommon) should return false")
	}
}

func TestPerms_CheckUPermission_CommonLevel(t *testing.T) {
	user := &model.TUser{
		Id:   "user-1",
		Name: "alice",
		Nick: "Alice",
	}

	result := CheckUPermission(user, PermCommon)
	if !result {
		t.Error("CheckUPermission should return true for PermCommon level")
	}
}

func TestPerms_CheckUPermission_AdminLevel_AdminUser(t *testing.T) {
	user := &model.TUser{
		Id:   "admin-1",
		Name: AdminUserName,
		Nick: "Administrator",
	}

	result := CheckUPermission(user, PermAdmin)
	if !result {
		t.Error("CheckUPermission should return true for admin user with PermAdmin level")
	}
}

func TestPerms_CheckUPermission_AdminLevel_NonAdminUser(t *testing.T) {
	user := &model.TUser{
		Id:   "user-1",
		Name: "alice",
		Nick: "Alice",
	}

	result := CheckUPermission(user, PermAdmin)
	if result {
		t.Error("CheckUPermission should return false for non-admin user with PermAdmin level")
	}
}

func TestPerms_CheckUPermission_UnknownLevel(t *testing.T) {
	user := &model.TUser{
		Id:   "user-1",
		Name: "alice",
		Nick: "Alice",
	}

	result := CheckUPermission(user, "unknown")
	if result {
		t.Error("CheckUPermission should return false for unknown permission level")
	}
}

func TestPerms_CheckCurrPermission_NilContext(t *testing.T) {
	// This would require a gin.Context which is not available in unit tests
	// We can't easily test this without setting up a full HTTP request
	t.Skip("CheckCurrPermission requires gin.Context - tested in route tests")
}

func TestPerms_ErrorSentinels(t *testing.T) {
	// Verify sentinel errors are properly defined
	if ErrPermissionDenied == nil {
		t.Error("ErrPermissionDenied should not be nil")
	}

	if ErrPermissionDenied.Error() != "permission denied" {
		t.Errorf("ErrPermissionDenied message = %q, want %q", ErrPermissionDenied.Error(), "permission denied")
	}

	// Test errors.Is compatibility
	err := ErrPermissionDenied
	if !errors.Is(err, ErrPermissionDenied) {
		t.Error("errors.Is should recognize ErrPermissionDenied")
	}
}

func TestPerms_Constants(t *testing.T) {
	// Verify permission level constants
	if PermCommon != "common" {
		t.Errorf("PermCommon = %q, want %q", PermCommon, "common")
	}

	if PermAdmin != "admin" {
		t.Errorf("PermAdmin = %q, want %q", PermAdmin, "admin")
	}

	if AdminUserName != "admin" {
		t.Errorf("AdminUserName = %q, want %q", AdminUserName, "admin")
	}
}
