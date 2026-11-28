package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestTimerEngineRefreshCtxEmptyID(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	err := te.RefreshCtx(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty timer ID")
	}
	if !strings.Contains(err.Error(), "timer id is empty") {
		t.Fatalf("expected error containing 'timer id is empty', got %q", err.Error())
	}
	if !errors.Is(err, ErrEmptyParams) {
		t.Fatalf("expected error to wrap ErrEmptyParams, got %v", err)
	}
}

func TestTimerEngineRefreshDelegatesToRefreshCtx(t *testing.T) {
	// Verify that Refresh("") produces the same error as RefreshCtx with empty ID.
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}
	err1 := te.Refresh("")
	err2 := te.RefreshCtx(context.Background(), "")
	if err1 == nil || err2 == nil {
		t.Fatal("expected both to return errors")
	}
	if err1.Error() != err2.Error() {
		t.Fatalf("Refresh and RefreshCtx should produce same error for empty ID:\n  Refresh:     %v\n  RefreshCtx:  %v", err1, err2)
	}
}
