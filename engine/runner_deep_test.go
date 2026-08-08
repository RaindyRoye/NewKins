package engine

import (
	"container/list"
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/runner/runners"
)

// --- Deeper check() branch coverage ---

func TestCheck_StageBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-100",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "wrong-build-id",
					Name:    "build",
					Steps:   []*runtime.Step{{Id: "step-1", Name: "compile", Step: "shell@ssh", BuildId: "wrong-build-id", StageId: "stage-1"}},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for build id mismatch")
	}
	if task.build.Error == "" {
		t.Fatal("expected error message for build id mismatch")
	}
}

func TestCheck_EmptyStageName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-101",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-101",
					Name:    "",
					Steps:   []*runtime.Step{{Id: "step-1", Name: "compile", Step: "shell@ssh", BuildId: "build-101", StageId: "stage-1"}},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for empty stage name")
	}
}

func TestCheck_EmptySteps(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-102",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-102",
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
		t.Fatal("expected check() to return false for empty steps")
	}
}

func TestCheck_DuplicateStageName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-103",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-103",
					Name:    "build",
					Steps:   []*runtime.Step{{Id: "step-1", Name: "a", Step: "shell@ssh", BuildId: "build-103", StageId: "stage-1"}},
				},
				{
					Id:      "stage-2",
					BuildId: "build-103",
					Name:    "build", // duplicate
					Steps:   []*runtime.Step{{Id: "step-2", Name: "b", Step: "shell@ssh", BuildId: "build-103", StageId: "stage-2"}},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for duplicate stage name")
	}
}

func TestCheck_StepBuildIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-104",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-104",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell@ssh", BuildId: "wrong", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for step build id mismatch")
	}
}

func TestCheck_StepStageIdMismatch(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-105",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-105",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell@ssh", BuildId: "build-105", StageId: "wrong-stage"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for step stage id mismatch")
	}
}

func TestCheck_EmptyStepPlugin(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-106",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-106",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "", BuildId: "build-106", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for empty step plugin")
	}
}

func TestCheck_EmptyStepName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-107",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-107",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "", Step: "shell@ssh", BuildId: "build-107", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for empty step name")
	}
}

func TestCheck_DuplicateStepName(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-108",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-108",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell@ssh", BuildId: "build-108", StageId: "stage-1"},
						{Id: "step-2", Name: "compile", Step: "shell@ssh", BuildId: "build-108", StageId: "stage-1"},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()
	if result {
		t.Fatal("expected check() to return false for duplicate step name")
	}
}

func TestCheck_StepTrimWhitespace(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-109",
			Repo: &runtime.Repository{CloneURL: ""},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-109",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "  shell@ssh  ", BuildId: "build-109", StageId: "stage-1",
							Commands: []string{"echo hello"}},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	// The step should be trimmed. Check that it's trimmed after check.
	// genRunjob will be called, but without DB it will panic/recover.
	// We just check the trimming happened.
	// Since genRunjob needs DB (comm.Db), we catch the panic.
	_ = task.check() // may return false due to genRunjob DB error, but step should be trimmed
	if task.build.Stages[0].Steps[0].Step != "shell@ssh" {
		t.Errorf("step should be trimmed, got %q", task.build.Stages[0].Steps[0].Step)
	}
}

func TestCheck_LocalRepoPath(t *testing.T) {
	// Create a temporary directory to simulate a local repo
	tmpDir := filepath.Join(os.TempDir(), "gokins-test-check-repo")
	if err := os.MkdirAll(tmpDir, 0750); err != nil {
		t.Skipf("cannot create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	task := &BuildTask{
		build: &runtime.Build{
			Id:   "build-110",
			Repo: &runtime.Repository{CloneURL: tmpDir},
			Stages: []*runtime.Stage{
				{
					Id:      "stage-1",
					BuildId: "build-110",
					Name:    "build",
					Steps: []*runtime.Step{
						{Id: "step-1", Name: "compile", Step: "shell@ssh", BuildId: "build-110", StageId: "stage-1",
							Commands: []string{"echo hello"}},
					},
				},
			},
		},
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	_ = task.check() // genRunjob needs DB, may fail, but isClone should be false
	if task.isClone {
		t.Error("expected isClone=false when CloneURL is a local directory")
	}
}

// --- BuildTask.getRepo ---

func TestGetRepo_NoClone(t *testing.T) {
	task := &BuildTask{
		build:   &runtime.Build{Id: "b1"},
		isClone: false,
	}
	err := task.getRepo()
	if err != nil {
		t.Fatalf("getRepo should return nil when isClone=false, got: %v", err)
	}
}

func TestGetRepo_CreatePaths(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "gokins-test-getrepo")
	defer os.RemoveAll(tmpDir)

	task := &BuildTask{
		build:     &runtime.Build{Id: "b2"},
		isClone:   true,
		repoPaths: tmpDir,
		repoPath:  "", // empty means no git clone needed
	}
	err := task.getRepo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Directory should be created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("expected repoPaths directory to be created")
	}
}

// --- BuildTask.clears ---

func TestClears_NilJobStep(t *testing.T) {
	task := &BuildTask{
		build: &runtime.Build{Id: "b3"},
		jobs:  make(map[string]*jobSync),
	}
	// Should not panic
	task.clears()
}

func TestClears_WithClone(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "gokins-test-clears")
	os.MkdirAll(tmpDir, 0750)
	defer os.RemoveAll(tmpDir)

	task := &BuildTask{
		build:     &runtime.Build{Id: "b4"},
		isClone:   true,
		repoPaths: tmpDir,
		jobs:      make(map[string]*jobSync),
	}
	task.clears()
	// Directory should be removed
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Error("expected repoPaths to be removed after clears()")
	}
}

// --- JobEngine.Pull with real job ---

func TestJobEnginePullWithJob(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	// Register an executer and add a job to it
	je.execs["shell@ssh"] = &executer{
		plug:  "shell@ssh",
		jobwt: list.New(),
	}
	step := &runtime.Step{
		Id:    "step-pull-1",
		Name:  "compile",
		Step:  "shell@ssh",
	}
	job := &jobSync{
		task:  &BuildTask{build: &runtime.Build{Id: "build-pull"}},
		step:  step,
		runjb: &runners.RunJob{Id: "step-pull-1", Name: "compile"},
		cmdmp: make(map[string]*cmdSync),
	}
	je.execs["shell@ssh"].jobwt.PushBack(job)

	// Pull should return the job
	result := je.Pull("runner-1", []string{"shell@ssh"})
	if result == nil {
		t.Fatal("expected Pull to return a job")
	}
	if result.Id != "step-pull-1" {
		t.Errorf("expected job id 'step-pull-1', got %q", result.Id)
	}
	// Job should be registered in jobs map
	je.joblk.RLock()
	_, ok := je.jobs["step-pull-1"]
	je.joblk.RUnlock()
	if !ok {
		t.Error("expected job to be registered in jobs map after Pull")
	}
}

func TestJobEnginePullMultiplePlugins(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	// First plugin has no jobs
	je.execs["plugin-a"] = &executer{
		plug:  "plugin-a",
		jobwt: list.New(),
	}
	// Second plugin has a job
	je.execs["plugin-b"] = &executer{
		plug:  "plugin-b",
		jobwt: list.New(),
	}
	step := &runtime.Step{Id: "step-multi", Name: "test", Step: "plugin-b"}
	job := &jobSync{
		task:  &BuildTask{build: &runtime.Build{Id: "build-multi"}},
		step:  step,
		runjb: &runners.RunJob{Id: "step-multi", Name: "test"},
		cmdmp: make(map[string]*cmdSync),
	}
	je.execs["plugin-b"].jobwt.PushBack(job)

	result := je.Pull("runner-1", []string{"plugin-a", "plugin-b"})
	if result == nil {
		t.Fatal("expected Pull to find job in plugin-b")
	}
	if result.Id != "step-multi" {
		t.Errorf("expected job from plugin-b, got %q", result.Id)
	}
}

func TestJobEnginePullEmptyQueue(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	je.execs["shell@ssh"] = &executer{
		plug:  "shell@ssh",
		jobwt: list.New(),
	}
	result := je.Pull("runner-1", []string{"shell@ssh"})
	if result != nil {
		t.Error("expected nil for empty queue")
	}
}

// --- Manager methods ---

func TestManagerBuildEgn_Nil(t *testing.T) {
	m := &Manager{}
	if m.BuildEgn() != nil {
		t.Error("expected nil BuildEgn")
	}
}

func TestManagerHRun_Nil(t *testing.T) {
	m := &Manager{}
	if m.HRun() != nil {
		t.Error("expected nil HRun")
	}
}

func TestManagerTimerEng_Nil(t *testing.T) {
	m := &Manager{}
	if m.TimerEng() != nil {
		t.Error("expected nil TimerEng")
	}
}

func TestManagerPlugins_NilJobEgn(t *testing.T) {
	m := &Manager{}
	plugs := m.Plugins()
	if plugs != nil {
		t.Errorf("expected nil Plugins when jobEgn is nil, got %v", plugs)
	}
}

func TestManagerPlugins_WithJobEgn(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	je.execs["p1"] = &executer{plug: "p1", jobwt: list.New()}
	m := &Manager{jobEgn: je}
	plugs := m.Plugins()
	if len(plugs) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugs))
	}
	if plugs[0] != "p1" {
		t.Errorf("expected 'p1', got %q", plugs[0])
	}
}

// --- baseRunner.CheckCancel ---

func TestBaseRunnerCheckCancel_NoBuild(t *testing.T) {
	// Save and restore Mgr
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	r := &baseRunner{}
	// Build not found should return true (cancelled)
	if !r.CheckCancel("nonexistent") {
		t.Error("CheckCancel should return true for non-existent build")
	}
}

func TestBaseRunnerCheckCancel_ActiveBuild(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "active-build"},
	}
	Mgr.buildEgn.tasks["active-build"] = bt

	r := &baseRunner{}
	// Active build (nil ctx means stopd() returns true)
	if !r.CheckCancel("active-build") {
		t.Error("CheckCancel should return true when task is stopped")
	}
}

// --- baseRunner.Update ---

func TestBaseRunnerUpdate_BuildNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	r := &baseRunner{}
	err := r.Update(&runners.UpdateJobInfo{BuildId: "nonexistent", JobId: "job"})
	if err == nil {
		t.Fatal("expected error for non-existent build")
	}
}

func TestBaseRunnerUpdate_JobNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-upd"},
		jobs:  make(map[string]*jobSync),
	}
	Mgr.buildEgn.tasks["build-upd"] = bt

	r := &baseRunner{}
	err := r.Update(&runners.UpdateJobInfo{BuildId: "build-upd", JobId: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for non-existent job")
	}
}

// --- baseRunner.FindJobId with real engine state ---

func TestBaseRunnerFindJobId_WithStages(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	step := &runtime.Step{Id: "step-find", Name: "compile", StageId: "stage-1"}
	job := &jobSync{
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}
	ts := &taskStage{
		stage: &runtime.Stage{Name: "build"},
		jobs:  map[string]*jobSync{"compile": job},
	}
	bt := &BuildTask{
		build:  &runtime.Build{Id: "build-find"},
		stages: map[string]*taskStage{"build": ts},
		jobs:   map[string]*jobSync{"step-find": job},
	}
	Mgr.buildEgn.tasks["build-find"] = bt

	r := &baseRunner{}
	id, ok := r.FindJobId("build-find", "build", "compile")
	if !ok {
		t.Fatal("expected FindJobId to find the job")
	}
	if id != "step-find" {
		t.Errorf("expected step id 'step-find', got %q", id)
	}

	// Not found step name
	_, ok = r.FindJobId("build-find", "build", "nonexistent")
	if ok {
		t.Error("expected false for non-existent step name")
	}

	// Not found stage name
	_, ok = r.FindJobId("build-find", "nonexistent", "compile")
	if ok {
		t.Error("expected false for non-existent stage name")
	}
}

// --- baseRunner.ReadDir with real build ---

func TestBaseRunnerReadDir_BuildNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	r := &baseRunner{}
	_, err := r.ReadDir(1, "nonexistent", "some/path")
	if err == nil {
		t.Fatal("expected error for non-existent build")
	}
}

func TestBaseRunnerReadDir_ValidBuild(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	tmpDir := filepath.Join(os.TempDir(), "gokins-test-readdir")
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0750)
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("hello"), 0600)
	defer os.RemoveAll(tmpDir)

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	bt := &BuildTask{
		build:     &runtime.Build{Id: "build-rd"},
		repoPaths: tmpDir,
		repoPath:  "/some/path", // non-empty to avoid returning nil
	}
	Mgr.buildEgn.tasks["build-rd"] = bt

	r := &baseRunner{}
	entries, err := r.ReadDir(1, "build-rd", ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 { // subdir + file.txt
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

// --- baseRunner.ReadFile ---

func TestBaseRunnerReadFile_BuildNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	r := &baseRunner{}
	_, _, err := r.ReadFile(1, "nonexistent", "path", 0)
	if err == nil {
		t.Fatal("expected error for non-existent build")
	}
}

func TestBaseRunnerReadFile_InvalidFSType(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-rf"},
	}
	Mgr.buildEgn.tasks["build-rf"] = bt

	r := &baseRunner{}
	_, _, err := r.ReadFile(99, "build-rf", "path", 0)
	if err == nil {
		t.Fatal("expected error for invalid fs type")
	}
}

func TestBaseRunnerReadFile_ValidFile(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	tmpDir := filepath.Join(os.TempDir(), "gokins-test-readfile")
	os.MkdirAll(tmpDir, 0750)
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello world"), 0600)
	defer os.RemoveAll(tmpDir)

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	bt := &BuildTask{
		build:     &runtime.Build{Id: "build-rf2"},
		repoPaths: tmpDir,
	}
	Mgr.buildEgn.tasks["build-rf2"] = bt

	r := &baseRunner{}
	size, reader, err := r.ReadFile(1, "build-rf2", "test.txt", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()
	if size != 11 { // "hello world" is 11 bytes
		t.Errorf("expected size 11, got %d", size)
	}
}

// --- baseRunner.GetEnv with real build ---

func TestBaseRunnerGetEnv_BuildNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	r := &baseRunner{}
	_, ok := r.GetEnv("nonexistent", "job", "key")
	if ok {
		t.Error("expected false for non-existent build")
	}
}

func TestBaseRunnerGetEnv_JobNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-env"},
		jobs:  make(map[string]*jobSync),
	}
	Mgr.buildEgn.tasks["build-env"] = bt

	r := &baseRunner{}
	_, ok := r.GetEnv("build-env", "nonexistent", "key")
	if ok {
		t.Error("expected false for non-existent job")
	}
}

// --- baseRunner.StatFile ---

func TestBaseRunnerStatFile_BuildNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	r := &baseRunner{}
	_, err := r.StatFile(1, "nonexistent", "job", "dir", "path")
	if err == nil {
		t.Fatal("expected error for non-existent build")
	}
}

func TestBaseRunnerStatFile_JobNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-sf"},
		jobs:  make(map[string]*jobSync),
	}
	Mgr.buildEgn.tasks["build-sf"] = bt

	r := &baseRunner{}
	_, err := r.StatFile(1, "build-sf", "nonexistent", "dir", "path")
	if err == nil {
		t.Fatal("expected error for non-existent job")
	}
}

func TestBaseRunnerStatFile_InvalidFSType(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-sf", Name: "test"},
		cmdmp: make(map[string]*cmdSync),
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-sf2"},
		jobs:  map[string]*jobSync{"step-sf": job},
	}
	Mgr.buildEgn.tasks["build-sf2"] = bt

	r := &baseRunner{}
	_, err := r.StatFile(99, "build-sf2", "step-sf", "dir", "path")
	if err == nil {
		t.Fatal("expected error for invalid fs type")
	}
}

// --- baseRunner.UploadFile ---

func TestBaseRunnerUploadFile_BuildNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	r := &baseRunner{}
	_, err := r.UploadFile(1, "nonexistent", "job", "dir", "path", 0)
	if err == nil {
		t.Fatal("expected error for non-existent build")
	}
}

func TestBaseRunnerUploadFile_JobNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-uf"},
		jobs:  make(map[string]*jobSync),
	}
	Mgr.buildEgn.tasks["build-uf"] = bt

	r := &baseRunner{}
	_, err := r.UploadFile(1, "build-uf", "nonexistent", "dir", "path", 0)
	if err == nil {
		t.Fatal("expected error for non-existent job")
	}
}

func TestBaseRunnerUploadFile_InvalidFSType(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uf", Name: "test"},
		cmdmp: make(map[string]*cmdSync),
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-uf2"},
		jobs:  map[string]*jobSync{"step-uf": job},
	}
	Mgr.buildEgn.tasks["build-uf2"] = bt

	r := &baseRunner{}
	_, err := r.UploadFile(99, "build-uf2", "step-uf", "dir", "path", 0)
	if err == nil {
		t.Fatal("expected error for invalid fs type")
	}
}

func TestBaseRunnerUploadFile_ValidUpload(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	tmpDir := filepath.Join(os.TempDir(), "gokins-test-upload")
	defer os.RemoveAll(tmpDir)

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uf2", Name: "test"},
		task:  &BuildTask{buildPath: tmpDir},
		cmdmp: make(map[string]*cmdSync),
	}
	bt := &BuildTask{
		build:     &runtime.Build{Id: "build-uf3"},
		buildPath: tmpDir,
		jobs:      map[string]*jobSync{"step-uf2": job},
	}
	Mgr.buildEgn.tasks["build-uf3"] = bt

	r := &baseRunner{}
	// Use fs=1 (artifacts dir)
	w, err := r.UploadFile(1, "build-uf3", "step-uf2", "", "test-upload.txt", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected non-nil writer")
	}
	defer w.Close()
	// Write some data
	_, err = w.Write([]byte("test data"))
	if err != nil {
		t.Fatalf("write error: %v", err)
	}
}

// --- baseRunner.GenEnv ---

func TestBaseRunnerGenEnv_BuildNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	r := &baseRunner{}
	err := r.GenEnv("nonexistent", "job", map[string]string{"key": "val"})
	if err == nil {
		t.Fatal("expected error for non-existent build")
	}
}

func TestBaseRunnerGenEnv_JobNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-gen"},
		jobs:  make(map[string]*jobSync),
	}
	Mgr.buildEgn.tasks["build-gen"] = bt

	r := &baseRunner{}
	err := r.GenEnv("build-gen", "nonexistent", map[string]string{"key": "val"})
	if err == nil {
		t.Fatal("expected error for non-existent job")
	}
}

// --- baseRunner.PushOutLine ---

func TestBaseRunnerPushOutLine_BuildNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	r := &baseRunner{}
	err := r.PushOutLine("nonexistent", "job", "cmd", "data", false)
	if err == nil {
		t.Fatal("expected error for non-existent build")
	}
}

func TestBaseRunnerPushOutLine_JobNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-pol"},
		jobs:  make(map[string]*jobSync),
	}
	Mgr.buildEgn.tasks["build-pol"] = bt

	r := &baseRunner{}
	err := r.PushOutLine("build-pol", "nonexistent", "cmd", "data", false)
	if err == nil {
		t.Fatal("expected error for non-existent job")
	}
}

func TestBaseRunnerPushOutLine_ValidPush(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	tmpDir := filepath.Join(os.TempDir(), "gokins-test-pushout")
	defer os.RemoveAll(tmpDir)

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	job := &jobSync{
		step: &runtime.Step{
			Id:      "step-pol",
			Name:    "test",
			BuildId: "build-pol2",
		},
		cmdmp: make(map[string]*cmdSync),
	}
	bt := &BuildTask{
		build:     &runtime.Build{Id: "build-pol2"},
		buildPath: tmpDir,
		jobs:      map[string]*jobSync{"step-pol": job},
	}
	Mgr.buildEgn.tasks["build-pol2"] = bt
	comm.WorkPath = tmpDir

	r := &baseRunner{}
	err := r.PushOutLine("build-pol2", "step-pol", "cmd-1", "log data", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify log file was created
	logPath := filepath.Join(tmpDir, common.PathBuild, "build-pol2", common.PathJobs, "step-pol", "build.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("expected log file to be created")
	}
}

// --- baseRunner.UpdateCmd ---

func TestBaseRunnerUpdateCmd_BuildNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	r := &baseRunner{}
	err := r.UpdateCmd("nonexistent", "job", "cmd", 1, 0)
	if err == nil {
		t.Fatal("expected error for non-existent build")
	}
}

func TestBaseRunnerUpdateCmd_JobNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-uc"},
		jobs:  make(map[string]*jobSync),
	}
	Mgr.buildEgn.tasks["build-uc"] = bt

	r := &baseRunner{}
	err := r.UpdateCmd("build-uc", "nonexistent", "cmd", 1, 0)
	if err == nil {
		t.Fatal("expected error for non-existent job")
	}
}

func TestBaseRunnerUpdateCmd_CmdNotFound(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uc", Name: "test"},
		cmdmp: make(map[string]*cmdSync),
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-uc2"},
		jobs:  map[string]*jobSync{"step-uc": job},
	}
	Mgr.buildEgn.tasks["build-uc2"] = bt

	r := &baseRunner{}
	err := r.UpdateCmd("build-uc2", "step-uc", "nonexistent", 1, 0)
	if err == nil {
		t.Fatal("expected error for non-existent cmd")
	}
}

func TestBaseRunnerUpdateCmd_ValidUpdate(t *testing.T) {
	origMgr := Mgr
	defer func() { Mgr = origMgr }()

	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-uc1"},
		status: common.BuildStatusPending,
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uc", Name: "test"},
		cmdmp: map[string]*cmdSync{"cmd-uc1": cmd},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-uc3"},
		jobs:  map[string]*jobSync{"step-uc": job},
	}
	Mgr.buildEgn.tasks["build-uc3"] = bt

	r := &baseRunner{}
	err := r.UpdateCmd("build-uc3", "step-uc", "cmd-uc1", 1, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cmd.RLock()
	if cmd.status != common.BuildStatusRunning {
		t.Errorf("expected status %q, got %q", common.BuildStatusRunning, cmd.status)
	}
	cmd.RUnlock()
}
