package route

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/common"
	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

// TestRuntimeController_logs_Success tests the logs handler with a real log file
func TestRuntimeController_logs_Success(t *testing.T) {
	// Setup test database
	db := setupRuntimeTestDB(t)
	defer db.Close()

	// Create a temporary log file
	buildId := "test-build-123"
	stepId := "test-step-456"
	logDir := filepath.Join(comm.WorkPath, common.PathBuild, buildId, common.PathJobs, stepId)
	err := os.MkdirAll(logDir, 0750)
	if err != nil {
		t.Fatalf("failed to create log directory: %v", err)
	}
	defer os.RemoveAll(filepath.Join(comm.WorkPath, common.PathBuild, buildId))

	logPath := filepath.Join(logDir, "build.log")
	logContent := `{"type":"log","content":"Building project..."}
{"type":"log","content":"Running tests..."}
{"type":"log","content":"Build completed successfully"}
`
	err = os.WriteFile(logPath, []byte(logContent), 0600)
	if err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("POST", "/runtime/logs", nil)
	req.Body = nil
	c.Request = req

	ctrl := RuntimeController{}
	ctrl.logs(c, &hbtp.Map{
		"stepId":  stepId,
		"buildId": buildId,
		"offset":  int64(0),
		"limit":   int64(100),
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify response structure
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["stepId"] != stepId {
		t.Errorf("expected stepId=%s, got %v", stepId, resp["stepId"])
	}

	if _, ok := resp["logs"]; !ok {
		t.Error("response should contain logs field")
	}

	if _, ok := resp["lastoff"]; !ok {
		t.Error("response should contain lastoff field")
	}
}

// TestRuntimeController_logs_WithOffset tests the logs handler with offset parameter
func TestRuntimeController_logs_WithOffset(t *testing.T) {
	// Setup test database
	db := setupRuntimeTestDB(t)
	defer db.Close()

	// Create a temporary log file
	buildId := "test-build-offset"
	stepId := "test-step-offset"
	logDir := filepath.Join(comm.WorkPath, common.PathBuild, buildId, common.PathJobs, stepId)
	err := os.MkdirAll(logDir, 0750)
	if err != nil {
		t.Fatalf("failed to create log directory: %v", err)
	}
	defer os.RemoveAll(filepath.Join(comm.WorkPath, common.PathBuild, buildId))

	logPath := filepath.Join(logDir, "build.log")
	logContent := `{"type":"log","content":"Line 1"}
{"type":"log","content":"Line 2"}
{"type":"log","content":"Line 3"}
`
	err = os.WriteFile(logPath, []byte(logContent), 0600)
	if err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// Create test request with offset
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("POST", "/runtime/logs", nil)
	c.Request = req

	ctrl := RuntimeController{}
	ctrl.logs(c, &hbtp.Map{
		"stepId":  stepId,
		"buildId": buildId,
		"offset":  int64(30), // Skip first ~30 bytes
		"limit":   int64(100),
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestRuntimeController_logs_WithLimit tests the logs handler with limit parameter
func TestRuntimeController_logs_WithLimit(t *testing.T) {
	// Setup test database
	db := setupRuntimeTestDB(t)
	defer db.Close()

	// Create a temporary log file
	buildId := "test-build-limit"
	stepId := "test-step-limit"
	logDir := filepath.Join(comm.WorkPath, common.PathBuild, buildId, common.PathJobs, stepId)
	err := os.MkdirAll(logDir, 0750)
	if err != nil {
		t.Fatalf("failed to create log directory: %v", err)
	}
	defer os.RemoveAll(filepath.Join(comm.WorkPath, common.PathBuild, buildId))

	logPath := filepath.Join(logDir, "build.log")
	logContent := `{"type":"log","content":"Line 1"}
{"type":"log","content":"Line 2"}
{"type":"log","content":"Line 3"}
{"type":"log","content":"Line 4"}
{"type":"log","content":"Line 5"}
`
	err = os.WriteFile(logPath, []byte(logContent), 0600)
	if err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// Create test request with limit
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("POST", "/runtime/logs", nil)
	c.Request = req

	ctrl := RuntimeController{}
	ctrl.logs(c, &hbtp.Map{
		"stepId":  stepId,
		"buildId": buildId,
		"offset":  int64(0),
		"limit":   int64(2), // Only get 2 lines
	})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify response contains limited logs
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	logs, ok := resp["logs"].([]interface{})
	if !ok {
		t.Fatal("logs field should be an array")
	}

	if len(logs) > 2 {
		t.Errorf("expected at most 2 log lines, got %d", len(logs))
	}
}
