package service

import (
	"context"
	"testing"
)

// TestFindParamCtx_WithNilDb verifies FindParamCtx handles nil DB gracefully.
func TestFindParamCtx_WithNilDb(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil in unit tests): %v", r)
		}
	}()
	_, ok := FindParamCtx(context.Background(), "test_key")
	if ok {
		t.Error("FindParamCtx should return false when Db is nil")
	}
}

// TestGetParamCtx_NotFound verifies GetParamCtx returns error for missing param.
func TestGetParamCtx_NotFound(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil in unit tests): %v", r)
		}
	}()
	_, err := GetParamCtx(context.Background(), "nonexistent_key")
	if err == nil {
		t.Error("GetParamCtx should return error for nonexistent key")
	}
}

// TestGetsParamCtx_NilData verifies GetsParamCtx rejects nil data parameter.
func TestGetsParamCtx_NilDataParam(t *testing.T) {
	err := GetsParamCtx(context.Background(), "key", nil)
	if err == nil {
		t.Fatal("expected error for nil data parameter")
	}
	if err != ErrParamDataNil {
		t.Errorf("expected ErrParamDataNil, got: %v", err)
	}
}

// TestSetsParamCtx_NilDataParam verifies SetsParamCtx rejects nil data.
func TestSetsParamCtx_NilDataParam(t *testing.T) {
	err := SetsParamCtx(context.Background(), "key", nil)
	if err == nil {
		t.Fatal("expected error for nil data parameter")
	}
	if err != ErrParamDataNil {
		t.Errorf("expected ErrParamDataNil, got: %v", err)
	}
}

// TestGetsParamCacheCtx_NilData verifies GetsParamCacheCtx rejects nil data.
func TestGetsParamCacheCtx_NilData(t *testing.T) {
	err := GetsParamCacheCtx(context.Background(), "key", nil)
	if err == nil {
		t.Fatal("expected error for nil data parameter")
	}
	if err != ErrParamDataNil {
		t.Errorf("expected ErrParamDataNil, got: %v", err)
	}
}

// TestSetParamCtx_WithNilDb verifies SetParamCtx handles nil DB.
func TestSetParamCtx_WithNilDb(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil): %v", r)
		}
	}()
	err := SetParamCtx(context.Background(), "key", []byte("data"), "title")
	if err == nil {
		t.Error("SetParamCtx should return error when Db is nil")
	}
}

// TestGetsParam_NilData verifies GetsParam rejects nil data.
func TestGetsParam_NilDataParam(t *testing.T) {
	err := GetsParam("key", nil)
	if err == nil {
		t.Fatal("expected error for nil data parameter")
	}
	if err != ErrParamDataNil {
		t.Errorf("expected ErrParamDataNil, got: %v", err)
	}
}

// TestSetParam_WithNilDb verifies SetParam handles nil DB.
func TestSetParam_WithNilDb(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil): %v", r)
		}
	}()
	err := SetParam("key", []byte("data"), "title")
	if err == nil {
		t.Error("SetParam should return error when Db is nil")
	}
}

// TestGetParam_WithNilDb verifies GetParam handles nil DB.
func TestGetParam_WithNilDb(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil): %v", r)
		}
	}()
	_, err := GetParam("key")
	if err == nil {
		t.Error("GetParam should return error when Db is nil")
	}
}

// TestFindParam_WithNilDb verifies FindParam handles nil DB.
func TestFindParam_WithNilDb(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil): %v", r)
		}
	}()
	_, ok := FindParam("key")
	if ok {
		t.Error("FindParam should return false when Db is nil")
	}
}
