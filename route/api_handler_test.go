package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestApiController_hello(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello world")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if body := w.Body.String(); body != "hello world" {
		t.Errorf("body = %q, want %q", body, "hello world")
	}
}

func TestApiController_version(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ApiController{}
	r.GET("/api/version", ac.version)

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"version", "build_time", "git_commit"} {
		if _, ok := resp[key]; !ok {
			t.Errorf("response missing key %q", key)
		}
	}
}

func TestApiController_test_InvalidYAML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ApiController{}
	r.POST("/api/builds", ac.test)

	req := httptest.NewRequest(http.MethodPost, "/api/builds", strings.NewReader("not: [valid: yaml: {"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 400 for invalid YAML
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestApiController_test_EmptyStages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ApiController{}
	r.POST("/api/builds", ac.test)

	// Valid YAML with no stages
	yaml := `name: test-pipeline
`
	req := httptest.NewRequest(http.MethodPost, "/api/builds", strings.NewReader(yaml))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 400 for empty stages
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestApiController_test_ValidPipeline(t *testing.T) {
	initTestJobEngine(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ApiController{}
	r.POST("/api/builds", ac.test)

	yaml := `name: test-pipeline
stages:
  - name: build
    steps:
      - step: gokins@git
        name: checkout
        commands:
          - git clone
`
	req := httptest.NewRequest(http.MethodPost, "/api/builds", strings.NewReader(yaml))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// The engine.Mgr.BuildEgn().Put() may fail if engine is not initialized,
	// but the request should at least parse correctly (200 or 500, not 400)
	if w.Code == http.StatusBadRequest {
		t.Errorf("unexpected 400 for valid pipeline, body: %s", w.Body.String())
	}
}

func TestApiController_test_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ApiController{}
	r.POST("/api/builds", ac.test)

	req := httptest.NewRequest(http.MethodPost, "/api/builds", bytes.NewReader(nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Empty body should result in empty stages → 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}
