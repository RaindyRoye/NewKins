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

func TestPipelineController_GetPath(t *testing.T) {
	c := &PipelineController{}
	if got := c.GetPath(); got != "/api/pipeline" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/pipeline")
	}
}

func TestOrgController_GetPath(t *testing.T) {
	c := &OrgController{}
	if got := c.GetPath(); got != "/api/org" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/org")
	}
}

func TestTriggerController_GetPath(t *testing.T) {
	c := &TriggerController{}
	if got := c.GetPath(); got != "/api/trigger" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/trigger")
	}
}

func TestHookController_GetPath(t *testing.T) {
	c := &HookController{}
	if got := c.GetPath(); got != "/trigger" {
		t.Errorf("GetPath() = %q, want %q", got, "/trigger")
	}
}

func TestRuntimeController_GetPath(t *testing.T) {
	c := &RuntimeController{}
	if got := c.GetPath(); got != "/api/runtime" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/runtime")
	}
}

func TestPipelineVersionController_GetPath(t *testing.T) {
	c := &PipelineVersionController{}
	if got := c.GetPath(); got != "/api/pipelineVersion" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/pipelineVersion")
	}
}

func TestArtifactController_GetPath(t *testing.T) {
	c := &ArtifactController{}
	if got := c.GetPath(); got != "/api/art" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/art")
	}
}

func TestYmlController_GetPath(t *testing.T) {
	c := &YmlController{}
	if got := c.GetPath(); got != "/api/yml" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/yml")
	}
}

func TestArtPublicController_GetPath(t *testing.T) {
	c := &ArtPublicController{}
	if got := c.GetPath(); got != "/api/art/pub" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/art/pub")
	}
}

func TestAllControllers_GetPathUnique(t *testing.T) {
	// Verify all controller paths are unique
	type pathController struct {
		name string
		path string
	}
	controllers := []pathController{
		{"ApiController", (&ApiController{}).GetPath()},
		{"InstallController", (&InstallController{}).GetPath()},
		{"UserController", (&UserController{}).GetPath()},
		{"LoginController", (&LoginController{}).GetPath()},
		{"PipelineController", (&PipelineController{}).GetPath()},
		{"OrgController", (&OrgController{}).GetPath()},
		{"TriggerController", (&TriggerController{}).GetPath()},
		{"HookController", (&HookController{}).GetPath()},
		{"RuntimeController", (&RuntimeController{}).GetPath()},
		{"PipelineVersionController", (&PipelineVersionController{}).GetPath()},
		{"ArtifactController", (&ArtifactController{}).GetPath()},
		{"YmlController", (&YmlController{}).GetPath()},
		{"ArtPublicController", (&ArtPublicController{}).GetPath()},
	}

	seen := make(map[string]string)
	for _, c := range controllers {
		if other, exists := seen[c.path]; exists {
			t.Errorf("duplicate path %q: %s and %s", c.path, c.name, other)
		}
		seen[c.path] = c.name
	}
}
