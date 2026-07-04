package util

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGinReqParseJson_NoContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Handler expects a pointer to a struct
	type TestParam struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	handlerCalled := false
	handler := func(c *gin.Context, m *TestParam) {
		handlerCalled = true
		// m should never be nil thanks to our fix
		if m == nil {
			t.Error("handler received nil pointer, expected zero-value struct")
			return
		}
		// With no JSON content-type, m should be zero-value
		if m.Name != "" {
			t.Errorf("expected empty Name, got %q", m.Name)
		}
		if m.Age != 0 {
			t.Errorf("expected Age=0, got %d", m.Age)
		}
		c.String(http.StatusOK, "ok")
	}

	r.POST("/test", GinReqParseJson(handler))

	body := bytes.NewBufferString(`{"name":"test","age":30}`)
	req := httptest.NewRequest("POST", "/test", body)
	// Intentionally NOT setting Content-Type to application/json
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGinReqParseJson_WithJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	type TestParam struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	handler := func(c *gin.Context, m *TestParam) {
		if m == nil {
			t.Error("handler received nil pointer")
			c.String(http.StatusInternalServerError, "nil")
			return
		}
		if m.Name != "Alice" {
			t.Errorf("expected Name=Alice, got %q", m.Name)
		}
		if m.Age != 25 {
			t.Errorf("expected Age=25, got %d", m.Age)
		}
		c.String(http.StatusOK, "ok")
	}

	r.POST("/test", GinReqParseJson(handler))

	body := bytes.NewBufferString(`{"name":"Alice","age":25}`)
	req := httptest.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
