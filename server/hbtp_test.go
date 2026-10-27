package server

import (
	"testing"

	"github.com/gokins/gokins/comm"
)

// TestRunHbtp_EmptyHost tests runHbtp returns early when HbtpHost is empty.
func TestRunHbtp_EmptyHost(t *testing.T) {
	origCfg := comm.Cfg
	origHbtpEgn := comm.HbtpEgn
	defer func() {
		comm.Cfg = origCfg
		comm.HbtpEgn = origHbtpEgn
	}()

	comm.Cfg.Server.HbtpHost = ""
	comm.HbtpEgn = nil

	// Should return immediately without panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("runHbtp panicked with empty HbtpHost: %v", r)
		}
	}()

	runHbtp()

	// HbtpEgn should not be initialized when host is empty
	if comm.HbtpEgn != nil {
		t.Error("HbtpEgn should not be set when HbtpHost is empty")
	}
}
