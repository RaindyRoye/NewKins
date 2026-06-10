package route

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gokins/gokins/comm"
)

func TestCheckUrl_Success(t *testing.T) {
	// Start a mock server that responds to the install check endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/install/check" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello gokins!"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	if !checkUrl(ts.URL) {
		t.Error("checkUrl should return true for a reachable server with correct endpoint")
	}
}

func TestCheckUrl_ServerReturns500(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	if checkUrl(ts.URL) {
		t.Error("checkUrl should return false when server returns 500")
	}
}

func TestCheckUrl_ServerReturns404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	if checkUrl(ts.URL) {
		t.Error("checkUrl should return false when server returns 404")
	}
}

func TestCheckUrl_UnreachableServer(t *testing.T) {
	// Use an address that will never respond
	if checkUrl("http://127.0.0.1:1") {
		t.Error("checkUrl should return false for unreachable server")
	}
}

func TestCheckUrl_InvalidURL(t *testing.T) {
	if checkUrl("://invalid") {
		t.Error("checkUrl should return false for invalid URL")
	}
}

func TestInitConfig_WritesFile(t *testing.T) {
	// Create a temp directory for the test
	tmpDir, err := os.MkdirTemp("", "gokins_config_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Save and restore comm.WorkPath and comm.Cfg
	oldWorkPath := comm.WorkPath
	oldCfg := comm.Cfg
	defer func() {
		comm.WorkPath = oldWorkPath
		comm.Cfg = oldCfg
	}()

	comm.WorkPath = tmpDir
	comm.Cfg.Server.Host = "http://localhost:8080"
	comm.Cfg.Server.LoginKey = "test-key-123"
	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = "/tmp/test.db"

	if err := initConfig(); err != nil {
		t.Fatalf("initConfig failed: %v", err)
	}

	// Verify the file was created
	configPath := filepath.Join(tmpDir, "app.yml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("config file was not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("config file is empty")
	}

	// Verify file permissions (should be 0600)
	if info.Mode().Perm() != 0600 {
		t.Errorf("config file permissions = %o, want 0600", info.Mode().Perm())
	}

	// Read and verify content contains expected values
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "localhost:8080") {
		t.Error("config file should contain the host value")
	}
	if !strings.Contains(content, "test-key-123") {
		t.Error("config file should contain the login key")
	}
	if !strings.Contains(content, "sqlite3") {
		t.Error("config file should contain the driver value")
	}
}

func TestInitConfig_InvalidWorkPath(t *testing.T) {
	oldWorkPath := comm.WorkPath
	defer func() { comm.WorkPath = oldWorkPath }()

	// Use a path that cannot be written to (a file, not a directory)
	tmpFile, err := os.CreateTemp("", "gokins_badpath_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Set WorkPath to the file path (not a directory) so mkdir/write fails
	comm.WorkPath = filepath.Join(tmpFile.Name(), "nonexistent")

	err = initConfig()
	if err == nil {
		t.Error("initConfig should fail with invalid WorkPath")
	}
}


