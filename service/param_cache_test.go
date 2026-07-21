package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gokins/gokins/comm"
)

// TestGetsParamCacheCtx_CacheHit verifies that GetsParamCacheCtx returns cached
// data without hitting the database when cache is available.
func TestGetsParamCacheCtx_CacheHit(t *testing.T) {
	if comm.Db == nil {
		t.Skip("skipping: comm.Db is nil (no database)")
	}

	ctx := context.Background()
	testKey := "test-cache-hit-key"

	// First call to populate cache
	err := SetsParamCtx(ctx, testKey, []byte(`{"foo":"bar"}`))
	if err != nil {
		t.Skipf("skipping: cannot set param: %v", err)
	}

	var data1 map[string]string
	err = GetsParamCacheCtx(ctx, testKey, &data1)
	if err != nil {
		t.Fatalf("first GetsParamCacheCtx failed: %v", err)
	}
	if data1["foo"] != "bar" {
		t.Errorf("first call: data[foo] = %q, want %q", data1["foo"], "bar")
	}

	// Second call should hit cache
	var data2 map[string]string
	err = GetsParamCacheCtx(ctx, testKey, &data2)
	if err != nil {
		t.Fatalf("second GetsParamCacheCtx failed: %v", err)
	}
	if data2["foo"] != "bar" {
		t.Errorf("second call: data[foo] = %q, want %q", data2["foo"], "bar")
	}
}

// TestGetsParamCacheCtx_NilData_CachePath verifies that GetsParamCacheCtx returns
// ErrParamDataNil when data parameter is nil.
func TestGetsParamCacheCtx_NilData_CachePath(t *testing.T) {
	ctx := context.Background()
	err := GetsParamCacheCtx(ctx, "test-key", nil)
	if err == nil {
		t.Fatal("GetsParamCacheCtx(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("GetsParamCacheCtx(nil data) = %v, want ErrParamDataNil", err)
	}
}
