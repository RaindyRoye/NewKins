package engine

import (
	"container/list"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
)

// --- Manager accessor methods ---

func TestManagerBuildEgn(t *testing.T) {
	egn := &BuildEngine{taskw: list.New(), tasks: make(map[string]*BuildTask)}
	mgr := &Manager{buildEgn: egn}
	if mgr.BuildEgn() != egn {
		t.Error("BuildEgn() should return the build engine")
	}
}

func TestManagerHRun(t *testing.T) {
	hr := &HbtpRunner{}
	mgr := &Manager{hrun: hr}
	if mgr.HRun() != hr {
		t.Error("HRun() should return the hbtp runner")
	}
}

func TestManagerTimerEng(t *testing.T) {
	te := &TimerEngine{}
	mgr := &Manager{timerEgn: te}
	if mgr.TimerEng() != te {
		t.Error("TimerEng() should return the timer engine")
	}
}

func TestManagerPlugins_NilJobEngine(t *testing.T) {
	mgr := &Manager{}
	if plgs := mgr.Plugins(); plgs != nil {
		t.Errorf("expected nil plugins when jobEgn is nil, got %v", plgs)
	}
}

func TestManagerPlugins_WithJobEngine(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	je.execs["shell"] = &executer{plug: "shell", jobwt: list.New()}
	mgr := &Manager{jobEgn: je}
	plgs := mgr.Plugins()
	if len(plgs) != 1 || plgs[0] != "shell" {
		t.Errorf("expected [shell], got %v", plgs)
	}
}

// --- BuildTask.taskCtx ---

func TestTaskCtx_WithOwnContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{ctx: ctx}
	if bt.taskCtx() != ctx {
		t.Error("taskCtx() should return the task's own context when set")
	}
}

func TestTaskCtx_FallbackToGlobal(t *testing.T) {
	// Set up comm.Ctx
	commCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	oldCtx := comm.Ctx
	comm.Ctx = commCtx
	defer func() { comm.Ctx = oldCtx }()

	bt := &BuildTask{}
	if bt.taskCtx() != commCtx {
		t.Error("taskCtx() should fall back to comm.Ctx when task ctx is nil")
	}
}

// --- BuildTask.clears ---

func TestClears_RemovesCloneDir(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "repo")
	if err := os.MkdirAll(repoDir, 0750); err != nil {
		t.Fatal(err)
	}
	// Create a file inside
	if err := os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	bt := &BuildTask{
		build:     &runtime.Build{Id: "b1"},
		isClone:   true,
		repoPaths: repoDir,
		jobs:      make(map[string]*jobSync),
	}
	bt.clears()

	if _, err := os.Stat(repoDir); !os.IsNotExist(err) {
		t.Error("clears() should remove cloned repo directory")
	}
}

func TestClears_RemovesJobArtDirs(t *testing.T) {
	dir := t.TempDir()
	buildPath := filepath.Join(dir, "build")
	artDir := filepath.Join(buildPath, common.PathJobs, "step-1", common.PathArts)
	if err := os.MkdirAll(artDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "artifact.tar"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}

	bt := &BuildTask{
		build:     &runtime.Build{Id: "b1"},
		buildPath: buildPath,
		isClone:   false,
		jobs: map[string]*jobSync{
			"j1": {
				step: &runtime.Step{Id: "step-1"},
			},
		},
	}
	bt.clears()

	if _, err := os.Stat(artDir); !os.IsNotExist(err) {
		t.Error("clears() should remove job artifact directories")
	}
}

func TestClears_NoClone_NoPanic(t *testing.T) {
	bt := &BuildTask{
		build:     &runtime.Build{Id: "b1"},
		isClone:   false,
		buildPath: "/nonexistent/path",
		jobs:      make(map[string]*jobSync),
	}
	// Should not panic
	bt.clears()
}

// --- BuildTask.getRepo ---

func TestGetRepo_NotClone(t *testing.T) {
	bt := &BuildTask{
		build:   &runtime.Build{Id: "b1"},
		isClone: false,
	}
	err := bt.getRepo()
	if err != nil {
		t.Errorf("getRepo() should return nil when isClone is false, got: %v", err)
	}
}

func TestGetRepo_CloneWithEmptyPath(t *testing.T) {
	dir := t.TempDir()
	bt := &BuildTask{
		build: &runtime.Build{
			Id: "b1",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
		},
		isClone:   true,
		repoPaths: filepath.Join(dir, "repo"),
		repoPath:  "",
	}
	err := bt.getRepo()
	if err != nil {
		t.Errorf("getRepo() should succeed with empty repoPath (no clone needed), got: %v", err)
	}
	// repoPaths should have been created
	if _, err := os.Stat(bt.repoPaths); os.IsNotExist(err) {
		t.Error("getRepo() should create the repo path directory")
	}
}

// --- BuildTask.Show ---

func TestShow_Stopped(t *testing.T) {
	// ctx is nil => stopd() returns true => Show returns false
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
	}
	_, ok := bt.Show()
	if ok {
		t.Error("Show() should return false when build is stopped")
	}
}

func TestShow_Active(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bt := &BuildTask{
		build: &runtime.Build{
			Id:         "b1",
			PipelineId: "p1",
			Status:     "running",
			Stages:     []*runtime.Stage{},
		},
		ctx:    ctx,
		stages: make(map[string]*taskStage),
	}
	show, ok := bt.Show()
	if !ok {
		t.Fatal("Show() should return true for active build")
	}
	if show.Id != "b1" {
		t.Errorf("expected Id 'b1', got %q", show.Id)
	}
	if show.PipelineId != "p1" {
		t.Errorf("expected PipelineId 'p1', got %q", show.PipelineId)
	}
	if show.Status != "running" {
		t.Errorf("expected Status 'running', got %q", show.Status)
	}
}

func TestShow_WithStagesAndJobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	step := &runtime.Step{
		Id:      "step-1",
		Name:    "build",
		Status:  "running",
		BuildId: "b1",
	}
	stg := &runtime.Stage{
		Id:      "stg-1",
		Name:    "build",
		BuildId: "b1",
		Steps:   []*runtime.Step{step},
	}
	job := &jobSync{
		step: step,
		cmdmp: map[string]*cmdSync{
			"cmd-1": {
				cmd:    &runners.CmdContent{Id: "cmd-1", Conts: "echo hi"},
				status: "running",
			},
		},
	}
	ts := &taskStage{
		stage: stg,
		jobs:  map[string]*jobSync{"build": job},
	}

	bt := &BuildTask{
		build: &runtime.Build{
			Id:         "b1",
			PipelineId: "p1",
			Status:     "running",
			Stages:     []*runtime.Stage{stg},
		},
		ctx:    ctx,
		stages: map[string]*taskStage{"build": ts},
	}

	show, ok := bt.Show()
	if !ok {
		t.Fatal("Show() should return true for active build")
	}
	if len(show.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(show.Stages))
	}
	if show.Stages[0].Id != "stg-1" {
		t.Errorf("expected stage id 'stg-1', got %q", show.Stages[0].Id)
	}
	if len(show.Stages[0].Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(show.Stages[0].Steps))
	}
	if show.Stages[0].Steps[0].Id != "step-1" {
		t.Errorf("expected step id 'step-1', got %q", show.Stages[0].Steps[0].Id)
	}
	if len(show.Stages[0].Steps[0].Cmds) != 1 {
		t.Fatalf("expected 1 cmd, got %d", len(show.Stages[0].Steps[0].Cmds))
	}
}

// --- BuildTask.GetJob ---

func TestGetJob_EmptyId(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		jobs:  make(map[string]*jobSync),
	}
	_, ok := bt.GetJob("")
	if ok {
		t.Error("GetJob should return false for empty ID")
	}
}

func TestGetJob_NotFound(t *testing.T) {
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		jobs:  make(map[string]*jobSync),
	}
	_, ok := bt.GetJob("nonexistent")
	if ok {
		t.Error("GetJob should return false for unknown ID")
	}
}

func TestGetJob_Found(t *testing.T) {
	job := &jobSync{step: &runtime.Step{Id: "j1"}}
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		jobs:  map[string]*jobSync{"j1": job},
	}
	got, ok := bt.GetJob("j1")
	if !ok {
		t.Fatal("GetJob should return true for existing job")
	}
	if got != job {
		t.Error("GetJob should return the correct job")
	}
}

// --- BuildTask.UpJob ---

func TestUpJob_NilJob(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	// Should not panic
	bt.UpJob(nil, "ok", "", 0)
}

func TestUpJob_EmptyStatus(t *testing.T) {
	job := &jobSync{step: &runtime.Step{}}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	// Should not panic
	bt.UpJob(job, "", "", 0)
}

// --- BuildTask.UpJobCmd ---

func TestUpJobCmd_NilCmd(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	// Should not panic
	bt.UpJobCmd(nil, 1, 0)
}

func TestUpJobCmd_Running(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "c1"},
		status: common.BuildStatusPending,
	}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJobCmd(cmd, 1, 0)
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusRunning {
		t.Errorf("expected status 'running', got %q", cmd.status)
	}
	if cmd.started.IsZero() {
		t.Error("expected started time to be set")
	}
}

func TestUpJobCmd_OkZeroCode(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "c1"},
		status: common.BuildStatusRunning,
	}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJobCmd(cmd, 2, 0)
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusOk {
		t.Errorf("expected status 'ok', got %q", cmd.status)
	}
	if cmd.finished.IsZero() {
		t.Error("expected finished time to be set")
	}
}

func TestUpJobCmd_ErrorNonZeroCode(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "c1"},
		status: common.BuildStatusRunning,
	}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJobCmd(cmd, 2, 1)
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusError {
		t.Errorf("expected status 'error', got %q", cmd.status)
	}
	if cmd.code != 1 {
		t.Errorf("expected code 1, got %d", cmd.code)
	}
}

func TestUpJobCmd_Cancel(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "c1"},
		status: common.BuildStatusRunning,
	}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJobCmd(cmd, 3, 137)
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusCancel {
		t.Errorf("expected status 'cancel', got %q", cmd.status)
	}
	if cmd.code != 137 {
		t.Errorf("expected code 137, got %d", cmd.code)
	}
}

func TestUpJobCmd_Error(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "c1"},
		status: common.BuildStatusRunning,
	}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJobCmd(cmd, -1, 2)
	cmd.RLock()
	defer cmd.RUnlock()
	if cmd.status != common.BuildStatusError {
		t.Errorf("expected status 'error', got %q", cmd.status)
	}
	if cmd.code != 2 {
		t.Errorf("expected code 2, got %d", cmd.code)
	}
}

func TestUpJobCmd_DefaultIgnored(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "c1"},
		status: common.BuildStatusPending,
	}
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	bt.UpJobCmd(cmd, 99, 0)
	cmd.RLock()
	defer cmd.RUnlock()
	// Default case should return without changing status
	if cmd.status != common.BuildStatusPending {
		t.Errorf("expected status to remain 'pending', got %q", cmd.status)
	}
}

// --- BuildTask.Write ---

func TestWrite_NoProgress(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	input := "Cloning into 'repo'..."
	n, err := bt.Write([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(input) {
		t.Errorf("expected n=%d, got %d", len(input), n)
	}
	// workpgss should remain 0
	if bt.workpgss != 0 {
		t.Errorf("expected workpgss 0, got %d", bt.workpgss)
	}
}

func TestWrite_EmptyInput(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}
	n, err := bt.Write([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected n=0, got %d", n)
	}
}

// --- BuildEngine.Stop ---

func TestBuildEngineStop(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	ctx, cancel := context.WithCancel(context.Background())
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		ctx:   ctx,
		cncl:  cancel,
	}
	c.tasks["b1"] = bt
	c.Stop()
	if !bt.stopd() {
		t.Error("Stop() should cancel all running tasks")
	}
}

func TestBuildEngineStop_Empty(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	// Should not panic
	c.Stop()
}

// --- BuildEngine nil guard ---

func TestBuildEngineGet_NilEngine(t *testing.T) {
	var c *BuildEngine
	_, ok := c.Get("anything")
	if ok {
		t.Error("Get on nil engine should return false")
	}
}

// --- JobEngine.rmExec ---

func TestRmExec(t *testing.T) {
	je := &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	job1 := &jobSync{step: &runtime.Step{Id: "j1"}}
	job2 := &jobSync{step: &runtime.Step{Id: "j2"}}
	ex := &executer{
		plug:  "test-plugin",
		tms:   time.Now(),
		jobwt: list.New(),
	}
	ex.jobwt.PushBack(job1)
	ex.jobwt.PushBack(job2)
	je.execs["test-plugin"] = ex

	je.rmExec("test-plugin", ex)

	// Both jobs should be marked as ended
	if !job1.ended {
		t.Error("expected job1 to be marked as ended")
	}
	if !job2.ended {
		t.Error("expected job2 to be marked as ended")
	}

	// Executer should be removed
	je.exelk.RLock()
	_, exists := je.execs["test-plugin"]
	je.exelk.RUnlock()
	if exists {
		t.Error("expected executer to be removed")
	}
}

// --- baseRunner.CheckCancel ---

func TestCheckCancel_BuildNotFound(t *testing.T) {
	// Set up a minimal Mgr with build engine
	oldMgr := Mgr
	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	if !br.CheckCancel("nonexistent") {
		t.Error("CheckCancel should return true when build is not found")
	}
}

func TestCheckCancel_BuildStopped(t *testing.T) {
	oldMgr := Mgr
	// Build with nil context => stopd() returns true
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	if !br.CheckCancel("b1") {
		t.Error("CheckCancel should return true when build is stopped")
	}
}

func TestCheckCancel_BuildActive(t *testing.T) {
	oldMgr := Mgr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{build: &runtime.Build{Id: "b1"}, ctx: ctx}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	if br.CheckCancel("b1") {
		t.Error("CheckCancel should return false when build is active")
	}
}

// --- baseRunner.ServerInfo ---

func TestServerInfo(t *testing.T) {
	oldCfg := comm.Cfg
	comm.Cfg.Server.Host = "https://ci.example.com"
	comm.Cfg.Server.DownToken = "tok123"
	defer func() { comm.Cfg = oldCfg }()

	br := &baseRunner{}
	info, err := br.ServerInfo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.WebHost != "https://ci.example.com" {
		t.Errorf("expected WebHost 'https://ci.example.com', got %q", info.WebHost)
	}
	if info.DownToken != "tok123" {
		t.Errorf("expected DownToken 'tok123', got %q", info.DownToken)
	}
}

// --- baseRunner.FindJobId ---

func TestFindJobId_EmptyParams(t *testing.T) {
	br := &baseRunner{}
	_, ok := br.FindJobId("", "stage", "step")
	if ok {
		t.Error("FindJobId should return false for empty buildID")
	}
	_, ok = br.FindJobId("b1", "", "step")
	if ok {
		t.Error("FindJobId should return false for empty stage name")
	}
	_, ok = br.FindJobId("b1", "stage", "")
	if ok {
		t.Error("FindJobId should return false for empty step name")
	}
}

func TestFindJobId_BuildNotFound(t *testing.T) {
	oldMgr := Mgr
	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	_, ok := br.FindJobId("b1", "stage", "step")
	if ok {
		t.Error("FindJobId should return false when build not found")
	}
}

func TestFindJobId_Found(t *testing.T) {
	oldMgr := Mgr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	step := &runtime.Step{Id: "step-id-1", Name: "build-step"}
	stg := &taskStage{
		stage: &runtime.Stage{Name: "build"},
		jobs: map[string]*jobSync{
			"build-step": {step: step},
		},
	}
	bt := &BuildTask{
		build:  &runtime.Build{Id: "b1"},
		ctx:    ctx,
		stages: map[string]*taskStage{"build": stg},
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	jobId, ok := br.FindJobId("b1", "build", "build-step")
	if !ok {
		t.Fatal("FindJobId should return true for matching job")
	}
	if jobId != "step-id-1" {
		t.Errorf("expected job id 'step-id-1', got %q", jobId)
	}
}

func TestFindJobId_StageNotFound(t *testing.T) {
	oldMgr := Mgr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bt := &BuildTask{
		build:  &runtime.Build{Id: "b1"},
		ctx:    ctx,
		stages: map[string]*taskStage{},
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	_, ok := br.FindJobId("b1", "nonexistent", "step")
	if ok {
		t.Error("FindJobId should return false when stage not found")
	}
}

func TestFindJobId_StepNotFound(t *testing.T) {
	oldMgr := Mgr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stg := &taskStage{
		stage: &runtime.Stage{Name: "build"},
		jobs:  map[string]*jobSync{},
	}
	bt := &BuildTask{
		build:  &runtime.Build{Id: "b1"},
		ctx:    ctx,
		stages: map[string]*taskStage{"build": stg},
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	_, ok := br.FindJobId("b1", "build", "nonexistent-step")
	if ok {
		t.Error("FindJobId should return false when step not found")
	}
}

// --- baseRunner.Update ---

func TestUpdate_BuildNotFound(t *testing.T) {
	oldMgr := Mgr
	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	err := br.Update(&runners.UpdateJobInfo{BuildId: "b1", JobId: "j1"})
	if err == nil {
		t.Fatal("expected error for nonexistent build")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("expected ErrBuildNotFound, got: %v", err)
	}
}

func TestUpdate_JobNotFound(t *testing.T) {
	oldMgr := Mgr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		ctx:   ctx,
		jobs:  make(map[string]*jobSync),
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	err := br.Update(&runners.UpdateJobInfo{BuildId: "b1", JobId: "j1"})
	if err == nil {
		t.Fatal("expected error for nonexistent job")
	}
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got: %v", err)
	}
}

// --- baseRunner.UpdateCmd ---

func TestUpdateCmd_BuildNotFound(t *testing.T) {
	oldMgr := Mgr
	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	err := br.UpdateCmd("b1", "j1", "c1", 1, 0)
	if err == nil {
		t.Fatal("expected error for nonexistent build")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("expected ErrBuildNotFound, got: %v", err)
	}
}

func TestUpdateCmd_JobNotFound(t *testing.T) {
	oldMgr := Mgr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		ctx:   ctx,
		jobs:  make(map[string]*jobSync),
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	err := br.UpdateCmd("b1", "j1", "c1", 1, 0)
	if err == nil {
		t.Fatal("expected error for nonexistent job")
	}
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got: %v", err)
	}
}

func TestUpdateCmd_CmdNotFound(t *testing.T) {
	oldMgr := Mgr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job := &jobSync{
		step:  &runtime.Step{Id: "j1"},
		cmdmp: make(map[string]*cmdSync),
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		ctx:   ctx,
		jobs:  map[string]*jobSync{"j1": job},
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	err := br.UpdateCmd("b1", "j1", "c1", 1, 0)
	if err == nil {
		t.Fatal("expected error for nonexistent cmd")
	}
	if !errors.Is(err, ErrCmdNotFound) {
		t.Errorf("expected ErrCmdNotFound, got: %v", err)
	}
}

// --- baseRunner.ReadDir ---

func TestReadDir_BuildNotFound(t *testing.T) {
	oldMgr := Mgr
	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	_, err := br.ReadDir(1, "b1", "path")
	if err == nil {
		t.Fatal("expected error for nonexistent build")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("expected ErrBuildNotFound, got: %v", err)
	}
}

func TestReadDir_RepoFS(t *testing.T) {
	oldMgr := Mgr
	dir := t.TempDir()
	// Create test files
	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("hello"), 0600)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0750)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{
		build:     &runtime.Build{Id: "b1"},
		ctx:       ctx,
		repoPaths: dir,
		repoPath:  dir,
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	entries, err := br.ReadDir(1, "b1", ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

// --- baseRunner.ReadFile ---

func TestReadFile_InvalidFS(t *testing.T) {
	oldMgr := Mgr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		ctx:   ctx,
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	_, _, err := br.ReadFile(99, "b1", "path", 0)
	if err == nil {
		t.Fatal("expected error for invalid FS type")
	}
	if !errors.Is(err, ErrInvalidFSType) {
		t.Errorf("expected ErrInvalidFSType, got: %v", err)
	}
}

func TestReadFile_Valid(t *testing.T) {
	oldMgr := Mgr
	dir := t.TempDir()
	content := "hello world\nline 2"
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte(content), 0600)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{
		build:     &runtime.Build{Id: "b1"},
		ctx:       ctx,
		repoPaths: dir,
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	size, reader, err := br.ReadFile(1, "b1", "test.txt", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()
	if size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), size)
	}
}

func TestReadFile_WithStart(t *testing.T) {
	oldMgr := Mgr
	dir := t.TempDir()
	content := "hello world"
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte(content), 0600)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{
		build:     &runtime.Build{Id: "b1"},
		ctx:       ctx,
		repoPaths: dir,
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	_, reader, err := br.ReadFile(1, "b1", "test.txt", 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()
	buf := make([]byte, 100)
	n, _ := reader.Read(buf)
	if string(buf[:n]) != "world" {
		t.Errorf("expected 'world' from offset 6, got %q", string(buf[:n]))
	}
}

// --- baseRunner.StatFile ---

func TestStatFile_EmptyParams(t *testing.T) {
	br := &baseRunner{}
	_, err := br.StatFile(1, "b1", "", "dir", "path")
	if err == nil {
		t.Fatal("expected error for empty jobId")
	}
	_, err = br.StatFile(1, "b1", "j1", "dir", "")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestStatFile_InvalidFS(t *testing.T) {
	oldMgr := Mgr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	job := &jobSync{
		step: &runtime.Step{Id: "j1"},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		ctx:   ctx,
		jobs:  map[string]*jobSync{"j1": job},
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	_, err := br.StatFile(99, "b1", "j1", "dir", "file.txt")
	if err == nil {
		t.Fatal("expected error for invalid FS type")
	}
	if !errors.Is(err, ErrInvalidFSType) {
		t.Errorf("expected ErrInvalidFSType, got: %v", err)
	}
}

// --- baseRunner.UploadFile ---

func TestUploadFile_EmptyParams(t *testing.T) {
	br := &baseRunner{}
	_, err := br.UploadFile(1, "b1", "", "dir", "file.txt", 0)
	if err == nil {
		t.Fatal("expected error for empty jobId")
	}
	_, err = br.UploadFile(1, "b1", "j1", "dir", "", 0)
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestUploadFile_BuildNotFound(t *testing.T) {
	oldMgr := Mgr
	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	_, err := br.UploadFile(1, "b1", "j1", "dir", "file.txt", 0)
	if err == nil {
		t.Fatal("expected error for nonexistent build")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("expected ErrBuildNotFound, got: %v", err)
	}
}

func TestUploadFile_InvalidFS(t *testing.T) {
	oldMgr := Mgr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	job := &jobSync{
		step: &runtime.Step{Id: "j1"},
	}
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		ctx:   ctx,
		jobs:  map[string]*jobSync{"j1": job},
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	_, err := br.UploadFile(99, "b1", "j1", "dir", "file.txt", 0)
	if err == nil {
		t.Fatal("expected error for invalid FS type")
	}
	if !errors.Is(err, ErrInvalidFSType) {
		t.Errorf("expected ErrInvalidFSType, got: %v", err)
	}
}

func TestUploadFile_Valid(t *testing.T) {
	oldMgr := Mgr
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	job := &jobSync{
		step: &runtime.Step{Id: "j1"},
		task: &BuildTask{buildPath: dir},
	}
	bt := &BuildTask{
		build:     &runtime.Build{Id: "b1"},
		ctx:       ctx,
		buildPath: dir,
		jobs:      map[string]*jobSync{"j1": job},
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	wc, err := br.UploadFile(2, "b1", "j1", "", "output.txt", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer wc.Close()
	_, err = wc.Write([]byte("test data"))
	if err != nil {
		t.Fatalf("write error: %v", err)
	}
}

// --- baseRunner.GenEnv ---

func TestGenEnv_EmptyParams(t *testing.T) {
	br := &baseRunner{}
	err := br.GenEnv("b1", "", nil)
	if err == nil {
		t.Fatal("expected error for empty params")
	}
}

func TestGenEnv_BuildNotFound(t *testing.T) {
	oldMgr := Mgr
	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	err := br.GenEnv("b1", "j1", map[string]interface{}{"key": "val"})
	if err == nil {
		t.Fatal("expected error for nonexistent build")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("expected ErrBuildNotFound, got: %v", err)
	}
}

// --- baseRunner.PushOutLine ---

func TestPushOutLine_BuildNotFound(t *testing.T) {
	oldMgr := Mgr
	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	err := br.PushOutLine("b1", "j1", "c1", "some output", false)
	if err == nil {
		t.Fatal("expected error for nonexistent build")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("expected ErrBuildNotFound, got: %v", err)
	}
}

func TestPushOutLine_JobNotFound(t *testing.T) {
	oldMgr := Mgr
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{
		build: &runtime.Build{Id: "b1"},
		ctx:   ctx,
		jobs:  make(map[string]*jobSync),
	}
	egn := &BuildEngine{
		taskw: list.New(),
		tasks: map[string]*BuildTask{"b1": bt},
	}
	Mgr = &Manager{buildEgn: egn}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	err := br.PushOutLine("b1", "j1", "c1", "some output", false)
	if err == nil {
		t.Fatal("expected error for nonexistent job")
	}
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got: %v", err)
	}
}

// --- baseRunner.GetEnv ---

func TestGetEnv_EmptyParams(t *testing.T) {
	br := &baseRunner{}
	_, ok := br.GetEnv("b1", "", "key")
	if ok {
		t.Error("GetEnv should return false for empty jobId")
	}
	_, ok = br.GetEnv("b1", "j1", "")
	if ok {
		t.Error("GetEnv should return false for empty key")
	}
}

func TestGetEnv_BuildNotFound(t *testing.T) {
	oldMgr := Mgr
	Mgr = &Manager{
		buildEgn: &BuildEngine{
			taskw: list.New(),
			tasks: make(map[string]*BuildTask),
		},
	}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	_, ok := br.GetEnv("b1", "j1", "key")
	if ok {
		t.Error("GetEnv should return false when build not found")
	}
}

// --- baseRunner.PullJob ---

func TestPullJob_NoJobAvailable(t *testing.T) {
	oldMgr := Mgr
	Mgr = &Manager{
		jobEgn: &JobEngine{
			execs: make(map[string]*executer),
			jobs:  make(map[string]*jobSync),
		},
	}
	defer func() { Mgr = oldMgr }()

	br := &baseRunner{}
	// PullJob has a 5-second timeout, so this will take ~5s
	// We can't test the full timeout in unit tests easily, but we can test
	// that it returns an error with empty plugins
	_, err := br.PullJob("runner1", nil)
	if err == nil {
		t.Fatal("expected error when no job is available")
	}
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got: %v", err)
	}
}
