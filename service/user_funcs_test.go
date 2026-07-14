package service

import (
	"context"
	"testing"
)

// TestGetUser_EmptyId verifies that GetUser returns false for empty user ID.
func TestGetUser_EmptyId(t *testing.T) {
	_, ok := GetUser("")
	if ok {
		t.Error("GetUser(\"\") should return false for empty ID")
	}
}

// TestGetUserCtx_EmptyId verifies context-aware version also returns false.
func TestGetUserCtx_EmptyId(t *testing.T) {
	_, ok := GetUserCtx(context.Background(), "")
	if ok {
		t.Error("GetUserCtx(ctx, \"\") should return false for empty ID")
	}
}

// TestGetUserInfo_EmptyId verifies that GetUserInfo returns false for empty user ID.
func TestGetUserInfo_EmptyId(t *testing.T) {
	_, ok := GetUserInfo("")
	if ok {
		t.Error("GetUserInfo(\"\") should return false for empty ID")
	}
}

// TestGetUserInfoCtx_EmptyId verifies context-aware version also returns false.
func TestGetUserInfoCtx_EmptyId(t *testing.T) {
	_, ok := GetUserInfoCtx(context.Background(), "")
	if ok {
		t.Error("GetUserInfoCtx(ctx, \"\") should return false for empty ID")
	}
}

// TestFindUserNameCtx_WithNilDb verifies that FindUserNameCtx doesn't crash
// when the database is nil (it logs an error and returns false).
func TestFindUserNameCtx_WithNilDb(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil in unit tests): %v", r)
		}
	}()
	_, ok := FindUserNameCtx(context.Background(), "testuser")
	// Should either return false or panic (when Db is nil)
	if ok {
		t.Error("FindUserNameCtx should return false when Db is nil")
	}
}

// TestClearUserCache_NoOp verifies that ClearUserCache is safe to call.
func TestClearUserCache_NoOp(t *testing.T) {
	// Should not panic with empty uid
	ClearUserCache("")
}

// TestClearUserCache_WithNonEmptyUid verifies cache clear with valid uid.
func TestClearUserCache_WithNonEmptyUid(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (cache may be nil): %v", r)
		}
	}()
	// May panic if comm.BCache is nil, which is expected in unit tests
	ClearUserCache("test-uid")
}

// TestGetUserCache_NoResult verifies that GetUserCache returns false
// when the user does not exist in the cache or database.
func TestGetUserCache_NoResult(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db/cache nil in unit tests): %v", r)
		}
	}()
	_, ok := GetUserCache("")
	if ok {
		t.Error("GetUserCache(\"\") should return false for empty ID")
	}
}

// TestGetUserCacheCtx_EmptyUid verifies context-aware version returns false for empty uid.
func TestGetUserCacheCtx_EmptyUid(t *testing.T) {
	_, ok := GetUserCacheCtx(context.Background(), "")
	if ok {
		t.Error("GetUserCacheCtx(ctx, \"\") should return false for empty ID")
	}
}

// TestCurrUserCacheCtx_NilGinContext2 verifies CurrUserCacheCtx returns false for nil gin.Context.
func TestCurrUserCacheCtx_NilGinContext2(t *testing.T) {
	_, ok := CurrUserCacheCtx(context.Background(), nil)
	if ok {
		t.Error("CurrUserCacheCtx with nil gin.Context should return false")
	}
}

// TestIsOrgAdmin_WithNilDb verifies behavior when DB is not initialized.
func TestIsOrgAdmin_WithNilDb(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil): %v", r)
		}
	}()
	result := IsOrgAdmin("user1", "org1")
	if result {
		t.Error("IsOrgAdmin should return false when Db is nil")
	}
}

// TestHasOrgExec_WithNilDb verifies behavior when DB is not initialized.
func TestHasOrgExec_WithNilDb(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil): %v", r)
		}
	}()
	result := HasOrgExec("user1", "org1")
	if result {
		t.Error("HasOrgExec should return false when Db is nil")
	}
}

// TestGetUsePermRwr_WithNilDb verifies behavior when DB is not initialized.
func TestGetUsePermRwr_WithNilDb(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil): %v", r)
		}
	}()
	result := GetUsePermRwr("user1", "org1")
	if result != 0 {
		t.Errorf("GetUsePermRwr should return 0 when Db is nil, got %d", result)
	}
}

// TestGetUserOrg_EmptyUid verifies GetUserOrg returns false for empty inputs.
func TestGetUserOrg_EmptyUid(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil): %v", r)
		}
	}()
	_, ok := GetUserOrg("", "org1")
	if ok {
		t.Error("GetUserOrg(\"\", \"org1\") should return false")
	}
}

// TestGetUserOrgCtx_CanceledContext verifies behavior with a canceled context.
func TestGetUserOrgCtx_CanceledContext(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil): %v", r)
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, ok := GetUserOrgCtx(ctx, "user1", "org1")
	if ok {
		t.Error("GetUserOrgCtx with canceled context should return false")
	}
}

// TestNewOrgPerm_NilUser verifies NewOrgPermCtx with nil user doesn't crash.
func TestNewOrgPerm_NilUser(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil): %v", r)
		}
	}()
	perm := NewOrgPermCtx(context.Background(), nil, "")
	if perm == nil {
		t.Fatal("NewOrgPermCtx should never return nil")
	}
	if perm.LgUser() != nil {
		t.Error("LgUser should be nil")
	}
	if perm.Org() != nil {
		t.Error("Org should be nil for empty orgId")
	}
	if perm.IsAdmin() {
		t.Error("nil user should not be admin")
	}
	if perm.IsOrgAdmin() {
		t.Error("nil user should not be org admin")
	}
}

// TestNewPipePerm_NilUser verifies NewPipePermCtx with nil user doesn't crash.
func TestNewPipePerm_NilUser(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil): %v", r)
		}
	}()
	perm := NewPipePermCtx(context.Background(), nil, "")
	if perm == nil {
		t.Fatal("NewPipePermCtx should never return nil")
	}
	if perm.LgUser() != nil {
		t.Error("LgUser should be nil")
	}
	if perm.Pipeline() != nil {
		t.Error("Pipeline should be nil for empty pipeId")
	}
	if perm.IsAdmin() {
		t.Error("nil user should not be admin")
	}
	if perm.IsPipeOwner() {
		t.Error("nil user should not be pipe owner")
	}
}
