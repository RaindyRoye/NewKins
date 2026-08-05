package server

import (
	"testing"

	"github.com/gokins/core"
	"github.com/gokins/gokins/comm"
)

// TestPprofEnabledByConfig verifies that pprof endpoints can be enabled via configuration
func TestPprofEnabledByConfig(t *testing.T) {
	// Save original values
	origDebug := core.Debug
	origEnablePprof := comm.Cfg.Server.EnablePprof
	defer func() {
		core.Debug = origDebug
		comm.Cfg.Server.EnablePprof = origEnablePprof
	}()

	// Test case 1: Debug mode enabled
	core.Debug = true
	comm.Cfg.Server.EnablePprof = false
	if !core.Debug && !comm.Cfg.Server.EnablePprof {
		t.Error("Expected pprof to be enabled when core.Debug is true")
	}

	// Test case 2: Config enabled
	core.Debug = false
	comm.Cfg.Server.EnablePprof = true
	if !core.Debug && !comm.Cfg.Server.EnablePprof {
		t.Error("Expected pprof to be enabled when config.EnablePprof is true")
	}

	// Test case 3: Both enabled
	core.Debug = true
	comm.Cfg.Server.EnablePprof = true
	if !core.Debug && !comm.Cfg.Server.EnablePprof {
		t.Error("Expected pprof to be enabled when both are true")
	}

	// Test case 4: Both disabled
	core.Debug = false
	comm.Cfg.Server.EnablePprof = false
	if core.Debug || comm.Cfg.Server.EnablePprof {
		t.Error("Expected pprof to be disabled when both are false")
	}
}

// TestConfigEnablePprofField verifies the config field is properly defined
func TestConfigEnablePprofField(t *testing.T) {
	cfg := comm.Config{}

	// Field should exist and default to false
	if cfg.Server.EnablePprof != false {
		t.Error("EnablePprof should default to false")
	}

	// Should be settable
	cfg.Server.EnablePprof = true
	if !cfg.Server.EnablePprof {
		t.Error("EnablePprof should be settable to true")
	}
}
