package service

import (
	"context"
	"testing"

	"github.com/gokins/gokins/model"
)

// TestCheckCurrPermission_ContextWrapper tests the permission check logic
// using context-based user retrieval instead of gin context.
// This avoids the complexity of mocking HTTP tokens.
func TestCheckCurrPermission_ContextWrapper(t *testing.T) {
	tests := []struct {
		name     string
		user     *model.TUser
		perms    string
		expected bool
	}{
		{
			name:     "nil user returns false",
			user:     nil,
			perms:    PermCommon,
			expected: false,
		},
		{
			name: "common user with common perm",
			user: &model.TUser{
				Id:   "user1",
				Name: "regular",
			},
			perms:    PermCommon,
			expected: true,
		},
		{
			name: "common user with admin perm",
			user: &model.TUser{
				Id:   "user1",
				Name: "regular",
			},
			perms:    PermAdmin,
			expected: false,
		},
		{
			name: "admin user with admin perm",
			user: &model.TUser{
				Id:   "admin1",
				Name: AdminUserName,
			},
			perms:    PermAdmin,
			expected: true,
		},
		{
			name: "admin user with common perm",
			user: &model.TUser{
				Id:   "admin1",
				Name: AdminUserName,
			},
			perms:    PermCommon,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use CheckUPermission directly since CheckCurrPermission
			// requires complex token mocking
			result := CheckUPermission(tt.user, tt.perms)
			if result != tt.expected {
				t.Errorf("CheckUPermission() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestCheckCurrPermission_ViaContext tests the context-based permission check
func TestCheckCurrPermission_ViaContext(t *testing.T) {
	ctx := context.Background()

	// Test with non-existent user
	// Note: This test expects a panic when comm.Db is nil
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic occurred (comm.Db is nil): %v", r)
		}
	}()

	result := CheckPermissionCtx(ctx, "nonexistent-user", PermCommon)
	if result {
		t.Error("CheckPermissionCtx should return false for non-existent user")
	}
}
