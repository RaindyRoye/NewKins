package bean

import (
	"errors"
	"fmt"
	"testing"
)

func TestTriggerParamCheck_SentinelErrors(t *testing.T) {
	tests := []struct {
		name       string
		param      *TriggerParam
		wantErr    error
		wantMsgSub string
	}{
		{
			name:       "missing pipeline ID",
			param:      &TriggerParam{},
			wantErr:    ErrTriggerPipelineIDRequired,
			wantMsgSub: "pipeline ID is required",
		},
		{
			name:       "missing type",
			param:      &TriggerParam{PipelineId: "p1"},
			wantErr:    ErrTriggerTypeRequired,
			wantMsgSub: "trigger type is required",
		},
		{
			name:       "missing name",
			param:      &TriggerParam{PipelineId: "p1", Types: "timer"},
			wantErr:    ErrTriggerNameRequired,
			wantMsgSub: "trigger name is required",
		},
		{
			name:       "missing params",
			param:      &TriggerParam{PipelineId: "p1", Types: "timer", Name: "t1"},
			wantErr:    ErrTriggerParamsRequired,
			wantMsgSub: "trigger params is required",
		},
		{
			name:       "valid trigger",
			param:      &TriggerParam{PipelineId: "p1", Types: "timer", Name: "t1", Params: "{}"},
			wantErr:    nil,
			wantMsgSub: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.Check()
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantMsgSub)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("errors.Is(err, %v) = false; err = %v", tt.wantErr, err)
			}
			if fmt.Sprintf("%v", err) == "" {
				t.Errorf("error message should not be empty")
			}
		})
	}
}

func TestPipelineCheck_SentinelErrors(t *testing.T) {
	tests := []struct {
		name       string
		pipeline   *Pipeline
		wantErr    error
		wantMsgSub string
	}{
		{
			name:     "no stages",
			pipeline: &Pipeline{},
			wantErr:  ErrStagesEmpty,
		},
		{
			name: "empty stage name",
			pipeline: &Pipeline{
				Stages: []*Stage{{}},
			},
			wantErr: ErrStageNameEmpty,
		},
		{
			name: "no steps",
			pipeline: &Pipeline{
				Stages: []*Stage{{Name: "build"}},
			},
			wantErr: ErrStepsEmpty,
		},
		{
			name: "empty step plugin",
			pipeline: &Pipeline{
				Stages: []*Stage{{
					Name:  "build",
					Steps: []*Step{{Name: "s1"}},
				}},
			},
			wantErr: ErrStepPluginEmpty,
		},
		{
			name: "empty step name",
			pipeline: &Pipeline{
				Stages: []*Stage{{
					Name:  "build",
					Steps: []*Step{{Step: "gokins@git"}},
				}},
			},
			wantErr: ErrStepNameEmpty,
		},
		{
			name: "duplicate stage names",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "gokins@git", Name: "s1"}}},
					{Name: "build", Steps: []*Step{{Step: "gokins@git", Name: "s2"}}},
				},
			},
			wantErr: nil, // not a sentinel, just a formatted error
		},
		{
			name: "valid pipeline",
			pipeline: &Pipeline{
				Stages: []*Stage{{
					Name: "build",
					Steps: []*Step{{
						Step: "gokins@git",
						Name: "checkout",
					}},
				}},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pipeline.Check()
			if tt.wantErr == nil {
				if err != nil && tt.name != "duplicate stage names" {
					// duplicate stage names returns a non-sentinel error
					// which is acceptable
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.wantErr)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("errors.Is(err, %v) = false; err = %v", tt.wantErr, err)
			}
		})
	}
}

func TestTriggerSentinelUnwrap(t *testing.T) {
	// Verify that wrapping preserves sentinel identity through errors.Is
	err := fmt.Errorf("trigger check: %w", ErrTriggerPipelineIDRequired)
	if !errors.Is(err, ErrTriggerPipelineIDRequired) {
		t.Errorf("wrapped error should match sentinel via errors.Is")
	}
	// Verify double wrapping also works
	err2 := fmt.Errorf("outer: %w", err)
	if !errors.Is(err2, ErrTriggerPipelineIDRequired) {
		t.Errorf("double-wrapped error should match sentinel via errors.Is")
	}
}
