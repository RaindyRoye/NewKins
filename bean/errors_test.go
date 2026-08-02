package bean

import (
	"errors"
	"testing"
)

func TestPipeline_Check_SentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		pipeline *Pipeline
		wantErr  error
	}{
		{
			name:     "empty stages",
			pipeline: &Pipeline{Stages: nil},
			wantErr:  ErrStagesEmpty,
		},
		{
			name: "stage name empty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "", Steps: []*Step{{Step: "git", Name: "clone"}}},
				},
			},
			wantErr: ErrStageNameEmpty,
		},
		{
			name: "steps empty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: nil},
				},
			},
			wantErr: ErrStepsEmpty,
		},
		{
			name: "step plugin empty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "  ", Name: "clone"}}},
				},
			},
			wantErr: ErrStepPluginEmpty,
		},
		{
			name: "step name empty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "git", Name: ""}}},
				},
			},
			wantErr: ErrStepNameEmpty,
		},
		{
			name: "duplicate stage name",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "git", Name: "clone"}}},
					{Name: "build", Steps: []*Step{{Step: "git", Name: "clone2"}}},
				},
			},
			wantErr: ErrDuplicateStage,
		},
		{
			name: "duplicate step name",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{
						{Step: "git", Name: "clone"},
						{Step: "git", Name: "clone"},
					}},
				},
			},
			wantErr: ErrDuplicateStep,
		},
		{
			name: "valid pipeline",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "git", Name: "clone"}}},
					{Name: "test", Steps: []*Step{{Step: "go", Name: "test"}}},
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pipeline.Check()
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
				t.Errorf("Check() error = %v, want error wrapping %v", err, tt.wantErr)
			}
		})
	}
}

func TestTriggerParam_Check_SentinelErrors(t *testing.T) {
	tests := []struct {
		name    string
		trigger *TriggerParam
		wantErr error
	}{
		{
			name:    "pipeline id required",
			trigger: &TriggerParam{PipelineId: ""},
			wantErr: ErrPipelineIdRequired,
		},
		{
			name:    "trigger type required",
			trigger: &TriggerParam{PipelineId: "pipe-1", Types: ""},
			wantErr: ErrTriggerTypeRequired,
		},
		{
			name:    "trigger name required",
			trigger: &TriggerParam{PipelineId: "pipe-1", Types: "webhook", Name: ""},
			wantErr: ErrTriggerNameRequired,
		},
		{
			name:    "trigger params required",
			trigger: &TriggerParam{PipelineId: "pipe-1", Types: "webhook", Name: "test", Params: ""},
			wantErr: ErrTriggerParamsRequired,
		},
		{
			name:    "valid trigger",
			trigger: &TriggerParam{PipelineId: "pipe-1", Types: "webhook", Name: "test", Params: "{}"},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trigger.Check()
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
