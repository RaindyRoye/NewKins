package route

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
)

func setupApiTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ApiController{}
	ac.Routes(r.Group("/api"))
	return r
}

func TestHello_Returns200(t *testing.T) {
	r := setupApiTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "hello world" {
		t.Errorf("body = %q, want %q", w.Body.String(), "hello world")
	}
}

func TestHello_POST_Returns200(t *testing.T) {
	r := setupApiTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestVersion_Returns200(t *testing.T) {
	r := setupApiTestRouter(t)

	// Set known values for the test
	oldVersion := comm.Version
	oldBuildTime := comm.BuildTime
	oldGitCommit := comm.GitCommit
	comm.Version = "1.2.3-test"
	comm.BuildTime = "2026-01-01T00:00:00Z"
	comm.GitCommit = "abc123"
	defer func() {
		comm.Version = oldVersion
		comm.BuildTime = oldBuildTime
		comm.GitCommit = oldGitCommit
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if resp["version"] != "1.2.3-test" {
		t.Errorf("version = %q, want %q", resp["version"], "1.2.3-test")
	}
	if resp["build_time"] != "2026-01-01T00:00:00Z" {
		t.Errorf("build_time = %q, want %q", resp["build_time"], "2026-01-01T00:00:00Z")
	}
	if resp["git_commit"] != "abc123" {
		t.Errorf("git_commit = %q, want %q", resp["git_commit"], "abc123")
	}
}

func TestVersion_ContentTypeIsJSON(t *testing.T) {
	r := setupApiTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json; charset=utf-8")
	}
}

func TestVersion_WithDefaultValues(t *testing.T) {
	r := setupApiTestRouter(t)

	oldVersion := comm.Version
	oldBuildTime := comm.BuildTime
	oldGitCommit := comm.GitCommit
	comm.Version = "1.3.7"
	comm.BuildTime = "unknown"
	comm.GitCommit = "unknown"
	defer func() {
		comm.Version = oldVersion
		comm.BuildTime = oldBuildTime
		comm.GitCommit = oldGitCommit
	}()

	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if resp["version"] != "1.3.7" {
		t.Errorf("version = %q, want %q", resp["version"], "1.3.7")
	}
	if resp["build_time"] != "unknown" {
		t.Errorf("build_time = %q, want %q", resp["build_time"], "unknown")
	}
	if resp["git_commit"] != "unknown" {
		t.Errorf("git_commit = %q, want %q", resp["git_commit"], "unknown")
	}
}
