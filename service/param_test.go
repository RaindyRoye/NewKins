package service

import (
	"errors"
	"testing"
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
