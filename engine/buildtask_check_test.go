package engine

import (
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

// --- BuildTask.check() comprehensive tests ---

func TestBuildTaskCheck_StageBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "wrong-build-id", // mismatch
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
		t.Fatal("expected check() to return false when stage BuildId mismatches")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestBuildTaskCheck_StageNameEmpty(t *testing.T) {
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
					Name:    "", // empty name
					Steps:   []*runtime.Step{},
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
		t.Errorf("expected error 'build Stage name is empty', got %q", task.build.Error)
	}
}

func TestBuildTaskCheck_StageStepsEmpty(t *testing.T) {
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
					Steps:   []*runtime.Step{}, // empty steps
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when stage steps is empty")
	}
}

func TestBuildTaskCheck_StageNameDuplicate(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-4",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-4",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", StageId: "stage-1", BuildId: "build-4", Step: "shell", Name: "test1"},
					},
				},
				{
					Id:      "stage-2",
					BuildId: "build-4",
					Name:    "build", // duplicate name
					Steps: []*runtime.Step{
						{Id: "step-2", StageId: "stage-2", BuildId: "build-4", Step: "shell", Name: "test2"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when stage name is duplicated")
	}
}

func TestBuildTaskCheck_StepBuildIdMismatch(t *testing.T) {
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
						{
							Id:      "step-1",
							StageId: "stage-1",
							BuildId: "wrong-build-id", // mismatch
							Step:    "shell",
							Name:    "test",
						},
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

func TestBuildTaskCheck_StepStageIdMismatch(t *testing.T) {
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
						{
							Id:      "step-1",
							StageId: "wrong-stage-id", // mismatch
							BuildId: "build-6",
							Step:    "shell",
							Name:    "test",
						},
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

func TestBuildTaskCheck_StepPluginEmpty(t *testing.T) {
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
						{
							Id:      "step-1",
							StageId: "stage-1",
							BuildId: "build-7",
							Step:    "", // empty plugin
							Name:    "test",
						},
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
}

func TestBuildTaskCheck_StepNameEmpty(t *testing.T) {
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
						{
							Id:      "step-1",
							StageId: "stage-1",
							BuildId: "build-8",
							Step:    "shell",
							Name:    "", // empty name
						},
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
}

func TestBuildTaskCheck_StepNameDuplicate(t *testing.T) {
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
						{
							Id:      "step-1",
							StageId: "stage-1",
							BuildId: "build-9",
							Step:    "shell",
							Name:    "test",
						},
						{
							Id:      "step-2",
							StageId: "stage-1",
							BuildId: "build-9",
							Step:    "shell",
							Name:    "test", // duplicate name
						},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when step name is duplicated")
	}
}

// --- gencmds map[any]any tests ---

func TestGencmds_MapAnyInterface(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[any]any{
			"key1": "echo from map[any]any",
			"key2": []any{"echo nested1", "echo nested2"},
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have at least some commands from the map values
	if len(runjb.Commands) == 0 {
		t.Fatal("expected some commands from map[any]any values, got 0")
	}
}

func TestGencmds_MapAnyInterfaceNestedArray(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[any]any{
			"key": []any{"cmd1", "cmd2", "cmd3"},
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 3 {
		t.Fatalf("expected 3 commands from nested array in map[any]any, got %d", len(runjb.Commands))
	}
}

// --- BuildTask.getRepo() tests ---

func TestBuildTaskGetRepo_NonClone(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-getrepo-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
		},
		isClone: false,
	}
	err := task.getRepo()
	if err != nil {
		t.Fatalf("getRepo() should return nil for non-clone, got: %v", err)
	}
}

func TestBuildTaskGetRepo_CloneWithEmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-getrepo-2",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
		},
		isClone:   true,
		repoPaths: tmpDir + "/repo",
		repoPath:  "", // empty, so no git clone
	}
	err := task.getRepo()
	if err != nil {
		t.Fatalf("getRepo() should succeed with empty repoPath, got: %v", err)
	}
}
