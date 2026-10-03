package util

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestGinReqParseJson_NonFunctionInput tests that non-function input returns nil
func TestGinReqParseJson_NonFunctionInput(t *testing.T) {
	// Test with string
	handler := GinReqParseJson("not a function")
	assert.Nil(t, handler, "GinReqParseJson should return nil for non-function input")

	// Test with int
	handler = GinReqParseJson(42)
	assert.Nil(t, handler, "GinReqParseJson should return nil for int input")

	// Test with nil
	handler = GinReqParseJson(nil)
	assert.Nil(t, handler, "GinReqParseJson should return nil for nil input")
}

// TestGinReqParseJson_ContextOnly tests a function with only gin.Context parameter
func TestGinReqParseJson_ContextOnly(t *testing.T) {
	called := false
	fn := func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "success")
	}

	handler := GinReqParseJson(fn)
	assert.NotNil(t, handler, "handler should not be nil for valid function")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	handler(c)

	assert.True(t, called, "function should have been called")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())
}

// TestGinReqParseJson_StructParam tests a function with struct parameter and JSON content
func TestGinReqParseJson_StructParam(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	called := false
	var receivedData TestStruct
	fn := func(c *gin.Context, data TestStruct) {
		called = true
		receivedData = data
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	assert.NotNil(t, handler)

	// Test with valid JSON
	reqData := TestStruct{Name: "Alice", Age: 30}
	jsonBytes, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	assert.True(t, called, "function should have been called")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, reqData.Name, receivedData.Name)
	assert.Equal(t, reqData.Age, receivedData.Age)
}

// TestGinReqParseJson_PointerStructParam tests a function with pointer-to-struct parameter
func TestGinReqParseJson_PointerStructParam(t *testing.T) {
	type TestStruct struct {
		ID    int    `json:"id"`
		Value string `json:"value"`
	}

	called := false
	var receivedData *TestStruct
	fn := func(c *gin.Context, data *TestStruct) {
		called = true
		receivedData = data
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	assert.NotNil(t, handler)

	reqData := TestStruct{ID: 123, Value: "test"}
	jsonBytes, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	assert.True(t, called, "function should have been called")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, receivedData)
	assert.Equal(t, reqData.ID, receivedData.ID)
	assert.Equal(t, reqData.Value, receivedData.Value)
}

// TestGinReqParseJson_MapParam tests a function with map parameter
func TestGinReqParseJson_MapParam(t *testing.T) {
	called := false
	var receivedData map[string]interface{}
	fn := func(c *gin.Context, data map[string]interface{}) {
		called = true
		receivedData = data
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	assert.NotNil(t, handler)

	reqData := map[string]interface{}{
		"key1": "value1",
		"key2": 42.0,
	}
	jsonBytes, _ := json.Marshal(reqData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	assert.True(t, called, "function should have been called")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, receivedData)
	assert.Equal(t, "value1", receivedData["key1"])
	assert.Equal(t, 42.0, receivedData["key2"])
}

// TestGinReqParseJson_InvalidJSON tests that invalid JSON returns 400
func TestGinReqParseJson_InvalidJSON(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	called := false
	fn := func(c *gin.Context, data TestStruct) {
		called = true
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	assert.NotNil(t, handler)

	// Send invalid JSON
	invalidJSON := []byte(`{"name": invalid json}`)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(invalidJSON))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	assert.False(t, called, "function should not have been called due to invalid JSON")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

// TestGinReqParseJson_NonJSONContent tests that non-JSON content doesn't bind
func TestGinReqParseJson_NonJSONContent(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	called := false
	var receivedData TestStruct
	fn := func(c *gin.Context, data TestStruct) {
		called = true
		receivedData = data
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	assert.NotNil(t, handler)

	// Send form data instead of JSON
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("name=Bob")))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	handler(c)

	assert.True(t, called, "function should have been called")
	assert.Equal(t, http.StatusOK, w.Code)
	// Data should be zero value since it wasn't bound
	assert.Equal(t, "", receivedData.Name)
}

// TestGinReqParseJson_PanicRecovery tests that panics are recovered
func TestGinReqParseJson_PanicRecovery(t *testing.T) {
	fn := func(c *gin.Context) {
		panic("intentional panic for testing")
	}

	handler := GinReqParseJson(fn)
	assert.NotNil(t, handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	// Should not panic
	handler(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "internal server error", w.Body.String())
}

// TestGinReqParseJson_MultipleParams tests that a function with multiple
// struct parameters stops at the first JSON parse error (body consumed).
func TestGinReqParseJson_MultipleParams(t *testing.T) {
	type Param1 struct {
		A int `json:"a"`
	}
	type Param2 struct {
		B string `json:"b"`
	}

	called := false
	fn := func(c *gin.Context, data1 Param1, data2 Param2) {
		called = true
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	assert.NotNil(t, handler)

	// JSON for Param1 only; Param2 will fail to bind → 400 before fn called
	json1, _ := json.Marshal(Param1{A: 10})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(json1))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	// The second struct param tries to read the (already consumed) body and gets 400
	assert.False(t, called, "function should not be called when second param bind fails")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestGinReqParseJson_NonStructParam tests that non-struct params get zero value
func TestGinReqParseJson_NonStructParam(t *testing.T) {
	called := false
	var receivedInt int
	fn := func(c *gin.Context, n int) {
		called = true
		receivedInt = n
		c.String(http.StatusOK, "ok")
	}

	handler := GinReqParseJson(fn)
	assert.NotNil(t, handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"x":1}`)))
	c.Request.Header.Set("Content-Type", "application/json")

	handler(c)

	assert.True(t, called, "function should have been called")
	assert.Equal(t, 0, receivedInt, "int param should be zero value (not a struct)")
}
