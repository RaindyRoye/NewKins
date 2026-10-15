package engine

import (
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
)

// --- BuildTask.check() deep branch coverage ---

func TestCheck_StageBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "wrong-build-id",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell@ssh", BuildId: "build-1", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when stage BuildId mismatches")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestCheck_StageNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-2",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-2",
					Name:    "",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell@ssh", BuildId: "build-2", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when stage name is empty")
	}
	if task.build.Error != "build Stage name is empty" {
		t.Errorf("expected error %q, got %q", "build Stage name is empty", task.build.Error)
	}
}

func TestCheck_StageWithEmptySteps(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-3",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-3",
					Name:    "build",
					Steps:   []*runtime.Step{},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when stage has no steps")
	}
	if task.build.Error != "build Stages is empty" {
		t.Errorf("expected error %q, got %q", "build Stages is empty", task.build.Error)
	}
}

func TestCheck_DuplicateStageNames(t *testing.T) {
	stage := func(name string) *runtime.Stage {
		return &runtime.Stage{
			Id:      "stage-" + name,
			BuildId: "build-4",
			Name:    name,
			Steps: []*runtime.Step{
				{Id: "step-" + name, Name: "compile", Step: "shell@ssh", BuildId: "build-4", StageId: "stage-" + name},
			},
		}
	}
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-4",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				stage("build"),
				stage("build"),
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for duplicate stage names")
	}
}

func TestCheck_StepBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-5",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-5",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell@ssh", BuildId: "wrong-build-id", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when step BuildId mismatches")
	}
}

func TestCheck_StepStageIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-6",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-6",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell@ssh", BuildId: "build-6", StageId: "wrong-stage-id"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when step StageId mismatches")
	}
}

func TestCheck_StepPluginEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-7",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-7",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "", BuildId: "build-7", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when step plugin is empty")
	}
	if task.build.Error != "build Step Plugin is empty" {
		t.Errorf("expected error %q, got %q", "build Step Plugin is empty", task.build.Error)
	}
}

func TestCheck_StepNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-8",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-8",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "", Step: "shell@ssh", BuildId: "build-8", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when step name is empty")
	}
	if task.build.Error != "build Step name is empty" {
		t.Errorf("expected error %q, got %q", "build Step name is empty", task.build.Error)
	}
}

func TestCheck_DuplicateJobNames(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-9",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-9",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell@ssh", BuildId: "build-9", StageId: "stage-1", Commands: "echo hello"},
						{Id: "step-2", Name: "compile", Step: "shell@ssh", BuildId: "build-9", StageId: "stage-1", Commands: "echo world"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for duplicate job names")
	}
}

func TestCheck_StepPluginTrimmed(t *testing.T) {
	// Step with whitespace-only plugin should become empty after trim
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-trim",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-trim",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "   ", BuildId: "build-trim", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when step plugin is whitespace only")
	}
}
