package bean

import (
	"errors"
	"testing"
)

// TestSentinelErrors verifies that Pipeline.Check() returns sentinel errors
// that can be checked with errors.Is().
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name       string
		pipeline   *Pipeline
		wantErr    error
		wantErrMsg string
	}{
		{
			name:       "ErrStagesEmpty on nil stages",
			pipeline:   &Pipeline{Stages: nil},
			wantErr:    ErrStagesEmpty,
			wantErrMsg: "stages is empty",
		},
		{
			name:       "ErrStagesEmpty on empty slice",
			pipeline:   &Pipeline{Stages: []*Stage{}},
			wantErr:    ErrStagesEmpty,
			wantErrMsg: "stages is empty",
		},
		{
			name: "ErrStageNameEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "", Steps: []*Step{{Step: "plugin", Name: "step1"}}},
				},
			},
			wantErr:    ErrStageNameEmpty,
			wantErrMsg: "stage name is empty",
		},
		{
			name: "ErrStepsEmpty on nil steps",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: nil},
				},
			},
			wantErr:    ErrStepsEmpty,
			wantErrMsg: "steps is empty",
		},
		{
			name: "ErrStepsEmpty on empty slice",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{}},
				},
			},
			wantErr:    ErrStepsEmpty,
			wantErrMsg: "steps is empty",
		},
		{
			name: "ErrDuplicateStage",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "plugin", Name: "step1"}}},
					{Name: "build", Steps: []*Step{{Step: "plugin", Name: "step2"}}},
				},
			},
			wantErr:    ErrDuplicateStage,
			wantErrMsg: "duplicate stage name: build",
		},
		{
			name: "ErrStepPluginEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "", Name: "step1"}}},
				},
			},
			wantErr:    ErrStepPluginEmpty,
			wantErrMsg: "step plugin is empty",
		},
		{
			name: "ErrStepPluginEmpty on whitespace",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "   ", Name: "step1"}}},
				},
			},
			wantErr:    ErrStepPluginEmpty,
			wantErrMsg: "step plugin is empty",
		},
		{
			name: "ErrStepNameEmpty",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "plugin", Name: ""}}},
				},
			},
			wantErr:    ErrStepNameEmpty,
			wantErrMsg: "step name is empty",
		},
		{
			name: "ErrDuplicateStep",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "plugin1", Name: "step1"},
							{Step: "plugin2", Name: "step1"},
						},
					},
				},
			},
			wantErr:    ErrDuplicateStep,
			wantErrMsg: "duplicate step name: step1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pipeline.Check()
			if err == nil {
				t.Fatalf("expected error %v, got nil", tt.wantErr)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error to wrap %v, got %v", tt.wantErr, err)
			}
			if err.Error() != tt.wantErrMsg {
				t.Errorf("expected error message %q, got %q", tt.wantErrMsg, err.Error())
			}
		})
	}
}

// TestSentinelErrorsValid verifies that valid pipelines return nil.
func TestSentinelErrorsValid(t *testing.T) {
	tests := []struct {
		name     string
		pipeline *Pipeline
	}{
		{
			name: "single stage single step",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "plugin", Name: "step1"}}},
				},
			},
		},
		{
			name: "multiple stages",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "plugin1", Name: "step1"}}},
					{Name: "test", Steps: []*Step{{Step: "plugin2", Name: "step2"}}},
				},
			},
		},
		{
			name: "multiple steps in stage",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "plugin1", Name: "step1"},
							{Step: "plugin2", Name: "step2"},
						},
					},
				},
			},
		},
		{
			name: "same step name in different stages",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{Name: "build", Steps: []*Step{{Step: "plugin1", Name: "step1"}}},
					{Name: "test", Steps: []*Step{{Step: "plugin2", Name: "step1"}}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pipeline.Check()
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

// TestTriggerParamSentinelErrors verifies that TriggerParam.Check() returns
// sentinel errors that can be checked with errors.Is().
func TestTriggerParamSentinelErrors(t *testing.T) {
	tests := []struct {
		name       string
		param      TriggerParam
		wantErrMsg string
	}{
		{
			name: "missing pipelineId",
			param: TriggerParam{
				PipelineId: "",
				Types:      "webhook",
				Name:       "trigger1",
				Params:     "{}",
			},
			wantErrMsg: "trigger field is required: pipelineId",
		},
		{
			name: "missing types",
			param: TriggerParam{
				PipelineId: "pipe1",
				Types:      "",
				Name:       "trigger1",
				Params:     "{}",
			},
			wantErrMsg: "trigger field is required: types",
		},
		{
			name: "missing name",
			param: TriggerParam{
				PipelineId: "pipe1",
				Types:      "webhook",
				Name:       "",
				Params:     "{}",
			},
			wantErrMsg: "trigger field is required: name",
		},
		{
			name: "missing params",
			param: TriggerParam{
				PipelineId: "pipe1",
				Types:      "webhook",
				Name:       "trigger1",
				Params:     "",
			},
			wantErrMsg: "trigger field is required: params",
		},
		{
			name:       "all fields empty",
			param:      TriggerParam{},
			wantErrMsg: "trigger field is required: pipelineId",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.Check()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, ErrTriggerFieldEmpty) {
				t.Errorf("expected error to wrap ErrTriggerFieldEmpty, got %v", err)
			}
			if err.Error() != tt.wantErrMsg {
				t.Errorf("expected error message %q, got %q", tt.wantErrMsg, err.Error())
			}
		})
	}
}

// TestTriggerParamSentinelErrorsValid verifies that valid TriggerParam returns nil.
func TestTriggerParamSentinelErrorsValid(t *testing.T) {
	param := TriggerParam{
		PipelineId: "pipe1",
		Types:      "webhook",
		Name:       "trigger1",
		Params:     "{}",
	}
	err := param.Check()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestSentinelErrorUnwrap verifies that sentinel errors can be unwrapped.
func TestSentinelErrorUnwrap(t *testing.T) {
	pipeline := &Pipeline{
		Stages: []*Stage{
			{Name: "build", Steps: []*Step{{Step: "plugin", Name: "step1"}}},
			{Name: "build", Steps: []*Step{{Step: "plugin", Name: "step2"}}},
		},
	}
	err := pipeline.Check()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify errors.Is works
	if !errors.Is(err, ErrDuplicateStage) {
		t.Error("errors.Is(err, ErrDuplicateStage) returned false")
	}

	// Verify the error message contains the stage name
	expected := "duplicate stage name: build"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}
