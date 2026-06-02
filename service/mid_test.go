package service

import (
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
