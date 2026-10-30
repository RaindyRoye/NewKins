package server

import (
	"testing"

	"github.com/gokins/gokins/comm"
)

// TestRunHbtp_EmptyHost verifies that runHbtp returns early when HbtpHost is not configured.
func TestRunHbtp_EmptyHost(t *testing.T) {
	origCfg := comm.Cfg
	defer func() { comm.Cfg = origCfg }()

	comm.Cfg.Server.HbtpHost = ""

	// Should not panic and should return early
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("runHbtp panicked with empty host: %v", r)
		}
	}()
	runHbtp()
}

// TestRunHbtp_InvalidHost verifies that runHbtp handles connection errors gracefully.
func TestRunHbtp_InvalidHost(t *testing.T) {
	origCfg := comm.Cfg
	defer func() { comm.Cfg = origCfg }()

	// Use an invalid host that will fail to bind
	comm.Cfg.Server.HbtpHost = "invalid-host:99999"

	// Should not panic; it will log error and cancel context
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("runHbtp panicked with invalid host: %v", r)
		}
	}()
	runHbtp()
}
