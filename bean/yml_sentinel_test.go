package bean

import (
	"errors"
	"testing"
)

// TestPipelineCheckSentinelErrors verifies that Pipeline.Check() returns
// sentinel errors that can be checked with errors.Is().
func TestPipelineCheckSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		pipeline *Pipeline
		wantErr  error
	}{
		{
			name:     "empty stages returns ErrStagesEmpty",
			pipeline: &Pipeline{},
			wantErr:  ErrStagesEmpty,
		},
		{
			name: "stage with empty name returns ErrStageNameEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "", Steps: []*Step{{Step: "plugin", Name: "s1"}}},
				},
			},
			wantErr: ErrStageNameEmpty,
		},
		{
			name: "stage with no steps returns ErrStepsEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: nil},
				},
			},
			wantErr: ErrStepsEmpty,
		},
		{
			name: "duplicate stage names returns ErrDuplicateStage",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "p1", Name: "s1"}}},
					{Name: "build", Steps: []*Step{{Step: "p2", Name: "s2"}}},
				},
			},
			wantErr: ErrDuplicateStage,
		},
		{
			name: "step with empty plugin returns ErrStepPluginEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "", Name: "s1"}}},
				},
			},
			wantErr: ErrStepPluginEmpty,
		},
		{
			name: "step with whitespace plugin returns ErrStepPluginEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "   ", Name: "s1"}}},
				},
			},
			wantErr: ErrStepPluginEmpty,
		},
		{
			name: "step with empty name returns ErrStepNameEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "plugin", Name: ""}}},
				},
			},
			wantErr: ErrStepNameEmpty,
		},
		{
			name: "duplicate step names returns ErrDuplicateStep",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "p1", Name: "s1"},
							{Step: "p2", Name: "s1"},
						},
					},
				},
			},
			wantErr: ErrDuplicateStep,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pipeline.Check()
			if err == nil {
				t.Fatalf("expected error wrapping %v, got nil", tt.wantErr)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error to wrap %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestPipelineCheckSuccess verifies that a valid pipeline passes all checks.
func TestPipelineCheckSuccess(t *testing.T) {
	p := &Pipeline{
		Stages: []*Stage{
			{
				Name: "build",
				Steps: []*Step{
					{Step: "clone", Name: "clone-repo"},
					{Step: "build", Name: "compile"},
				},
			},
			{
				Name: "test",
				Steps: []*Step{
					{Step: "test", Name: "unit-test"},
				},
			},
			{
				Name: "deploy",
				Steps: []*Step{
					{Step: "deploy", Name: "deploy-prod"},
				},
			},
		},
	}
	if err := p.Check(); err != nil {
		t.Fatalf("expected valid pipeline to pass Check(), got %v", err)
	}
}

// TestPipelineSentinelErrorsAreDistinct verifies that all sentinel errors
// are distinct values (no accidental aliasing).
func TestPipelineSentinelErrorsAreDistinct(t *testing.T) {
	sentinels := []error{
		ErrStagesEmpty,
		ErrStageNameEmpty,
		ErrStepsEmpty,
		ErrDuplicateStage,
		ErrStepPluginEmpty,
		ErrStepNameEmpty,
		ErrDuplicateStep,
	}
	for i := 0; i < len(sentinels); i++ {
		for j := i + 1; j < len(sentinels); j++ {
			if errors.Is(sentinels[i], sentinels[j]) {
				t.Errorf("sentinel %d and %d should be distinct: %v vs %v", i, j, sentinels[i], sentinels[j])
			}
		}
	}
}
