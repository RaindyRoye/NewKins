package engine

import (
	"context"
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupCheckTestDB creates an in-memory SQLite DB for check() tests that
// exercise genRunjob (which inserts TCmdLine records).
func setupCheckTestDB(t *testing.T) {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	oldDb := comm.Db
	comm.Db = eng
	t.Cleanup(func() {
		comm.Db = oldDb
		_ = eng.Close()
	})
	if err := eng.Sync2(&model.TCmdLine{}); err != nil {
		t.Fatalf("sync TCmdLine: %v", err)
	}
}

// makeCheckBuild creates a minimal Build with one stage and one step for check() tests.
// Callers can then modify fields to test specific validation paths.
func makeCheckBuild(buildID string) *runtime.Build {
	stageID := "stage-1"
	stepID := "step-1"
	return &runtime.Build{
		Id: buildID,
		Repo: &runtime.Repository{
			CloneURL: "", // empty → isClone stays true, repoPath=""
		},
		Vars: map[string]*runtime.Variables{},
		Stages: []*runtime.Stage{
			{
				Id:      stageID,
				BuildId: buildID,
				Name:    "build",
				Steps: []*runtime.Step{
					{
						Id:       stepID,
						BuildId:  buildID,
						StageId:  stageID,
						Name:     "compile",
						Step:     "shell",
						Commands: []string{"echo hello"},
					},
				},
			},
		},
	}
}

func newCheckTask(build *runtime.Build) *BuildTask {
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel // keep cancel alive; caller may use it
	return &BuildTask{
		build:  build,
		ctx:    ctx,
		cncl:   cancel,
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
}

// --- Stage-level validation ---

func TestCheck_StageMismatchedBuildID(t *testing.T) {
	build := makeCheckBuild("build-1")
	build.Stages[0].BuildId = "wrong-build"
	task := newCheckTask(build)

	if task.check() {
		t.Fatal("check() should return false for mismatched stage BuildId")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("event = %q, want %q", task.build.Event, common.BuildEventCheckParam)
	}
	if task.build.Error == "" {
		t.Error("error should be set for mismatched stage BuildId")
	}
}

func TestCheck_StageEmptyName(t *testing.T) {
	build := makeCheckBuild("build-1")
	build.Stages[0].Name = ""
	task := newCheckTask(build)

	if task.check() {
		t.Fatal("check() should return false for empty stage name")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("event = %q, want %q", task.build.Event, common.BuildEventCheckParam)
	}
	if task.build.Error != "build Stage name is empty" {
		t.Errorf("error = %q, want %q", task.build.Error, "build Stage name is empty")
	}
}

func TestCheck_StageEmptySteps(t *testing.T) {
	build := makeCheckBuild("build-1")
	build.Stages[0].Steps = []*runtime.Step{}
	task := newCheckTask(build)

	if task.check() {
		t.Fatal("check() should return false for stage with no steps")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("event = %q, want %q", task.build.Event, common.BuildEventCheckParam)
	}
}

func TestCheck_DuplicateStageNames(t *testing.T) {
	setupCheckTestDB(t)
	build := makeCheckBuild("build-1")
	// Add a second stage with the same name
	build.Stages = append(build.Stages, &runtime.Stage{
		Id:      "stage-2",
		BuildId: "build-1",
		Name:    "build", // duplicate
		Steps: []*runtime.Step{
			{
				Id:       "step-2",
				BuildId:  "build-1",
				StageId:  "stage-2",
				Name:     "test",
				Step:     "shell",
				Commands: []string{"echo test"},
			},
		},
	})
	task := newCheckTask(build)

	if task.check() {
		t.Fatal("check() should return false for duplicate stage names")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("event = %q, want %q", task.build.Event, common.BuildEventCheckParam)
	}
}

// --- Step-level validation ---

func TestCheck_StepMismatchedBuildID(t *testing.T) {
	build := makeCheckBuild("build-1")
	build.Stages[0].Steps[0].BuildId = "wrong-build"
	task := newCheckTask(build)

	if task.check() {
		t.Fatal("check() should return false for mismatched step BuildId")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("event = %q, want %q", task.build.Event, common.BuildEventCheckParam)
	}
}

func TestCheck_StepMismatchedStageID(t *testing.T) {
	build := makeCheckBuild("build-1")
	build.Stages[0].Steps[0].StageId = "wrong-stage"
	task := newCheckTask(build)

	if task.check() {
		t.Fatal("check() should return false for mismatched step StageId")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("event = %q, want %q", task.build.Event, common.BuildEventCheckParam)
	}
}

func TestCheck_StepEmptyPlugin(t *testing.T) {
	build := makeCheckBuild("build-1")
	build.Stages[0].Steps[0].Step = "   " // whitespace-only after TrimSpace
	task := newCheckTask(build)

	if task.check() {
		t.Fatal("check() should return false for empty step plugin")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("event = %q, want %q", task.build.Event, common.BuildEventCheckParam)
	}
	if task.build.Error != "build Step Plugin is empty" {
		t.Errorf("error = %q, want %q", task.build.Error, "build Step Plugin is empty")
	}
}

func TestCheck_StepEmptyName(t *testing.T) {
	build := makeCheckBuild("build-1")
	build.Stages[0].Steps[0].Name = ""
	task := newCheckTask(build)

	if task.check() {
		t.Fatal("check() should return false for empty step name")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("event = %q, want %q", task.build.Event, common.BuildEventCheckParam)
	}
	if task.build.Error != "build Step name is empty" {
		t.Errorf("error = %q, want %q", task.build.Error, "build Step name is empty")
	}
}

func TestCheck_DuplicateStepNames(t *testing.T) {
	setupCheckTestDB(t)
	build := makeCheckBuild("build-1")
	// Add a second step with the same name in the same stage
	build.Stages[0].Steps = append(build.Stages[0].Steps, &runtime.Step{
		Id:       "step-2",
		BuildId:  "build-1",
		StageId:  "stage-1",
		Name:     "compile", // duplicate
		Step:     "shell",
		Commands: []string{"echo second"},
	})
	task := newCheckTask(build)

	if task.check() {
		t.Fatal("check() should return false for duplicate step names in same stage")
	}
	if task.build.Event != common.BuildEventCheckParam {
		t.Errorf("event = %q, want %q", task.build.Event, common.BuildEventCheckParam)
	}
}

// --- Clone URL paths ---

func TestCheck_CloneURLEmpty_IsCloneTrue(t *testing.T) {
	setupCheckTestDB(t)

	build := makeCheckBuild("build-clone-1")
	build.Repo.CloneURL = ""
	task := newCheckTask(build)

	result := task.check()
	if !result {
		t.Fatalf("check() should succeed with empty CloneURL; error: %s", task.build.Error)
	}
	if !task.isClone {
		t.Error("isClone should be true when CloneURL is empty")
	}
}

func TestCheck_CloneURLExistingDir_IsCloneFalse(t *testing.T) {
	setupCheckTestDB(t)

	localDir := t.TempDir()
	build := makeCheckBuild("build-clone-2")
	build.Repo.CloneURL = localDir
	task := newCheckTask(build)

	result := task.check()
	if !result {
		t.Fatalf("check() should succeed with existing dir as CloneURL; error: %s", task.build.Error)
	}
	if task.isClone {
		t.Error("isClone should be false when CloneURL is an existing directory")
	}
	if task.repoPath != localDir {
		t.Errorf("repoPath = %q, want %q", task.repoPath, localDir)
	}
}

// --- genRunjob command types via check() ---

func TestCheck_StringCommands(t *testing.T) {
	setupCheckTestDB(t)

	build := makeCheckBuild("build-cmd-1")
	build.Stages[0].Steps[0].Commands = "echo hello"
	task := newCheckTask(build)

	if !task.check() {
		t.Fatalf("check() should succeed with string commands; error: %s", task.build.Error)
	}
	if len(task.jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(task.jobs))
	}
}

func TestCheck_SliceStringCommands(t *testing.T) {
	setupCheckTestDB(t)

	build := makeCheckBuild("build-cmd-2")
	build.Stages[0].Steps[0].Commands = []string{"echo a", "echo b"}
	task := newCheckTask(build)

	if !task.check() {
		t.Fatalf("check() should succeed with []string commands; error: %s", task.build.Error)
	}
	if len(task.jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(task.jobs))
	}
}

func TestCheck_GokinsGitPlugin(t *testing.T) {
	setupCheckTestDB(t)

	build := makeCheckBuild("build-git-1")
	build.Stages[0].Steps[0].Step = "gokins@git"
	build.Stages[0].Steps[0].Commands = nil
	task := newCheckTask(build)
	task.repoPaths = "/tmp/some-repo"

	if !task.check() {
		t.Fatalf("check() should succeed with gokins@git plugin; error: %s", task.build.Error)
	}
}

// --- Variable substitution in genRunjob ---

func TestCheck_WithVariables(t *testing.T) {
	setupCheckTestDB(t)

	build := makeCheckBuild("build-var-1")
	build.Vars = map[string]*runtime.Variables{
		"MY_VAR": {Value: "hello", Secret: false},
	}
	build.Stages[0].Steps[0].Commands = []string{"echo ${MY_VAR}"}
	task := newCheckTask(build)

	if !task.check() {
		t.Fatalf("check() should succeed with variables; error: %s", task.build.Error)
	}
}

func TestCheck_WithSecretVariables(t *testing.T) {
	setupCheckTestDB(t)

	build := makeCheckBuild("build-var-2")
	build.Vars = map[string]*runtime.Variables{
		"SECRET_KEY": {Value: "s3cret", Secret: true},
	}
	build.Stages[0].Steps[0].Commands = []string{"echo ${SECRET_KEY}"}
	task := newCheckTask(build)

	if !task.check() {
		t.Fatalf("check() should succeed with secret variables; error: %s", task.build.Error)
	}
}

// --- Multiple stages ---

func TestCheck_MultipleStages(t *testing.T) {
	setupCheckTestDB(t)

	build := makeCheckBuild("build-multi-1")
	build.Stages = append(build.Stages, &runtime.Stage{
		Id:      "stage-2",
		BuildId: "build-multi-1",
		Name:    "test",
		Steps: []*runtime.Step{
			{
				Id:       "step-2",
				BuildId:  "build-multi-1",
				StageId:  "stage-2",
				Name:     "run-tests",
				Step:     "shell",
				Commands: []string{"echo test"},
			},
		},
	})
	task := newCheckTask(build)

	if !task.check() {
		t.Fatalf("check() should succeed with multiple stages; error: %s", task.build.Error)
	}
	if len(task.stages) != 2 {
		t.Errorf("expected 2 stages in task, got %d", len(task.stages))
	}
	if len(task.jobs) != 2 {
		t.Errorf("expected 2 jobs in task, got %d", len(task.jobs))
	}
}
