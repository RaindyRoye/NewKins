package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoginController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c := &LoginController{}
	c.Routes(r.Group("/api/lg"))

	// Verify routes are registered by checking the route tree
	routes := r.Routes()
	found := map[string]bool{}
	for _, route := range routes {
		found[route.Method+":"+route.Path] = true
	}
	if !found["POST:/api/lg/info"] {
		t.Error("POST /api/lg/info route not registered")
	}
	if !found["POST:/api/lg/login"] {
		t.Error("POST /api/lg/login route not registered")
	}
}

func TestLogin_Info_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c := &LoginController{}
	c.Routes(r.Group("/api/lg"))

	req := httptest.NewRequest("POST", "/api/lg/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if login, ok := resp["login"].(bool); !ok || login != false {
		t.Errorf("expected login=false, got %v", resp["login"])
	}
}

func TestLogin_Info_ContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c := &LoginController{}
	c.Routes(r.Group("/api/lg"))

	req := httptest.NewRequest("POST", "/api/lg/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestLogin_Login_EmptyParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c := &LoginController{}
	c.Routes(r.Group("/api/lg"))

	body := bytes.NewBufferString(`{"name":"","pass":""}`)
	req := httptest.NewRequest("POST", "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty params, got %d", w.Code)
	}
	if w.Body.String() != "param err" {
		t.Errorf("expected 'param err', got %q", w.Body.String())
	}
}

func TestLogin_Login_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c := &LoginController{}
	c.Routes(r.Group("/api/lg"))

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest("POST", "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestLogin_Login_NameOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c := &LoginController{}
	c.Routes(r.Group("/api/lg"))

	body := bytes.NewBufferString(`{"name":"testuser","pass":""}`)
	req := httptest.NewRequest("POST", "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing password, got %d", w.Code)
	}
}

func TestLogin_Login_PassOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c := &LoginController{}
	c.Routes(r.Group("/api/lg"))

	body := bytes.NewBufferString(`{"name":"","pass":"testpass"}`)
	req := httptest.NewRequest("POST", "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing username, got %d", w.Code)
	}
}

func TestLogin_Login_WhitespaceName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c := &LoginController{}
	c.Routes(r.Group("/api/lg"))

	body := bytes.NewBufferString(`{"name":"   ","pass":"testpass"}`)
	req := httptest.NewRequest("POST", "/api/lg/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for whitespace-only name, got %d", w.Code)
	}
}

func TestLogin_Login_NoContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c := &LoginController{}
	c.Routes(r.Group("/api/lg"))

	// POST without JSON content-type - handler receives zero-value struct
	body := bytes.NewBufferString(`{"name":"testuser","pass":"testpass"}`)
	req := httptest.NewRequest("POST", "/api/lg/login", body)
	// Intentionally NOT setting Content-Type to application/json
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// With no JSON content-type, GinReqParseJson passes zero-value struct
	// which has empty name and pass, so we expect "param err"
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for no content-type, got %d", w.Code)
	}
}
