package bean

import (
	"errors"
	"testing"
)

func TestSentinelErrors_Pipeline_DuplicateStage(t *testing.T) {
	p := &Pipeline{
		Stages: []*Stage{
			{
				Name: "build",
				Steps: []*Step{
					{Step: "shell", Name: "step1"},
				},
			},
			{
				Name: "build", // duplicate
				Steps: []*Step{
					{Step: "shell", Name: "step2"},
				},
			},
		},
	}

	err := p.Check()
	if err == nil {
		t.Fatal("expected error for duplicate stage name")
	}
	if !errors.Is(err, ErrDuplicateStage) {
		t.Errorf("expected error to wrap ErrDuplicateStage, got: %v", err)
	}
}

func TestSentinelErrors_Pipeline_DuplicateStep(t *testing.T) {
	p := &Pipeline{
		Stages: []*Stage{
			{
				Name: "build",
				Steps: []*Step{
					{Step: "shell", Name: "compile"},
					{Step: "shell", Name: "compile"}, // duplicate
				},
			},
		},
	}

	err := p.Check()
	if err == nil {
		t.Fatal("expected error for duplicate step name")
	}
	if !errors.Is(err, ErrDuplicateStep) {
		t.Errorf("expected error to wrap ErrDuplicateStep, got: %v", err)
	}
}
