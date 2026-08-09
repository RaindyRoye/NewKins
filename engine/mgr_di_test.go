package engine

import (
	"context"
	"testing"
)

func TestNewManager_DefaultContext(t *testing.T) {
	// NewManager with nil Ctx should use context.Background()
	m, err := NewManager(ManagerConfig{})
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil Manager")
	}
	if m.ctx == nil {
		t.Error("expected ctx to be set to context.Background()")
	}
	// Cleanup: stop the engines started by NewManager
	m.Stop()
}

func TestNewManager_CustomContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, err := NewManager(ManagerConfig{
		Ctx:      ctx,
		WorkPath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}
	if m.ctx != ctx {
		t.Error("expected ctx to match the provided context")
	}
	m.Stop()
}

func TestNewManager_CustomWorkPath(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := NewManager(ManagerConfig{
		WorkPath: tmpDir,
	})
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}
	if m.workPath != tmpDir {
		t.Errorf("expected workPath=%q, got %q", tmpDir, m.workPath)
	}
	m.Stop()
}

func TestNewManager_InitializesAllEngines(t *testing.T) {
	m, err := NewManager(ManagerConfig{
		WorkPath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}
	defer m.Stop()

	if m.buildEgn == nil {
		t.Error("expected buildEgn to be initialized")
	}
	if m.jobEgn == nil {
		t.Error("expected jobEgn to be initialized")
	}
	if m.tmrEgn == nil {
		t.Error("expected tmrEgn to be initialized")
	}
	if m.brun == nil {
		t.Error("expected brun to be initialized")
	}
	if m.hrun == nil {
		t.Error("expected hrun to be initialized")
	}
}

func TestManager_Stop_NilFields(t *testing.T) {
	// Stop should not panic when all fields are nil
	m := &Manager{}
	m.Stop()
}

func TestManager_StartWithContext_NilContext(t *testing.T) {
	m := &Manager{}
	err := m.StartWithContext()
	if err != nil {
		t.Fatalf("StartWithContext() error: %v", err)
	}
	if m.ctx == nil {
		t.Error("expected ctx to be set after StartWithContext with nil ctx")
	}
}

func TestManager_StartWithContext_ActiveContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := &Manager{ctx: ctx}
	err := m.StartWithContext()
	if err != nil {
		t.Fatalf("StartWithContext() error: %v", err)
	}

	// Cancel to stop cleanup goroutine
	cancel()
}
