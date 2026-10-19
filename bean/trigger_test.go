package bean

import "testing"

// TestTriggerParam_Check validates the field-presence rules enforced by TriggerParam.Check.
// Each table entry isolates a single validation path so regressions in error messages
// or precedence are caught immediately.
func TestTriggerParam_Check(t *testing.T) {
	tests := []struct {
		name    string
		param   TriggerParam
		wantErr bool
		errMsg  string
	}{
		{
			name: "all fields valid",
			param: TriggerParam{
				PipelineId: "pipe-1",
				Types:      "webhook",
				Name:       "deploy",
				Params:     `{"ref":"main"}`,
			},
			wantErr: false,
		},
		{
			name: "empty pipeline id",
			param: TriggerParam{
				PipelineId: "",
				Types:      "webhook",
				Name:       "deploy",
				Params:     `{"ref":"main"}`,
			},
			wantErr: true,
			errMsg:  "pipeline ID is required",
		},
		{
			name: "empty trigger type",
			param: TriggerParam{
				PipelineId: "pipe-1",
				Types:      "",
				Name:       "deploy",
				Params:     `{"ref":"main"}`,
			},
			wantErr: true,
			errMsg:  "trigger type is required",
		},
		{
			name: "empty name",
			param: TriggerParam{
				PipelineId: "pipe-1",
				Types:      "webhook",
				Name:       "",
				Params:     `{"ref":"main"}`,
			},
			wantErr: true,
			errMsg:  "trigger name is required",
		},
		{
			name: "empty params",
			param: TriggerParam{
				PipelineId: "pipe-1",
				Types:      "webhook",
				Name:       "deploy",
				Params:     "",
			},
			wantErr: true,
			errMsg:  "trigger params is required",
		},
		{
			name:    "all fields empty",
			param:   TriggerParam{},
			wantErr: true,
			errMsg:  "pipeline ID is required",
		},
		{
			name: "optional fields may be empty",
			param: TriggerParam{
				PipelineId: "pipe-1",
				Types:      "webhook",
				Name:       "deploy",
				Params:     `{"ref":"main"}`,
				Id:         "",
				Desc:       "",
				Enabled:    false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.Check()
			if (err != nil) != tt.wantErr {
				t.Errorf("TriggerParam.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("TriggerParam.Check() error = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}
