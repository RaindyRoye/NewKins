package util

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// testReqBody is a sample struct used to test JSON binding.
type testReqBody struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func TestGinReqParseJson_NilFn(t *testing.T) {
	handler := GinReqParseJson(nil)
	if handler != nil {
		t.Error("GinReqParseJson(nil) should return nil")
	}
}

func TestGinReqParseJson_NonFunction(t *testing.T) {
	// Pass a non-function value
	handler := GinReqParseJson("not a function")
	if handler != nil {
		t.Error("GinReqParseJson(string) should return nil for non-function")
	}
}

func TestGinReqParseJson_IntArg(t *testing.T) {
	handler := GinReqParseJson(42)
	if handler != nil {
		t.Error("GinReqParseJson(int) should return nil for non-function")
	}
}

func TestGinReqParseJson_ContextOnly(t *testing.T) {
	// A handler that only takes *gin.Context
	called := false
	handler := GinReqParseJson(func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "ok")
	})

	if handler == nil {
		t.Fatal("GinReqParseJson should return non-nil handler for valid function")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	handler(c)

	if !called {
		t.Error("handler function was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestGinReqParseJson_StructBinding(t *testing.T) {
	var received *testReqBody

	handler := GinReqParseJson(func(c *gin.Context, body *testReqBody) {
		received = body
		c.String(http.StatusOK, "ok")
	})

	if handler == nil {
		t.Fatal("GinReqParseJson should return non-nil handler")
	}

	reqBody := &testReqBody{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler(c)

	if received == nil {
		t.Fatal("handler should have received the body parameter")
	}
	if received.Name != "Alice" {
		t.Errorf("expected name 'Alice', got %q", received.Name)
	}
	if received.Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %q", received.Email)
	}
	if received.Age != 30 {
		t.Errorf("expected age 30, got %d", received.Age)
	}
}

func TestGinReqParseJson_StructValueBinding(t *testing.T) {
	// Test with value type (not pointer)
	var received testReqBody

	handler := GinReqParseJson(func(c *gin.Context, body testReqBody) {
		received = body
		c.String(http.StatusOK, "ok")
	})

	if handler == nil {
		t.Fatal("GinReqParseJson should return non-nil handler")
	}

	reqBody := testReqBody{
		Name:  "Bob",
		Email: "bob@example.com",
		Age:   25,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler(c)

	if received.Name != "Bob" {
		t.Errorf("expected name 'Bob', got %q", received.Name)
	}
	if received.Email != "bob@example.com" {
		t.Errorf("expected email 'bob@example.com', got %q", received.Email)
	}
}

func TestGinReqParseJson_InvalidJSON(t *testing.T) {
	called := false

	handler := GinReqParseJson(func(c *gin.Context, body *testReqBody) {
		called = true
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler(c)

	// Handler should NOT be called when JSON binding fails
	if called {
		t.Error("handler should not be called when JSON binding fails")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGinReqParseJson_NonJsonContentType(t *testing.T) {
	// When content-type is not JSON, pointer args remain nil (zero value for pointers).
	// The handler is still called.
	handlerCalled := false
	var receivedBody *testReqBody

	handler := GinReqParseJson(func(c *gin.Context, body *testReqBody) {
		handlerCalled = true
		receivedBody = body
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("name=Charlie"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Request = req

	handler(c)

	// Should still call handler
	if !handlerCalled {
		t.Error("handler should be called even for non-JSON content")
	}
	// Pointer arg should be nil (zero value) since JSON binding was skipped
	if receivedBody != nil {
		t.Error("pointer arg should be nil for non-JSON content type")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestGinReqParseJson_MapBinding(t *testing.T) {
	var received map[string]string

	handler := GinReqParseJson(func(c *gin.Context, body map[string]string) {
		received = body
		c.String(http.StatusOK, "ok")
	})

	if handler == nil {
		t.Fatal("GinReqParseJson should return non-nil handler")
	}

	bodyBytes, _ := json.Marshal(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler(c)

	if received == nil {
		t.Fatal("handler should receive map body")
	}
	if received["key1"] != "value1" {
		t.Errorf("expected key1='value1', got %q", received["key1"])
	}
	if received["key2"] != "value2" {
		t.Errorf("expected key2='value2', got %q", received["key2"])
	}
}

func TestGinReqParseJson_PanicRecovery(t *testing.T) {
	handler := GinReqParseJson(func(c *gin.Context) {
		panic("test panic in handler")
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// Should not panic
	handler(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d after panic, got %d", http.StatusInternalServerError, w.Code)
	}
	if w.Body.String() != "internal server error" {
		t.Errorf("expected 'internal server error', got %q", w.Body.String())
	}
}

func TestGinReqParseJson_MultipleArgs(t *testing.T) {
	var receivedCtx bool
	var receivedBody *testReqBody

	handler := GinReqParseJson(func(c *gin.Context, body *testReqBody) {
		receivedCtx = c != nil
		receivedBody = body
		c.String(http.StatusOK, "ok")
	})

	reqBody := &testReqBody{Name: "Dave", Email: "dave@example.com", Age: 40}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler(c)

	if !receivedCtx {
		t.Error("handler should receive non-nil context")
	}
	if receivedBody == nil {
		t.Fatal("handler should receive body")
	}
	if receivedBody.Name != "Dave" {
		t.Errorf("expected name 'Dave', got %q", receivedBody.Name)
	}
}

func TestGinReqParseJson_IntArgInHandler(t *testing.T) {
	// A handler with a non-struct, non-map arg (int) should get zero value
	var received int

	handler := GinReqParseJson(func(c *gin.Context, num int) {
		received = num
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	handler(c)

	if received != 0 {
		t.Errorf("expected zero int value, got %d", received)
	}
}

func TestGinReqParseJson_EmptyBody(t *testing.T) {
	var received *testReqBody

	handler := GinReqParseJson(func(c *gin.Context, body *testReqBody) {
		received = body
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler(c)

	// Empty JSON object should still bind
	if received == nil {
		t.Fatal("handler should receive body even for empty JSON object")
	}
	if received.Name != "" {
		t.Errorf("expected empty name, got %q", received.Name)
	}
}

func TestGinReqParseJson_StructPointerBinding(t *testing.T) {
	// Test with pointer to struct (as second arg is pointer)
	handler := GinReqParseJson(func(c *gin.Context, body *testReqBody) {
		if body == nil {
			t.Error("body should not be nil for valid JSON")
		}
		c.String(http.StatusOK, "ok")
	})

	reqBody := &testReqBody{Name: "Eve", Email: "eve@example.com", Age: 28}
	bodyBytes, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
