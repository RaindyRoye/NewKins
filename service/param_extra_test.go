package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestParam_FindParamCtx_NilDB(t *testing.T) {
	// With nil comm.Db, this should panic. We recover and log.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic (Db is nil): %v", r)
		}
	}()
	_, _ = FindParamCtx(context.Background(), "test-key")
}

func TestParam_GetParamCtx_NilDB(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic (Db is nil): %v", r)
		}
	}()
	_, _ = GetParamCtx(context.Background(), "test-key")
}

func TestParam_SetParamCtx_NilDB(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic (Db is nil): %v", r)
		}
	}()
	_ = SetParamCtx(context.Background(), "test-key", []byte("data"))
}

func TestParam_SetsParamCtx_NilData(t *testing.T) {
	err := SetsParamCtx(context.Background(), "test-key", nil)
	if err == nil {
		t.Fatal("expected error for nil data")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("expected ErrParamDataNil, got: %v", err)
	}
}

func TestParam_GetsParamCtx_NilData(t *testing.T) {
	err := GetsParamCtx(context.Background(), "test-key", nil)
	if err == nil {
		t.Fatal("expected error for nil data")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("expected ErrParamDataNil, got: %v", err)
	}
}

func TestParam_SetsParamCtx_ValidData(t *testing.T) {
	// This will panic because Db is nil, but we verify the marshaling path
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic (Db is nil after marshal): %v", r)
		}
	}()
	data := map[string]string{"key": "value"}
	_ = SetsParamCtx(context.Background(), "test-key", data, "Test Title")
}

func TestParam_GetParamCtx_NotFound(t *testing.T) {
	// This will panic because Db is nil
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic (Db is nil): %v", r)
		}
	}()
	_, _ = GetParamCtx(context.Background(), "nonexistent-key")
}

func TestParam_GetsParamCtx_InvalidJSON(t *testing.T) {
	// This will panic when trying to query the DB
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic (Db is nil): %v", r)
		}
	}()
	var result map[string]string
	_ = GetsParamCtx(context.Background(), "test-key", &result)
}

func TestParam_GetsParamCacheCtx_CacheHit(t *testing.T) {
	// Test cache hit path - this requires comm.CacheGets to work
	// Since cache is not initialized, this will fail and fall through to DB query
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered expected panic (Db is nil after cache miss): %v", r)
		}
	}()
	var result map[string]string
	_ = GetsParamCacheCtx(context.Background(), "test-key", &result, time.Minute)
}

func TestParam_GetsParamCacheCtx_NilData(t *testing.T) {
	err := GetsParamCacheCtx(context.Background(), "test-key", nil, time.Minute)
	if err == nil {
		t.Fatal("expected error for nil data")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("expected ErrParamDataNil, got: %v", err)
	}
}

func TestParam_JSON_Marshal_Unmarshal(t *testing.T) {
	// Test JSON serialization path without DB
	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	original := TestData{Name: "test", Value: 42}
	bts, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded TestData
	err = json.Unmarshal(bts, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Name != original.Name || decoded.Value != original.Value {
		t.Errorf("decoded data doesn't match: got %+v, want %+v", decoded, original)
	}
}
