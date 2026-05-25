package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestCreateToken_Success(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": "user123",
	}
	key := "test-secret-key"
	timeout := time.Hour

	token, err := CreateToken(claims, key, timeout)
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}
	if token == "" {
		t.Fatal("CreateToken returned empty token")
	}

	// Verify the token can be parsed back
	parsed, err := jwt.Parse(token, func(tk *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})
	if err != nil {
		t.Fatalf("failed to parse created token: %v", err)
	}
	if !parsed.Valid {
		t.Error("created token should be valid")
	}

	mc, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("claims should be MapClaims")
	}
	if mc["uid"] != "user123" {
		t.Errorf("expected uid 'user123', got %v", mc["uid"])
	}
	// Should have times and timeout set
	if _, ok := mc["times"]; !ok {
		t.Error("expected 'times' claim to be set")
	}
	if _, ok := mc["timeout"]; !ok {
		t.Error("expected 'timeout' claim to be set")
	}
}

func TestCreateToken_ZeroTimeout(t *testing.T) {
	claims := jwt.MapClaims{
		"uid": "user456",
	}
	key := "test-secret-key"

	token, err := CreateToken(claims, key, 0)
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	parsed, err := jwt.Parse(token, func(tk *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	mc := parsed.Claims.(jwt.MapClaims)
	// timeout should NOT be set when tmout is 0
	if _, ok := mc["timeout"]; ok {
		t.Error("timeout should not be set when tmout is 0")
	}
	// times should still be set
	if _, ok := mc["times"]; !ok {
		t.Error("expected 'times' claim to be set")
	}
}

func TestGetTokens_ValidToken(t *testing.T) {
	key := "my-secret-key"
	claims := jwt.MapClaims{
		"uid":  "testuser",
		"role": "admin",
	}
	token, err := CreateToken(claims, key, time.Hour)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	result := GetTokens(token, key)
	if result == nil {
		t.Fatal("GetTokens should return non-nil claims for valid token")
	}
	if result["uid"] != "testuser" {
		t.Errorf("expected uid 'testuser', got %v", result["uid"])
	}
	if result["role"] != "admin" {
		t.Errorf("expected role 'admin', got %v", result["role"])
	}
}

func TestGetTokens_EmptyString(t *testing.T) {
	result := GetTokens("", "any-key")
	if result != nil {
		t.Error("GetTokens should return nil for empty string")
	}
}

func TestGetTokens_InvalidToken(t *testing.T) {
	result := GetTokens("not-a-valid-jwt", "key")
	if result != nil {
		t.Error("GetTokens should return nil for invalid token")
	}
}

func TestGetTokens_WrongKey(t *testing.T) {
	claims := jwt.MapClaims{"uid": "user1"}
	token, err := CreateToken(claims, "correct-key", time.Hour)
	if err != nil {
		t.Fatalf("CreateToken failed: %v", err)
	}

	result := GetTokens(token, "wrong-key")
	if result != nil {
		t.Error("GetTokens should return nil when key doesn't match")
	}
}

func TestGetTokenAuth_WithToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "TOKEN my-auth-token-123")
	c.Request = req

	got := getTokenAuth(c)
	if got != "my-auth-token-123" {
		t.Errorf("expected 'my-auth-token-123', got %q", got)
	}
}

func TestGetTokenAuth_EmptyHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request = req

	got := getTokenAuth(c)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestGetTokenAuth_NoTokenPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	c.Request = req

	got := getTokenAuth(c)
	// "TOKEN " prefix is stripped, so "Bearer some-token" stays as-is
	if got != "Bearer some-token" {
		t.Errorf("expected 'Bearer some-token', got %q", got)
	}
}

func TestGetToken_FromCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a valid token
	key := "cookie-test-key"
	claims := jwt.MapClaims{"uid": "cookieuser"}
	token, _ := CreateToken(claims, key, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "gokinstk",
		Value: token,
	})
	c.Request = req

	result := GetToken(c, key)
	if result == nil {
		t.Fatal("GetToken should return claims from cookie")
	}
	if result["uid"] != "cookieuser" {
		t.Errorf("expected uid 'cookieuser', got %v", result["uid"])
	}
}

func TestGetToken_FromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	key := "query-test-key"
	claims := jwt.MapClaims{"uid": "queryuser"}
	token, _ := CreateToken(claims, key, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/test?authToken="+token, nil)
	c.Request = req

	result := GetToken(c, key)
	if result == nil {
		t.Fatal("GetToken should return claims from query param")
	}
	if result["uid"] != "queryuser" {
		t.Errorf("expected uid 'queryuser', got %v", result["uid"])
	}
}

func TestGetToken_AuthHeaderPriority(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	key := "priority-test-key"

	// Create two different tokens
	authClaims := jwt.MapClaims{"uid": "authuser"}
	authToken, _ := CreateToken(authClaims, key, time.Hour)

	queryClaims := jwt.MapClaims{"uid": "queryuser"}
	queryToken, _ := CreateToken(queryClaims, key, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/test?authToken="+queryToken, nil)
	req.Header.Set("Authorization", "TOKEN "+authToken)
	c.Request = req

	result := GetToken(c, key)
	if result == nil {
		t.Fatal("GetToken should return claims")
	}
	// Authorization header should take priority
	if result["uid"] != "authuser" {
		t.Errorf("expected auth header to take priority, got uid %v", result["uid"])
	}
}

func TestGetToken_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request = req

	result := GetToken(c, "any-key")
	if result != nil {
		t.Error("GetToken should return nil when no token is present")
	}
}
