package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
)

// TestParseConfig tests the parseConfig function
func TestParseConfig(t *testing.T) {
	// Save original values
	origWorkPath := comm.WorkPath
	defer func() { comm.WorkPath = origWorkPath }()

	// Create temp directory
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Test case 1: No config file exists
	err := parseConfig()
	if err == nil {
		t.Error("Expected error when config file doesn't exist, got nil")
	}

	// Test case 2: Valid config file (app.yml)
	validConfig := `
server:
  host: "http://localhost:8030"
  webHost: ":8030"
  runLimit: 5
datasource:
  driver: "sqlite"
  url: "./test.db"
`
	err = os.WriteFile(filepath.Join(tmpDir, "app.yml"), []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	err = parseConfig()
	if err != nil {
		t.Errorf("Expected no error with valid config, got: %v", err)
	}

	// Test case 3: Invalid YAML
	invalidConfig := `
server:
  host: "http://localhost:8030"
  [invalid yaml
`
	err = os.WriteFile(filepath.Join(tmpDir, "app.yml"), []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	err = parseConfig()
	if err == nil {
		t.Error("Expected error with invalid YAML, got nil")
	}
}

// TestInitCache tests the initCache function
func TestInitCache(t *testing.T) {
	// Save original values
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	}()

	// Create temp directory
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Test successful cache initialization
	err := initCache()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify cache was created
	if comm.BCache == nil {
		t.Error("Expected BCache to be initialized, got nil")
	}

	// Verify cache file exists
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Expected cache.dat file to be created")
	}

	// Close the cache
	if comm.BCache != nil {
		comm.BCache.Close()
	}
}

// TestInitCacheReadOnlyDirectory tests initCache with read-only directory
func TestInitCacheReadOnlyDirectory(t *testing.T) {
	// Skip on Windows
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping read-only test on Windows")
	}

	// Save original values
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	}()

	// Create temp directory
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Make directory read-only
	err := os.Chmod(tmpDir, 0555)
	if err != nil {
		t.Fatalf("Failed to make directory read-only: %v", err)
	}
	defer os.Chmod(tmpDir, 0755) // Restore permissions for cleanup

	// Test cache initialization with read-only directory
	err = initCache()
	if err == nil {
		t.Error("Expected error with read-only directory, got nil")
		if comm.BCache != nil {
			comm.BCache.Close()
		}
	}
}

// TestRunHbtpEmptyHost tests runHbtp with empty host
func TestRunHbtpEmptyHost(t *testing.T) {
	// Save original values
	origCfg := comm.Cfg
	defer func() { comm.Cfg = origCfg }()

	// Set empty HbtpHost
	comm.Cfg.Server.HbtpHost = ""

	// Should return immediately without error
	runHbtp()
}

// TestRunHbtpInvalidHost tests runHbtp with invalid host
func TestRunHbtpInvalidHost(t *testing.T) {
	// Save original values
	origCfg := comm.Cfg
	origCtx := comm.Ctx
	defer func() {
		comm.Cfg = origCfg
		comm.Ctx = origCtx
	}()

	// Create a new context for this test
	ctx, cancel := context.WithCancel(context.Background())
	comm.Ctx = ctx
	defer cancel()

	// Set invalid HbtpHost (should fail to bind)
	comm.Cfg.Server.HbtpHost = "invalid-host:99999"

	// runHbtp should handle the error gracefully
	// Note: This test may not trigger the error path in all environments
	runHbtp()

	// Give goroutines time to execute
	time.Sleep(100 * time.Millisecond)
}

// TestParseConfigWithAppYaml tests parseConfig with app.yaml (instead of app.yml)
func TestParseConfigWithAppYaml(t *testing.T) {
	// Save original values
	origWorkPath := comm.WorkPath
	defer func() { comm.WorkPath = origWorkPath }()

	// Create temp directory
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Write valid config as app.yaml (not app.yml)
	validConfig := `
server:
  host: "http://localhost:8030"
  webHost: ":8030"
  runLimit: 5
datasource:
  driver: "sqlite"
  url: "./test.db"
`
	err := os.WriteFile(filepath.Join(tmpDir, "app.yaml"), []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	err = parseConfig()
	if err != nil {
		t.Errorf("Expected no error with valid app.yaml, got: %v", err)
	}
}

// TestParseConfigMissingRequiredFields tests parseConfig with missing required fields
func TestParseConfigMissingRequiredFields(t *testing.T) {
	// Save original values
	origWorkPath := comm.WorkPath
	defer func() { comm.WorkPath = origWorkPath }()

	// Create temp directory
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Write config with missing datasource URL
	incompleteConfig := `
server:
  host: "http://localhost:8030"
  webHost: ":8030"
datasource:
  driver: "sqlite"
`
	err := os.WriteFile(filepath.Join(tmpDir, "app.yml"), []byte(incompleteConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	err = parseConfig()
	// Note: This may or may not error depending on validation rules
	// The test verifies the function handles incomplete config gracefully
	t.Logf("parseConfig result with incomplete config: %v", err)
}
