package engine

import (
	"container/list"
	"os"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/core/utils"
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

func TestBuildTaskCheck_StageBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-1",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{Id: "s1", BuildId: "wrong-id", Name: "stage1",
					Steps: []*runtime.Step{{Id: "st1", BuildId: "build-1", StageId: "s1", Step: "shell", Name: "step1"}}},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false on stage BuildId mismatch")
	}
}

func TestBuildTaskCheck_DuplicateStageName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-2",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{Id: "s1", BuildId: "build-2", Name: "dup",
					Steps: []*runtime.Step{{Id: "st1", BuildId: "build-2", StageId: "s1", Step: "shell", Name: "a"}}},
				{Id: "s2", BuildId: "build-2", Name: "dup",
					Steps: []*runtime.Step{{Id: "st2", BuildId: "build-2", StageId: "s2", Step: "shell", Name: "b"}}},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false on duplicate stage name")
	}
}

func TestBuildTaskCheck_EmptyStageName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-3",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{Id: "s1", BuildId: "build-3", Name: "",
					Steps: []*runtime.Step{{Id: "st1", BuildId: "build-3", StageId: "s1", Step: "shell", Name: "a"}}},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false on empty stage name")
	}
}

func TestBuildTaskCheck_EmptySteps(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-4",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{Id: "s1", BuildId: "build-4", Name: "stage1", Steps: []*runtime.Step{}},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false when stage has empty steps")
	}
}

func TestBuildTaskCheck_StepBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-5",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{Id: "s1", BuildId: "build-5", Name: "stage1",
					Steps: []*runtime.Step{{Id: "st1", BuildId: "wrong", StageId: "s1", Step: "shell", Name: "step1"}}},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false on step BuildId mismatch")
	}
}

func TestBuildTaskCheck_StepStageIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-6",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{Id: "s1", BuildId: "build-6", Name: "stage1",
					Steps: []*runtime.Step{{Id: "st1", BuildId: "build-6", StageId: "wrong", Step: "shell", Name: "step1"}}},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false on step StageId mismatch")
	}
}

func TestBuildTaskCheck_EmptyStepPlugin(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-7",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{Id: "s1", BuildId: "build-7", Name: "stage1",
					Steps: []*runtime.Step{{Id: "st1", BuildId: "build-7", StageId: "s1", Step: "", Name: "step1"}}},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false on empty step plugin")
	}
}

func TestBuildTaskCheck_EmptyStepName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-8",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{Id: "s1", BuildId: "build-8", Name: "stage1",
					Steps: []*runtime.Step{{Id: "st1", BuildId: "build-8", StageId: "s1", Step: "shell", Name: ""}}},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false on empty step name")
	}
}

func TestBuildTaskCheck_DuplicateStepName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-9",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{Id: "s1", BuildId: "build-9", Name: "stage1",
					Steps: []*runtime.Step{
						{Id: "st1", BuildId: "build-9", StageId: "s1", Step: "shell", Name: "dup"},
						{Id: "st2", BuildId: "build-9", StageId: "s1", Step: "shell", Name: "dup"},
					}},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	if task.check() {
		t.Fatal("check() should return false on duplicate step name")
	}
}

// --- JobEngine.Pull with job retrieval ---

func TestJobEnginePull_ReturnsJobWhenQueued(t *testing.T) {
	je := newTestJobEngine()
	// Pre-register executer for the plugin
	je.execs["myplugin"] = &executer{
		plug:  "myplugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	// Queue a job
	job := &jobSync{
		step:  &runtime.Step{Id: "step-1", Step: "myplugin"},
		cmdmp: make(map[string]*cmdSync),
		runjb: &runners.RunJob{Id: "step-1", Step: "myplugin", Name: "step-1"},
	}
	err := je.Put(job)
	if err != nil {
		t.Fatalf("Put() error: %v", err)
	}

	result := je.Pull("runner1", []string{"myplugin"})
	if result == nil {
		t.Fatal("Pull() should return the queued job")
	}
	if result.Id != "step-1" {
		t.Errorf("Pull() job Id = %q, want 'step-1'", result.Id)
	}
}

func TestJobEnginePull_NoJobAvailable(t *testing.T) {
	je := newTestJobEngine()
	// Register plugin with empty queue
	je.execs["myplugin"] = &executer{plug: "myplugin", jobwt: list.New()}

	result := je.Pull("runner1", []string{"myplugin"})
	if result != nil {
		t.Error("Pull() should return nil when queue is empty")
	}
}

func TestJobEnginePull_TriesMultiplePlugins(t *testing.T) {
	je := newTestJobEngine()
	// Queue job only on "plugin-b"
	je.execs["plugin-b"] = &executer{
		plug:  "plugin-b",
		jobwt: list.New(),
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-x", Step: "plugin-b"},
		cmdmp: make(map[string]*cmdSync),
		runjb: &runners.RunJob{Id: "step-x"},
	}
	je.execs["plugin-b"].jobwt.PushBack(job)

	// Pull with multiple plugin names — should find the job in "plugin-b"
	result := je.Pull("runner1", []string{"plugin-a", "plugin-b"})
	if result == nil {
		t.Fatal("Pull() should find job in second plugin")
	}
	if result.Id != "step-x" {
		t.Errorf("Pull() job Id = %q, want 'step-x'", result.Id)
	}
}

func TestJobEnginePull_UpdatesExecuterTimestamp(t *testing.T) {
	je := newTestJobEngine()
	je.execs["myplugin"] = &executer{
		plug:  "myplugin",
		tms:   time.Now().Add(-time.Hour), // old timestamp
		jobwt: list.New(),
	}
	oldTime := je.execs["myplugin"].tms
	_ = je.Pull("runner1", []string{"myplugin"})

	je.exelk.RLock()
	newTime := je.execs["myplugin"].tms
	je.exelk.RUnlock()
	if !newTime.After(oldTime) {
		t.Error("Pull() should update executer timestamp")
	}
}

// --- JobEngine.rmExec ---

func TestJobEngineRmExec(t *testing.T) {
	je := newTestJobEngine()
	ex := &executer{
		plug:  "myplugin",
		jobwt: list.New(),
	}
	// Add a couple of jobs
	j1 := &jobSync{step: &runtime.Step{Id: "s1"}}
	j2 := &jobSync{step: &runtime.Step{Id: "s2"}}
	ex.jobwt.PushBack(j1)
	ex.jobwt.PushBack(j2)

	je.execs["myplugin"] = ex
	je.rmExec("myplugin", ex)

	// Executer should be removed
	je.exelk.RLock()
	_, ok := je.execs["myplugin"]
	je.exelk.RUnlock()
	if ok {
		t.Error("rmExec should remove the executer")
	}
	// Jobs should be marked as ended
	if !j1.ended {
		t.Error("rmExec should mark job 1 as ended")
	}
	if !j2.ended {
		t.Error("rmExec should mark job 2 as ended")
	}
}

// --- JobEngine.run (cleanup expired execs and ended jobs) ---

func TestJobEngineRun_CleansEndedJobs(t *testing.T) {
	je := &JobEngine{
		tmr:   utils.NewTimer(time.Second * 30),
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	// Add an ended job
	je.jobs["ended-1"] = &jobSync{
		step:  &runtime.Step{Id: "ended-1"},
		ended: true,
	}
	// Add an active job
	je.jobs["active-1"] = &jobSync{
		step:  &runtime.Step{Id: "active-1"},
		ended: false,
	}
	// Run should clean up ended jobs
	// The run() method checks tmr.Tick(), so we force it by creating a timer
	// that ticks immediately (we call run directly)
	je.run()

	// Only active job should remain
	je.joblk.RLock()
	defer je.joblk.RUnlock()
	if _, ok := je.jobs["ended-1"]; ok {
		t.Error("run() should remove ended jobs")
	}
	if _, ok := je.jobs["active-1"]; !ok {
		t.Error("run() should keep active jobs")
	}
}

// --- getRepo with isClone=false ---

func TestGetRepo_NoClone(t *testing.T) {
	task := &BuildTask{
		build:   &runtime.Build{Id: "test-build"},
		isClone: false,
	}
	err := task.getRepo()
	if err != nil {
		t.Fatalf("getRepo() with isClone=false should not error, got: %v", err)
	}
}

func TestGetRepo_WithCloneNoPath(t *testing.T) {
	// isClone=true but repoPath="" should succeed (skips actual git clone)
	task := &BuildTask{
		build:     &runtime.Build{Id: "test-build"},
		isClone:   true,
		repoPaths: "/tmp/test-repo-nonexistent-" + t.Name(),
		repoPath:  "", // empty means skip clone
	}
	err := task.getRepo()
	if err != nil {
		t.Fatalf("getRepo() with empty repoPath should not error, got: %v", err)
	}
	// Clean up
	_ = os.RemoveAll(task.repoPaths)
}
