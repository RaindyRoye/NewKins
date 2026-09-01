package bean

import (
	"errors"
	"testing"
)

func TestTriggerParamCheck(t *testing.T) {
	tests := []struct {
		name    string
		param   TriggerParam
		wantErr bool
		errMsg  string
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
			name: "empty pipeline ID",
			param: TriggerParam{
				PipelineId: "",
				Types:      "webhook",
				Name:       "my-trigger",
				Params:     `{"hookType":"github"}`,
			},
			wantErr: true,
			errMsg:  "pipeline ID is required",
		},
		{
			name: "empty types",
			param: TriggerParam{
				PipelineId: "pipe-1",
				Types:      "",
				Name:       "my-trigger",
				Params:     `{"hookType":"github"}`,
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
				Params:     `{"hookType":"github"}`,
			},
			wantErr: true,
			errMsg:  "trigger name is required",
		},
		{
			name: "empty params",
			param: TriggerParam{
				PipelineId: "pipe-1",
				Types:      "webhook",
				Name:       "my-trigger",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.Check()
			if (err != nil) != tt.wantErr {
				t.Errorf("TriggerParam.Check() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("TriggerParam.Check() error message = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestPipelineCheck(t *testing.T) {
	tests := []struct {
		name     string
		pipeline *Pipeline
		wantErr  error
	}{
		{
			name:     "empty pipeline",
			pipeline: &Pipeline{},
			wantErr:  ErrStagesEmpty,
		},
		{
			name: "nil stages",
			pipeline: &Pipeline{
				Stages: nil,
			},
			wantErr: ErrStagesEmpty,
		},
		{
			name: "empty stages slice",
			pipeline: &Pipeline{
				Stages: []*Stage{},
			},
			wantErr: ErrStagesEmpty,
		},
		{
			name: "stage with empty name",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name:  "",
						Steps: []*Step{{Step: "plugin", Name: "step1"}},
					},
				},
			},
			wantErr: ErrStageNameEmpty,
		},
		{
			name: "stage with empty steps",
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
			name: "stage with nil steps",
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
			name: "duplicate stage names",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name:  "build",
						Steps: []*Step{{Step: "plugin", Name: "step1"}},
					},
					{
						Name:  "build",
						Steps: []*Step{{Step: "plugin2", Name: "step2"}},
					},
				},
			},
			wantErr: ErrDuplicateStage,
		},
		{
			name: "step with empty plugin",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "", Name: "step1"},
						},
					},
				},
			},
			wantErr: ErrStepPluginEmpty,
		},
		{
			name: "step with whitespace-only plugin",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "   ", Name: "step1"},
						},
					},
				},
			},
			wantErr: ErrStepPluginEmpty,
		},
		{
			name: "step with empty name",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "plugin", Name: ""},
						},
					},
				},
			},
			wantErr: ErrStepNameEmpty,
		},
		{
			name: "duplicate step names within same stage",
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
			wantErr: ErrDuplicateStep,
		},
		{
			name: "same step name in different stages is OK",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name:  "build",
						Steps: []*Step{{Step: "plugin1", Name: "step1"}},
					},
					{
						Name:  "deploy",
						Steps: []*Step{{Step: "plugin2", Name: "step1"}},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "valid pipeline with multiple stages",
			pipeline: &Pipeline{
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
					t.Errorf("Pipeline.Check() error = %v, wantErr nil", err)
				}
				return
			}
			if err == nil {
				t.Errorf("Pipeline.Check() error = nil, wantErr %v", tt.wantErr)
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Pipeline.Check() error = %v, want error matching %v (errors.Is returned false)", err, tt.wantErr)
			}
		})
	}
}

func TestPipelineConvertCmd(t *testing.T) {
	tests := []struct {
		name     string
		pipeline *Pipeline
		validate func(t *testing.T, p *Pipeline)
	}{
		{
			name: "string command stays as string",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "shell", Name: "run", Commands: "echo hello"},
						},
					},
				},
			},
			validate: func(t *testing.T, p *Pipeline) {
				cmd := p.Stages[0].Steps[0].Commands
				if s, ok := cmd.(string); !ok || s != "echo hello" {
					t.Errorf("expected string 'echo hello', got %T(%v)", cmd, cmd)
				}
			},
		},
		{
			name: "slice of interfaces converted to string slice",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "shell", Name: "run", Commands: []any{"echo hello", "echo world"}},
						},
					},
				},
			},
			validate: func(t *testing.T, p *Pipeline) {
				cmd := p.Stages[0].Steps[0].Commands
				sl, ok := cmd.([]string)
				if !ok {
					t.Fatalf("expected []string, got %T(%v)", cmd, cmd)
				}
				if len(sl) != 2 || sl[0] != "echo hello" || sl[1] != "echo world" {
					t.Errorf("unexpected result: %v", sl)
				}
			},
		},
		{
			name: "nil command stays nil",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "shell", Name: "run", Commands: nil},
						},
					},
				},
			},
			validate: func(t *testing.T, p *Pipeline) {
				cmd := p.Stages[0].Steps[0].Commands
				if cmd != nil {
					t.Errorf("expected nil, got %T(%v)", cmd, cmd)
				}
			},
		},
		{
			name: "other type converted to string via fmt",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "shell", Name: "run", Commands: 12345},
						},
					},
				},
			},
			validate: func(t *testing.T, p *Pipeline) {
				cmd := p.Stages[0].Steps[0].Commands
				if s, ok := cmd.(string); !ok || s != "12345" {
					t.Errorf("expected string '12345', got %T(%v)", cmd, cmd)
				}
			},
		},
		{
			name: "multiple stages and steps",
			pipeline: &Pipeline{
				Stages: []*Stage{
					{
						Name: "build",
						Steps: []*Step{
							{Step: "shell", Name: "s1", Commands: "cmd1"},
							{Step: "shell", Name: "s2", Commands: []any{"cmd2a", "cmd2b"}},
						},
					},
					{
						Name: "deploy",
						Steps: []*Step{
							{Step: "deploy", Name: "d1", Commands: nil},
						},
					},
				},
			},
			validate: func(t *testing.T, p *Pipeline) {
				// First step: string
				if s, ok := p.Stages[0].Steps[0].Commands.(string); !ok || s != "cmd1" {
					t.Errorf("stage[0].step[0]: expected 'cmd1', got %v", p.Stages[0].Steps[0].Commands)
				}
				// Second step: []string
				if sl, ok := p.Stages[0].Steps[1].Commands.([]string); !ok || len(sl) != 2 {
					t.Errorf("stage[0].step[1]: expected []string with 2 items, got %T", p.Stages[0].Steps[1].Commands)
				}
				// Third step: nil
				if p.Stages[1].Steps[0].Commands != nil {
					t.Errorf("stage[1].step[0]: expected nil, got %v", p.Stages[1].Steps[0].Commands)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pipeline.ConvertCmd()
			tt.validate(t, tt.pipeline)
		})
	}
}

func TestPipelineToJson(t *testing.T) {
	p := &Pipeline{
		Version: "1.0",
		Stages: []*Stage{
			{
				Name: "build",
				Steps: []*Step{
					{Step: "shell", Name: "run", Commands: []any{"echo hello"}},
				},
			},
		},
	}
	bts, err := p.ToJson()
	if err != nil {
		t.Fatalf("Pipeline.ToJson() error = %v", err)
	}
	if len(bts) == 0 {
		t.Error("Pipeline.ToJson() returned empty bytes")
	}
	// Verify that ConvertCmd was called (Commands should be []string now)
	cmd := p.Stages[0].Steps[0].Commands
	if _, ok := cmd.([]string); !ok {
		t.Errorf("expected Commands to be []string after ToJson, got %T", cmd)
	}
}
