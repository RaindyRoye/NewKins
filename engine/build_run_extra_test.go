package engine

import (
	"container/list"
	"sync"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
)

// --- BuildEngine.run() tests ---

// TestBuildEngineRun_Ln2ExceedsLimit verifies run() returns early when active task count >= RunLimit
func TestBuildEngineRun_Ln2ExceedsLimit(t *testing.T) {
	comm.Cfg.Server.RunLimit = 1
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	c.tasks["existing"] = &BuildTask{build: &runtime.Build{Id: "existing"}}
	c.taskw.PushBack(&runtime.Build{Id: "queued"})
	c.run()
	if c.taskw.Len() != 1 {
		t.Errorf("expected queue length 1 (unchanged), got %d", c.taskw.Len())
	}
}

// TestBuildEngineRun_EmptyQueue verifies run() returns early when queue is empty
func TestBuildEngineRun_EmptyQueue(t *testing.T) {
	comm.Cfg.Server.RunLimit = 5
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	c.run()
	if len(c.tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(c.tasks))
	}
}

// TestBuildEngineRun_DequeueAndLaunch verifies run() dequeues and launches a build
func TestBuildEngineRun_DequeueAndLaunch(t *testing.T) {
	comm.Cfg.Server.RunLimit = 5
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bd := &runtime.Build{
		Id:         "build-run-1",
		PipelineId: "pipe-1",
		Status:     common.BuildStatusPending,
		Stages:     []*runtime.Stage{},
		Repo:       &runtime.Repository{},
	}
	c.taskw.PushBack(bd)
	c.run()
	time.Sleep(50 * time.Millisecond)
	if c.taskw.Len() != 0 {
		t.Errorf("expected empty queue after run(), got %d items", c.taskw.Len())
	}
}

// TestBuildEngineRun_ConcurrentDequeues verifies run() handles concurrent invocations safely
func TestBuildEngineRun_ConcurrentDequeues(t *testing.T) {
	comm.Cfg.Server.RunLimit = 10
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	for i := 0; i < 5; i++ {
		c.taskw.PushBack(&runtime.Build{
			Id:         "concurrent-build-" + string(rune('a'+i)),
			PipelineId: "pipe-1",
			Status:     common.BuildStatusPending,
			Stages:     []*runtime.Stage{},
			Repo:       &runtime.Repository{},
		})
	}
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.run()
		}()
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)
	if c.taskw.Len() != 0 {
		t.Errorf("expected empty queue, got %d items", c.taskw.Len())
	}
}

// TestBuildEngineRun_LimitExactlyMet verifies boundary behavior when tasks == RunLimit
func TestBuildEngineRun_LimitExactlyMet(t *testing.T) {
	comm.Cfg.Server.RunLimit = 2
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	c.tasks["t1"] = &BuildTask{build: &runtime.Build{Id: "t1"}}
	c.tasks["t2"] = &BuildTask{build: &runtime.Build{Id: "t2"}}
	c.taskw.PushBack(&runtime.Build{Id: "queued"})
	c.run()
	if c.taskw.Len() != 1 {
		t.Errorf("expected queue length 1 (unchanged), got %d", c.taskw.Len())
	}
}

// TestBuildEngineStartBuild_RemovesFromTasks verifies startBuild removes the task from the map
func TestBuildEngineStartBuild_RemovesFromTasks(t *testing.T) {
	comm.Cfg.Server.RunLimit = 5
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	bd := &runtime.Build{
		Id:         "build-start-1",
		PipelineId: "pipe-1",
		Status:     common.BuildStatusPending,
		Stages:     []*runtime.Stage{},
		Repo:       &runtime.Repository{},
	}
	bt := NewBuildTask(c, bd)
	c.tskslk.Lock()
	c.tasks[bd.Id] = bt
	c.tskslk.Unlock()
	c.startBuild(bt)
	c.tskslk.RLock()
	_, exists := c.tasks[bd.Id]
	c.tskslk.RUnlock()
	if exists {
		t.Error("startBuild should remove the task from the map after completion")
	}
}

// --- BuildTask.getRepo() tests ---

// TestGetRepo_NotCloning verifies getRepo() returns nil when isClone is false
func TestGetRepo_NotCloning(t *testing.T) {
	bt := &BuildTask{
		build:    &runtime.Build{Id: "no-clone"},
		isClone:  false,
		repoPath: "some-path",
	}
	if err := bt.getRepo(); err != nil {
		t.Errorf("getRepo() with isClone=false should return nil, got: %v", err)
	}
}

// TestGetRepo_CloneWithEmptyRepoPath verifies getRepo() succeeds when isClone=true but repoPath is empty
func TestGetRepo_CloneWithEmptyRepoPath(t *testing.T) {
	bt := &BuildTask{
		build:     &runtime.Build{Id: "empty-repo"},
		isClone:   true,
		repoPath:  "",
		repoPaths: t.TempDir() + "/repo",
	}
	if err := bt.getRepo(); err != nil {
		t.Errorf("getRepo() with empty repoPath should succeed, got: %v", err)
	}
}

// TestGetRepo_CloneCreateDirFails verifies getRepo() returns error when directory creation fails
func TestGetRepo_CloneCreateDirFails(t *testing.T) {
	bt := &BuildTask{
		build:     &runtime.Build{Id: "mkdir-fail"},
		isClone:   true,
		repoPaths: "/dev/null/impossible/path",
	}
	if err := bt.getRepo(); err == nil {
		t.Error("getRepo() should fail when directory creation fails")
	}
}

// --- cmdSync status transitions via UpJobCmd ---

// TestCmdSyncStatus_AllTransitions verifies all UpJobCmd state transitions
func TestCmdSyncStatus_AllTransitions(t *testing.T) {
	bt := &BuildTask{build: &runtime.Build{}}

	// fs=1: running
	cmd := &cmdSync{status: common.BuildStatusPending}
	bt.UpJobCmd(cmd, 1, 0)
	if cmd.status != common.BuildStatusRunning {
		t.Errorf("fs=1: expected running, got %q", cmd.status)
	}
	if cmd.started.IsZero() {
		t.Error("fs=1: started should be set")
	}

	// fs=2 with code=0: ok
	cmd = &cmdSync{status: common.BuildStatusRunning}
	bt.UpJobCmd(cmd, 2, 0)
	if cmd.status != common.BuildStatusOk {
		t.Errorf("fs=2 code=0: expected ok, got %q", cmd.status)
	}

	// fs=2 with code=1: error
	cmd = &cmdSync{status: common.BuildStatusRunning}
	bt.UpJobCmd(cmd, 2, 1)
	if cmd.status != common.BuildStatusError {
		t.Errorf("fs=2 code=1: expected error, got %q", cmd.status)
	}

	// fs=3: cancel
	cmd = &cmdSync{status: common.BuildStatusRunning}
	bt.UpJobCmd(cmd, 3, 0)
	if cmd.status != common.BuildStatusCancel {
		t.Errorf("fs=3: expected cancel, got %q", cmd.status)
	}

	// fs=-1: error
	cmd = &cmdSync{status: common.BuildStatusRunning}
	bt.UpJobCmd(cmd, -1, 127)
	if cmd.status != common.BuildStatusError || cmd.code != 127 {
		t.Errorf("fs=-1: expected error/127, got %q/%d", cmd.status, cmd.code)
	}

	// unknown fs: no-op
	cmd = &cmdSync{status: common.BuildStatusRunning}
	bt.UpJobCmd(cmd, 99, 0)
	if cmd.status != common.BuildStatusRunning {
		t.Errorf("fs=99: expected unchanged running, got %q", cmd.status)
	}
}

// --- Manager accessor tests ---

// TestManagerAccessors verifies Manager.BuildEgn/HRun/TimerEng/Plugins accessors
func TestManagerAccessors(t *testing.T) {
	egn := &BuildEngine{taskw: list.New(), tasks: make(map[string]*BuildTask)}
	hrun := &HbtpRunner{}
	teng := &TimerEngine{tasks: make(map[string]*timerExec)}

	// Nil jobEgn → Plugins returns nil
	mgr := &Manager{}
	if plugs := mgr.Plugins(); plugs != nil {
		t.Errorf("nil jobEgn: expected nil plugins, got %v", plugs)
	}

	// With all fields set
	mgr = &Manager{buildEgn: egn, hrun: hrun, timerEgn: teng}
	if mgr.BuildEgn() != egn {
		t.Error("BuildEgn() mismatch")
	}
	if mgr.HRun() != hrun {
		t.Error("HRun() mismatch")
	}
	if mgr.TimerEng() != teng {
		t.Error("TimerEng() mismatch")
	}
}
