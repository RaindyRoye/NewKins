package route

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func setupArtPubTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	c := &ArtPublicController{}
	c.Routes(r.Group("/api/art/pub"))
	return r
}

func TestDown_MissingTimesParam(t *testing.T) {
	r := setupArtPubTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/art/pub/down/test-id/some/path?random=abc&sign=def", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing times param, got %d", w.Code)
	}
	if w.Body.String() != "param err" {
		t.Errorf("expected body 'param err', got %q", w.Body.String())
	}
}

func TestDown_MissingRandomParam(t *testing.T) {
	r := setupArtPubTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/art/pub/down/test-id/some/path?times=2026-01-01T00:00:00Z&sign=def", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing random param, got %d", w.Code)
	}
}

func TestDown_MissingSignParam(t *testing.T) {
	r := setupArtPubTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/art/pub/down/test-id/some/path?times=2026-01-01T00:00:00Z&random=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing sign param, got %d", w.Code)
	}
}

func TestDown_MissingAllParams(t *testing.T) {
	r := setupArtPubTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/art/pub/down/test-id/some/path", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing all params, got %d", w.Code)
	}
}

func TestDown_InvalidTimesFormat(t *testing.T) {
	r := setupArtPubTestRouter(t)

	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/down/test-id/some/path?times=not-a-date&random=abc&sign=def", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid times format, got %d", w.Code)
	}
	if w.Body.String() != "param err:times" {
		t.Errorf("expected body 'param err:times', got %q", w.Body.String())
	}
}

func TestDown_ExpiredTimes(t *testing.T) {
	r := setupArtPubTestRouter(t)

	// Use a timestamp more than 20 hours ago
	expired := time.Now().Add(-25 * time.Hour).Format(time.RFC3339Nano)
	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/down/test-id/some/path?times="+url.QueryEscape(expired)+"&random=abc&sign=def", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestTimeout {
		t.Errorf("expected 408 for expired times, got %d", w.Code)
	}
	if w.Body.String() != "request timeout" {
		t.Errorf("expected body 'request timeout', got %q", w.Body.String())
	}
}

func TestDown_InvalidSignature(t *testing.T) {
	r := setupArtPubTestRouter(t)

	// Use a fresh timestamp but wrong sign
	tms := time.Now().Format(time.RFC3339Nano)
	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/down/test-id/some/path?times="+url.QueryEscape(tms)+"&random=abc&sign=wrong-sign", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for invalid signature, got %d", w.Code)
	}
	if w.Body.String() != "No Permission" {
		t.Errorf("expected body 'No Permission', got %q", w.Body.String())
	}
}

func TestDowns_MissingTimesParam(t *testing.T) {
	r := setupArtPubTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/art/pub/downs/test-id/pkg-name/some/path?random=abc&sign=def", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing times param, got %d", w.Code)
	}
}

func TestDowns_MissingRandomParam(t *testing.T) {
	r := setupArtPubTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/art/pub/downs/test-id/pkg-name/some/path?times=2026-01-01T00:00:00Z&sign=def", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing random param, got %d", w.Code)
	}
}

func TestDowns_MissingSignParam(t *testing.T) {
	r := setupArtPubTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/art/pub/downs/test-id/pkg-name/some/path?times=2026-01-01T00:00:00Z&random=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing sign param, got %d", w.Code)
	}
}

func TestDowns_InvalidTimesFormat(t *testing.T) {
	r := setupArtPubTestRouter(t)

	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/downs/test-id/pkg-name/some/path?times=invalid&random=abc&sign=def", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid times format, got %d", w.Code)
	}
}

func TestDowns_ExpiredTimes(t *testing.T) {
	r := setupArtPubTestRouter(t)

	expired := time.Now().Add(-25 * time.Hour).Format(time.RFC3339Nano)
	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/downs/test-id/pkg-name/some/path?times="+url.QueryEscape(expired)+"&random=abc&sign=def", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestTimeout {
		t.Errorf("expected 408 for expired times, got %d", w.Code)
	}
}

func TestDowns_InvalidSignature(t *testing.T) {
	r := setupArtPubTestRouter(t)

	tms := time.Now().Format(time.RFC3339Nano)
	req := httptest.NewRequest(http.MethodGet,
		"/api/art/pub/downs/test-id/pkg-name/some/path?times="+url.QueryEscape(tms)+"&random=abc&sign=wrong", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for invalid signature, got %d", w.Code)
	}
}

func TestDownFile_NotFoundFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request = req

	ctrl := ArtPublicController{}
	ctrl.downFile(c, "/nonexistent/path/to/file.txt")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent file, got %d", w.Code)
	}
	if w.Body.String() != "Not Found File" {
		t.Errorf("expected body 'Not Found File', got %q", w.Body.String())
	}
}

func TestDownFile_FoundRegularFile(t *testing.T) {
	// Create a temp file to serve
	tmpFile := t.TempDir() + "/testfile.txt"
	if err := os.WriteFile(tmpFile, []byte("hello artifact"), 0644); err != nil {
		t.Fatalf("create temp file: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request = req

	ctrl := ArtPublicController{}
	ctrl.downFile(c, tmpFile)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for found file, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/octet-stream" {
		t.Errorf("Content-Type = %q, want application/octet-stream", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Content-Disposition") == "" {
		t.Error("Content-Disposition header should be set")
	}
	if w.Body.String() != "hello artifact" {
		t.Errorf("body = %q, want 'hello artifact'", w.Body.String())
	}
}
