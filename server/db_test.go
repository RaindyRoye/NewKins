package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
)

func TestInitCache_Success(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if origBCache != nil {
			comm.BCache = origBCache
		}
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}

	if comm.BCache == nil {
		t.Fatal("BCache should be initialized")
	}

	// Verify cache file exists
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("cache.dat file should exist")
	}

	// Clean up
	_ = comm.BCache.Close()
}

func TestInitCache_InvalidPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	}()

	// Use a path that cannot be created (root of filesystem in read-only context)
	comm.WorkPath = "/proc/invalid/path/that/does/not/exist"

	err := initCache()
	if err == nil {
		t.Fatal("expected error when cache path is invalid")
	}
}

func TestInitCache_RemovesOldCache(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if origBCache != nil {
			comm.BCache = origBCache
		}
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Create an old cache file
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("old data"), 0600); err != nil {
		t.Fatalf("create old cache: %v", err)
	}

	// initCache should remove and recreate it
	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}

	if comm.BCache == nil {
		t.Fatal("BCache should be initialized")
	}

	// Verify new cache is different from old data
	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("stat cache: %v", err)
	}
	if info.Size() == 8 { // "old data" is 8 bytes
		t.Error("cache should be recreated, not contain old data")
	}

	_ = comm.BCache.Close()
}
