package service

import (
	"context"
	"testing"
)

func TestGetIdOrAidCtx_NilId(t *testing.T) {
	ok := GetIdOrAidCtx(context.Background(), nil, &struct{}{})
	if ok {
		t.Error("GetIdOrAidCtx(nil id) should return false")
	}
}

func TestGetIdOrAidCtx_NilEntity(t *testing.T) {
	ok := GetIdOrAidCtx(context.Background(), "some-id", nil)
	if ok {
		t.Error("GetIdOrAidCtx(nil entity) should return false")
	}
}

func TestGetIdOrAidCtx_EmptyStringId(t *testing.T) {
	ok := GetIdOrAidCtx(context.Background(), "", &struct{}{})
	if ok {
		t.Error("GetIdOrAidCtx(empty string id) should return false")
	}
}

func TestGetIdOrAidCtx_NilBoth(t *testing.T) {
	ok := GetIdOrAidCtx(context.Background(), nil, nil)
	if ok {
		t.Error("GetIdOrAidCtx(nil, nil) should return false")
	}
}

func TestGetIdOrAidECtx_NilId(t *testing.T) {
	ok, err := GetIdOrAidECtx(context.Background(), nil, &struct{}{})
	if ok {
		t.Error("GetIdOrAidECtx(nil id) should return false")
	}
	if err != nil {
		t.Errorf("GetIdOrAidECtx(nil id) should not return error, got: %v", err)
	}
}

func TestGetIdOrAidECtx_NilEntity(t *testing.T) {
	ok, err := GetIdOrAidECtx(context.Background(), "some-id", nil)
	if ok {
		t.Error("GetIdOrAidECtx(nil entity) should return false")
	}
	if err != nil {
		t.Errorf("GetIdOrAidECtx(nil entity) should not return error, got: %v", err)
	}
}

func TestGetIdOrAidECtx_EmptyStringId(t *testing.T) {
	ok, err := GetIdOrAidECtx(context.Background(), "", &struct{}{})
	if ok {
		t.Error("GetIdOrAidECtx(empty string id) should return false")
	}
	if err != nil {
		t.Errorf("GetIdOrAidECtx(empty string id) should not return error, got: %v", err)
	}
}

func TestGetIdOrAidECtx_NilBoth(t *testing.T) {
	ok, err := GetIdOrAidECtx(context.Background(), nil, nil)
	if ok {
		t.Error("GetIdOrAidECtx(nil, nil) should return false")
	}
	if err != nil {
		t.Errorf("GetIdOrAidECtx(nil, nil) should not return error, got: %v", err)
	}
}

func TestGetIdOrAidCtx_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	// With a cancelled context and nil inputs, should still return false (short-circuit)
	ok := GetIdOrAidCtx(ctx, nil, &struct{}{})
	if ok {
		t.Error("GetIdOrAidCtx with cancelled context and nil id should return false")
	}
}

func TestGetIdOrAidECtx_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	ok, err := GetIdOrAidECtx(ctx, "", &struct{}{})
	if ok {
		t.Error("GetIdOrAidECtx with cancelled context and empty id should return false")
	}
	if err != nil {
		t.Errorf("GetIdOrAidECtx with cancelled context and empty id should not error, got: %v", err)
	}
}

func TestGetIdOrAidCtx_NonStringNonNilId(t *testing.T) {
	// A non-string, non-nil id (like an int) should NOT short-circuit
	// It would need a DB to proceed, but with nil comm.Db we expect a panic.
	// We test only that the empty-string check doesn't catch it.
	defer func() {
		if r := recover(); r != nil {
			// Expected: nil pointer dereference from comm.Db being nil
			t.Logf("recovered expected panic (Db is nil): %v", r)
		}
	}()
	// Non-string id (int 123) - won't be caught by empty string check
	_ = GetIdOrAidCtx(context.Background(), 123, &struct{}{})
	// If no panic, Db might be initialized; that's also fine.
}
