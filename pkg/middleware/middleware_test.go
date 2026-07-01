package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockController implements GinController for testing
type mockController struct {
	path string
}

func (m *mockController) GetPath() string { return m.path }
func (m *mockController) Routes(g gin.IRoutes) {
	g.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
}

func TestRegisterController(t *testing.T) {
	tests := []struct {
		name       string
		engine     *gin.Engine
		controller GinController
		wantPath   string
	}{
		{
			name:       "root path controller",
			engine:     gin.New(),
			controller: &mockController{path: "/"},
			wantPath:   "/test",
		},
		{
			name:       "grouped path controller",
			engine:     gin.New(),
			controller: &mockController{path: "/api"},
			wantPath:   "/api/test",
		},
		{
			name:       "nil engine does not panic",
			engine:     nil,
			controller: &mockController{path: "/api"},
			wantPath:   "",
		},
		{
			name:       "nil controller does not panic",
			engine:     gin.New(),
			controller: nil,
			wantPath:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			RegisterController(tt.engine, tt.controller)

			if tt.engine != nil && tt.controller != nil {
				req := httptest.NewRequest(http.MethodGet, tt.wantPath, nil)
				w := httptest.NewRecorder()
				tt.engine.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					t.Errorf("expected status 200, got %d", w.Code)
				}
			}
		})
	}
}

func TestJSONBinder_NonFunction(t *testing.T) {
	handler := JSONBinder("not a function")
	if handler != nil {
		t.Error("expected nil handler for non-function input")
	}
}

func TestJSONBinder_ValidJSON(t *testing.T) {
	type TestPayload struct {
		Name string `json:"name"`
	}

	handler := JSONBinder(func(c *gin.Context, p TestPayload) {
		if p.Name != "test" {
			t.Errorf("expected name 'test', got %q", p.Name)
		}
		c.String(http.StatusOK, "ok")
	})

	engine := gin.New()
	engine.POST("/test", handler)

	body := `{"name": "test"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestJSONBinder_InvalidJSON(t *testing.T) {
	type TestPayload struct {
		Name string `json:"name"`
	}

	handler := JSONBinder(func(c *gin.Context, p TestPayload) {
		c.String(http.StatusOK, "ok")
	})

	engine := gin.New()
	engine.POST("/test", handler)

	body := `{"invalid json`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestJSONBinder_PointerParam(t *testing.T) {
	type TestPayload struct {
		Value int `json:"value"`
	}

	handler := JSONBinder(func(c *gin.Context, p *TestPayload) {
		if p == nil {
			t.Error("expected non-nil pointer")
		} else if p.Value != 42 {
			t.Errorf("expected value 42, got %d", p.Value)
		}
		c.String(http.StatusOK, "ok")
	})

	engine := gin.New()
	engine.POST("/test", handler)

	body := `{"value": 42}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestJSONBinder_MapParam(t *testing.T) {
	handler := JSONBinder(func(c *gin.Context, m map[string]string) {
		if m["key"] != "value" {
			t.Errorf("expected m['key']='value', got %q", m["key"])
		}
		c.String(http.StatusOK, "ok")
	})

	engine := gin.New()
	engine.POST("/test", handler)

	body := `{"key": "value"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestJSONBinder_NonJSONContentType(t *testing.T) {
	type TestPayload struct {
		Name string `json:"name"`
	}

	handler := JSONBinder(func(c *gin.Context, p TestPayload) {
		// p should be zero value since we didn't bind JSON
		if p.Name != "" {
			t.Errorf("expected empty name for non-JSON content, got %q", p.Name)
		}
		c.String(http.StatusOK, "ok")
	})

	engine := gin.New()
	engine.POST("/test", handler)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("plain text"))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestJSONBinder_PanicRecovery(t *testing.T) {
	handler := JSONBinder(func(c *gin.Context) {
		panic("test panic")
	})

	engine := gin.New()
	engine.POST("/test", handler)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "internal server error") {
		t.Errorf("expected 'internal server error' in response, got %q", w.Body.String())
	}
}

func TestInternalError(t *testing.T) {
	engine := gin.New()
	engine.GET("/test", func(c *gin.Context) {
		InternalError(c, "database query failed", errors.New("connection refused"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	if w.Body.String() != "internal server error" {
		t.Errorf("expected generic error message, got %q", w.Body.String())
	}
}

func TestRespondError(t *testing.T) {
	engine := gin.New()
	engine.GET("/test", func(c *gin.Context) {
		RespondError(c, http.StatusBadRequest, "invalid input", errors.New("name is required"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	if w.Body.String() != "invalid input" {
		t.Errorf("expected 'invalid input', got %q", w.Body.String())
	}
}

func TestCORSAllowAll_OPTIONS(t *testing.T) {
	engine := gin.New()
	engine.Use(CORSAllowAll())
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for OPTIONS, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("expected CORS origin header, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSAllowAll_POST(t *testing.T) {
	engine := gin.New()
	engine.Use(CORSAllowAll())
	engine.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("expected CORS origin header on POST, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSAllowAll_GET(t *testing.T) {
	engine := gin.New()
	engine.Use(CORSAllowAll())
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	// GET requests should not have CORS headers set (only POST and OPTIONS)
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("GET should not set CORS headers, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}
