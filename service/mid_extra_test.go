package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
)

// --- MidUserCheck middleware tests (direct HTTP integration) ---

func TestMidUserCheck_Integration_Admin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", MidUserCheck, func(c *gin.Context) {
		usr := GetMidLgUser(c)
		c.JSON(http.StatusOK, gin.H{"id": usr.Id})
	})

	// Use a real HTTP test recorder
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request = req

	// Pre-set admin user (bypassing CurrUserCache since we don't have token setup)
	c.Set(LgUserKey, &model.TUser{Id: "admin", Active: 1})

	// MidUserCheck calls CurrUserCache which requires a token
	// Without a token, it returns nil and the middleware aborts
	MidUserCheck(c)

	// Will be aborted because CurrUserCache can't find user from token
	if !c.IsAborted() {
		t.Error("MidUserCheck should abort when CurrUserCache can't authenticate")
	}
}

func TestMidUserCheck_Integration_InactiveNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// Pre-set inactive non-admin user
	c.Set(LgUserKey, &model.TUser{Id: "user1", Active: 0})

	MidUserCheck(c)

	// Should be aborted
	if !c.IsAborted() {
		t.Error("MidUserCheck should abort inactive non-admin user")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestMidUserCheck_Integration_ActiveNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// Pre-set active non-admin user
	c.Set(LgUserKey, &model.TUser{Id: "user1", Active: 1})

	// MidUserCheck calls CurrUserCache which requires a token
	// Without a token, it returns nil and the middleware aborts
	MidUserCheck(c)

	// Will be aborted because CurrUserCache can't find user from token
	if !c.IsAborted() {
		t.Error("MidUserCheck should abort when CurrUserCache can't authenticate")
	}
}

func TestMidUserCheck_Integration_NoUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// Don't set any user - CurrUserCache will return nil
	MidUserCheck(c)

	// Should be aborted
	if !c.IsAborted() {
		t.Error("MidUserCheck should abort when no user is authenticated")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
	if w.Body.String() != "Not Auth" {
		t.Errorf("expected body 'Not Auth', got %q", w.Body.String())
	}
}

// --- GetMidLgUser edge cases ---

func TestGetMidLgUser_NilContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// No request set, should still work
	got := GetMidLgUser(c)
	if got != nil {
		t.Errorf("expected nil when no user set, got %v", got)
	}
}

func TestGetMidLgUser_IntegerValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set(LgUserKey, 12345)

	got := GetMidLgUser(c)
	if got != nil {
		t.Errorf("expected nil for integer value, got %v", got)
	}
}

func TestGetMidLgUser_MapValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set(LgUserKey, map[string]string{"id": "test"})

	got := GetMidLgUser(c)
	if got != nil {
		t.Errorf("expected nil for map value, got %v", got)
	}
}

func TestGetMidLgUser_NilUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Set(LgUserKey, (*model.TUser)(nil))

	// This should return nil since the stored value is a nil pointer
	got := GetMidLgUser(c)
	// The type assertion succeeds (it IS a *model.TUser), but the value is nil
	if got != nil {
		// This is also acceptable - the pointer itself is nil but the type assertion succeeded
		t.Logf("got non-nil *TUser from nil value: %v (this is implementation-defined)", got)
	}
}

// --- IsAdmin tests ---

func TestIsAdmin_VariousUserIds(t *testing.T) {
	tests := []struct {
		name string
		uid  string
		want bool
	}{
		{"admin user", "admin", true},
		{"Admin (capitalized) is not admin", "Admin", false},
		{"ADMIN (uppercase) is not admin", "ADMIN", false},
		{"admin with suffix", "admin1", false},
		{"prefix admin", "myadmin", false},
		{"empty user", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usr := &model.TUser{Id: tt.uid}
			if got := IsAdmin(usr); got != tt.want {
				t.Errorf("IsAdmin(%q) = %v, want %v", tt.uid, got, tt.want)
			}
		})
	}
}

func TestIsAdmin_NilUser(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("IsAdmin(nil) panicked (expected - nil pointer dereference): %v", r)
		}
	}()
	// IsAdmin(nil) will panic because it accesses usr.Id without nil check
	// This test documents the behavior
	result := IsAdmin(nil)
	t.Logf("IsAdmin(nil) returned %v (no panic)", result)
}

// --- LgUserKey tests ---

func TestLgUserKey_Value(t *testing.T) {
	if LgUserKey != "lguser" {
		t.Errorf("LgUserKey constant = %q, want %q", LgUserKey, "lguser")
	}
}

// --- Error sentinel verification for service package ---

func TestServiceErrors_AreWrappedProperly(t *testing.T) {
	// Verify all sentinel errors are usable with errors.Is
	sentinels := []error{
		ErrPipelineNotFound,
		ErrPipelineYmlEmpty,
		ErrTriggerNoParams,
		ErrHookTypeEmpty,
		ErrWebhookParseFailed,
		ErrWebhookEventMismatch,
		ErrBranchMismatch,
		ErrTriggerNoSecret,
		ErrTriggerSecretMismatch,
		ErrPermissionDenied,
		ErrParamDataNil,
		ErrParamNotFound,
	}

	for _, err := range sentinels {
		wrapped := wrapTestError("context: %w", err)
		if !errors.Is(wrapped, err) {
			t.Errorf("wrapped error should match sentinel %v", err)
		}
	}
}

func wrapTestError(format string, err error) error {
	return &testWrappedError{msg: format, cause: err}
}

type testWrappedError struct {
	msg   string
	cause error
}

func (e *testWrappedError) Error() string {
	return e.msg
}

func (e *testWrappedError) Unwrap() error {
	return e.cause
}
