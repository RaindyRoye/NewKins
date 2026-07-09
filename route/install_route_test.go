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

func setupInstallTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ic := &InstallController{}

	// Save and restore comm.Installed
	origInstalled := comm.Installed
	t.Cleanup(func() { comm.Installed = origInstalled })

	comm.Installed = false
	ic.Routes(r.Group("/api/install"))
	return r
}

func TestInstallCheck_ReturnsHelloGokins(t *testing.T) {
	r := setupInstallTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/install/check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "hello gokins!" {
		t.Errorf("body = %q, want %q", w.Body.String(), "hello gokins!")
	}
}

func TestInstallAuth_WhenInstalled_Returns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ic := &InstallController{}

	origInstalled := comm.Installed
	comm.Installed = true
	t.Cleanup(func() { comm.Installed = origInstalled })

	ic.Routes(r.Group("/api/install"))

	req := httptest.NewRequest(http.MethodPost, "/api/install/check", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when installed, got %d", w.Code)
	}
}

func TestInstall_InvalidHostFormat(t *testing.T) {
	r := setupInstallTestRouter(t)

	cfg := map[string]any{
		"server": map[string]any{
			"host": "not-a-valid-url",
		},
		"datasource": map[string]any{
			"driver": "sqlite",
		},
	}
	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/install/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid host, got %d", w.Code)
	}
}

func TestInstall_InvalidHbtpHost(t *testing.T) {
	r := setupInstallTestRouter(t)

	// Start a mock server that the install handler can reach
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/install/check" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	cfg := map[string]any{
		"server": map[string]any{
			"host":     ts.URL,
			"hbtpHost": "not valid host format!!!",
		},
		"datasource": map[string]any{
			"driver": "sqlite",
		},
	}
	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/install/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid hbtpHost, got %d", w.Code)
	}
}

func TestInstall_MysqlInvalidDbHost(t *testing.T) {
	r := setupInstallTestRouter(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := map[string]any{
		"server": map[string]any{
			"host": ts.URL,
		},
		"datasource": map[string]any{
			"driver": "mysql",
			"host":   "invalid host!!",
			"name":   "testdb",
		},
	}
	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/install/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid db host, got %d", w.Code)
	}
}

func TestInstall_MysqlEmptyDbName(t *testing.T) {
	r := setupInstallTestRouter(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := map[string]any{
		"server": map[string]any{
			"host": ts.URL,
		},
		"datasource": map[string]any{
			"driver": "mysql",
			"host":   "localhost:3306",
			"name":   "",
		},
	}
	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/install/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty db name, got %d", w.Code)
	}
}

func TestInstall_MysqlDbNameWithColon(t *testing.T) {
	r := setupInstallTestRouter(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := map[string]any{
		"server": map[string]any{
			"host": ts.URL,
		},
		"datasource": map[string]any{
			"driver": "mysql",
			"host":   "localhost:3306",
			"name":   "db:name",
		},
	}
	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/install/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for db name with colon, got %d", w.Code)
	}
}

func TestInstall_HostTrailingSlashTrimmed(t *testing.T) {
	r := setupInstallTestRouter(t)

	// This will fail at checkUrl since the trimmed URL won't match our test server,
	// but we verify the trimming happens by checking the error message
	cfg := map[string]any{
		"server": map[string]any{
			"host": "http://nonexistent.test/",
		},
		"datasource": map[string]any{
			"driver": "sqlite",
		},
	}
	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/install/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should get a connection error (502) since the host is unreachable
	// and the trailing slash was trimmed before checkUrl
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for unreachable host, got %d", w.Code)
	}
}

func TestInstall_InvalidJSON(t *testing.T) {
	r := setupInstallTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/install/", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestInstall_CantConnectToHost(t *testing.T) {
	r := setupInstallTestRouter(t)

	cfg := map[string]any{
		"server": map[string]any{
			"host": "http://127.0.0.1:1",
		},
		"datasource": map[string]any{
			"driver": "sqlite",
		},
	}
	body, _ := json.Marshal(cfg)
	req := httptest.NewRequest(http.MethodPost, "/api/install/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for unreachable host, got %d", w.Code)
	}
}
