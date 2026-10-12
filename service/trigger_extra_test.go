package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gokins/gokins/model"
)

// --- TriggerPerm Tests ---

func TestTriggerPermCtx_TriggerWithEmptyPipelineId(t *testing.T) {
	ctx := context.Background()
	tt := &model.TTrigger{
		Id:         "trigger1",
		Uid:        "user1",
		PipelineId: "",
	}
	err := TriggerPermCtx(ctx, tt)
	// With empty pipeline ID, NewPipePermCtx will not find the pipeline
	if err == nil {
		t.Fatal("TriggerPermCtx with empty pipeline ID should return error")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("TriggerPermCtx = %v, want ErrPipelineNotFound", err)
	}
}

func TestTriggerPermCtx_TriggerWithNonexistentPipeline(t *testing.T) {
	ctx := context.Background()
	tt := &model.TTrigger{
		Id:         "trigger1",
		Uid:        "user1",
		PipelineId: "nonexistent-pipeline",
	}
	// NewPipePermCtx now handles nil Db gracefully, no panic expected.
	err := TriggerPermCtx(ctx, tt)
	// Should return ErrPipelineNotFound
	if err == nil {
		t.Fatal("TriggerPermCtx with nonexistent pipeline should return error")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("TriggerPermCtx = %v, want ErrPipelineNotFound", err)
	}
}

// --- TriggerPerm edge cases ---

func TestTriggerPermCtx_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	tt := &model.TTrigger{
		Id:         "trigger1",
		Uid:        "user1",
		PipelineId: "pipe1",
	}
	// NewPipePermCtx now handles nil Db gracefully, no panic expected.
	err := TriggerPermCtx(ctx, tt)
	// With canceled context and no DB, should still return an error
	if err == nil {
		t.Fatal("TriggerPermCtx with canceled context should return error")
	}
}

// --- Verify error types ---

func TestTriggerPermCtx_ErrorTypes(t *testing.T) {
	tests := []struct {
		name    string
		trigger *model.TTrigger
		wantErr error
	}{
		{
			"nil trigger",
			nil,
			ErrPipelineNotFound,
		},
		{
			"empty pipeline ID",
			&model.TTrigger{Id: "t1", Uid: "u1", PipelineId: ""},
			ErrPipelineNotFound,
		},
		{
			"nonexistent pipeline",
			&model.TTrigger{Id: "t1", Uid: "u1", PipelineId: "fake-pipe"},
			ErrPipelineNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			// NewPipePermCtx now handles nil Db gracefully, no panic expected.
			err := TriggerPermCtx(ctx, tt.trigger)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("TriggerPermCtx() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
