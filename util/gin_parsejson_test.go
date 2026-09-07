package util

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGinReqParseJson_NonFunc(t *testing.T) {
	// Non-function should return nil handler
	handler := GinReqParseJson("not a function")
	if handler != nil {
		t.Error("GinReqParseJson with non-func should return nil")
	}

	handler = GinReqParseJson(42)
	if handler != nil {
		t.Error("GinReqParseJson with int should return nil")
	}
}

func TestGinReqParseJson_SimpleHandler(t *testing.T) {
	e := gin.New()
	called := false
	handler := GinReqParseJson(func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "ok")
	})
	if handler == nil {
		t.Fatal("GinReqParseJson with valid func should return handler")
	}

	e.POST("/test", handler)
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", nil)
	e.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGinReqParseJson_WithJSONBody(t *testing.T) {
	type testReq struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	e := gin.New()
	var receivedName string
	var receivedAge int
	handler := GinReqParseJson(func(c *gin.Context, req *testReq) {
		receivedName = req.Name
		receivedAge = req.Age
		c.String(http.StatusOK, "ok")
	})
	e.POST("/test", handler)

	body, _ := json.Marshal(testReq{Name: "alice", Age: 30})
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if receivedName != "alice" {
		t.Errorf("expected name 'alice', got %q", receivedName)
	}
	if receivedAge != 30 {
		t.Errorf("expected age 30, got %d", receivedAge)
	}
}

func TestGinReqParseJson_InvalidJSON(t *testing.T) {
	type testReq struct {
		Name string `json:"name"`
	}

	e := gin.New()
	handler := GinReqParseJson(func(c *gin.Context, req *testReq) {
		// Should not reach here
		c.String(http.StatusOK, "ok")
	})
	e.POST("/test", handler)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test",
		strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid JSON, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid request body") {
		t.Errorf("expected error message about invalid body, got %q", w.Body.String())
	}
}

func TestGinReqParseJson_MapArg(t *testing.T) {
	e := gin.New()
	var receivedVal string
	handler := GinReqParseJson(func(c *gin.Context, m *map[string]string) {
		if v, ok := (*m)["key"]; ok {
			receivedVal = v
		}
		c.String(http.StatusOK, "ok")
	})
	e.POST("/test", handler)

	body, _ := json.Marshal(map[string]string{"key": "value"})
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if receivedVal != "value" {
		t.Errorf("expected 'value', got %q", receivedVal)
	}
}

func TestGinReqParseJson_NonJSONContentType(t *testing.T) {
	type testReq struct {
		Name string `json:"name"`
	}

	e := gin.New()
	handlerCalled := false
	handler := GinReqParseJson(func(c *gin.Context, req *testReq) {
		handlerCalled = true
		// With non-JSON content type, req is the zero value (nil pointer)
		if req != nil {
			t.Errorf("expected nil pointer for non-JSON content type, got %+v", req)
		}
		c.String(http.StatusOK, "ok")
	})
	e.POST("/test", handler)

	// Send form-encoded data instead of JSON
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test",
		strings.NewReader("name=bob"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	e.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !handlerCalled {
		t.Error("handler should still be called with non-JSON content type")
	}
}

func TestGinReqParseJson_RecoverPanic(t *testing.T) {
	e := gin.New()
	handler := GinReqParseJson(func(c *gin.Context) {
		panic("test panic in handler")
	})
	e.POST("/test", handler)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", nil)
	e.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 after panic, got %d", w.Code)
	}
	if w.Body.String() != "internal server error" {
		t.Errorf("expected 'internal server error', got %q", w.Body.String())
	}
}

func TestGinReqParseJson_NonStructArg(t *testing.T) {
	// Test with a non-struct, non-map pointer argument (should skip binding)
	e := gin.New()
	called := false
	handler := GinReqParseJson(func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "ok")
	})
	e.POST("/test", handler)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", nil)
	e.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
}
