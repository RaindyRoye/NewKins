package comm

import (
	"sync"
	"testing"
)

// --- MarkInstalled ---

func TestMarkInstalled_ClosesChannel(t *testing.T) {
	// Reset state for test isolation
	Installed = false
	installOnce = syncOnce()
	InstalledCh = make(chan struct{})

	MarkInstalled()
	if !Installed {
		t.Error("MarkInstalled should set Installed to true")
	}
	// Channel should be closed
	select {
	case <-InstalledCh:
		// ok, channel is closed
	default:
		t.Error("InstalledCh should be closed after MarkInstalled")
	}
}

func TestMarkInstalled_Idempotent(t *testing.T) {
	// Reset state
	Installed = false
	installOnce = syncOnce()
	InstalledCh = make(chan struct{})

	// Call multiple times - should not panic
	MarkInstalled()
	MarkInstalled()
	MarkInstalled()
	if !Installed {
		t.Error("Installed should remain true")
	}
}

// --- Cancel ---

func TestCancel_CancelsContext(t *testing.T) {
	ResetCtx() // start fresh
	Cancel()
	select {
	case <-Ctx.Done():
		// ok
	default:
		t.Error("Cancel() should cancel the context")
	}
}

func TestCancel_NilCancelFunc(t *testing.T) {
	// Save and restore
	oldCtx := Ctx
	oldCncl := cncl
	defer func() {
		Ctx = oldCtx
		cncl = oldCncl
	}()

	cncl = nil
	// Should not panic
	Cancel()
}

// --- ResetCtx ---

func TestResetCtx_CreatesNewContext(t *testing.T) {
	ResetCtx()
	if Ctx == nil {
		t.Fatal("ResetCtx should create a non-nil context")
	}
	if cncl == nil {
		t.Fatal("ResetCtx should create a non-nil cancel function")
	}
	// Context should not be canceled
	select {
	case <-Ctx.Done():
		t.Error("new context should not be canceled")
	default:
		// ok
	}
}

func TestResetCtx_CancelsPreviousContext(t *testing.T) {
	ResetCtx()
	prevCtx := Ctx

	ResetCtx() // should cancel prevCtx
	select {
	case <-prevCtx.Done():
		// ok, previous context was canceled
	default:
		t.Error("ResetCtx should cancel the previous context")
	}
}

func TestResetCtx_DifferentContexts(t *testing.T) {
	ResetCtx()
	ctx1 := Ctx

	ResetCtx()
	ctx2 := Ctx

	if ctx1 == ctx2 {
		t.Error("ResetCtx should create a different context each time")
	}
}

// syncOnce returns a fresh sync.Once for test isolation.
// We need this because sync.Once cannot be reset.
func syncOnce() sync.Once {
	return sync.Once{}
}

// --- Build Info Variables ---

func TestBuildInfoDefaults(t *testing.T) {
	// Verify the build info variables have their expected default values.
	// These are set at build time via -ldflags, but should default to
	// "unknown" when not set (e.g., during development or testing).
	if Version == "" {
		t.Error("Version should have a non-empty default value")
	}
	if BuildTime == "" {
		t.Error("BuildTime should have a non-empty default value")
	}
	if GitCommit == "" {
		t.Error("GitCommit should have a non-empty default value")
	}
}

func TestBuildInfoSettable(t *testing.T) {
	// Verify build info variables can be set (as ldflags would do at build time).
	oldVersion := Version
	oldBuild := BuildTime
	oldCommit := GitCommit
	defer func() {
		Version = oldVersion
		BuildTime = oldBuild
		GitCommit = oldCommit
	}()

	Version = "9.9.9-test"
	BuildTime = "2026-01-01T00:00:00Z"
	GitCommit = "abc1234"

	if Version != "9.9.9-test" {
		t.Errorf("Version = %q, want %q", Version, "9.9.9-test")
	}
	if BuildTime != "2026-01-01T00:00:00Z" {
		t.Errorf("BuildTime = %q, want %q", BuildTime, "2026-01-01T00:00:00Z")
	}
	if GitCommit != "abc1234" {
		t.Errorf("GitCommit = %q, want %q", GitCommit, "abc1234")
	}
}
