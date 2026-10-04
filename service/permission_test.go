package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
)

func TestCheckCurrPermission_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a request without auth token
	req, _ := http.NewRequest("GET", "/test", nil)
	c.Request = req

	result := CheckCurrPermission(c, "common")
	if result {
		t.Error("CheckCurrPermission without auth should return false")
	}
}

func TestCheckCurrPermission_WithAdminUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/test", nil)
	c.Request = req

	// Set up a mock admin user in the context
	adminUser := &model.TUser{
		Id:     "admin",
		Name:   "admin",
		Nick:   "Admin User",
		Active: 1,
	}
	c.Set(LgUserKey, adminUser)

	// Mock the CurrUserCache function behavior by setting the user directly
	// Since we can't easily mock the cache, we test the permission logic
	result := CheckUPermission(adminUser, "admin")
	if !result {
		t.Error("Admin user should have admin permission")
	}
}

func TestCheckCurrPermission_WithRegularUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/test", nil)
	c.Request = req

	regularUser := &model.TUser{
		Id:     "user1",
		Name:   "testuser",
		Nick:   "Test User",
		Active: 1,
	}
	c.Set(LgUserKey, regularUser)

	// Test common permission (should pass for any authenticated user)
	result := CheckUPermission(regularUser, "common")
	if !result {
		t.Error("Regular user should have common permission")
	}

	// Test admin permission (should fail for regular user)
	result = CheckUPermission(regularUser, "admin")
	if result {
		t.Error("Regular user should not have admin permission")
	}
}

func TestCheckCurrPermission_UnknownPermission(t *testing.T) {
	regularUser := &model.TUser{
		Id:     "user1",
		Name:   "testuser",
		Active: 1,
	}

	result := CheckUPermission(regularUser, "unknown_permission")
	if result {
		t.Error("Unknown permission should return false")
	}
}

func TestCheckCurrPermission_EmptyPermission(t *testing.T) {
	regularUser := &model.TUser{
		Id:     "user1",
		Name:   "testuser",
		Active: 1,
	}

	result := CheckUPermission(regularUser, "")
	if result {
		t.Error("Empty permission should return false")
	}
}
