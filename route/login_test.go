package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
)

func setupLoginTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	lc := &LoginController{}
	lc.Routes(r.Group("/api/lg"))
	return r
}

func TestLogin_EmptyName(t *testing.T) {
	r := setupLoginTestRouter(t)

	body, _ := json.Marshal(map[string]any{
		"name": "",
		"pass": "somepass",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", w.Code)
	}
	if w.Body.String() != "param err" {
		t.Errorf("expected 'param err', got %q", w.Body.String())
	}
}

func TestLogin_EmptyPass(t *testing.T) {
	r := setupLoginTestRouter(t)

	body, _ := json.Marshal(map[string]any{
		"name": "admin",
		"pass": "",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pass, got %d", w.Code)
	}
}

func TestLogin_WhitespaceOnlyName(t *testing.T) {
	r := setupLoginTestRouter(t)

	body, _ := json.Marshal(map[string]any{
		"name": "   ",
		"pass": "somepass",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Name is trimmed to empty, so should get param err
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for whitespace-only name, got %d", w.Code)
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	r := setupLoginTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/lg/login", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

// Note: TestLogin_UserNotFound requires a database connection and is covered
// by integration tests in comm/db_integration_test.go

func TestLoginInfo_NoAuth(t *testing.T) {
	r := setupLoginTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/lg/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["login"] != false {
		t.Errorf("expected login=false, got %v", resp["login"])
	}
}

func TestApiController_Hello(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ApiController{}
	ac.Routes(r.Group("/api"))

	req := httptest.NewRequest(http.MethodGet, "/api/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "hello world" {
		t.Errorf("expected 'hello world', got %q", w.Body.String())
	}
}

func TestApiController_Version(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ApiController{}
	ac.Routes(r.Group("/api"))

	// Set known values for testing
	origVersion := comm.Version
	origBuildTime := comm.BuildTime
	origGitCommit := comm.GitCommit
	comm.Version = "1.0.0-test"
	comm.BuildTime = "2024-01-01"
	comm.GitCommit = "abc123"
	defer func() {
		comm.Version = origVersion
		comm.BuildTime = origBuildTime
		comm.GitCommit = origGitCommit
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["version"] != "1.0.0-test" {
		t.Errorf("expected version '1.0.0-test', got %v", resp["version"])
	}
	if resp["build_time"] != "2024-01-01" {
		t.Errorf("expected build_time '2024-01-01', got %v", resp["build_time"])
	}
	if resp["git_commit"] != "abc123" {
		t.Errorf("expected git_commit 'abc123', got %v", resp["git_commit"])
	}
}

func TestApiController_Test_InvalidYAML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ApiController{}
	ac.Routes(r.Group("/api"))

	req := httptest.NewRequest(http.MethodPost, "/api/builds", bytes.NewBufferString("{{invalid yaml"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid YAML, got %d", w.Code)
	}
}

func TestApiController_Test_EmptyStages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ApiController{}
	ac.Routes(r.Group("/api"))

	yamlContent := "version: '1'\nstages: []\n"
	req := httptest.NewRequest(http.MethodPost, "/api/builds", bytes.NewBufferString(yamlContent))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty stages, got %d", w.Code)
	}
}

func TestApiController_Test_ValidYAMLNoStages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ApiController{}
	ac.Routes(r.Group("/api"))

	// Valid YAML but no stages - should get prebuild error
	yamlContent := "version: '1'\n"
	req := httptest.NewRequest(http.MethodPost, "/api/builds", bytes.NewBufferString(yamlContent))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should fail in prebuild since no stages
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for YAML with no stages, got %d", w.Code)
	}
}
