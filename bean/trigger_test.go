package bean

import (
	"errors"
	"testing"
)

func TestTriggerParamCheck_SentinelErrors(t *testing.T) {
	tests := []struct {
		name    string
		param   *TriggerParam
		wantErr error
	}{
		{
			name:    "valid trigger",
			param:   &TriggerParam{PipelineId: "p1", Types: "push", Name: "test", Params: "{}"},
			wantErr: nil,
		},
		{
			name:    "missing pipeline ID",
			param:   &TriggerParam{Types: "push", Name: "test", Params: "{}"},
			wantErr: ErrPipelineIDRequired,
		},
		{
			name:    "missing type",
			param:   &TriggerParam{PipelineId: "p1", Name: "test", Params: "{}"},
			wantErr: ErrTriggerTypeRequired,
		},
		{
			name:    "missing name",
			param:   &TriggerParam{PipelineId: "p1", Types: "push", Params: "{}"},
			wantErr: ErrTriggerNameRequired,
		},
		{
			name:    "missing params",
			param:   &TriggerParam{PipelineId: "p1", Types: "push", Name: "test"},
			wantErr: ErrTriggerParamsRequired,
		},
		{
			name:    "all empty - first error wins",
			param:   &TriggerParam{},
			wantErr: ErrPipelineIDRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.Check()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Check() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Errorf("Check() error = nil, want %v", tt.wantErr)
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Check() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTriggerParamSentinelErrors(t *testing.T) {
	// Verify sentinel errors are distinct
	sentinels := []error{
		ErrPipelineIDRequired,
		ErrTriggerTypeRequired,
		ErrTriggerNameRequired,
		ErrTriggerParamsRequired,
	}

	for i, err1 := range sentinels {
		for j, err2 := range sentinels {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("sentinel errors should be distinct: %v and %v", err1, err2)
			}
		}
	}
}

func TestTriggerParamCheckWithWhitespace(t *testing.T) {
	// Empty strings should fail validation, whitespace-only strings currently pass
	// This test documents current behavior
	param := &TriggerParam{
		PipelineId: "   ",
		Types:      "   ",
		Name:       "   ",
		Params:     "   ",
	}
	err := param.Check()
	if err != nil {
		t.Logf("Note: whitespace-only strings pass validation (err=%v)", err)
	}
}
