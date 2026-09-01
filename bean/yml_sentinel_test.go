package bean

import (
	"errors"
	"testing"
)

// TestPipelineCheckSentinelErrors verifies that Pipeline.Check() returns
// sentinel errors that can be matched with errors.Is().
func TestPipelineCheckSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		pipeline *Pipeline
		wantErr  error
	}{
		{
			name:     "empty stages returns ErrStagesEmpty",
			pipeline: &Pipeline{Stages: []*Stage{}},
			wantErr:  ErrStagesEmpty,
		},
		{
			name: "nil stages returns ErrStagesEmpty",
			pipeline: &Pipeline{
				Stages: nil,
			},
			wantErr: ErrStagesEmpty,
		},
		{
			name: "empty stage name returns ErrStageNameEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name:  "",
						Steps: []*Step{{Step: "shell", Name: "test"}},
					},
				},
			},
			wantErr: ErrStageNameEmpty,
		},
		{
			name: "empty steps returns ErrStepsEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name:  "build",
						Steps: []*Step{},
					},
				},
			},
			wantErr: ErrStepsEmpty,
		},
		{
			name: "nil steps returns ErrStepsEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name:  "build",
						Steps: nil,
					},
				},
			},
			wantErr: ErrStepsEmpty,
		},
		{
			name: "duplicate stage names returns ErrDuplicateStage",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name:  "build",
						Steps: []*Step{{Step: "shell", Name: "test1"}},
					},
					{
						Name:  "build",
						Steps: []*Step{{Step: "shell", Name: "test2"}},
					},
				},
			},
			wantErr: ErrDuplicateStage,
		},
		{
			name: "empty step plugin returns ErrStepPluginEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "", Name: "test"},
						},
					},
				},
			},
			wantErr: ErrStepPluginEmpty,
		},
		{
			name: "whitespace-only step plugin returns ErrStepPluginEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "   ", Name: "test"},
						},
					},
				},
			},
			wantErr: ErrStepPluginEmpty,
		},
		{
			name: "empty step name returns ErrStepNameEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "shell", Name: ""},
						},
					},
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
							{Step: "shell", Name: "test"},
							{Step: "shell", Name: "test"},
						},
					},
				},
			},
			wantErr: ErrDuplicateStep,
		},
		{
			name: "valid pipeline returns nil",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "shell", Name: "compile"},
							{Step: "shell", Name: "test"},
						},
					},
					{
						Name: "deploy",
						Steps: []*Step{
							{Step: "shell", Name: "deploy"},
						},
					},
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
					t.Errorf("Check() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Errorf("Check() = nil, want error matching %v", tt.wantErr)
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Check() error = %v, want error matching %v (errors.Is returned false)", err, tt.wantErr)
			}
		})
	}
}

// TestPipelineCheckErrorMessageContent verifies that wrapped errors contain
// contextual information (e.g., duplicate stage/step names).
func TestPipelineCheckErrorMessageContent(t *testing.T) {
	t.Run("duplicate stage error contains stage name", func(t *testing.T) {
		pipeline := &Pipeline{
			Stages: []*Stage{
				{
					Name:  "build",
					Steps: []*Step{{Step: "shell", Name: "test1"}},
				},
				{
					Name:  "build",
					Steps: []*Step{{Step: "shell", Name: "test2"}},
				},
			},
		}
		err := pipeline.Check()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrDuplicateStage) {
			t.Errorf("error does not wrap ErrDuplicateStage: %v", err)
		}
		// Verify error message contains the duplicate name
		errStr := err.Error()
		if !contains(errStr, "build") {
			t.Errorf("error message %q does not contain stage name 'build'", errStr)
		}
	})

	t.Run("duplicate step error contains step name", func(t *testing.T) {
		pipeline := &Pipeline{
			Stages: []*Stage{
				{
					Name: "build",
					Steps: []*Step{
						{Step: "shell", Name: "compile"},
						{Step: "shell", Name: "compile"},
					},
				},
			},
		}
		err := pipeline.Check()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrDuplicateStep) {
			t.Errorf("error does not wrap ErrDuplicateStep: %v", err)
		}
		// Verify error message contains the duplicate name
		errStr := err.Error()
		if !contains(errStr, "compile") {
			t.Errorf("error message %q does not contain step name 'compile'", errStr)
		}
	})
}

// contains checks if s contains substr (helper to avoid importing strings).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
