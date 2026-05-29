package route

import (
	"testing"

	"github.com/gokins/core/runtime"
)

func TestPrebuild_NilBuild(t *testing.T) {
	err := prebuild(nil)
	if err == nil {
		t.Fatal("expected error for nil build, got nil")
	}
	if err.Error() != "build is empty" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestPrebuild_EmptyStages(t *testing.T) {
	b := &runtime.Build{}
	err := prebuild(b)
	if err == nil {
		t.Fatal("expected error for empty stages, got nil")
	}
	if err.Error() != "stages is empty" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestPrebuild_NilStages(t *testing.T) {
	b := &runtime.Build{
		Stages: nil,
	}
	err := prebuild(b)
	if err == nil {
		t.Fatal("expected error for nil stages, got nil")
	}
}

func TestPrebuild_ValidBuild(t *testing.T) {
	b := &runtime.Build{
		Stages: []*runtime.Stage{
			{
				Steps: []*runtime.Step{
					{},
					{},
				},
			},
			{
				Steps: []*runtime.Step{
					{},
				},
			},
		},
	}

	err := prebuild(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check build ID is set
	if b.Id == "" {
		t.Error("build Id should be set")
	}
	if b.PipelineId == "" {
		t.Error("build PipelineId should be set")
	}

	// Check repo is set
	if b.Repo == nil {
		t.Fatal("build Repo should be set")
	}
	if b.Repo.CloneURL == "" {
		t.Error("build Repo.CloneURL should be set")
	}

	// Check stages
	if len(b.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(b.Stages))
	}
	for i, stage := range b.Stages {
		if stage.Id == "" {
			t.Errorf("stage[%d] Id should be set", i)
		}
		if stage.BuildId != b.Id {
			t.Errorf("stage[%d] BuildId should be %q, got %q", i, b.Id, stage.BuildId)
		}
	}

	// Check steps
	if len(b.Stages[0].Steps) != 2 {
		t.Fatalf("expected 2 steps in stage[0], got %d", len(b.Stages[0].Steps))
	}
	if len(b.Stages[1].Steps) != 1 {
		t.Fatalf("expected 1 step in stage[1], got %d", len(b.Stages[1].Steps))
	}

	allStepIds := map[string]bool{}
	for si, stage := range b.Stages {
		for sti, step := range stage.Steps {
			if step.Id == "" {
				t.Errorf("stage[%d].step[%d] Id should be set", si, sti)
			}
			if step.StageId != stage.Id {
				t.Errorf("stage[%d].step[%d] StageId should be %q, got %q", si, sti, stage.Id, step.StageId)
			}
			if step.BuildId != b.Id {
				t.Errorf("stage[%d].step[%d] BuildId should be %q, got %q", si, sti, b.Id, step.BuildId)
			}
			if allStepIds[step.Id] {
				t.Errorf("stage[%d].step[%d] duplicate Id: %s", si, sti, step.Id)
			}
			allStepIds[step.Id] = true
		}
	}
}

func TestPrebuild_UniqueIds(t *testing.T) {
	// Run prebuild twice and verify IDs are different
	b1 := &runtime.Build{
		Stages: []*runtime.Stage{
			{Steps: []*runtime.Step{{}}},
		},
	}
	b2 := &runtime.Build{
		Stages: []*runtime.Stage{
			{Steps: []*runtime.Step{{}}},
		},
	}

	if err := prebuild(b1); err != nil {
		t.Fatalf("prebuild b1: %v", err)
	}
	if err := prebuild(b2); err != nil {
		t.Fatalf("prebuild b2: %v", err)
	}

	if b1.Id == b2.Id {
		t.Error("two builds should have different Ids")
	}
	if b1.PipelineId == b2.PipelineId {
		t.Error("two builds should have different PipelineIds")
	}
	if b1.Stages[0].Id == b2.Stages[0].Id {
		t.Error("two stages should have different Ids")
	}
	if b1.Stages[0].Steps[0].Id == b2.Stages[0].Steps[0].Id {
		t.Error("two steps should have different Ids")
	}
}

func TestApiController_GetPath(t *testing.T) {
	c := &ApiController{}
	if got := c.GetPath(); got != "/api" {
		t.Errorf("GetPath() = %q, want %q", got, "/api")
	}
}

func TestInstallController_GetPath(t *testing.T) {
	c := &InstallController{}
	if got := c.GetPath(); got != "/api/install" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/install")
	}
}

func TestUserController_GetPath(t *testing.T) {
	c := &UserController{}
	if got := c.GetPath(); got != "/api/user" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/user")
	}
}

func TestLoginController_GetPath(t *testing.T) {
	c := &LoginController{}
	if got := c.GetPath(); got != "/api/lg" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/lg")
	}
}
