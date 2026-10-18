package service

import (
	"testing"

	"github.com/gokins/gokins/model"
)

// TestIsAdmin_NilSafety_AllCases verifies IsAdmin behavior across all edge cases
// including nil pointer, empty ID, and various user IDs.
// This test documents the fix for a nil pointer dereference vulnerability where IsAdmin previously
// accessed usr.Id without checking if usr was nil.
func TestIsAdmin_NilSafety_AllCases(t *testing.T) {
	tests := []struct {
		name string
		usr  *model.TUser
		want bool
	}{
		{
			name: "nil pointer returns false",
			usr:  nil,
			want: false,
		},
		{
			name: "admin user returns true",
			usr:  &model.TUser{Id: "admin"},
			want: true,
		},
		{
			name: "non-admin user returns false",
			usr:  &model.TUser{Id: "alice"},
			want: false,
		},
		{
			name: "empty ID returns false",
			usr:  &model.TUser{Id: ""},
			want: false,
		},
		{
			name: "user with nil pointer fields",
			usr:  &model.TUser{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("IsAdmin panicked for case %q: %v", tt.name, r)
				}
			}()

			got := IsAdmin(tt.usr)
			if got != tt.want {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}
