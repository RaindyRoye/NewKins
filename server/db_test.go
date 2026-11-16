package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
)

func TestInitCache_Success(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origCache
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("comm.BCache should be initialized")
	}

	// Verify the cache file was created
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); err != nil {
		t.Errorf("cache file not found: %v", err)
	}
}

func TestInitCache_ExistingCacheRemoved(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origCache
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Create an existing cache file
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("old data"), 0600); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("comm.BCache should be initialized")
	}
}

func TestInitCache_InvalidPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origCache
	}()

	// Use a path that cannot be created (e.g., a file instead of a directory)
	tmpDir := t.TempDir()
	comm.WorkPath = filepath.Join(tmpDir, "nonexistent", "deep", "path")

	err := initCache()
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}
