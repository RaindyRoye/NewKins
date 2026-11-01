package bean

import (
	"errors"
	"testing"
)

func TestTriggerParamCheck_Valid(t *testing.T) {
	tp := &TriggerParam{
		PipelineId: "pipe-1",
		Types:      "webhook",
		Name:       "test-trigger",
		Params:     `{"key":"value"}`,
	}
	if err := tp.Check(); err != nil {
		t.Fatalf("unexpected error for valid params: %v", err)
	}
}

func TestTriggerParamCheck_PipelineIdRequired(t *testing.T) {
	tp := &TriggerParam{
		PipelineId: "",
		Types:      "webhook",
		Name:       "test",
		Params:     "{}",
	}
	err := tp.Check()
	if err == nil {
		t.Fatal("expected error for empty PipelineId")
	}
	if !errors.Is(err, ErrPipelineIdRequired) {
		t.Errorf("expected ErrPipelineIdRequired, got %v", err)
	}
}

func TestTriggerParamCheck_TriggerTypeRequired(t *testing.T) {
	tp := &TriggerParam{
		PipelineId: "pipe-1",
		Types:      "",
		Name:       "test",
		Params:     "{}",
	}
	err := tp.Check()
	if err == nil {
		t.Fatal("expected error for empty Types")
	}
	if !errors.Is(err, ErrTriggerTypeRequired) {
		t.Errorf("expected ErrTriggerTypeRequired, got %v", err)
	}
}

func TestTriggerParamCheck_TriggerNameRequired(t *testing.T) {
	tp := &TriggerParam{
		PipelineId: "pipe-1",
		Types:      "webhook",
		Name:       "",
		Params:     "{}",
	}
	err := tp.Check()
	if err == nil {
		t.Fatal("expected error for empty Name")
	}
	if !errors.Is(err, ErrTriggerNameRequired) {
		t.Errorf("expected ErrTriggerNameRequired, got %v", err)
	}
}

func TestTriggerParamCheck_TriggerParamsRequired(t *testing.T) {
	tp := &TriggerParam{
		PipelineId: "pipe-1",
		Types:      "webhook",
		Name:       "test",
		Params:     "",
	}
	err := tp.Check()
	if err == nil {
		t.Fatal("expected error for empty Params")
	}
	if !errors.Is(err, ErrTriggerParamsRequired) {
		t.Errorf("expected ErrTriggerParamsRequired, got %v", err)
	}
}

func TestTriggerParamCheck_ErrorOrder(t *testing.T) {
	// All fields empty - should fail on first check (PipelineId)
	tp := &TriggerParam{}
	err := tp.Check()
	if !errors.Is(err, ErrPipelineIdRequired) {
		t.Errorf("expected first error to be ErrPipelineIdRequired, got %v", err)
	}
}
