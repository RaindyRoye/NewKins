package engine

import (
	"container/list"
	"errors"
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

// newTestEngine creates a minimal BuildEngine for testing error paths.
// It bypasses the StartBuildEngine goroutine and DB dependency.
func newTestEngine() *BuildEngine {
	return &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
}

func TestBaseRunner_CheckCancel_NoBuild(t *testing.T) {
	// Save and restore global Mgr
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	// A missing build should be treated as "already cancelled"
	if !r.CheckCancel("nonexistent-build-id") {
		t.Error("CheckCancel for nonexistent build should return true")
	}
}

func TestBaseRunner_CheckCancel_ActiveBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	// Insert a running build
	build := &runtime.Build{
		Id:         "active-build",
		PipelineId: "pipe-1",
		Status:     "running",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()

	r := &baseRunner{}
	// A running build with nil ctx should be considered "stopped" (stopd() returns true)
	if !r.CheckCancel("active-build") {
		t.Error("CheckCancel for build with nil ctx should return true (stopped)")
	}
}

func TestBaseRunner_Update_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	err := r.Update(&runners.UpdateJobInfo{
		BuildId: "missing-build",
		JobId:   "job-1",
		Status:  "ok",
	})
	if err == nil {
		t.Fatal("Update with missing build should error")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("Update error = %v, want wrapping ErrBuildNotFound", err)
	}
}

func TestBaseRunner_Update_NoJob(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	// Insert a build but no job
	build := &runtime.Build{
		Id:         "existing-build",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()

	r := &baseRunner{}
	err := r.Update(&runners.UpdateJobInfo{
		BuildId: "existing-build",
		JobId:   "missing-job",
		Status:  "ok",
	})
	if err == nil {
		t.Fatal("Update with missing job should error")
	}
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("Update error = %v, want wrapping ErrJobNotFound", err)
	}
}

func TestBaseRunner_UpdateCmd_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	err := r.UpdateCmd("missing-build", "job-1", "cmd-1", 1, 0)
	if err == nil {
		t.Fatal("UpdateCmd with missing build should error")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("UpdateCmd error = %v, want wrapping ErrBuildNotFound", err)
	}
}

func TestBaseRunner_UpdateCmd_NoJob(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	build := &runtime.Build{
		Id:         "existing-build-2",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()

	r := &baseRunner{}
	err := r.UpdateCmd("existing-build-2", "missing-job", "cmd-1", 1, 0)
	if err == nil {
		t.Fatal("UpdateCmd with missing job should error")
	}
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("UpdateCmd error = %v, want wrapping ErrJobNotFound", err)
	}
}

func TestBaseRunner_UpdateCmd_NoCmd(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	build := &runtime.Build{
		Id:         "existing-build-3",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	// Add a job but no commands in it
	step := &runtime.Step{Id: "step-1", BuildId: build.Id}
	job := &jobSync{
		task:  task,
		step:  step,
		cmdmp: make(map[string]*cmdSync),
	}
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()
	task.joblk.Lock()
	if task.jobs == nil {
		task.jobs = make(map[string]*jobSync)
	}
	task.jobs["existing-job"] = job
	task.joblk.Unlock()

	r := &baseRunner{}
	err := r.UpdateCmd("existing-build-3", "existing-job", "missing-cmd", 1, 0)
	if err == nil {
		t.Fatal("UpdateCmd with missing cmd should error")
	}
	if !errors.Is(err, ErrCmdNotFound) {
		t.Errorf("UpdateCmd error = %v, want wrapping ErrCmdNotFound", err)
	}
}

func TestBaseRunner_PushOutLine_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	err := r.PushOutLine("missing-build", "job-1", "cmd-1", "log line", false)
	if err == nil {
		t.Fatal("PushOutLine with missing build should error")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("PushOutLine error = %v, want wrapping ErrBuildNotFound", err)
	}
}

func TestBaseRunner_PushOutLine_NoJob(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	build := &runtime.Build{
		Id:         "existing-build-4",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()

	r := &baseRunner{}
	err := r.PushOutLine("existing-build-4", "missing-job", "cmd-1", "log line", false)
	if err == nil {
		t.Fatal("PushOutLine with missing job should error")
	}
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("PushOutLine error = %v, want wrapping ErrJobNotFound", err)
	}
}

func TestBaseRunner_ReadDir_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	_, err := r.ReadDir(1, "missing-build", "some/path")
	if err == nil {
		t.Fatal("ReadDir with missing build should error")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("ReadDir error = %v, want wrapping ErrBuildNotFound", err)
	}
}

func TestBaseRunner_ReadFile_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	_, _, err := r.ReadFile(1, "missing-build", "some/path", 0)
	if err == nil {
		t.Fatal("ReadFile with missing build should error")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("ReadFile error = %v, want wrapping ErrBuildNotFound", err)
	}
}

func TestBaseRunner_ReadFile_InvalidFSType(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	build := &runtime.Build{
		Id:         "existing-build-5",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()

	r := &baseRunner{}
	// fs=99 should resolve to empty path → ErrInvalidFSType
	_, _, err := r.ReadFile(99, "existing-build-5", "some/path", 0)
	if err == nil {
		t.Fatal("ReadFile with invalid fs type should error")
	}
	if !errors.Is(err, ErrInvalidFSType) {
		t.Errorf("ReadFile error = %v, want wrapping ErrInvalidFSType", err)
	}
}

func TestBaseRunner_StatFile_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	_, err := r.StatFile(1, "missing-build", "job-1", "dir", "path")
	if err == nil {
		t.Fatal("StatFile with missing build should error")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("StatFile error = %v, want wrapping ErrBuildNotFound", err)
	}
}

func TestBaseRunner_StatFile_NoJob(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	build := &runtime.Build{
		Id:         "existing-build-6",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()

	r := &baseRunner{}
	_, err := r.StatFile(1, "existing-build-6", "missing-job", "dir", "path")
	if err == nil {
		t.Fatal("StatFile with missing job should error")
	}
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("StatFile error = %v, want wrapping ErrJobNotFound", err)
	}
}

func TestBaseRunner_UploadFile_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	_, err := r.UploadFile(1, "missing-build", "job-1", "dir", "path", 0)
	if err == nil {
		t.Fatal("UploadFile with missing build should error")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("UploadFile error = %v, want wrapping ErrBuildNotFound", err)
	}
}

func TestBaseRunner_UploadFile_NoJob(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	build := &runtime.Build{
		Id:         "existing-build-7",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()

	r := &baseRunner{}
	_, err := r.UploadFile(1, "existing-build-7", "missing-job", "dir", "path", 0)
	if err == nil {
		t.Fatal("UploadFile with missing job should error")
	}
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("UploadFile error = %v, want wrapping ErrJobNotFound", err)
	}
}

func TestBaseRunner_FindJobId_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	_, ok := r.FindJobId("missing-build", "stage", "step")
	if ok {
		t.Error("FindJobId with missing build should return false")
	}
}

func TestBaseRunner_FindJobId_NoStage(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	build := &runtime.Build{
		Id:         "existing-build-8",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	task.stages = make(map[string]*taskStage)
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()

	r := &baseRunner{}
	_, ok := r.FindJobId("existing-build-8", "missing-stage", "step")
	if ok {
		t.Error("FindJobId with missing stage should return false")
	}
}

func TestBaseRunner_GetEnv_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	_, ok := r.GetEnv("missing-build", "job-1", "key")
	if ok {
		t.Error("GetEnv with missing build should return false")
	}
}

func TestBaseRunner_GetEnv_NoJob(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	build := &runtime.Build{
		Id:         "existing-build-9",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()

	r := &baseRunner{}
	_, ok := r.GetEnv("existing-build-9", "missing-job", "key")
	if ok {
		t.Error("GetEnv with missing job should return false")
	}
}

func TestBaseRunner_GenEnv_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	err := r.GenEnv("missing-build", "job-1", map[string]string{"FOO": "bar"})
	if err == nil {
		t.Fatal("GenEnv with missing build should error")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("GenEnv error = %v, want wrapping ErrBuildNotFound", err)
	}
}

func TestBaseRunner_GenEnv_NoJob(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	build := &runtime.Build{
		Id:         "existing-build-10",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()

	r := &baseRunner{}
	err := r.GenEnv("existing-build-10", "missing-job", map[string]string{"FOO": "bar"})
	if err == nil {
		t.Fatal("GenEnv with missing job should error")
	}
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("GenEnv error = %v, want wrapping ErrJobNotFound", err)
	}
}

func TestBaseRunner_FindArtVersionId_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	_, err := r.FindArtVersionId("missing-build", "idnt", "name")
	if err == nil {
		t.Fatal("FindArtVersionId with missing build should error")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("FindArtVersionId error = %v, want wrapping ErrBuildNotFound", err)
	}
}

func TestBaseRunner_NewArtVersionId_NoBuild(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	r := &baseRunner{}
	_, err := r.NewArtVersionId("missing-build", "idnt", "name")
	if err == nil {
		t.Fatal("NewArtVersionId with missing build should error")
	}
	if !errors.Is(err, ErrBuildNotFound) {
		t.Errorf("NewArtVersionId error = %v, want wrapping ErrBuildNotFound", err)
	}
}

func TestBaseRunner_StatFile_InvalidFSType(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	build := &runtime.Build{
		Id:         "existing-build-11",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	step := &runtime.Step{Id: "step-1", BuildId: build.Id}
	job := &jobSync{
		task: task,
		step: step,
	}
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()
	task.joblk.Lock()
	if task.jobs == nil {
		task.jobs = make(map[string]*jobSync)
	}
	task.jobs["existing-job"] = job
	task.joblk.Unlock()

	r := &baseRunner{}
	// fs=99 should resolve to empty path → ErrInvalidFSType
	_, err := r.StatFile(99, "existing-build-11", "existing-job", "dir", "path")
	if err == nil {
		t.Fatal("StatFile with invalid fs type should error")
	}
	if !errors.Is(err, ErrInvalidFSType) {
		t.Errorf("StatFile error = %v, want wrapping ErrInvalidFSType", err)
	}
}

func TestBaseRunner_UploadFile_InvalidFSType(t *testing.T) {
	saved := Mgr.buildEgn
	Mgr.buildEgn = newTestEngine()
	defer func() { Mgr.buildEgn = saved }()

	build := &runtime.Build{
		Id:         "existing-build-12",
		PipelineId: "pipe-1",
	}
	task := NewBuildTask(Mgr.buildEgn, build)
	step := &runtime.Step{Id: "step-1", BuildId: build.Id}
	job := &jobSync{
		task: task,
		step: step,
	}
	Mgr.buildEgn.tskslk.Lock()
	Mgr.buildEgn.tasks[build.Id] = task
	Mgr.buildEgn.tskslk.Unlock()
	task.joblk.Lock()
	if task.jobs == nil {
		task.jobs = make(map[string]*jobSync)
	}
	task.jobs["existing-job"] = job
	task.joblk.Unlock()

	r := &baseRunner{}
	_, err := r.UploadFile(99, "existing-build-12", "existing-job", "dir", "path", 0)
	if err == nil {
		t.Fatal("UploadFile with invalid fs type should error")
	}
	if !errors.Is(err, ErrInvalidFSType) {
		t.Errorf("UploadFile error = %v, want wrapping ErrInvalidFSType", err)
	}
}
