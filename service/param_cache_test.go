package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	bolt "go.etcd.io/bbolt"
	"xorm.io/xorm"
)

// setupParamCacheTest creates an isolated in-memory SQLite DB and a temporary
// bbolt cache for testing GetsParamCacheCtx. Returns nothing; cleanup is
// registered via t.Cleanup.
func setupParamCacheTest(t *testing.T) {
	t.Helper()

	// Setup SQLite database
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	oldDb := comm.Db
	comm.Db = eng
	t.Cleanup(func() {
		comm.Db = oldDb
		_ = eng.Close()
	})
	if err := eng.Sync2(&model.TParam{}); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}

	// Setup bbolt cache
	tmpFile, err := os.CreateTemp("", "param_cache_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp cache file: %v", err)
	}
	_ = tmpFile.Close()

	cache, err := bolt.Open(tmpFile.Name(), 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		t.Fatalf("failed to open bbolt: %v", err)
	}

	oldCache := comm.BCache
	comm.BCache = cache
	t.Cleanup(func() {
		_ = cache.Close()
		comm.BCache = oldCache
		_ = os.Remove(tmpFile.Name())
	})
}

// --- GetsParamCacheCtx Tests ---

func TestGetsParamCacheCtx_CacheMiss_ThenHit(t *testing.T) {
	setupParamCacheTest(t)
	ctx := context.Background()

	// Create a param in the database
	type config struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	original := config{Host: "localhost", Port: 8080}
	if err := SetsParamCtx(ctx, "test-config", original); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	// First call: cache miss -> should fetch from DB and populate cache
	var result config
	if err := GetsParamCacheCtx(ctx, "test-config", &result); err != nil {
		t.Fatalf("GetsParamCacheCtx (cache miss): %v", err)
	}
	if result.Host != "localhost" || result.Port != 8080 {
		t.Errorf("result = %+v, want %+v", result, original)
	}

	// Second call: cache hit -> should return from cache
	var result2 config
	if err := GetsParamCacheCtx(ctx, "test-config", &result2); err != nil {
		t.Fatalf("GetsParamCacheCtx (cache hit): %v", err)
	}
	if result2.Host != "localhost" || result2.Port != 8080 {
		t.Errorf("result2 = %+v, want %+v", result2, original)
	}
}

func TestGetsParamCacheCtx_NotFound(t *testing.T) {
	setupParamCacheTest(t)
	ctx := context.Background()

	var result map[string]string
	err := GetsParamCacheCtx(ctx, "nonexistent", &result)
	if err == nil {
		t.Fatal("GetsParamCacheCtx should return error for nonexistent key")
	}
}

func TestGetsParamCacheCtx_NilData_Integ(t *testing.T) {
	setupParamCacheTest(t)
	ctx := context.Background()

	err := GetsParamCacheCtx(ctx, "any-key", nil)
	if err == nil {
		t.Fatal("GetsParamCacheCtx(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("GetsParamCacheCtx(nil data) = %v, want wrapped ErrParamDataNil", err)
	}
}

func TestGetsParamCacheCtx_CacheBypassOnNilBCache(t *testing.T) {
	// Setup only DB, no cache (BCache stays nil)
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	oldDb := comm.Db
	comm.Db = eng
	t.Cleanup(func() {
		comm.Db = oldDb
		_ = eng.Close()
	})
	if err := eng.Sync2(&model.TParam{}); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}

	// Ensure BCache is nil for this test
	oldCache := comm.BCache
	comm.BCache = nil
	t.Cleanup(func() {
		comm.BCache = oldCache
	})

	ctx := context.Background()

	type item struct {
		Value string `json:"value"`
	}
	if err := SetsParamCtx(ctx, "no-cache-key", item{Value: "hello"}); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	// CacheGets will return ErrCacheNotInit, so it falls through to DB
	var result item
	if err := GetsParamCacheCtx(ctx, "no-cache-key", &result); err != nil {
		t.Fatalf("GetsParamCacheCtx with nil cache: %v", err)
	}
	if result.Value != "hello" {
		t.Errorf("result.Value = %q, want %q", result.Value, "hello")
	}
}

func TestGetsParamCacheCtx_WithCustomTTL(t *testing.T) {
	setupParamCacheTest(t)
	ctx := context.Background()

	type payload struct {
		Count int `json:"count"`
	}
	if err := SetsParamCtx(ctx, "ttl-key", payload{Count: 42}); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	// Call with a custom TTL duration
	var result payload
	if err := GetsParamCacheCtx(ctx, "ttl-key", &result, 5*time.Minute); err != nil {
		t.Fatalf("GetsParamCacheCtx with TTL: %v", err)
	}
	if result.Count != 42 {
		t.Errorf("result.Count = %d, want 42", result.Count)
	}

	// Verify the value is still cached
	var result2 payload
	if err := GetsParamCacheCtx(ctx, "ttl-key", &result2); err != nil {
		t.Fatalf("GetsParamCacheCtx second call: %v", err)
	}
	if result2.Count != 42 {
		t.Errorf("result2.Count = %d, want 42", result2.Count)
	}
}

// --- GetsParamCache (global context wrapper) ---

func TestGetsParamCache_GlobalContext(t *testing.T) {
	setupParamCacheTest(t)

	type setting struct {
		Debug bool `json:"debug"`
	}
	if err := SetsParamCtx(comm.Ctx, "global-setting", setting{Debug: true}); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	var result setting
	if err := GetsParamCache("global-setting", &result); err != nil {
		t.Fatalf("GetsParamCache: %v", err)
	}
	if !result.Debug {
		t.Error("result.Debug = false, want true")
	}
}

func TestGetsParamCache_NilData(t *testing.T) {
	err := GetsParamCache("any-key", nil)
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("GetsParamCache(nil) = %v, want wrapped ErrParamDataNil", err)
	}
}
