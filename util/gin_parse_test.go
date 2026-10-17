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

// TestGinReqParseJson_NonFunc verifies that passing a non-function returns nil handler.
func TestGinReqParseJson_NonFunc(t *testing.T) {
	h := GinReqParseJson("not a function")
	if h != nil {
		t.Error("expected nil handler for non-function argument")
	}
}

// TestGinReqParseJson_SimpleHandler verifies a simple handler with no extra args.
func TestGinReqParseJson_SimpleHandler(t *testing.T) {
	called := false
	h := GinReqParseJson(func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "ok")
	})
	if h == nil {
		t.Fatal("expected non-nil handler")
	}

	e := gin.New()
	e.GET("/test", h)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	e.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestGinReqParseJson_StructArg verifies JSON binding to a struct argument.
func TestGinReqParseJson_StructArg(t *testing.T) {
	type Payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	var received Payload
	h := GinReqParseJson(func(c *gin.Context, p Payload) {
		received = p
		c.String(http.StatusOK, "ok")
	})

	e := gin.New()
	e.POST("/test", h)

	body, _ := json.Marshal(Payload{Name: "Alice", Age: 30})
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(w, req)

	if received.Name != "Alice" {
		t.Errorf("expected name 'Alice', got '%s'", received.Name)
	}
	if received.Age != 30 {
		t.Errorf("expected age 30, got %d", received.Age)
	}
}

// TestGinReqParseJson_PointerStructArg verifies JSON binding to a pointer struct argument.
func TestGinReqParseJson_PointerStructArg(t *testing.T) {
	type Payload struct {
		Value string `json:"value"`
	}

	var received *Payload
	h := GinReqParseJson(func(c *gin.Context, p *Payload) {
		received = p
		c.String(http.StatusOK, "ok")
	})

	e := gin.New()
	e.POST("/test", h)

	body, _ := json.Marshal(Payload{Value: "hello"})
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(w, req)

	if received == nil {
		t.Fatal("expected non-nil received pointer")
	}
	if received.Value != "hello" {
		t.Errorf("expected value 'hello', got '%s'", received.Value)
	}
}

// TestGinReqParseJson_MapArg verifies JSON binding to a map argument.
func TestGinReqParseJson_MapArg(t *testing.T) {
	var received map[string]any
	h := GinReqParseJson(func(c *gin.Context, p map[string]any) {
		received = p
		c.String(http.StatusOK, "ok")
	})

	e := gin.New()
	e.POST("/test", h)

	body, _ := json.Marshal(map[string]any{"key": "val"})
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(w, req)

	if received == nil {
		t.Fatal("expected non-nil map")
	}
	if received["key"] != "val" {
		t.Errorf("expected key='val', got '%v'", received["key"])
	}
}

// TestGinReqParseJson_InvalidJSON verifies that invalid JSON returns 400.
func TestGinReqParseJson_InvalidJSON(t *testing.T) {
	type Payload struct {
		Name string `json:"name"`
	}

	h := GinReqParseJson(func(c *gin.Context, p Payload) {
		c.String(http.StatusOK, "ok")
	})

	e := gin.New()
	e.POST("/test", h)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

// TestGinReqParseJson_NonJsonContentType verifies that non-JSON content type skips binding.
func TestGinReqParseJson_NonJsonContentType(t *testing.T) {
	type Payload struct {
		Name string `json:"name"`
	}

	var received Payload
	h := GinReqParseJson(func(c *gin.Context, p Payload) {
		received = p
		c.String(http.StatusOK, "ok")
	})

	e := gin.New()
	e.POST("/test", h)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader("form data"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	e.ServeHTTP(w, req)

	// Struct should be zero value since JSON binding was skipped
	if received.Name != "" {
		t.Errorf("expected empty name for non-JSON content type, got '%s'", received.Name)
	}
}

// TestGinReqParseJson_MultipleArgs verifies handler behavior with two struct args.
// Note: GinReqParseJson calls BindJSON for each struct arg. Since Gin's BindJSON
// consumes the request body, the second bind fails and the handler returns 400.
// This is a known limitation - only one bindable argument is supported per request.
func TestGinReqParseJson_MultipleArgs(t *testing.T) {
	type First struct {
		A string `json:"a"`
	}
	type Second struct {
		B int `json:"b"`
	}

	called := false
	h := GinReqParseJson(func(c *gin.Context, f First, s *Second) {
		called = true
		c.String(http.StatusOK, "ok")
	})

	e := gin.New()
	e.POST("/multi", h)

	body, _ := json.Marshal(First{A: "test"})
	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/multi", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(w, req)

	// Handler should not be called because second bind fails
	if called {
		t.Error("handler should not be called when second bind fails")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestGinReqParseJson_NonBindableType verifies that non-struct/non-map types are left as zero.
func TestGinReqParseJson_NonBindableType(t *testing.T) {
	var gotInt int
	var gotStr string
	h := GinReqParseJson(func(c *gin.Context, n int, s string) {
		gotInt = n
		gotStr = s
		c.String(http.StatusOK, "ok")
	})

	e := gin.New()
	e.POST("/test", h)

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(w, req)

	if gotInt != 0 {
		t.Errorf("expected int=0, got %d", gotInt)
	}
	if gotStr != "" {
		t.Errorf("expected empty string, got '%s'", gotStr)
	}
}
