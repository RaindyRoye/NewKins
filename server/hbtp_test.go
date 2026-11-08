package server

import (
	"testing"

	"github.com/gokins/gokins/comm"
)

// TestRunHbtp_EmptyHost tests that runHbtp returns immediately when
// HbtpHost is empty, without starting the hbtp engine.
func TestRunHbtp_EmptyHost(t *testing.T) {
	origCfg := comm.Cfg
	origHbtpEgn := comm.HbtpEgn
	defer func() {
		comm.Cfg = origCfg
		comm.HbtpEgn = origHbtpEgn
	}()

	comm.Cfg.Server.HbtpHost = ""

	// Should not panic and should return immediately
	runHbtp()

	// HbtpEgn should remain nil since we didn't start it
	if comm.HbtpEgn != nil {
		t.Error("HbtpEgn should remain nil when HbtpHost is empty")
	}
}
