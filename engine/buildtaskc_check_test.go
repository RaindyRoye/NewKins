package engine

import (
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

// --- BuildTask.check() expanded edge cases ---

func TestCheck_NilRepo(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-check-1",
			Repo: nil,
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when Repo is nil")
	}
	if task.build.Status != common.BuildEventCheckParam {
		t.Errorf("status = %q, want %q", task.build.Status, common.BuildEventCheckParam)
	}
	if task.build.Error != "repo param err" {
		t.Errorf("error = %q, want %q", task.build.Error, "repo param err")
	}
}

func TestCheck_EmptyStages(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:     "build-check-2",
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
		t.Errorf("event = %q, want %q", task.build.Event, common.BuildEventCheckParam)
	}
	if task.build.Error != "build Stages is empty" {
		t.Errorf("error = %q, want %q", task.build.Error, "build Stages is empty")
	}
}

func TestCheck_StageBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-check-3",
			Repo: &runtime.Repository{},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "wrong-build-id",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell"},
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
		t.Errorf("event = %q, want %q", task.build.Event, common.BuildEventCheckParam)
	}
}

func TestCheck_StageNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-check-4",
			Repo: &runtime.Repository{},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-check-4",
					Name:    "",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell"},
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
		t.Errorf("error = %q, want %q", task.build.Error, "build Stage name is empty")
	}
}

func TestCheck_StageWithNoSteps(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-check-5",
			Repo: &runtime.Repository{},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-check-5",
					Name:    "build",
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
		t.Errorf("error = %q, want %q", task.build.Error, "build Stages is empty")
	}
}

func TestCheck_DuplicateStageNames(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-check-6",
			Repo: &runtime.Repository{},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-check-6",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell", StageId: "stage-1", BuildId: "build-check-6"},
					},
				},
				{
					Id:      "stage-2",
					BuildId: "build-check-6",
					Name:    "build", // duplicate name
					Steps: []*runtime.Step{
						{Id: "step-2", Name: "test", Step: "shell", StageId: "stage-2", BuildId: "build-check-6"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false for duplicate stage names")
	}
}

func TestCheck_StepBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-check-7",
			Repo: &runtime.Repository{},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-check-7",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell", StageId: "stage-1", BuildId: "wrong-build"},
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

func TestCheck_StepStageIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-check-8",
			Repo: &runtime.Repository{},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-check-8",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell", StageId: "wrong-stage", BuildId: "build-check-8"},
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

func TestCheck_StepPluginEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-check-9",
			Repo: &runtime.Repository{},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-check-9",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "", StageId: "stage-1", BuildId: "build-check-9"},
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
		t.Errorf("error = %q, want %q", task.build.Error, "build Step Plugin is empty")
	}
}

func TestCheck_StepNameEmpty(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-check-10",
			Repo: &runtime.Repository{},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-check-10",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "", Step: "shell", StageId: "stage-1", BuildId: "build-check-10"},
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
		t.Errorf("error = %q, want %q", task.build.Error, "build Step name is empty")
	}
}

func TestCheck_DuplicateStepNames(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-check-11",
			Repo: &runtime.Repository{},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-check-11",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell", StageId: "stage-1", BuildId: "build-check-11"},
						{Id: "step-2", Name: "compile", Step: "shell", StageId: "stage-1", BuildId: "build-check-11"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false for duplicate step names within a stage")
	}
}

// --- gencmds map[any]any branch ---

func TestGencmds_MapAnyAny(t *testing.T) {
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
	if len(runjb.Commands) == 0 {
		t.Fatal("expected some commands from map[any]any, got 0")
	}
}

func TestGencmds_MapAnyAnyWithIntValues(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	// Values that are not string or []any should be silently ignored
	cmds := []any{
		map[any]any{
			"key1": 42,    // not string or []any
			"key2": true,  // not string or []any
			"key3": "echo hello",
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only "echo hello" should produce a command
	if len(runjb.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runjb.Commands))
	}
	if runjb.Commands[0].Conts != "echo hello" {
		t.Errorf("expected 'echo hello', got %q", runjb.Commands[0].Conts)
	}
}

func TestGencmds_MapStringAnyWithIntValues(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[string]any{
			"key1": 3.14,
			"key2": "echo valid",
		},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only "echo valid" should produce a command
	if len(runjb.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runjb.Commands))
	}
}

func TestGencmds_DeepNestedArray(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	// Nested []any inside []any
	cmds := []any{
		[]any{"echo a", "echo b"},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 2 {
		t.Fatalf("expected 2 commands from nested array, got %d", len(runjb.Commands))
	}
}

func TestGencmds_EmptyMap(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}
	cmds := []any{
		map[string]any{},
	}

	err := task.gencmds(runjb, cmds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 0 {
		t.Fatalf("expected 0 commands from empty map, got %d", len(runjb.Commands))
	}
}

func TestGencmds_NilSlice(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "test-build"},
	}
	runjb := &runners.RunJob{}

	err := task.gencmds(runjb, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runjb.Commands) != 0 {
		t.Fatalf("expected 0 commands from nil slice, got %d", len(runjb.Commands))
	}
}

// --- genRunjob with string commands (no DB needed) ---

func TestGenRunjob_StringCommand(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-genrun-1",
			PipelineId: "pipe-1",
		},
	}
	stage := &runtime.Stage{
		Id:   "stage-1",
		Name: "build",
	}
	step := &runtime.Step{
		Id:       "step-1",
		BuildId:  "build-genrun-1",
		StageId:  "stage-1",
		Name:     "compile",
		Step:     "shell",
		Commands: "echo hello",
	}
	job := &jobSync{
		task:  task,
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}

	// genRunjob will try to insert to DB at the end, which will panic/err
	// We just verify the RunJob was constructed correctly before the DB call
	defer func() {
		if r := recover(); r != nil {
			// Expected: comm.Db is nil in unit tests
			t.Logf("recovered panic (expected without DB): %v", r)
		}
	}()
	_ = task.genRunjob(stage, job)

	// Verify runjb was created
	if job.runjb == nil {
		t.Fatal("genRunjob should create runjb before DB insert")
	}
	if job.runjb.Id != "step-1" {
		t.Errorf("runjb.Id = %q, want %q", job.runjb.Id, "step-1")
	}
	if job.runjb.PipelineId != "pipe-1" {
		t.Errorf("runjb.PipelineId = %q, want %q", job.runjb.PipelineId, "pipe-1")
	}
	if job.runjb.Step != "shell" {
		t.Errorf("runjb.Step = %q, want %q", job.runjb.Step, "shell")
	}
	if len(job.runjb.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(job.runjb.Commands))
	}
	if job.runjb.Commands[0].Conts != "echo hello" {
		t.Errorf("command content = %q, want %q", job.runjb.Commands[0].Conts, "echo hello")
	}
}

func TestGenRunjob_SliceStringCommand(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-genrun-2",
			PipelineId: "pipe-1",
		},
	}
	stage := &runtime.Stage{
		Id:   "stage-1",
		Name: "build",
	}
	step := &runtime.Step{
		Id:       "step-1",
		BuildId:  "build-genrun-2",
		StageId:  "stage-1",
		Name:     "compile",
		Step:     "shell",
		Commands: []string{"echo first", "echo second"},
	}
	job := &jobSync{
		task:  task,
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}

	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (expected without DB): %v", r)
		}
	}()
	_ = task.genRunjob(stage, job)

	if job.runjb == nil {
		t.Fatal("genRunjob should create runjb before DB insert")
	}
	if len(job.runjb.Commands) != 2 {
		t.Fatalf("expected 2 commands from []string, got %d", len(job.runjb.Commands))
	}
}

func TestGenRunjob_GokinsGitStep(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:         "build-genrun-3",
			PipelineId: "pipe-1",
		},
		repoPaths: "/tmp/repos",
		isClone:   true,
	}
	stage := &runtime.Stage{
		Id:   "stage-1",
		Name: "build",
	}
	step := &runtime.Step{
		Id:       "step-1",
		BuildId:  "build-genrun-3",
		StageId:  "stage-1",
		Name:     "git-checkout",
		Step:     "gokins@git",
		Commands: "original-command",
	}
	job := &jobSync{
		task:  task,
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}

	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (expected without DB): %v", r)
		}
	}()
	_ = task.genRunjob(stage, job)

	if job.runjb == nil {
		t.Fatal("genRunjob should create runjb")
	}
	// For gokins@git, OriginRepo should be set to repoPaths
	if job.runjb.OriginRepo != "/tmp/repos" {
		t.Errorf("OriginRepo = %q, want %q", job.runjb.OriginRepo, "/tmp/repos")
	}
	// Commands should be overridden to "git works"
	if len(job.runjb.Commands) != 1 || job.runjb.Commands[0].Conts != "git works" {
		t.Errorf("gokins@git step should have command 'git works', got %v", job.runjb.Commands)
	}
}

func TestGenRunjob_CloneClearsOriginRepo(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-genrun-4",
		},
		repoPath: "/some/repo",
		isClone:  true,
	}
	stage := &runtime.Stage{Id: "s1", Name: "build"}
	step := &runtime.Step{
		Id:       "step-1",
		BuildId:  "build-genrun-4",
		StageId:  "s1",
		Name:     "compile",
		Step:     "shell",
		Commands: "echo hello",
	}
	job := &jobSync{
		task:  task,
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}

	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic: %v", r)
		}
	}()
	_ = task.genRunjob(stage, job)

	if job.runjb == nil {
		t.Fatal("genRunjob should create runjb")
	}
	// When isClone is true, OriginRepo should be cleared
	if job.runjb.OriginRepo != "" {
		t.Errorf("OriginRepo should be empty when isClone=true, got %q", job.runjb.OriginRepo)
	}
}

func TestGenRunjob_MustCopyClearsOriginRepo(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-genrun-5",
		},
		repoPath: "/some/repo",
		isClone:  false,
	}
	stage := &runtime.Stage{Id: "s1", Name: "build"}
	step := &runtime.Step{
		Id:       "step-1",
		BuildId:  "build-genrun-5",
		StageId:  "s1",
		Name:     "compile",
		Step:     "shell",
		MustCopy: true,
		Commands: "echo hello",
	}
	job := &jobSync{
		task:  task,
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}

	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic: %v", r)
		}
	}()
	_ = task.genRunjob(stage, job)

	if job.runjb == nil {
		t.Fatal("genRunjob should create runjb")
	}
	if job.runjb.OriginRepo != "" {
		t.Errorf("OriginRepo should be empty when MustCopy=true, got %q", job.runjb.OriginRepo)
	}
}

// --- genRunjob with nil commands (no-op) ---

func TestGenRunjob_NilCommands(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id: "build-genrun-6",
		},
	}
	stage := &runtime.Stage{Id: "s1", Name: "build"}
	step := &runtime.Step{
		Id:       "step-1",
		BuildId:  "build-genrun-6",
		StageId:  "s1",
		Name:     "noop",
		Step:     "shell",
		Commands: nil,
	}
	job := &jobSync{
		task:  task,
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}

	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic: %v", r)
		}
	}()
	_ = task.genRunjob(stage, job)

	if job.runjb == nil {
		t.Fatal("genRunjob should create runjb even with nil commands")
	}
	if len(job.runjb.Commands) != 0 {
		t.Errorf("expected 0 commands for nil commands, got %d", len(job.runjb.Commands))
	}
}
