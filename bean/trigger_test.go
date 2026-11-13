package bean

import (
	"errors"
	"testing"
)

func TestTriggerParamCheckSentinelErrors(t *testing.T) {
	tests := []struct {
		name       string
		param      TriggerParam
		wantErr    bool
		sentinel   error
		errMessage string
	}{
		{
			name: "valid trigger param",
			param: TriggerParam{
				PipelineId: "pipe-1",
				Types:      "webhook",
				Name:       "my-trigger",
				Params:     `{"hookType":"github"}`,
			},
			wantErr: false,
		},
		{
			name: "empty pipeline ID returns ErrPipelineIdRequired",
			param: TriggerParam{
				PipelineId: "",
				Types:      "webhook",
				Name:       "my-trigger",
				Params:     `{"hookType":"github"}`,
			},
			wantErr:    true,
			sentinel:   ErrPipelineIdRequired,
			errMessage: "pipeline ID is required",
		},
		{
			name: "empty types returns ErrTriggerTypeRequired",
			param: TriggerParam{
				PipelineId: "pipe-1",
				Types:      "",
				Name:       "my-trigger",
				Params:     `{"hookType":"github"}`,
			},
			wantErr:    true,
			sentinel:   ErrTriggerTypeRequired,
			errMessage: "trigger type is required",
		},
		{
			name: "empty name returns ErrTriggerNameRequired",
			param: TriggerParam{
				PipelineId: "pipe-1",
				Types:      "webhook",
				Name:       "",
				Params:     `{"hookType":"github"}`,
			},
			wantErr:    true,
			sentinel:   ErrTriggerNameRequired,
			errMessage: "trigger name is required",
		},
		{
			name: "empty params returns ErrTriggerParamsRequired",
			param: TriggerParam{
				PipelineId: "pipe-1",
				Types:      "webhook",
				Name:       "my-trigger",
				Params:     "",
			},
			wantErr:    true,
			sentinel:   ErrTriggerParamsRequired,
			errMessage: "trigger params is required",
		},
		{
			name:       "all fields empty fails on first check (pipeline ID)",
			param:      TriggerParam{},
			wantErr:    true,
			sentinel:   ErrPipelineIdRequired,
			errMessage: "pipeline ID is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.Check()
			if (err != nil) != tt.wantErr {
				t.Errorf("TriggerParam.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				if !errors.Is(err, tt.sentinel) {
					t.Errorf("TriggerParam.Check() error should wrap %v, got %v", tt.sentinel, err)
				}
				if tt.errMessage != "" && err.Error() != tt.errMessage {
					t.Errorf("TriggerParam.Check() error message = %q, want %q", err.Error(), tt.errMessage)
				}
			}
		})
	}
}

func TestTriggerParamCheckValidationOrder(t *testing.T) {
	// When multiple fields are empty, Check() should return the first
	// error in validation order: PipelineId → Types → Name → Params
	p := TriggerParam{}

	err := p.Check()
	if !errors.Is(err, ErrPipelineIdRequired) {
		t.Errorf("expected ErrPipelineIdRequired first, got %v", err)
	}

	p.PipelineId = "pipe-1"
	err = p.Check()
	if !errors.Is(err, ErrTriggerTypeRequired) {
		t.Errorf("expected ErrTriggerTypeRequired second, got %v", err)
	}

	p.Types = "webhook"
	err = p.Check()
	if !errors.Is(err, ErrTriggerNameRequired) {
		t.Errorf("expected ErrTriggerNameRequired third, got %v", err)
	}

	p.Name = "trigger"
	err = p.Check()
	if !errors.Is(err, ErrTriggerParamsRequired) {
		t.Errorf("expected ErrTriggerParamsRequired fourth, got %v", err)
	}

	p.Params = "{}"
	if err := p.Check(); err != nil {
		t.Errorf("expected nil error when all fields set, got %v", err)
	}
}

func TestTriggerSentinelErrorsAreDistinct(t *testing.T) {
	// Verify that sentinel errors are distinct values
	sentinels := []error{
		ErrPipelineIdRequired,
		ErrTriggerTypeRequired,
		ErrTriggerNameRequired,
		ErrTriggerParamsRequired,
	}
	for i := 0; i < len(sentinels); i++ {
		for j := i + 1; j < len(sentinels); j++ {
			if errors.Is(sentinels[i], sentinels[j]) {
				t.Errorf("sentinel %d and %d should be distinct: %v vs %v", i, j, sentinels[i], sentinels[j])
			}
		}
	}
}

func TestTriggerSentinelErrorsAreStable(t *testing.T) {
	// Verify that the same sentinel error is returned on multiple calls
	p1 := TriggerParam{}
	p2 := TriggerParam{}
	err1 := p1.Check()
	err2 := p2.Check()
	// Sentinel errors are never wrapped, so errors.Is checks identity.
	if !errors.Is(err1, err2) {
		t.Errorf("sentinel errors should be the same value: %p vs %p", err1, err2)
	}
}
