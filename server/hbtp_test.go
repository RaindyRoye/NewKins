package server

import (
	"testing"

	"github.com/gokins/gokins/comm"
)

func TestRunHbtp_EmptyHost(t *testing.T) {
	// runHbtp should return immediately when HbtpHost is empty.
	// This exercises the early-return path at 0% coverage.
	origCfg := comm.Cfg
	defer func() { comm.Cfg = origCfg }()

	comm.Cfg.Server.HbtpHost = ""
	comm.HbtpEgn = nil

	// Should return without panic or side effects
	runHbtp()

	if comm.HbtpEgn != nil {
		t.Error("expected HbtpEgn to remain nil when HbtpHost is empty")
	}
}
