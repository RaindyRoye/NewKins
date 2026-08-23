package engine

import (
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
)

// helper to create a minimal valid build for check() tests.
func newValidBuild() *runtime.Build {
	const id = "b1"
	return &runtime.Build{
		Id: id,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      "stage-1",
				BuildId: id,
				Name:    "build",
				Steps: []*runtime.Step{
					{
						Id:       "step-1",
						BuildId:  id,
						StageId:  "stage-1",
						Step:     "shell",
						Name:     "run",
						Commands: "echo hello",
					},
				},
			},
		},
	}
}

func newCheckTask(bd *runtime.Build) *BuildTask {
	return &BuildTask{
		build:  bd,
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
}

func TestCheck_StageBuildIdMismatch(t *testing.T) {
	bd := newValidBuild()
	bd.Stages[0].BuildId = "wrong-id"
	task := newCheckTask(bd)
	if task.check() {
		t.Fatal("expected false for mismatched stage BuildId")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestCheck_StageNameEmpty(t *testing.T) {
	bd := newValidBuild()
	bd.Stages[0].Name = ""
	task := newCheckTask(bd)
	if task.check() {
		t.Fatal("expected false for empty stage name")
	}
	if task.build.Error != "build Stage name is empty" {
		t.Errorf("expected error 'build Stage name is empty', got %q", task.build.Error)
	}
}

func TestCheck_StageStepsEmpty(t *testing.T) {
	bd := newValidBuild()
	bd.Stages[0].Steps = []*runtime.Step{}
	task := newCheckTask(bd)
	if task.check() {
		t.Fatal("expected false for empty steps")
	}
	if task.build.Error != "build Stages is empty" {
		t.Errorf("expected error 'build Stages is empty', got %q", task.build.Error)
	}
}

func TestCheck_DuplicateStageNames(t *testing.T) {
	bd := newValidBuild()
	// Add a second stage with the same name
	bd.Stages = append(bd.Stages, &runtime.Stage{
		Id:      "stage-2",
		BuildId: "b1",
		Name:    "build",
		Steps: []*runtime.Step{
			{
				Id:       "step-2",
				BuildId:  "b1",
				StageId:  "stage-2",
				Step:     "shell",
				Name:     "run2",
				Commands: "echo hello2",
			},
		},
	})
	task := newCheckTask(bd)
	if task.check() {
		t.Fatal("expected false for duplicate stage names")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestCheck_StepBuildIdMismatch(t *testing.T) {
	bd := newValidBuild()
	bd.Stages[0].Steps[0].BuildId = "wrong-id"
	task := newCheckTask(bd)
	if task.check() {
		t.Fatal("expected false for mismatched step BuildId")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestCheck_StepStageIdMismatch(t *testing.T) {
	bd := newValidBuild()
	bd.Stages[0].Steps[0].StageId = "wrong-stage-id"
	task := newCheckTask(bd)
	if task.check() {
		t.Fatal("expected false for mismatched step StageId")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestCheck_StepPluginEmpty(t *testing.T) {
	bd := newValidBuild()
	bd.Stages[0].Steps[0].Step = ""
	task := newCheckTask(bd)
	if task.check() {
		t.Fatal("expected false for empty step plugin")
	}
	if task.build.Error != "build Step Plugin is empty" {
		t.Errorf("expected error 'build Step Plugin is empty', got %q", task.build.Error)
	}
}

func TestCheck_StepNameEmpty(t *testing.T) {
	bd := newValidBuild()
	bd.Stages[0].Steps[0].Name = ""
	task := newCheckTask(bd)
	if task.check() {
		t.Fatal("expected false for empty step name")
	}
	if task.build.Error != "build Step name is empty" {
		t.Errorf("expected error 'build Step name is empty', got %q", task.build.Error)
	}
}

func TestCheck_DuplicateStepNames(t *testing.T) {
	bd := newValidBuild()
	bd.Stages[0].Steps = append(bd.Stages[0].Steps, &runtime.Step{
		Id:       "step-2",
		BuildId:  "b1",
		StageId:  "stage-1",
		Step:     "shell",
		Name:     "run",
		Commands: "echo world",
	})
	task := newCheckTask(bd)
	if task.check() {
		t.Fatal("expected false for duplicate step names")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

// Note: Tests that exercise genRunjob (which requires comm.Db) cannot be tested
// without a database. We test only the validation logic paths that return before
// reaching genRunjob.

func TestCheck_IsCloneDefaultsTrue(t *testing.T) {
	bd := &runtime.Build{
		Id:   "b1",
		Repo: &runtime.Repository{CloneURL: ""},
		Stages: []*runtime.Stage{
			{
				Id: "s1", BuildId: "b1", Name: "stage",
				Steps: []*runtime.Step{},
			},
		},
	}
	task := newCheckTask(bd)
	// check() will fail at "Steps is empty" but before that it sets isClone=true
	_ = task.check()
	if !task.isClone {
		t.Error("expected isClone to be true by default")
	}
}
