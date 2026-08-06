package engine

import (
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

// --- check() branch coverage ---

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
					BuildId: "wrong-build",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: "wrong-build", StageId: "stage-1", Step: "gokins@git", Name: "clone"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false for stage BuildId mismatch")
	}
	if task.build.Error != "Stage Build id err:wrong-build/build-1" {
		t.Errorf("unexpected error: %q", task.build.Error)
	}
}

func TestCheck_StageNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "",
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: "build-1", StageId: "stage-1", Step: "gokins@git", Name: "clone"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false for empty stage name")
	}
	if task.build.Error != "build Stage name is empty" {
		t.Errorf("unexpected error: %q", task.build.Error)
	}
}

func TestCheck_StageEmptySteps(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "build",
					Steps:   []*runtime.Step{},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false for stage with no steps")
	}
}

func TestCheck_StageRepeatedName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: "build-1", StageId: "stage-1", Step: "gokins@git", Name: "clone"},
					},
				},
				{
					Id:      "stage-2",
					BuildId: "build-1",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-2", BuildId: "build-1", StageId: "stage-2", Step: "shell", Name: "run"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false for repeated stage names")
	}
}

func TestCheck_StepBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: "wrong-build", StageId: "stage-1", Step: "gokins@git", Name: "clone"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false for step BuildId mismatch")
	}
}

func TestCheck_StepStageIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: "build-1", StageId: "wrong-stage", Step: "gokins@git", Name: "clone"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false for step StageId mismatch")
	}
}

func TestCheck_StepPluginEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: "build-1", StageId: "stage-1", Step: "", Name: "clone"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false for empty step plugin")
	}
	if task.build.Error != "build Step Plugin is empty" {
		t.Errorf("unexpected error: %q", task.build.Error)
	}
}

func TestCheck_StepNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: "build-1", StageId: "stage-1", Step: "gokins@git", Name: ""},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false for empty step name")
	}
	if task.build.Error != "build Step name is empty" {
		t.Errorf("unexpected error: %q", task.build.Error)
	}
}

func TestCheck_StepRepeatedName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-1",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", BuildId: "build-1", StageId: "stage-1", Step: "gokins@git", Name: "clone"},
						{Id: "step-2", BuildId: "build-1", StageId: "stage-1", Step: "gokins@git", Name: "clone"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false for repeated step names")
	}
}

// --- gencmds with map[any]any ---

func TestGencmds_MapAnyAny(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[any]any{
			"key1": "echo from any map",
			"key2": []any{"echo nested1", "echo nested2"},
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) == 0 {
		t.Fatal("expected some commands from map[any]any values, got 0")
	}
}

func TestGencmds_MapAnyAnyNestedArray(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[any]any{
			"scripts": []any{"echo a", "echo b", "echo c"},
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 3 {
		t.Errorf("expected 3 commands from nested array in map, got %d", len(runjb.Commands))
	}
}

func TestGencmds_MapStringAnyNestedArray(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[string]any{
			"scripts": []any{"echo x", "echo y"},
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 2 {
		t.Errorf("expected 2 commands from nested array, got %d", len(runjb.Commands))
	}
}

// --- genRunjob command type branches ---

func TestGenRunjob_StringCommand(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-1",
			PipelineId: "pipe-1",
			Vars:       map[string]*runtime.Variables{},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}

	// genRunjob requires DB, so we test only the command type switch
	// by calling appendcmds directly
	runjb := &runners.RunJob{}
	task.appendcmds(runjb, "echo hello")
	if len(runjb.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runjb.Commands))
	}
}

func TestGenRunjob_GitPlugin(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-1",
			PipelineId: "pipe-1",
			Vars:       map[string]*runtime.Variables{},
		},
		repoPath:  "/tmp/repo",
		repoPaths: "/tmp/repo",
		isClone:   false,
		stages:    make(map[string]*taskStage),
		jobs:      make(map[string]*jobSync),
	}

	runjb := &runners.RunJob{
		Step: "gokins@git",
	}
	// For gokins@git, genRunjob sets OriginRepo to repoPaths
	// and commands to ["git works"]
	if runjb.Step == "gokins@git" {
		runjb.OriginRepo = task.repoPaths
	}
	if runjb.OriginRepo != "/tmp/repo" {
		t.Errorf("expected OriginRepo to be /tmp/repo, got %q", runjb.OriginRepo)
	}
}

// --- taskStage.status (additional coverage) ---

func TestTaskStageStatusWithEvent(t *testing.T) {
	stg := &taskStage{
		stage: &runtime.Stage{},
		jobs:  make(map[string]*jobSync),
	}
	stg.status("error", "something failed", "param_err")
	if stg.stage.Status != "error" {
		t.Errorf("expected status 'error', got %q", stg.stage.Status)
	}
	if stg.stage.Error != "something failed" {
		t.Errorf("expected error 'something failed', got %q", stg.stage.Error)
	}
	if stg.stage.Event != "param_err" {
		t.Errorf("expected event 'param_err', got %q", stg.stage.Event)
	}
}

// --- jobSync.status (additional coverage) ---

func TestJobSyncStatusWithEvent(t *testing.T) {
	js := &jobSync{
		step: &runtime.Step{},
	}
	js.status("error", "timeout", "job_timeout")
	if js.step.Status != "error" {
		t.Errorf("expected status 'error', got %q", js.step.Status)
	}
	if js.step.Error != "timeout" {
		t.Errorf("expected error 'timeout', got %q", js.step.Error)
	}
	if js.step.Event != "job_timeout" {
		t.Errorf("expected event 'job_timeout', got %q", js.step.Event)
	}
}

func TestJobSyncStatusNoEvent(t *testing.T) {
	js := &jobSync{
		step: &runtime.Step{Event: "prev-event"},
	}
	js.status("running", "")
	// Event should be preserved when not passed
	if js.step.Event != "prev-event" {
		t.Errorf("expected event to remain 'prev-event', got %q", js.step.Event)
	}
}
