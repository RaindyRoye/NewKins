package service

import (
	"context"
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

func TestCurrUserCache_NilContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	
	// Should handle nil gin.Context gracefully
	usr, ok := CurrUserCache(nil)
	if ok || usr != nil {
		t.Errorf("CurrUserCache(nil) = (%v, %v), want (nil, false)", usr, ok)
	}
	
	// Should handle nil request gracefully
	usr, ok = CurrUserCache(c)
	if ok || usr != nil {
		t.Errorf("CurrUserCache with nil request = (%v, %v), want (nil, false)", usr, ok)
	}
}

func TestCurrUserCacheCtx_NilContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	ctx := context.Background()
	
	// Should handle nil gin.Context gracefully
	usr, ok := CurrUserCacheCtx(ctx, nil)
	if ok || usr != nil {
		t.Errorf("CurrUserCacheCtx(ctx, nil) = (%v, %v), want (nil, false)", usr, ok)
	}
	
	// Should handle nil request gracefully
	usr, ok = CurrUserCacheCtx(ctx, c)
	if ok || usr != nil {
		t.Errorf("CurrUserCacheCtx with nil request = (%v, %v), want (nil, false)", usr, ok)
	}
}

func TestLgUserKey_Constant(t *testing.T) {
	if LgUserKey != "lguser" {
		t.Errorf("LgUserKey = %q, want %q", LgUserKey, "lguser")
	}
}
