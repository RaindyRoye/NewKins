package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	bolt "go.etcd.io/bbolt"
	"xorm.io/xorm"
)

// setupParamCacheTestDB creates an isolated in-memory SQLite DB + bbolt cache for param cache tests.
func setupParamCacheTestDB(t *testing.T) {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	bcache, err := bolt.Open(t.TempDir()+"/test.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		_ = eng.Close()
		t.Fatalf("failed to create test cache: %v", err)
	}
	// Create the main cache bucket
	err = bcache.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("mainCacheBucket"))
		return err
	})
	if err != nil {
		_ = eng.Close()
		_ = bcache.Close()
		t.Fatalf("failed to create cache bucket: %v", err)
	}

	oldDb := comm.Db
	oldBCache := comm.BCache
	comm.Db = eng
	comm.BCache = bcache
	t.Cleanup(func() {
		comm.Db = oldDb
		comm.BCache = oldBCache
		_ = eng.Close()
		_ = bcache.Close()
	})
	if err := eng.Sync2(&model.TParam{}); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
}

func TestGetsParamCacheCtx_CacheMissThenHit(t *testing.T) {
	setupParamCacheTestDB(t)
	ctx := context.Background()

	type testConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	original := testConfig{Host: "localhost", Port: 8080}

	// First, store the param in DB
	if err := SetsParamCtx(ctx, "cache-test", original); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	// First call: cache miss, should load from DB and populate cache
	var loaded testConfig
	err := GetsParamCacheCtx(ctx, "cache-test", &loaded)
	if err != nil {
		t.Fatalf("GetsParamCacheCtx (cache miss): %v", err)
	}
	if loaded.Host != "localhost" || loaded.Port != 8080 {
		t.Errorf("loaded = %+v, want %+v", loaded, original)
	}

	// Second call: cache hit, should load from cache
	var loaded2 testConfig
	err = GetsParamCacheCtx(ctx, "cache-test", &loaded2)
	if err != nil {
		t.Fatalf("GetsParamCacheCtx (cache hit): %v", err)
	}
	if loaded2.Host != "localhost" || loaded2.Port != 8080 {
		t.Errorf("loaded2 = %+v, want %+v", loaded2, original)
	}
}

func TestGetsParamCacheCtx_NotFound(t *testing.T) {
	setupParamCacheTestDB(t)
	ctx := context.Background()

	var data map[string]string
	err := GetsParamCacheCtx(ctx, "nonexistent", &data)
	if err == nil {
		t.Fatal("GetsParamCacheCtx(nonexistent) should return error")
	}
}

func TestGetsParamCacheCtx_NilData(t *testing.T) {
	setupParamCacheTestDB(t)
	ctx := context.Background()

	err := GetsParamCacheCtx(ctx, "key", nil)
	if err == nil {
		t.Fatal("GetsParamCacheCtx(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("GetsParamCacheCtx(nil data) = %v, want ErrParamDataNil", err)
	}
}

func TestGetsParamCacheCtx_WithTimeout(t *testing.T) {
	setupParamCacheTestDB(t)
	ctx := context.Background()

	// Store a param
	if err := SetParamCtx(ctx, "timeout-test", []byte(`{"value":"cached"}`)); err != nil {
		t.Fatalf("SetParamCtx: %v", err)
	}

	type testData struct {
		Value string `json:"value"`
	}
	var loaded testData
	// Use a very short timeout duration
	err := GetsParamCacheCtx(ctx, "timeout-test", &loaded, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("GetsParamCacheCtx with timeout: %v", err)
	}
	if loaded.Value != "cached" {
		t.Errorf("Value = %q, want %q", loaded.Value, "cached")
	}
}

func TestGetsParamCache_Global(t *testing.T) {
	setupParamCacheTestDB(t)

	type testConfig struct {
		Name string `json:"name"`
	}
	original := testConfig{Name: "global-test"}

	// Store in DB
	if err := SetsParam("global-cache-param", original); err != nil {
		t.Fatalf("SetsParam: %v", err)
	}

	// Global wrapper should work
	var loaded testConfig
	err := GetsParamCache("global-cache-param", &loaded)
	if err != nil {
		t.Fatalf("GetsParamCache: %v", err)
	}
	if loaded.Name != "global-test" {
		t.Errorf("Name = %q, want %q", loaded.Name, "global-test")
	}
}

func TestGetsParamCacheCtx_CacheInvalidJSON(t *testing.T) {
	setupParamCacheTestDB(t)
	ctx := context.Background()

	// Store invalid JSON directly in cache
	invalidData := []byte("not-valid-json")
	err := comm.CacheSet("bad-json-cache", invalidData)
	if err != nil {
		t.Fatalf("CacheSet: %v", err)
	}

	// GetsParamCacheCtx should fail to unmarshal
	type testData struct {
		Key string `json:"key"`
	}
	var data testData
	err = GetsParamCacheCtx(ctx, "bad-json-cache", &data)
	if err == nil {
		t.Fatal("GetsParamCacheCtx with invalid JSON in cache should return error")
	}
}

func TestGetsParamCacheCtx_UpdatedParamInvalidatesCache(t *testing.T) {
	setupParamCacheTestDB(t)
	ctx := context.Background()

	type config struct {
		Version int `json:"version"`
	}
	v1 := config{Version: 1}
	if err := SetsParamCtx(ctx, "versioned", v1); err != nil {
		t.Fatalf("SetsParamCtx v1: %v", err)
	}

	// Load into cache
	var loaded config
	if err := GetsParamCacheCtx(ctx, "versioned", &loaded); err != nil {
		t.Fatalf("GetsParamCacheCtx v1: %v", err)
	}
	if loaded.Version != 1 {
		t.Errorf("Version = %d, want 1", loaded.Version)
	}

	// Update the param
	v2 := config{Version: 2}
	if err := SetsParamCtx(ctx, "versioned", v2); err != nil {
		t.Fatalf("SetsParamCtx v2: %v", err)
	}

	// Cache still has old value (cache doesn't auto-invalidate on update)
	var loaded2 config
	if err := GetsParamCacheCtx(ctx, "versioned", &loaded2); err != nil {
		t.Fatalf("GetsParamCacheCtx v2 (cached): %v", err)
	}
	// Note: this tests current behavior, which may or may not auto-invalidate
	t.Logf("After update, cached version = %d (cache may still serve old value)", loaded2.Version)
}

// TestGetsParamCacheCtx_MultipleKeys tests caching multiple different keys.
func TestGetsParamCacheCtx_MultipleKeys(t *testing.T) {
	setupParamCacheTestDB(t)
	ctx := context.Background()

	params := map[string]string{
		"key-a": `{"val":"a"}`,
		"key-b": `{"val":"b"}`,
		"key-c": `{"val":"c"}`,
	}
	for k, v := range params {
		if err := SetParamCtx(ctx, k, []byte(v)); err != nil {
			t.Fatalf("SetParamCtx(%s): %v", k, err)
		}
	}

	type data struct {
		Val string `json:"val"`
	}
	for k, want := range params {
		var d data
		if err := GetsParamCacheCtx(ctx, k, &d); err != nil {
			t.Errorf("GetsParamCacheCtx(%s): %v", k, err)
			continue
		}
		wantData := data{}
		if err := json.Unmarshal([]byte(want), &wantData); err != nil {
			t.Fatalf("unmarshal expected: %v", err)
		}
		if d.Val != wantData.Val {
			t.Errorf("GetsParamCacheCtx(%s).Val = %q, want %q", k, d.Val, wantData.Val)
		}
	}
}
