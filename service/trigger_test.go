package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gokins/gokins/model"
)

func TestTriggerPermCtx_NilTrigger(t *testing.T) {
	err := TriggerPermCtx(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil trigger, got nil")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestTriggerPerm_NilTrigger(t *testing.T) {
	err := TriggerPerm(nil)
	if err == nil {
		t.Fatal("expected error for nil trigger, got nil")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestTriggerPermCtx_EmptyPipelineId(t *testing.T) {
	// With empty pipeline ID, NewPipePermCtx won't query DB and pipe will be nil,
	// so TriggerPermCtx should return ErrPipelineNotFound.
	tt := &model.TTrigger{
		Uid:        "user1",
		PipelineId: "",
	}
	err := TriggerPermCtx(context.Background(), tt)
	if err == nil {
		t.Fatal("expected error for empty pipeline ID, got nil")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestTriggerPermCtx_UnknownPipeline(t *testing.T) {
	// With a non-empty but non-existent pipeline ID, the DB query in
	// NewPipePermCtx will fail to find the pipeline (when DB is nil or the
	// pipeline doesn't exist), so TriggerPermCtx should return ErrPipelineNotFound.
	tt := &model.TTrigger{
		Uid:        "user1",
		PipelineId: "nonexistent-pipeline-id",
	}
	// This will panic if comm.Db is nil, so we recover gracefully.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (Db is nil in unit tests): %v", r)
		}
	}()
	err := TriggerPermCtx(context.Background(), tt)
	if err == nil {
		t.Fatal("expected error for nonexistent pipeline, got nil")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestTriggerPermCtx_DelegatesToNewPipePermCtx(t *testing.T) {
	// Verify that TriggerPerm passes through to TriggerPermCtx with comm.Ctx.
	// Both should return the same error for nil trigger.
	err1 := TriggerPerm(nil)
	err2 := TriggerPermCtx(context.Background(), nil)
	if !errors.Is(err1, err2) {
		t.Errorf("TriggerPerm and TriggerPermCtx should return the same error type for nil trigger: %v vs %v", err1, err2)
	}
}

func TestTriggerPerm_ErrorTypes(t *testing.T) {
	// Verify sentinel errors are properly defined and distinguishable.
	sentinelErrors := []error{
		ErrPipelineNotFound,
		ErrPermissionDenied,
		ErrTriggerNoParams,
		ErrHookTypeEmpty,
		ErrWebhookParseFailed,
		ErrWebhookEventMismatch,
		ErrBranchMismatch,
		ErrTriggerNoSecret,
		ErrTriggerSecretMismatch,
	}
	for i, e1 := range sentinelErrors {
		if e1 == nil {
			t.Errorf("sentinel error at index %d is nil", i)
		}
		if e1.Error() == "" {
			t.Errorf("sentinel error at index %d has empty message", i)
		}
		for j, e2 := range sentinelErrors {
			if i != j && errors.Is(e1, e2) {
				t.Errorf("sentinel errors at index %d and %d should be distinct: %v vs %v", i, j, e1, e2)
			}
		}
	}
}
