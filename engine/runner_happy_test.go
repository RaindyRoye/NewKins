package engine

import (
	"bytes"
	"container/list"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gokins/core/common"
	rt "github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
)

// --- baseRunner.CheckCancel ---

func TestBaseRunner_CheckCancel_NotFound(t *testing.T) {
	origBuildEgn := Mgr.buildEgn
	defer func() { Mgr.buildEgn = origBuildEgn }()

	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	r := &baseRunner{}
	// Build not found means CheckCancel returns true
	if !r.CheckCancel("nonexistent") {
		t.Error("CheckCancel should return true when build not found")
	}
}

func TestBaseRunner_CheckCancel_ActiveBuild(t *testing.T) {
	origBuildEgn := Mgr.buildEgn
	defer func() { Mgr.buildEgn = origBuildEgn }()

	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	bt := &BuildTask{
		build: &rt.Build{Id: "build-active"},
	}
	Mgr.buildEgn.tasks["build-active"] = bt

	r := &baseRunner{}
	if !r.CheckCancel("build-active") {
		t.Error("CheckCancel should return true when build ctx is nil (stopped)")
	}
}

func TestBaseRunner_CheckCancel_RunningBuild(t *testing.T) {
	origBuildEgn := Mgr.buildEgn
	defer func() { Mgr.buildEgn = origBuildEgn }()

	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	// Create a build with an active context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bt := &BuildTask{
		build: &rt.Build{Id: "build-running"},
		ctx:   ctx,
	}
	Mgr.buildEgn.tasks["build-running"] = bt

	r := &baseRunner{}
	if r.CheckCancel("build-running") {
		t.Error("CheckCancel should return false for a running build with active context")
	}
}

// --- baseRunner.FindJobId (with in-memory engine) ---

func TestBaseRunner_FindJobId_Found(t *testing.T) {
	origBuildEgn := Mgr.buildEgn
	defer func() { Mgr.buildEgn = origBuildEgn }()

	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	stage := &rt.Stage{
		Id:   "stage-1",
		Name: "build",
	}
	step := &rt.Step{
		Id:   "step-1",
		Name: "compile",
	}
	job := &jobSync{
		step: step,
	}
	ts := &taskStage{
		stage: stage,
		jobs:  map[string]*jobSync{"compile": job},
	}

	bt := &BuildTask{
		build:  &rt.Build{Id: "build-1"},
		stages: map[string]*taskStage{"build": ts},
	}
	Mgr.buildEgn.tasks["build-1"] = bt

	r := &baseRunner{}
	id, ok := r.FindJobId("build-1", "build", "compile")
	if !ok {
		t.Fatal("FindJobId should find the job")
	}
	if id != "step-1" {
		t.Errorf("FindJobId = %q, want %q", id, "step-1")
	}
}

func TestBaseRunner_FindJobId_StepNotFound(t *testing.T) {
	origBuildEgn := Mgr.buildEgn
	defer func() { Mgr.buildEgn = origBuildEgn }()

	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	stage := &rt.Stage{Id: "stage-1", Name: "build"}
	ts := &taskStage{
		stage: stage,
		jobs:  map[string]*jobSync{},
	}
	bt := &BuildTask{
		build:  &rt.Build{Id: "build-1"},
		stages: map[string]*taskStage{"build": ts},
	}
	Mgr.buildEgn.tasks["build-1"] = bt

	r := &baseRunner{}
	_, ok := r.FindJobId("build-1", "build", "nonexistent")
	if ok {
		t.Error("FindJobId should return false for nonexistent step")
	}
}

func TestBaseRunner_FindJobId_StageNotFound(t *testing.T) {
	origBuildEgn := Mgr.buildEgn
	defer func() { Mgr.buildEgn = origBuildEgn }()

	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	bt := &BuildTask{
		build:  &rt.Build{Id: "build-1"},
		stages: map[string]*taskStage{},
	}
	Mgr.buildEgn.tasks["build-1"] = bt

	r := &baseRunner{}
	_, ok := r.FindJobId("build-1", "nonexistent", "compile")
	if ok {
		t.Error("FindJobId should return false for nonexistent stage")
	}
}

// --- baseRunner.PushOutLine ---

func TestBaseRunner_PushOutLine_Success(t *testing.T) {
	origBuildEgn := Mgr.buildEgn
	defer func() {
		Mgr.buildEgn = origBuildEgn
	}()

	tmp := t.TempDir()

	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	job := &jobSync{
		step: &rt.Step{
			Id:      "step-1",
			BuildId: "build-1",
		},
	}
	bt := &BuildTask{
		build:     &rt.Build{Id: "build-1"},
		buildPath: tmp,
		jobs:      map[string]*jobSync{"step-1": job},
	}
	Mgr.buildEgn.tasks["build-1"] = bt

	// Override comm.WorkPath for this test
	origCommWorkPath := comm.WorkPath
	comm.WorkPath = tmp
	defer func() { comm.WorkPath = origCommWorkPath }()

	r := &baseRunner{}
	err := r.PushOutLine("build-1", "step-1", "cmd-1", "log line content", false)
	if err != nil {
		t.Fatalf("PushOutLine error: %v", err)
	}

	// Verify log file was created
	logDir := filepath.Join(tmp, common.PathBuild, "build-1", common.PathJobs, "step-1")
	logFile := filepath.Join(logDir, "build.log")
	data, err := os.ReadFile(logFile) //nolint:gosec // G304: test file path is controlled by t.TempDir()
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Parse the JSON log entry
	var logEntry struct {
		Id      string `json:"Id"`
		Content string `json:"Content"`
		Errs    bool   `json:"Errs"`
	}
	// The file has one JSON object per line
	_ = json.Unmarshal(bytes.SplitN(data, []byte{'\n'}, 2)[0], &logEntry)
	if logEntry.Id != "cmd-1" {
		t.Errorf("log entry Id = %q, want %q", logEntry.Id, "cmd-1")
	}
	if logEntry.Content != "log line content" {
		t.Errorf("log entry Content = %q, want %q", logEntry.Content, "log line content")
	}
}

func TestBaseRunner_PushOutLine_BuildNotFound(t *testing.T) {
	origBuildEgn := Mgr.buildEgn
	defer func() { Mgr.buildEgn = origBuildEgn }()

	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	r := &baseRunner{}
	err := r.PushOutLine("nonexistent", "step-1", "cmd-1", "content", false)
	if err == nil {
		t.Error("PushOutLine should fail when build not found")
	}
}

func TestBaseRunner_PushOutLine_JobNotFound(t *testing.T) {
	origBuildEgn := Mgr.buildEgn
	defer func() { Mgr.buildEgn = origBuildEgn }()

	Mgr.buildEgn = &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	bt := &BuildTask{
		build: &rt.Build{Id: "build-1"},
		jobs:  make(map[string]*jobSync),
	}
	Mgr.buildEgn.tasks["build-1"] = bt

	r := &baseRunner{}
	err := r.PushOutLine("build-1", "nonexistent", "cmd-1", "content", false)
	if err == nil {
		t.Error("PushOutLine should fail when job not found")
	}
}

// --- baseRunner.PullJob ---

func TestBaseRunner_PullJob_NoJobAvailable(t *testing.T) {
	origJobEgn := Mgr.jobEgn
	defer func() { Mgr.jobEgn = origJobEgn }()

	Mgr.jobEgn = newTestJobEngine()

	r := &baseRunner{}
	// PullJob polls for 5 seconds; with no jobs it should time out
	// We test this by calling with no registered plugins
	_, err := r.PullJob("runner1", []string{"unknown-plugin"})
	if err == nil {
		t.Error("PullJob should fail when no jobs are available")
	}
}

// --- Manager.Plugins ---

func TestManager_Plugins(t *testing.T) {
	origJobEgn := Mgr.jobEgn
	defer func() { Mgr.jobEgn = origJobEgn }()

	je := newTestJobEngine()
	je.execs["shell"] = &executer{plug: "shell", jobwt: list.New()}
	je.execs["git"] = &executer{plug: "git", jobwt: list.New()}
	Mgr.jobEgn = je

	plugs := Mgr.Plugins()
	if len(plugs) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugs))
	}
}

// --- BuildEngine.startBuild ---

func TestBuildEngine_StartBuild(t *testing.T) {
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}

	// Create a build task with a build that has no repo (check will fail)
	bd := &rt.Build{Id: "build-start"}
	bt := NewBuildTask(c, bd)

	// startBuild should run the task and remove it from the map
	c.startBuild(bt)

	// After startBuild, the task should be removed from the map
	_, ok := c.Get("build-start")
	if ok {
		t.Error("startBuild should remove the task after completion")
	}
}

// --- TimerEngine.execItem ---

func TestTimerEngine_ExecItem_FutureTick(t *testing.T) {
	te := &TimerEngine{
		tasks: make(map[string]*timerExec),
	}

	// Create a timer with tick in the future (should not execute)
	future := &timerExec{
		tt:   &model.TTrigger{Id: "t1"},
		typ:  1,
		tick: time.Now().Add(time.Hour),
	}
	te.tasks["t1"] = future

	// execItem should not trigger for future ticks
	te.execItem(future)
	// No panic means success
}
