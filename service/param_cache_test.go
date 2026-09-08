package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	bolt "go.etcd.io/bbolt"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupParamCacheTestDB creates an isolated in-memory SQLite DB and bbolt cache for param cache tests.
func setupParamCacheTestDB(t *testing.T) (*xorm.Engine, *bolt.DB) {
	t.Helper()
	
	// Setup SQLite
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	oldDb := comm.Db
	comm.Db = eng
	
	// Setup bbolt cache in temp directory
	tmpDir := t.TempDir()
	boltPath := filepath.Join(tmpDir, "test-cache.db")
	cache, err := bolt.Open(boltPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		eng.Close()
		t.Fatalf("failed to create test cache: %v", err)
	}
	oldCache := comm.BCache
	comm.BCache = cache
	
	t.Cleanup(func() {
		comm.Db = oldDb
		comm.BCache = oldCache
		_ = eng.Close()
		_ = cache.Close()
		_ = os.Remove(boltPath)
	})
	
	if err := eng.Sync2(&model.TParam{}); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	
	return eng, cache
}

// --- GetsParamCacheCtx tests ---

func TestGetsParamCacheCtx_CacheMiss(t *testing.T) {
	eng, _ := setupParamCacheTestDB(t)
	ctx := context.Background()
	
	// Insert param into DB (cache is empty)
	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	original := testData{Name: "cache-test", Value: 42}
	
	if err := SetsParamCtx(ctx, "cache-key", original, "Cache Test"); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}
	
	// First call should miss cache and load from DB
	var loaded testData
	err := GetsParamCacheCtx(ctx, "cache-key", &loaded)
	if err != nil {
		t.Fatalf("GetsParamCacheCtx: %v", err)
	}
	if loaded.Name != "cache-test" || loaded.Value != 42 {
		t.Errorf("loaded = %+v, want %+v", loaded, original)
	}
	
	// Verify param exists in DB
	p := &model.TParam{}
	ok, err := eng.Where("name=?", "cache-key").Get(p)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("param should exist in DB")
	}
}

func TestGetsParamCacheCtx_CacheHit(t *testing.T) {
	_, _ = setupParamCacheTestDB(t)
	ctx := context.Background()
	
	type config struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	original := config{Host: "localhost", Port: 8080}
	
	// Store in DB
	if err := SetsParamCtx(ctx, "config-key", original); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}
	
	// First call populates cache
	var loaded1 config
	if err := GetsParamCacheCtx(ctx, "config-key", &loaded1); err != nil {
		t.Fatalf("first GetsParamCacheCtx: %v", err)
	}
	
	// Second call should hit cache
	var loaded2 config
	if err := GetsParamCacheCtx(ctx, "config-key", &loaded2); err != nil {
		t.Fatalf("second GetsParamCacheCtx: %v", err)
	}
	if loaded2.Host != "localhost" || loaded2.Port != 8080 {
		t.Errorf("loaded2 = %+v, want %+v", loaded2, original)
	}
}

func TestGetsParamCacheCtx_CustomTimeout(t *testing.T) {
	_, _ = setupParamCacheTestDB(t)
	ctx := context.Background()
	
	data := map[string]string{"key": "value"}
	
	// Store with custom timeout
	if err := SetsParamCtx(ctx, "timeout-key", data); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}
	
	// Load with custom cache timeout
	var loaded map[string]string
	err := GetsParamCacheCtx(ctx, "timeout-key", &loaded, 30*time.Minute)
	if err != nil {
		t.Fatalf("GetsParamCacheCtx: %v", err)
	}
	if loaded["key"] != "value" {
		t.Errorf("loaded[key] = %q, want %q", loaded["key"], "value")
	}
}

func TestGetsParamCacheCtx_NotFound(t *testing.T) {
	_, _ = setupParamCacheTestDB(t)
	ctx := context.Background()
	
	var data map[string]string
	err := GetsParamCacheCtx(ctx, "nonexistent", &data)
	if err == nil {
		t.Fatal("GetsParamCacheCtx should return error for nonexistent key")
	}
}

func TestGetsParamCacheCtx_NilDataCache(t *testing.T) {
	_, _ = setupParamCacheTestDB(t)
	ctx := context.Background()

	err := GetsParamCacheCtx(ctx, "any-key", nil)
	if err == nil {
		t.Fatal("GetsParamCacheCtx should return error for nil data")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("error = %v, want wrapping ErrParamDataNil", err)
	}
}

func TestGetsParamCacheCtx_CacheNotInit(t *testing.T) {
	eng, cache := setupParamCacheTestDB(t)
	ctx := context.Background()
	
	// Temporarily disable cache
	oldCache := comm.BCache
	comm.BCache = nil
	defer func() { comm.BCache = oldCache }()
	
	// Insert param into DB
	if err := SetsParamCtx(ctx, "no-cache", []byte("data")); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}
	
	var data map[string]string
	err := GetsParamCacheCtx(ctx, "no-cache", &data)
	// Should fail because cache is not initialized
	if err == nil {
		t.Fatal("GetsParamCacheCtx should return error when cache is not initialized")
	}
	
	// Restore cache and verify DB still has the data
	comm.BCache = cache
	p := &model.TParam{}
	ok, err := eng.Where("name=?", "no-cache").Get(p)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !ok {
		t.Fatal("param should exist in DB")
	}
}

// --- GetsParamCache (global context wrapper) ---

func TestGetsParamCache_GlobalContext(t *testing.T) {
	_, _ = setupParamCacheTestDB(t)
	
	type testData struct {
		Count int `json:"count"`
	}
	original := testData{Count: 100}
	
	// Store using global context
	if err := SetsParam("global-cache", original); err != nil {
		t.Fatalf("SetsParam: %v", err)
	}
	
	// Load using global context
	var loaded testData
	err := GetsParamCache("global-cache", &loaded)
	if err != nil {
		t.Fatalf("GetsParamCache: %v", err)
	}
	if loaded.Count != 100 {
		t.Errorf("loaded.Count = %d, want 100", loaded.Count)
	}
}

func TestGetsParamCache_WithTimeout(t *testing.T) {
	_, _ = setupParamCacheTestDB(t)
	
	data := map[string]int{"value": 42}
	
	if err := SetsParam("timeout-global", data); err != nil {
		t.Fatalf("SetsParam: %v", err)
	}
	
	var loaded map[string]int
	err := GetsParamCache("timeout-global", &loaded, 1*time.Hour)
	if err != nil {
		t.Fatalf("GetsParamCache: %v", err)
	}
	if loaded["value"] != 42 {
		t.Errorf("loaded[value] = %d, want 42", loaded["value"])
	}
}
