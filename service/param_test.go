package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gokins/gokins/comm"
)

func TestSetsParam_NilData(t *testing.T) {
	err := SetsParam("test-key", nil)
	if err == nil {
		t.Fatal("SetsParam(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("SetsParam(nil data) = %v, want ErrParamDataNil", err)
	}
}

func TestGetsParam_NilData(t *testing.T) {
	err := GetsParam("test-key", nil)
	if err == nil {
		t.Fatal("GetsParam(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("GetsParam(nil data) = %v, want ErrParamDataNil", err)
	}
}

func TestSetsParamCtx_NilData(t *testing.T) {
	ctx := context.Background()
	err := SetsParamCtx(ctx, "test-key", nil)
	if err == nil {
		t.Fatal("SetsParamCtx(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("SetsParamCtx(nil data) = %v, want ErrParamDataNil", err)
	}
}

func TestGetsParamCtx_NilData(t *testing.T) {
	ctx := context.Background()
	err := GetsParamCtx(ctx, "test-key", nil)
	if err == nil {
		t.Fatal("GetsParamCtx(nil data) should return error")
	}
	if !errors.Is(err, ErrParamDataNil) {
		t.Errorf("GetsParamCtx(nil data) = %v, want ErrParamDataNil", err)
	}
}

func TestGetsParamCtx_CancelledContext(t *testing.T) {
	// Skip if database is not initialized (e.g. in CI without DB)
	if comm.Db == nil {
		t.Skip("skipping: comm.Db is nil (no database)")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	var data map[string]string
	err := GetsParamCtx(ctx, "test-key", &data)
	if err == nil {
		t.Fatal("GetsParamCtx with canceled context should return error")
	}
}

func TestGetParamCtx_CancelledContext(t *testing.T) {
	// Skip if database is not initialized (e.g. in CI without DB)
	if comm.Db == nil {
		t.Skip("skipping: comm.Db is nil (no database)")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := GetParamCtx(ctx, "test-key")
	if err == nil {
		t.Fatal("GetParamCtx with canceled context should return error")
	}
}
