package engine

import (
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
)

// --- check(): nil repo ---

func TestCheck_NilRepo(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:     "build-1",
			Repo:   nil,
			Stages: []*runtime.Stage{},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when Repo is nil")
	}
	// status() sets Status (not Event) to the first arg, Error to the second
	if task.build.Status != common.BuildEventCheckParam {
		t.Errorf("expected status %q, got %q", common.BuildEventCheckParam, task.build.Status)
	}
	if task.build.Error != "repo param err" {
		t.Errorf("expected error 'repo param err', got %q", task.build.Error)
	}
}

// --- check(): empty stages ---

func TestCheck_EmptyStages(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:     "build-2",
			Repo:   &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when Stages is empty")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
	if task.build.Error != "build Stages is empty" {
		t.Errorf("expected error 'build Stages is empty', got %q", task.build.Error)
	}
}

func TestCheck_NilStages(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:     "build-nil-stages",
			Repo:   &runtime.Repository{CloneURL: ""},
			Stages: nil,
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when Stages is nil")
	}
}

// --- check(): stage BuildId mismatch ---

func TestCheck_StageBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-3",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					Name:    "build",
					BuildId: "wrong-build-id",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "test", Step: "shell"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when stage BuildId mismatches")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

// --- check(): stage name empty ---

func TestCheck_StageNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-4",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					Name:    "",
					BuildId: "build-4",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "test", Step: "shell"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when stage name is empty")
	}
	if task.build.Error != "build Stage name is empty" {
		t.Errorf("expected error 'build Stage name is empty', got %q", task.build.Error)
	}
}

// --- check(): stage steps empty ---

func TestCheck_StageStepsEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-5",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					Name:    "build",
					BuildId: "build-5",
					Steps:   []*runtime.Step{},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when stage has no steps")
	}
	if task.build.Error != "build Stages is empty" {
		t.Errorf("expected error 'build Stages is empty', got %q", task.build.Error)
	}
}

// --- check(): duplicate stage names ---

func TestCheck_DuplicateStageNames(t *testing.T) {
	stage := func(id, name string) *runtime.Stage {
		return &runtime.Stage{
			Id:      id,
			Name:    name,
			BuildId: "build-6",
			Steps: []*runtime.Step{
				{
					Id:      "step-" + id,
					Name:    "step-" + id,
					Step:    "shell",
					BuildId: "build-6",
					StageId: id,
				},
			},
		}
	}
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-6",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				stage("s1", "deploy"),
				stage("s2", "deploy"),
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when stages have duplicate names")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

// --- check(): step BuildId mismatch ---

func TestCheck_StepBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-7",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					Name:    "build",
					BuildId: "build-7",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							Name:    "test",
							Step:    "shell",
							BuildId: "wrong-build-id",
							StageId: "stage-1",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when step BuildId mismatches")
	}
}

// --- check(): step StageId mismatch ---

func TestCheck_StepStageIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-8",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					Name:    "build",
					BuildId: "build-8",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							Name:    "test",
							Step:    "shell",
							BuildId: "build-8",
							StageId: "wrong-stage-id",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when step StageId mismatches")
	}
}

// --- check(): step plugin (Step field) empty ---

func TestCheck_StepPluginEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-9",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					Name:    "build",
					BuildId: "build-9",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							Name:    "test",
							Step:    "  ", // whitespace only, should be trimmed to empty
							BuildId: "build-9",
							StageId: "stage-1",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when step plugin is empty")
	}
	if task.build.Error != "build Step Plugin is empty" {
		t.Errorf("expected error 'build Step Plugin is empty', got %q", task.build.Error)
	}
}

// --- check(): step name empty ---

func TestCheck_StepNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-10",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					Name:    "build",
					BuildId: "build-10",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							Name:    "",
							Step:    "shell",
							BuildId: "build-10",
							StageId: "stage-1",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when step name is empty")
	}
	if task.build.Error != "build Step name is empty" {
		t.Errorf("expected error 'build Step name is empty', got %q", task.build.Error)
	}
}

// --- check(): duplicate step names within a stage ---

func TestCheck_DuplicateStepNames(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-11",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					Name:    "build",
					BuildId: "build-11",
					Steps: []*runtime.Step{
						{
							Id:      "step-1",
							Name:    "compile",
							Step:    "shell",
							BuildId: "build-11",
							StageId: "stage-1",
						},
						{
							Id:      "step-2",
							Name:    "compile",
							Step:    "shell",
							BuildId: "build-11",
							StageId: "stage-1",
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when steps have duplicate names")
	}
}

// --- check(): repo with existing directory (isClone = false) ---

func TestCheck_RepoCloneURLIsLocalDir(t *testing.T) {
	// Use /tmp as a known existing directory
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-local-dir",
			Repo: &runtime.Repository{CloneURL: "/tmp"},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					Name:    "build",
					BuildId: "build-local-dir",
					Steps: []*runtime.Step{
						{
							Id:       "step-1",
							Name:     "test",
							Step:     "shell",
							BuildId:  "build-local-dir",
							StageId:  "stage-1",
							Commands: []any{"echo hi"},
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	// check() should set isClone = false and repoPath = CloneURL
	_ = task.check()
	if task.isClone {
		t.Error("expected isClone to be false when CloneURL is an existing directory")
	}
	if task.repoPaths != "/tmp" {
		t.Errorf("expected repoPaths to be '/tmp', got %q", task.repoPaths)
	}
}

// --- check(): isClone = true for non-existent CloneURL ---

func TestCheck_RepoCloneURLNonExistent(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-clone",
			Repo: &runtime.Repository{CloneURL: "https://example.com/repo.git"},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					Name:    "build",
					BuildId: "build-clone",
					Steps: []*runtime.Step{
						{
							Id:       "step-1",
							Name:     "test",
							Step:     "shell",
							BuildId:  "build-clone",
							StageId:  "stage-1",
							Commands: []any{"echo hi"},
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	_ = task.check()
	if !task.isClone {
		t.Error("expected isClone to be true when CloneURL is not an existing directory")
	}
}

// --- check(): step name trimming ---

func TestCheck_StepNameTrimmed(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-trim",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					Name:    "build",
					BuildId: "build-trim",
					Steps: []*runtime.Step{
						{
							Id:       "step-1",
							Name:     "test",
							Step:     "  shell  ",
							BuildId:  "build-trim",
							StageId:  "stage-1",
							Commands: []any{"echo hi"},
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	_ = task.check()
	// Step field should have been trimmed
	if task.build.Stages[0].Steps[0].Step != "shell" {
		t.Errorf("expected step to be trimmed to 'shell', got %q", task.build.Stages[0].Steps[0].Step)
	}
}
