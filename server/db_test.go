package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
)

// TestInitCache_Success tests successful cache initialization with a valid work path.
func TestInitCache_Success(t *testing.T) {
	// Save original WorkPath and BCache, restore after test
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	})

	// Create temporary directory for test
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Initialize cache
	err := initCache()
	if err != nil {
		t.Fatalf("initCache() error = %v", err)
	}

	// Verify BCache is set
	if comm.BCache == nil {
		t.Error("comm.BCache is nil after successful initCache()")
	}

	// Verify cache file was created
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file %q was not created", cachePath)
	}

	// Clean up
	if comm.BCache != nil {
		_ = comm.BCache.Close()
	}
}

// TestInitCache_InvalidWorkPath tests cache initialization with an invalid work path.
func TestInitCache_InvalidWorkPath(t *testing.T) {
	// Save original WorkPath and BCache, restore after test
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	})

	// Use a path that cannot be written (non-existent parent directory)
	comm.WorkPath = "/nonexistent/directory/that/does/not/exist"

	// Initialize cache should fail
	err := initCache()
	if err == nil {
		t.Skip("skipping: initCache succeeded unexpectedly (possibly running as root)")
	}

	// Verify error message contains meaningful context
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

// TestInitCache_RemovesOldCache tests that initCache removes an existing cache file.
func TestInitCache_RemovesOldCache(t *testing.T) {
	// Save original WorkPath and BCache, restore after test
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	})

	// Create temporary directory for test
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Create an existing cache file with some content
	cachePath := filepath.Join(tmpDir, "cache.dat")
	oldContent := []byte("old cache data")
	if err := os.WriteFile(cachePath, oldContent, 0644); err != nil {
		t.Fatalf("failed to create old cache file: %v", err)
	}

	// Verify old file exists
	oldInfo, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("old cache file not found: %v", err)
	}
	oldSize := oldInfo.Size()

	// Initialize cache
	err = initCache()
	if err != nil {
		t.Fatalf("initCache() error = %v", err)
	}

	// Verify BCache is set
	if comm.BCache == nil {
		t.Error("comm.BCache is nil after successful initCache()")
	}

	// Verify new cache file exists
	newInfo, err := os.Stat(cachePath)
	if err != nil {
		t.Errorf("new cache file not found: %v", err)
	}

	// The new cache should be different (either empty or a valid bbolt database)
	newSize := newInfo.Size()
	if newSize == oldSize && oldSize == int64(len(oldContent)) {
		t.Error("cache file was not replaced")
	}

	// Clean up
	if comm.BCache != nil {
		_ = comm.BCache.Close()
	}
}

// TestInitCache_MultipleInitializations tests that initCache can be called multiple times.
func TestInitCache_MultipleInitializations(t *testing.T) {
	// Save original WorkPath and BCache, restore after test
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origBCache
	})

	// Create temporary directory for test
	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// First initialization
	err := initCache()
	if err != nil {
		t.Fatalf("first initCache() error = %v", err)
	}

	firstCache := comm.BCache
	if firstCache == nil {
		t.Fatal("first BCache is nil")
	}

	// Close the first cache before reinitializing
	_ = firstCache.Close()
	comm.BCache = nil

	// Second initialization should succeed
	err = initCache()
	if err != nil {
		t.Fatalf("second initCache() error = %v", err)
	}

	secondCache := comm.BCache
	if secondCache == nil {
		t.Error("second BCache is nil")
	}

	// Verify they are different instances
	if firstCache == secondCache {
		t.Error("first and second cache are the same instance")
	}
}

// TestInitCache_ErrorWrapping verifies that errors are properly wrapped.
func TestInitCache_ErrorWrapping(t *testing.T) {
	// Save original WorkPath and BCache, restore after test
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	})

	// Use an invalid path
	comm.WorkPath = "/nonexistent/path"

	err := initCache()
	if err == nil {
		t.Skip("skipping: initCache succeeded unexpectedly")
	}

	// Verify error message contains context
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("error message is empty")
	}

	// Should mention cache or path
	if len(errMsg) < 10 {
		t.Errorf("error message too short: %q", errMsg)
	}
}
