package util

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecretKey = "test-secret-key"
	testCookieName = "gokinstk"
)

func TestSetToken_CookieSecurityAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	key := testSecretKey
	claims := jwt.MapClaims{"uid": "user1"}

	_, err := SetToken(c, claims, key, false)
	if err != nil {
		t.Fatalf("SetToken failed: %v", err)
	}

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected at least one cookie to be set")
	}

	var found bool
	for _, cke := range cookies {
		if cke.Name == testCookieName {
			found = true
			// Check HttpOnly
			if !cke.HttpOnly {
				t.Error("cookie should have HttpOnly flag")
			}
			// Check Path
			if cke.Path != "/" {
				t.Errorf("cookie path should be '/', got %q", cke.Path)
			}
			// Check SameSite (Lax mode)
			if cke.SameSite != http.SameSiteLaxMode {
				t.Errorf("cookie SameSite should be Lax, got %v", cke.SameSite)
			}
			// For HTTP (non-TLS), Secure should be false
			if cke.Secure {
				t.Error("cookie Secure should be false for non-TLS requests")
			}
		}
	}
	if !found {
		t.Error("gokinstk cookie not found in response")
	}
}

func TestSetToken_SecureFlagForTLS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Simulate TLS request
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	c.Request = req

	key := testSecretKey
	claims := jwt.MapClaims{"uid": "user1"}

	_, err := SetToken(c, claims, key, false)
	if err != nil {
		t.Fatalf("SetToken failed: %v", err)
	}

	cookies := w.Result().Cookies()
	for _, cke := range cookies {
		if cke.Name == testCookieName {
			if !cke.Secure {
				t.Error("cookie should have Secure flag for TLS requests")
			}
		}
	}
}

func TestSetToken_RememberMeMaxAge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	key := testSecretKey
	claims := jwt.MapClaims{"uid": "user1"}

	_, err := SetToken(c, claims, key, true) // remember me = true
	if err != nil {
		t.Fatalf("SetToken failed: %v", err)
	}

	cookies := w.Result().Cookies()
	for _, cke := range cookies {
		if cke.Name == testCookieName {
			// 5 days = 432000 seconds
			expectedMaxAge := 60 * 60 * 24 * 5
			if cke.MaxAge != expectedMaxAge {
				t.Errorf("expected MaxAge %d for remember-me, got %d", expectedMaxAge, cke.MaxAge)
			}
		}
	}
}

func TestSetToken_NormalMaxAge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	key := testSecretKey
	claims := jwt.MapClaims{"uid": "user1"}

	_, err := SetToken(c, claims, key, false) // remember me = false
	if err != nil {
		t.Fatalf("SetToken failed: %v", err)
	}

	cookies := w.Result().Cookies()
	for _, cke := range cookies {
		if cke.Name == testCookieName {
			expectedMaxAge := 60 * 60 * 5
			if cke.MaxAge != expectedMaxAge {
				t.Errorf("expected MaxAge %d for normal session, got %d", expectedMaxAge, cke.MaxAge)
			}
		}
	}
}

func TestClearToken_SecurityAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	err := ClearToken(c)
	if err != nil {
		t.Fatalf("ClearToken failed: %v", err)
	}

	cookies := w.Result().Cookies()
	for _, cke := range cookies {
		if cke.Name == testCookieName {
			if cke.MaxAge != -1 {
				t.Errorf("expected MaxAge -1 for cleared cookie, got %d", cke.MaxAge)
			}
			if cke.Path != "/" {
				t.Errorf("cookie path should be '/', got %q", cke.Path)
			}
			if cke.SameSite != http.SameSiteLaxMode {
				t.Errorf("cookie SameSite should be Lax, got %v", cke.SameSite)
			}
			if !cke.HttpOnly {
				t.Error("cleared cookie should retain HttpOnly flag")
			}
		}
	}
}

func TestSetToken_WithDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	key := testSecretKey
	claims := jwt.MapClaims{"uid": "user1"}

	_, err := SetToken(c, claims, key, false, "example.com")
	if err != nil {
		t.Fatalf("SetToken failed: %v", err)
	}

	cookies := w.Result().Cookies()
	for _, cke := range cookies {
		if cke.Name == testCookieName {
			if cke.Domain != "example.com" {
				t.Errorf("expected domain 'example.com', got %q", cke.Domain)
			}
		}
	}
}

func TestSetToken_CookieRoundtrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := "roundtrip-key"
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	claims := jwt.MapClaims{"uid": "roundtrip-user"}
	_, err := SetToken(c, claims, key, false)
	if err != nil {
		t.Fatalf("SetToken failed: %v", err)
	}

	cookies := w.Result().Cookies()
	for _, cke := range cookies {
		if cke.Name == testCookieName {
			parsed := GetTokens(cke.Value, key)
			if parsed == nil {
				t.Fatal("failed to parse cookie token")
			}
			if parsed["uid"] != "roundtrip-user" {
				t.Errorf("expected uid 'roundtrip-user', got %v", parsed["uid"])
			}
			return
		}
	}
	t.Error("gokinstk cookie not found")
}

func TestGetTokenAuth_UrlEncodedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := testSecretKey
	claims := jwt.MapClaims{"uid": "encoded-user"}
	token, _ := CreateToken(claims, key, time.Hour)

	// URL-encode the token dots
	encoded := strings.ReplaceAll(token, ".", "%2E")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "TOKEN "+encoded)
	c.Request = req

	parsed := GetToken(c, key)
	if parsed == nil {
		t.Fatal("expected to parse URL-encoded token")
	}
	if parsed["uid"] != "encoded-user" {
		t.Errorf("expected uid 'encoded-user', got %v", parsed["uid"])
	}
}

func TestClearToken_WithDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	err := ClearToken(c, "example.com")
	if err != nil {
		t.Fatalf("ClearToken failed: %v", err)
	}

	cookies := w.Result().Cookies()
	for _, cke := range cookies {
		if cke.Name == testCookieName {
			if cke.Domain != "example.com" {
				t.Errorf("expected domain 'example.com', got %q", cke.Domain)
			}
			if cke.MaxAge != -1 {
				t.Errorf("expected MaxAge -1, got %d", cke.MaxAge)
			}
		}
	}
}
