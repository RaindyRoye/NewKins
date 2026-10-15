package engine

import (
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
)

// newTestBuildTask creates a minimal BuildTask for check() validation tests.
// The returned task has empty stages and jobs maps ready for population.
func newTestBuildTask(buildID string) *BuildTask {
	return &BuildTask{
		build: &runtime.Build{
			Id:     buildID,
			Repo:   &runtime.Repository{},
			Stages: []*runtime.Stage{},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
}

// makeStage creates a valid runtime.Stage for testing.
func makeStage(buildID, stageID, name string, steps []*runtime.Step) *runtime.Stage {
	return &runtime.Stage{
		Id:      stageID,
		BuildId: buildID,
		Name:    name,
		Steps:   steps,
	}
}

// makeStep creates a valid runtime.Step for testing.
func makeStep(buildID, stageID, stepID, name, stepPlugin string) *runtime.Step {
	return &runtime.Step{
		Id:       stepID,
		BuildId:  buildID,
		StageId:  stageID,
		Name:     name,
		Step:     stepPlugin,
		Commands: "echo test",
	}
}

func TestCheck_StageBuildIDMismatch(t *testing.T) {
	task := newTestBuildTask("build-1")
	task.build.Stages = []*runtime.Stage{
		makeStage("build-WRONG", "stage-1", "build", []*runtime.Step{
			makeStep("build-1", "stage-1", "step-1", "compile", "shell"),
		}),
	}
	if task.check() {
		t.Fatal("check() should return false when stage BuildId mismatches")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
	expected := "Stage Build id err:build-WRONG/build-1"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestCheck_StageNameEmpty(t *testing.T) {
	task := newTestBuildTask("build-1")
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "", []*runtime.Step{
			makeStep("build-1", "stage-1", "step-1", "compile", "shell"),
		}),
	}
	if task.check() {
		t.Fatal("check() should return false when stage name is empty")
	}
	if task.build.Error != "build Stage name is empty" {
		t.Errorf("expected error 'build Stage name is empty', got %q", task.build.Error)
	}
}

func TestCheck_StageStepsEmpty(t *testing.T) {
	task := newTestBuildTask("build-1")
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{}),
	}
	if task.check() {
		t.Fatal("check() should return false when stage has no steps")
	}
	if task.build.Error != "build Stages is empty" {
		t.Errorf("expected error 'build Stages is empty', got %q", task.build.Error)
	}
}

func TestCheck_DuplicateStageName(t *testing.T) {
	task := newTestBuildTask("build-1")
	// Use nil Commands to avoid genRunjob DB operations
	step := makeStep("build-1", "stage-1", "step-1", "compile", "shell")
	step.Commands = nil
	step2 := makeStep("build-1", "stage-2", "step-2", "test", "shell")
	step2.Commands = nil
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{step}),
		makeStage("build-1", "stage-2", "build", []*runtime.Step{step2}), // duplicate name
	}
	if task.check() {
		t.Fatal("check() should return false when stage names are duplicated")
	}
	expected := "build Stages.build is repeat"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestCheck_StepBuildIDMismatch(t *testing.T) {
	task := newTestBuildTask("build-1")
	badStep := makeStep("build-WRONG", "stage-1", "step-1", "compile", "shell")
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{badStep}),
	}
	if task.check() {
		t.Fatal("check() should return false when step BuildId mismatches")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestCheck_StepStageIDMismatch(t *testing.T) {
	task := newTestBuildTask("build-1")
	badStep := makeStep("build-1", "stage-WRONG", "step-1", "compile", "shell")
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{badStep}),
	}
	if task.check() {
		t.Fatal("check() should return false when step StageId mismatches")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestCheck_StepPluginEmpty(t *testing.T) {
	task := newTestBuildTask("build-1")
	badStep := makeStep("build-1", "stage-1", "step-1", "compile", "")
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{badStep}),
	}
	if task.check() {
		t.Fatal("check() should return false when step plugin is empty")
	}
	if task.build.Error != "build Step Plugin is empty" {
		t.Errorf("expected error 'build Step Plugin is empty', got %q", task.build.Error)
	}
}

func TestCheck_StepPluginWhitespaceOnly(t *testing.T) {
	task := newTestBuildTask("build-1")
	badStep := makeStep("build-1", "stage-1", "step-1", "compile", "   \t  ")
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{badStep}),
	}
	if task.check() {
		t.Fatal("check() should return false when step plugin is whitespace-only")
	}
	if task.build.Error != "build Step Plugin is empty" {
		t.Errorf("expected error 'build Step Plugin is empty', got %q", task.build.Error)
	}
}

func TestCheck_StepNameEmpty(t *testing.T) {
	task := newTestBuildTask("build-1")
	badStep := makeStep("build-1", "stage-1", "step-1", "", "shell")
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{badStep}),
	}
	if task.check() {
		t.Fatal("check() should return false when step name is empty")
	}
	if task.build.Error != "build Step name is empty" {
		t.Errorf("expected error 'build Step name is empty', got %q", task.build.Error)
	}
}

func TestCheck_DuplicateStepName(t *testing.T) {
	task := newTestBuildTask("build-1")
	// Use nil Commands to avoid genRunjob DB operations
	step1 := makeStep("build-1", "stage-1", "step-1", "compile", "shell")
	step1.Commands = nil
	step2 := makeStep("build-1", "stage-1", "step-2", "compile", "docker") // same name
	step2.Commands = nil
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{step1, step2}),
	}
	if task.check() {
		t.Fatal("check() should return false when step names are duplicated within a stage")
	}
	expected := "build Job.compile is repeat"
	if task.build.Error != expected {
		t.Errorf("expected error %q, got %q", expected, task.build.Error)
	}
}

func TestCheck_StepPluginTrimmed(t *testing.T) {
	task := newTestBuildTask("build-1")
	// Step plugin with leading/trailing whitespace — should be trimmed and accepted
	step := makeStep("build-1", "stage-1", "step-1", "compile", "  shell  ")
	// We need valid Commands that genRunjob can handle without DB
	step.Commands = nil // nil commands won't cause errors in genRunjob
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{step}),
	}
	// check() should pass through validation — genRunjob may return nil for nil commands
	// We're testing that the trim happens correctly
	// After check, step.Step should be "shell" (trimmed)
	task.check()
	if step.Step != "shell" {
		t.Errorf("expected step plugin to be trimmed to 'shell', got %q", step.Step)
	}
}

func TestCheck_MultipleStagesAndSteps_Valid(t *testing.T) {
	task := newTestBuildTask("build-1")
	step1 := makeStep("build-1", "stage-1", "step-1", "compile", "shell")
	step1.Commands = nil // avoid DB operations in genRunjob
	step2 := makeStep("build-1", "stage-2", "step-2", "test", "shell")
	step2.Commands = nil
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{step1}),
		makeStage("build-1", "stage-2", "test", []*runtime.Step{step2}),
	}
	// With nil commands, genRunjob should succeed (no commands to process)
	result := task.check()
	if !result {
		t.Fatalf("check() should return true for valid multi-stage build, got error: %q", task.build.Error)
	}
	// Verify stages were populated
	if len(task.stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(task.stages))
	}
	// Verify jobs were populated
	if len(task.jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(task.jobs))
	}
}

func TestCheck_RepoWithCloneURL(t *testing.T) {
	task := newTestBuildTask("build-1")
	task.build.Repo = &runtime.Repository{
		CloneURL: "/nonexistent/path/that/doesnt/exist",
	}
	step := makeStep("build-1", "stage-1", "step-1", "compile", "shell")
	step.Commands = nil
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{step}),
	}
	// CloneURL points to a non-existent path, so isClone should remain true
	result := task.check()
	if !result {
		t.Fatalf("check() should return true, got error: %q", task.build.Error)
	}
	if !task.isClone {
		t.Error("isClone should be true for non-existent CloneURL path")
	}
}

func TestCheck_RepoEmptyCloneURL(t *testing.T) {
	task := newTestBuildTask("build-1")
	task.build.Repo = &runtime.Repository{
		CloneURL: "",
	}
	step := makeStep("build-1", "stage-1", "step-1", "compile", "shell")
	step.Commands = nil
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{step}),
	}
	result := task.check()
	if !result {
		t.Fatalf("check() should return true with empty CloneURL, got error: %q", task.build.Error)
	}
	// isClone should be true (default) when CloneURL is empty
	if !task.isClone {
		t.Error("isClone should be true when CloneURL is empty")
	}
}

func TestCheck_ValidationOrder_RepoFirst(t *testing.T) {
	// Verify that nil Repo check comes before Stages check
	task := &BuildTask{
		build: &runtime.Build{
			Id:     "build-1",
			Repo:   nil,
			Stages: []*runtime.Stage{}, // also invalid
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	task.check()
	// Should fail on nil repo, not on empty stages
	if task.build.Error != "repo param err" {
		t.Errorf("expected 'repo param err' (checked first), got %q", task.build.Error)
	}
}

func TestCheck_ValidationOrder_StagesBeforeStepValidation(t *testing.T) {
	// Verify that empty stages is checked before individual stage validation
	task := newTestBuildTask("build-1")
	task.build.Stages = []*runtime.Stage{}
	task.check()
	if task.build.Error != "build Stages is empty" {
		t.Errorf("expected 'build Stages is empty', got %q", task.build.Error)
	}
}

func TestCheck_MultipleStepsInStage(t *testing.T) {
	task := newTestBuildTask("build-1")
	step1 := makeStep("build-1", "stage-1", "step-1", "compile", "shell")
	step1.Commands = nil
	step2 := makeStep("build-1", "stage-1", "step-2", "lint", "shell")
	step2.Commands = nil
	step3 := makeStep("build-1", "stage-1", "step-3", "test", "shell")
	step3.Commands = nil
	task.build.Stages = []*runtime.Stage{
		makeStage("build-1", "stage-1", "build", []*runtime.Step{step1, step2, step3}),
	}
	result := task.check()
	if !result {
		t.Fatalf("check() should return true for multiple valid steps, got error: %q", task.build.Error)
	}
	if len(task.jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(task.jobs))
	}
}
