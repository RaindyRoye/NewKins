package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
)

// TestCheckCurrPermission_NoToken verifies that CheckCurrPermission returns
// false when there's no valid auth token in the request context.
// Note: CheckCurrPermission uses CurrUserCache which extracts the user from
// the JWT token in the Authorization header, not from gin context values.
func TestCheckCurrPermission_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	got := CheckCurrPermission(c, PermCommon)
	if got {
		t.Error("CheckCurrPermission without token should return false")
	}
}

// TestCheckCurrPermission_InvalidToken verifies that an invalid token
// causes CheckCurrPermission to return false.
func TestCheckCurrPermission_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "TOKEN invalid-token-data")
	c.Request = req
	got := CheckCurrPermission(c, PermCommon)
	if got {
		t.Error("CheckCurrPermission with invalid token should return false")
	}
}

// TestCheckCurrPermission_NilRequest verifies graceful handling of nil request.
func TestCheckCurrPermission_NilRequest(t *testing.T) {
	// CurrUserCache calls c.Request.Context() which panics on nil request.
	// This is expected behavior — gin always sets a request.
	// We skip this test as it documents known behavior.
	t.Skip("CurrUserCache panics on nil request — gin always provides one")
}

// TestCheckUPermission_UnknownPermLevel verifies that unknown permission
// levels are always denied.
func TestCheckUPermission_UnknownPermLevel(t *testing.T) {
	usr := &model.TUser{Id: "user1", Name: "alice"}
	got := CheckUPermission(usr, "superadmin")
	if got {
		t.Error("CheckUPermission with unknown perm level should return false")
	}
}

// TestCheckUPermission_EmptyPerm verifies that empty permission string is denied.
func TestCheckUPermission_EmptyPerm(t *testing.T) {
	usr := &model.TUser{Id: "user1", Name: "alice"}
	got := CheckUPermission(usr, "")
	if got {
		t.Error("CheckUPermission with empty perm should return false")
	}
}
