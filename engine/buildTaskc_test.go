package engine

import (
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

func TestGencmds_StringCommands(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{"echo hello", "ls -la", "pwd"}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(runjb.Commands))
	}
	if runjb.Commands[0].Conts != "echo hello" {
		t.Errorf("expected 'echo hello', got %q", runjb.Commands[0].Conts)
	}
	if runjb.Commands[1].Conts != "ls -la" {
		t.Errorf("expected 'ls -la', got %q", runjb.Commands[1].Conts)
	}
	if runjb.Commands[2].Conts != "pwd" {
		t.Errorf("expected 'pwd', got %q", runjb.Commands[2].Conts)
	}
}

func TestGencmds_EmptyCommands(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(runjb.Commands))
	}
}

func TestGencmds_NestedArray(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		[]any{"echo a", "echo b"},
		"echo c",
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// nested array items are flattened
	if len(runjb.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(runjb.Commands))
	}
}

func TestGencmds_MapStringInterface(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[string]any{
			"key1": "echo from map",
			"key2": []any{"echo nested1", "echo nested2"},
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have at least some commands from the map values
	if len(runjb.Commands) == 0 {
		t.Fatal("expected some commands from map values, got 0")
	}
}

func TestGencmds_UnknownTypesSkipped(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	// int is not a recognized type, so it should be silently ignored
	cmds := []any{42, true, 3.14}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 0 {
		t.Fatalf("expected 0 commands for unrecognized types, got %d", len(runjb.Commands))
	}
}

func TestAppendcmds(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}

	task.appendcmds(runjb, "echo hello")
	task.appendcmds(runjb, "echo world")

	if len(runjb.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(runjb.Commands))
	}
	if runjb.Commands[0].Conts != "echo hello" {
		t.Errorf("expected 'echo hello', got %q", runjb.Commands[0].Conts)
	}
	if runjb.Commands[1].Conts != "echo world" {
		t.Errorf("expected 'echo world', got %q", runjb.Commands[1].Conts)
	}
	// Each command should have a unique ID
	if runjb.Commands[0].Id == runjb.Commands[1].Id {
		t.Error("expected unique command IDs")
	}
}

func TestGencmds_MixedTypes(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		"echo first",
		[]any{"echo second", "echo third"},
		"echo fourth",
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(runjb.Commands))
	}
}

func TestBuildTaskCheck_NilRepo(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "test-build",
			Repo: nil,
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when Repo is nil")
	}
	// check() calls c.status(BuildEventCheckParam, "repo param err") which sets Status and Error
	if task.build.Status != common.BuildEventCheckParam {
		t.Errorf("expected build status to be %q, got %q", common.BuildEventCheckParam, task.build.Status)
	}
	if task.build.Error != "repo param err" {
		t.Errorf("expected error 'repo param err', got %q", task.build.Error)
	}
}

func TestBuildTaskCheck_EmptyStages(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when Stages is empty")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected build event to be %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
	if task.build.Error != "build Stages is empty" {
		t.Errorf("expected error 'build Stages is empty', got %q", task.build.Error)
	}
}

func TestBuildTaskCheck_StageNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "https://example.com/repo.git",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "test-build",
					Name:    "", // empty
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "s1", Step: "shell", BuildId: "test-build", StageId: "stage-1"},
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
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected build event to be %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestBuildTaskCheck_StageBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "https://example.com/repo.git",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "wrong-build-id", // mismatch
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "s1", Step: "shell", BuildId: "test-build", StageId: "stage-1"},
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
		t.Errorf("expected build event to be %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestBuildTaskCheck_StageNoSteps(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "https://example.com/repo.git",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "test-build",
					Name:    "build",
					Steps:   []*runtime.Step{}, // empty
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
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected build event to be %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestBuildTaskCheck_StepPluginEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "https://example.com/repo.git",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "s1", Step: "", BuildId: "test-build", StageId: "stage-1"}, // empty plugin
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
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected build event to be %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestBuildTaskCheck_StepNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "https://example.com/repo.git",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "", Step: "shell", BuildId: "test-build", StageId: "stage-1"}, // empty name
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
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected build event to be %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestBuildTaskCheck_DuplicateStageName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "https://example.com/repo.git",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "s1", Step: "shell", BuildId: "test-build", StageId: "stage-1"},
					},
				},
				{
					Id:      "stage-2",
					BuildId: "test-build",
					Name:    "build", // duplicate name
					Steps: []*runtime.Step{
						{Id: "step-2", Name: "s2", Step: "shell", BuildId: "test-build", StageId: "stage-2"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when stage names are duplicated")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected build event to be %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestBuildTaskCheck_DuplicateStepName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "https://example.com/repo.git",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell", BuildId: "test-build", StageId: "stage-1"},
						{Id: "step-2", Name: "compile", Step: "shell", BuildId: "test-build", StageId: "stage-1"}, // duplicate
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false when step names are duplicated within a stage")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected build event to be %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestBuildTaskCheck_StepBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "https://example.com/repo.git",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "s1", Step: "shell", BuildId: "wrong-id", StageId: "stage-1"}, // mismatch
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
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected build event to be %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestBuildTaskCheck_StepStageIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "test-build",
			Repo: &runtime.Repository{
				CloneURL: "https://example.com/repo.git",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "test-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "s1", Step: "shell", BuildId: "test-build", StageId: "wrong-stage"}, // mismatch
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
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("expected build event to be %q, got %q", common.BuildEventCheckParam, task.build.Event)
	}
}

func TestBuildTaskGetRepo_NoClone(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	// isClone is false, so getRepo should return nil immediately
	err := task.getRepo()
	if err != nil {
		t.Errorf("getRepo() with isClone=false should return nil, got: %v", err)
	}
}
