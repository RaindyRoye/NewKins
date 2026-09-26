package service

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	bolt "go.etcd.io/bbolt"
)

func TestGetsParamCache_CacheHit(t *testing.T) {
	// Initialize cache
	tmpFile := t.TempDir() + "/cache.db"
	bcache, err := bolt.Open(tmpFile, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer bcache.Close()

	origBCache := comm.BCache
	comm.BCache = bcache
	defer func() { comm.BCache = origBCache }()

	// Set cache value
	testData := map[string]string{"key": "value"}
	err = comm.CacheSets("test-key", testData, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Test cache hit
	var result map[string]string
	err = GetsParamCache("test-key", &result)
	if err != nil {
		t.Errorf("GetsParamCache failed: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("Expected key=value, got %v", result)
	}
}

func TestGetsParamCache_CacheMiss_NoDB(t *testing.T) {
	// Initialize cache
	tmpFile := t.TempDir() + "/cache.db"
	bcache, err := bolt.Open(tmpFile, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to open cache: %v", err)
	}
	defer bcache.Close()

	origBCache := comm.BCache
	comm.BCache = bcache
	defer func() { comm.BCache = origBCache }()

	// Ensure Db is nil to trigger the panic scenario
	origDb := comm.Db
	comm.Db = nil
	defer func() { comm.Db = origDb }()

	// Test should panic when Db is nil - we recover and verify it panicked
	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil Db
			t.Logf("Expected panic occurred: %v", r)
		} else {
			t.Error("Expected panic when Db is nil, but did not panic")
		}
	}()

	// This should panic because comm.Db is nil
	var result map[string]string
	_ = GetsParamCacheCtx(context.Background(), "nonexistent-key", &result)
}

func TestGetsParamCache_CacheNotInit(t *testing.T) {
	origBCache := comm.BCache
	comm.BCache = nil
	defer func() { comm.BCache = origBCache }()

	// Also save and restore Db since it might be nil from previous test
	origDb := comm.Db
	defer func() { comm.Db = origDb }()

	// Test should panic when both cache and Db are nil
	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil Db
			t.Logf("Expected panic occurred: %v", r)
		} else {
			t.Error("Expected panic when cache not initialized and Db is nil, but did not panic")
		}
	}()

	var result map[string]string
	_ = GetsParamCache("test-key", &result)
}
