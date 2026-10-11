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

func TestGinReqParseJson_NonFuncReturnsNil(t *testing.T) {
	// Passing a non-function should return nil
	h := GinReqParseJson("not a function")
	if h != nil {
		t.Fatal("expected nil handler for non-function input")
	}
}

func TestGinReqParseJson_NonFuncIntReturnsNil(t *testing.T) {
	h := GinReqParseJson(42)
	if h != nil {
		t.Fatal("expected nil handler for int input")
	}
}

func TestGinReqParseJson_BasicHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	fn := func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "hello")
	}

	handler := GinReqParseJson(fn)
	if handler == nil {
		t.Fatal("expected non-nil handler for valid function")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)

	handler(c)
	if !called {
		t.Error("handler was not invoked")
	}
}

func TestGinReqParseJson_WithStructParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type TestPayload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	var receivedName string
	var receivedAge int
	fn := func(c *gin.Context, p *TestPayload) {
		receivedName = p.Name
		receivedAge = p.Age
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	body := TestPayload{Name: "Alice", Age: 30}
	bts, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", bytes.NewReader(bts))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	if receivedName != "Alice" {
		t.Errorf("expected name Alice, got %q", receivedName)
	}
	if receivedAge != 30 {
		t.Errorf("expected age 30, got %d", receivedAge)
	}
}

func TestGinReqParseJson_WithStructValueParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type Payload struct {
		Value string `json:"value"`
	}

	var receivedValue string
	fn := func(c *gin.Context, p Payload) {
		receivedValue = p.Value
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	body := Payload{Value: "test123"}
	bts, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test", bytes.NewReader(bts))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)
	if receivedValue != "test123" {
		t.Errorf("expected value 'test123', got %q", receivedValue)
	}
}

func TestGinReqParseJson_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type Payload struct {
		Name string `json:"name"`
	}

	fn := func(c *gin.Context, p *Payload) {
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test",
		strings.NewReader(`{invalid`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestGinReqParseJson_MapParam(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var received map[string]string
	fn := func(c *gin.Context, m *map[string]string) {
		received = *m
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	body := map[string]string{"key": "val"}
	bts, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test",
		bytes.NewReader(bts))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)
	if received["key"] != "val" {
		t.Errorf("expected map key=val, got %v", received)
	}
}

func TestGinReqParseJson_NonJSONContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type Payload struct {
		Name string `json:"name"`
	}

	called := false
	var zeroName string
	fn := func(c *gin.Context, p *Payload) {
		called = true
		zeroName = p.Name
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/test",
		strings.NewReader("text body"))
	c.Request.Header.Set("Content-Type", "text/plain")

	handler(c)
	if !called {
		t.Error("handler should still be called with non-JSON content type")
	}
	// The struct should remain zero-valued since it wasn't bound from JSON
	if zeroName != "" {
		t.Errorf("expected zero-value name, got %q", zeroName)
	}
}

func TestGinReqParseJson_PanicRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fn := func(c *gin.Context) {
		panic("test panic")
	}

	handler := GinReqParseJson(fn)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)

	// Should not propagate the panic
	handler(c)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 after panic recovery, got %d", w.Code)
	}
}
