package engine

import (
	"container/list"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/runner/runners"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestBuildEngine creates a BuildEngine and registers a BuildTask in Mgr.
// Returns a cleanup function to restore the original Mgr state.
func setupTestBuildEngine(t *testing.T, buildID string, stages map[string]*taskStage, jobs map[string]*jobSync) (*BuildTask, func()) {
	t.Helper()
	e := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	build := &runtime.Build{
		Id:         buildID,
		PipelineId: "pipe-1",
		Status:     common.BuildStatusRunning,
		Stages:     []*runtime.Stage{},
	}
	bt := &BuildTask{
		egn:       e,
		build:     build,
		stages:    stages,
		jobs:      jobs,
		buildPath: filepath.Join(t.TempDir(), "build", buildID),
		repoPaths: filepath.Join(t.TempDir(), "repo"),
		repoPath:  "/tmp/repo",
	}
	ctx, cancel := context.WithCancel(context.Background())
	bt.ctx = ctx
	bt.cncl = cancel
	e.tasks[buildID] = bt

	origBuildEgn := Mgr.buildEgn
	Mgr.buildEgn = e

	cleanup := func() {
		cancel()
		Mgr.buildEgn = origBuildEgn
	}
	return bt, cleanup
}

// --- baseRunner.CheckCancel ---

func TestBaseRunner_CheckCancel_BuildNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	// Build "nonexistent" not found → should return true (treated as canceled)
	assert.True(t, r.CheckCancel("nonexistent"), "CheckCancel should return true when build not found")
}

func TestBaseRunner_CheckCancel_ActiveBuild(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-active", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	// build-active has a live context → should return false
	assert.False(t, r.CheckCancel("build-active"), "CheckCancel should return false for active build")
}

func TestBaseRunner_CheckCancel_CancelledBuild(t *testing.T) {
	e := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	bt := &BuildTask{
		build: &runtime.Build{Id: "build-canceled"},
		ctx:   ctx,
		cncl:  cancel,
	}
	e.tasks["build-canceled"] = bt
	origBuildEgn := Mgr.buildEgn
	Mgr.buildEgn = e
	defer func() { Mgr.buildEgn = origBuildEgn }()

	r := &baseRunner{}
	assert.True(t, r.CheckCancel("build-canceled"), "CheckCancel should return true for canceled build")
}

// --- baseRunner.Update ---

func TestBaseRunner_Update_BuildNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	err := r.Update(&runners.UpdateJobInfo{BuildId: "nonexistent", JobId: "job-1"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

func TestBaseRunner_Update_JobNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-u1",
		make(map[string]*taskStage),
		make(map[string]*jobSync),
	)
	defer cleanup()

	r := &baseRunner{}
	err := r.Update(&runners.UpdateJobInfo{BuildId: "build-u1", JobId: "nonexistent-job"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobNotFound))
}

func TestBaseRunner_Update_Success(t *testing.T) {
	job := &jobSync{
		step:  &runtime.Step{Id: "step-1", Name: "test-step"},
		cmdmp: make(map[string]*cmdSync),
	}
	stages := make(map[string]*taskStage)
	jobs := map[string]*jobSync{"step-1": job}

	_, cleanup := setupTestBuildEngine(t, "build-u2", stages, jobs)
	defer cleanup()

	r := &baseRunner{}
	err := r.Update(&runners.UpdateJobInfo{
		BuildId:  "build-u2",
		JobId:    "step-1",
		Status:   common.BuildStatusOk,
		Error:    "",
		ExitCode: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, common.BuildStatusOk, job.step.Status)
}

// --- baseRunner.UpdateCmd ---

func TestBaseRunner_UpdateCmd_BuildNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	err := r.UpdateCmd("nonexistent", "job-1", "cmd-1", 1, 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

func TestBaseRunner_UpdateCmd_JobNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-uc1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	err := r.UpdateCmd("build-uc1", "nonexistent-job", "cmd-1", 1, 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobNotFound))
}

func TestBaseRunner_UpdateCmd_CmdNotFound(t *testing.T) {
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uc", Name: "test-step"},
		cmdmp: make(map[string]*cmdSync),
	}
	jobs := map[string]*jobSync{"step-uc": job}

	_, cleanup := setupTestBuildEngine(t, "build-uc2", make(map[string]*taskStage), jobs)
	defer cleanup()

	r := &baseRunner{}
	err := r.UpdateCmd("build-uc2", "step-uc", "nonexistent-cmd", 1, 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCmdNotFound))
}

func TestBaseRunner_UpdateCmd_Success_Running(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-1", Conts: "echo hello"},
		status: common.BuildStatusPending,
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uc3", Name: "test-step"},
		cmdmp: map[string]*cmdSync{"cmd-1": cmd},
	}
	jobs := map[string]*jobSync{"step-uc3": job}

	_, cleanup := setupTestBuildEngine(t, "build-uc3", make(map[string]*taskStage), jobs)
	defer cleanup()

	r := &baseRunner{}
	// fs=1 means running
	err := r.UpdateCmd("build-uc3", "step-uc3", "cmd-1", 1, 0)
	require.NoError(t, err)
	assert.Equal(t, common.BuildStatusRunning, cmd.status)
	assert.False(t, cmd.started.IsZero())
}

func TestBaseRunner_UpdateCmd_Success_OkWithCode(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-2", Conts: "echo test"},
		status: common.BuildStatusRunning,
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uc4", Name: "test-step"},
		cmdmp: map[string]*cmdSync{"cmd-2": cmd},
	}
	jobs := map[string]*jobSync{"step-uc4": job}

	_, cleanup := setupTestBuildEngine(t, "build-uc4", make(map[string]*taskStage), jobs)
	defer cleanup()

	r := &baseRunner{}
	// fs=2 means ok, code=0 means success
	err := r.UpdateCmd("build-uc4", "step-uc4", "cmd-2", 2, 0)
	require.NoError(t, err)
	assert.Equal(t, common.BuildStatusOk, cmd.status)
	assert.False(t, cmd.finished.IsZero())
}

func TestBaseRunner_UpdateCmd_ErrorWithExitCode(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-3", Conts: "exit 1"},
		status: common.BuildStatusRunning,
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uc5", Name: "test-step"},
		cmdmp: map[string]*cmdSync{"cmd-3": cmd},
	}
	jobs := map[string]*jobSync{"step-uc5": job}

	_, cleanup := setupTestBuildEngine(t, "build-uc5", make(map[string]*taskStage), jobs)
	defer cleanup()

	r := &baseRunner{}
	// fs=2 means ok, but code=1 means error
	err := r.UpdateCmd("build-uc5", "step-uc5", "cmd-3", 2, 1)
	require.NoError(t, err)
	assert.Equal(t, common.BuildStatusError, cmd.status)
	assert.Equal(t, 1, cmd.code)
}

func TestBaseRunner_UpdateCmd_Cancel(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-4", Conts: "sleep 100"},
		status: common.BuildStatusRunning,
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uc6", Name: "test-step"},
		cmdmp: map[string]*cmdSync{"cmd-4": cmd},
	}
	jobs := map[string]*jobSync{"step-uc6": job}

	_, cleanup := setupTestBuildEngine(t, "build-uc6", make(map[string]*taskStage), jobs)
	defer cleanup()

	r := &baseRunner{}
	// fs=3 means cancel
	err := r.UpdateCmd("build-uc6", "step-uc6", "cmd-4", 3, 0)
	require.NoError(t, err)
	assert.Equal(t, common.BuildStatusCancel, cmd.status)
}

func TestBaseRunner_UpdateCmd_ErrorStatus(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-5", Conts: "failing"},
		status: common.BuildStatusRunning,
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uc7", Name: "test-step"},
		cmdmp: map[string]*cmdSync{"cmd-5": cmd},
	}
	jobs := map[string]*jobSync{"step-uc7": job}

	_, cleanup := setupTestBuildEngine(t, "build-uc7", make(map[string]*taskStage), jobs)
	defer cleanup()

	r := &baseRunner{}
	// fs=-1 means error
	err := r.UpdateCmd("build-uc7", "step-uc7", "cmd-5", -1, 127)
	require.NoError(t, err)
	assert.Equal(t, common.BuildStatusError, cmd.status)
	assert.Equal(t, 127, cmd.code)
}

func TestBaseRunner_UpdateCmd_UnknownStatus(t *testing.T) {
	cmd := &cmdSync{
		cmd:    &runners.CmdContent{Id: "cmd-6", Conts: "test"},
		status: common.BuildStatusPending,
	}
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uc8", Name: "test-step"},
		cmdmp: map[string]*cmdSync{"cmd-6": cmd},
	}
	jobs := map[string]*jobSync{"step-uc8": job}

	_, cleanup := setupTestBuildEngine(t, "build-uc8", make(map[string]*taskStage), jobs)
	defer cleanup()

	r := &baseRunner{}
	// fs=99 is unknown, should be a no-op (returns nil)
	err := r.UpdateCmd("build-uc8", "step-uc8", "cmd-6", 99, 0)
	require.NoError(t, err)
	// Status should remain unchanged
	assert.Equal(t, common.BuildStatusPending, cmd.status)
}

// --- baseRunner.FindJobId ---

func TestBaseRunner_FindJobId_BuildNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	id, ok := r.FindJobId("nonexistent", "stage-1", "step-1")
	assert.False(t, ok)
	assert.Empty(t, id)
}

func TestBaseRunner_FindJobId_StageNotFound(t *testing.T) {
	stages := make(map[string]*taskStage)
	_, cleanup := setupTestBuildEngine(t, "build-fj1", stages, make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	id, ok := r.FindJobId("build-fj1", "nonexistent-stage", "step-1")
	assert.False(t, ok)
	assert.Empty(t, id)
}

func TestBaseRunner_FindJobId_StepFound(t *testing.T) {
	job := &jobSync{
		step: &runtime.Step{Id: "step-real", Name: "compile"},
	}
	stages := map[string]*taskStage{
		"build": {
			stage: &runtime.Stage{Name: "build"},
			jobs: map[string]*jobSync{
				"compile": job,
			},
		},
	}

	_, cleanup := setupTestBuildEngine(t, "build-fj2", stages, make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	id, ok := r.FindJobId("build-fj2", "build", "compile")
	assert.True(t, ok)
	assert.Equal(t, "step-real", id)
}

func TestBaseRunner_FindJobId_StepNotFound(t *testing.T) {
	stages := map[string]*taskStage{
		"build": {
			stage: &runtime.Stage{Name: "build"},
			jobs: map[string]*jobSync{
				"compile": {step: &runtime.Step{Id: "step-x", Name: "compile"}},
			},
		},
	}

	_, cleanup := setupTestBuildEngine(t, "build-fj3", stages, make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	id, ok := r.FindJobId("build-fj3", "build", "nonexistent-step")
	assert.False(t, ok)
	assert.Empty(t, id)
}

// --- baseRunner.ReadDir ---

func TestBaseRunner_ReadDir_BuildNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	_, err := r.ReadDir(1, "nonexistent", "path")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

func TestBaseRunner_ReadDir_EmptyBuildID(t *testing.T) {
	r := &baseRunner{}
	_, err := r.ReadDir(1, "", "path")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
}

func TestBaseRunner_ReadDir_EmptyPath(t *testing.T) {
	r := &baseRunner{}
	_, err := r.ReadDir(1, "build", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyParams))
}

func TestBaseRunner_ReadDir_RepoPath_FsType1(t *testing.T) {
	// Create actual files in the repo path
	repoDir := t.TempDir()
	_, cleanup := setupTestBuildEngine(t, "build-rd1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()
	// Override the build's repoPaths
	Mgr.buildEgn.tasks["build-rd1"].repoPaths = repoDir

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "file1.txt"), []byte("test"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "file2.go"), []byte("code"), 0600))
	subDir := filepath.Join(repoDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0750))

	r := &baseRunner{}
	entries, err := r.ReadDir(1, "build-rd1", ".")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 3) // file1.txt, file2.go, subdir

	// Check that dir flags are correct
	for _, e := range entries {
		if e.Name == "subdir" {
			assert.True(t, e.IsDir, "subdir should be a directory")
		}
		if e.Name == "file1.txt" {
			assert.False(t, e.IsDir, "file1.txt should not be a directory")
			assert.Equal(t, int64(4), e.Size)
		}
	}
}

func TestBaseRunner_ReadDir_ArtifactsPath_FsType2(t *testing.T) {
	artDir := t.TempDir()
	_, cleanup := setupTestBuildEngine(t, "build-rd2", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	// Set comm.WorkPath and create artifact file
	origWorkPath := comm.WorkPath
	comm.WorkPath = artDir
	defer func() { comm.WorkPath = origWorkPath }()

	artPath := filepath.Join(artDir, common.PathArtifacts)
	require.NoError(t, os.MkdirAll(artPath, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(artPath, "artifact.jar"), []byte("binary"), 0600))

	r := &baseRunner{}
	entries, err := r.ReadDir(2, "build-rd2", ".")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "artifact.jar", entries[0].Name)
}

func TestBaseRunner_ReadDir_BuildJobsPath_FsType3(t *testing.T) {
	bt, cleanup := setupTestBuildEngine(t, "build-rd3", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	// Create jobs path under buildPath
	jobsDir := filepath.Join(bt.buildPath, common.PathJobs)
	require.NoError(t, os.MkdirAll(jobsDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(jobsDir, "log.txt"), []byte("logs"), 0600))

	r := &baseRunner{}
	entries, err := r.ReadDir(3, "build-rd3", ".")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "log.txt", entries[0].Name)
}

func TestBaseRunner_ReadDir_InvalidFsType(t *testing.T) {
	bt, cleanup := setupTestBuildEngine(t, "build-rd4", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	// fs=99 → pths remains empty, so os.ReadDir("") fails
	r := &baseRunner{}
	_, err := r.ReadDir(99, "build-rd4", ".")
	// Should error because pths is empty
	require.Error(t, err)
	_ = bt // suppress unused warning
}

func TestBaseRunner_ReadDir_EmptyRepoPathNoError(t *testing.T) {
	bt, cleanup := setupTestBuildEngine(t, "build-rd5", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	// Set repoPath to empty so the function returns nil, nil
	bt.repoPath = ""
	bt.repoPaths = "/nonexistent/path"

	r := &baseRunner{}
	entries, err := r.ReadDir(1, "build-rd5", ".")
	// When repoPath is "", error returns nil, nil
	assert.NoError(t, err)
	assert.Nil(t, entries)
}

// --- baseRunner.ReadFile ---

func TestBaseRunner_ReadFile_BuildNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	_, _, err := r.ReadFile(1, "nonexistent", "path", 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

func TestBaseRunner_ReadFile_InvalidFsType(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-rf1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	_, _, err := r.ReadFile(99, "build-rf1", "file.txt", 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidFSType))
}

func TestBaseRunner_ReadFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	bt, cleanup := setupTestBuildEngine(t, "build-rf2", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()
	bt.repoPaths = tmpDir

	content := "hello world content"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte(content), 0600))

	r := &baseRunner{}
	size, reader, err := r.ReadFile(1, "build-rf2", "test.txt", 0)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()
	assert.Equal(t, int64(len(content)), size)

	buf := make([]byte, 1024)
	n, _ := reader.Read(buf)
	assert.Equal(t, content, string(buf[:n]))
}

func TestBaseRunner_ReadFile_WithStartOffset(t *testing.T) {
	tmpDir := t.TempDir()
	bt, cleanup := setupTestBuildEngine(t, "build-rf3", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()
	bt.repoPaths = tmpDir

	content := "0123456789ABCDEF"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "offset.txt"), []byte(content), 0600))

	r := &baseRunner{}
	size, reader, err := r.ReadFile(1, "build-rf3", "offset.txt", 5)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()
	assert.Equal(t, int64(len(content)), size) // size is full file size

	buf := make([]byte, 1024)
	n, _ := reader.Read(buf)
	assert.Equal(t, "56789ABCDEF", string(buf[:n]))
}

func TestBaseRunner_ReadFile_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	bt, cleanup := setupTestBuildEngine(t, "build-rf4", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()
	bt.repoPaths = tmpDir

	r := &baseRunner{}
	_, _, err := r.ReadFile(1, "build-rf4", "nonexistent.txt", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stat file")
}

// --- baseRunner.StatFile ---

func TestBaseRunner_StatFile_BuildNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	_, err := r.StatFile(1, "nonexistent", "job-1", "dir", "file.txt")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

func TestBaseRunner_StatFile_JobNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-sf1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	_, err := r.StatFile(1, "build-sf1", "nonexistent-job", "dir", "file.txt")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobNotFound))
}

func TestBaseRunner_StatFile_InvalidFsType(t *testing.T) {
	job := &jobSync{
		step:  &runtime.Step{Id: "step-sf", Name: "test-step"},
		cmdmp: make(map[string]*cmdSync),
		task:  nil, // will be set below
	}
	jobs := map[string]*jobSync{"step-sf": job}
	bt, cleanup := setupTestBuildEngine(t, "build-sf2", make(map[string]*taskStage), jobs)
	defer cleanup()
	job.task = bt

	r := &baseRunner{}
	_, err := r.StatFile(99, "build-sf2", "step-sf", "dir", "file.txt")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidFSType))
}

func TestBaseRunner_StatFile_Success_FsType1(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	artDir := filepath.Join(tmpDir, common.PathArtifacts, "mydir")
	require.NoError(t, os.MkdirAll(artDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(artDir, "file.jar"), []byte("binary-content"), 0600))

	job := &jobSync{
		step:  &runtime.Step{Id: "step-sf3", Name: "test-step"},
		cmdmp: make(map[string]*cmdSync),
		task:  nil,
	}
	jobs := map[string]*jobSync{"step-sf3": job}
	bt, cleanup := setupTestBuildEngine(t, "build-sf3", make(map[string]*taskStage), jobs)
	defer cleanup()
	job.task = bt

	r := &baseRunner{}
	stat, err := r.StatFile(1, "build-sf3", "step-sf3", "mydir", "file.jar")
	require.NoError(t, err)
	assert.Equal(t, "file.jar", stat.Name)
	assert.False(t, stat.IsDir)
	assert.Equal(t, int64(14), stat.Size)
}

func TestBaseRunner_StatFile_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	artDir := filepath.Join(tmpDir, common.PathArtifacts, "mydir")
	require.NoError(t, os.MkdirAll(artDir, 0750))
	// Don't create the file

	job := &jobSync{
		step:  &runtime.Step{Id: "step-sf4", Name: "test-step"},
		cmdmp: make(map[string]*cmdSync),
		task:  nil,
	}
	jobs := map[string]*jobSync{"step-sf4": job}
	bt, cleanup := setupTestBuildEngine(t, "build-sf4", make(map[string]*taskStage), jobs)
	defer cleanup()
	job.task = bt

	r := &baseRunner{}
	_, err := r.StatFile(1, "build-sf4", "step-sf4", "mydir", "missing.jar")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stat file")
}

// --- baseRunner.UploadFile ---

func TestBaseRunner_UploadFile_BuildNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	_, err := r.UploadFile(1, "nonexistent", "job-1", "dir", "file.txt", 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

func TestBaseRunner_UploadFile_JobNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-uf1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	_, err := r.UploadFile(1, "build-uf1", "nonexistent-job", "dir", "file.txt", 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobNotFound))
}

func TestBaseRunner_UploadFile_InvalidFsType(t *testing.T) {
	job := &jobSync{
		step:  &runtime.Step{Id: "step-uf", Name: "test-step"},
		cmdmp: make(map[string]*cmdSync),
		task:  nil,
	}
	jobs := map[string]*jobSync{"step-uf": job}
	bt, cleanup := setupTestBuildEngine(t, "build-uf2", make(map[string]*taskStage), jobs)
	defer cleanup()
	job.task = bt

	r := &baseRunner{}
	_, err := r.UploadFile(99, "build-uf2", "step-uf", "dir", "file.txt", 0)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidFSType))
}

func TestBaseRunner_UploadFile_Success_FsType1(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	job := &jobSync{
		step:  &runtime.Step{Id: "step-uf3", Name: "test-step"},
		cmdmp: make(map[string]*cmdSync),
		task:  nil,
	}
	jobs := map[string]*jobSync{"step-uf3": job}
	bt, cleanup := setupTestBuildEngine(t, "build-uf3", make(map[string]*taskStage), jobs)
	defer cleanup()
	job.task = bt

	r := &baseRunner{}
	writer, err := r.UploadFile(1, "build-uf3", "step-uf3", "updir", "uploaded.txt", 0)
	require.NoError(t, err)
	defer func() { _ = writer.Close() }()

	n, err := writer.Write([]byte("uploaded content"))
	require.NoError(t, err)
	assert.Equal(t, 16, n)

	// Verify file was created
	data, err := os.ReadFile(filepath.Join(tmpDir, common.PathArtifacts, "updir", "uploaded.txt")) //nolint:gosec
	require.NoError(t, err)
	assert.Equal(t, "uploaded content", string(data))
}

func TestBaseRunner_UploadFile_WithStartOffset(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	// Pre-create the file with some content
	artDir := filepath.Join(tmpDir, common.PathArtifacts, "updir2")
	require.NoError(t, os.MkdirAll(artDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(artDir, "partial.txt"), []byte("AAAAABBBBB"), 0600))

	job := &jobSync{
		step:  &runtime.Step{Id: "step-uf4", Name: "test-step"},
		cmdmp: make(map[string]*cmdSync),
		task:  nil,
	}
	jobs := map[string]*jobSync{"step-uf4": job}
	bt, cleanup := setupTestBuildEngine(t, "build-uf4", make(map[string]*taskStage), jobs)
	defer cleanup()
	job.task = bt

	r := &baseRunner{}
	writer, err := r.UploadFile(1, "build-uf4", "step-uf4", "updir2", "partial.txt", 5)
	require.NoError(t, err)
	defer func() { _ = writer.Close() }()

	_, err = writer.Write([]byte("CCCCC"))
	require.NoError(t, err)

	// Verify the file: AAAAA + CCCCC
	data, err := os.ReadFile(filepath.Join(artDir, "partial.txt")) //nolint:gosec
	require.NoError(t, err)
	assert.Equal(t, "AAAAACCCCC", string(data))
}

// --- baseRunner.PushOutLine ---

func TestBaseRunner_PushOutLine_BuildNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	err := r.PushOutLine("nonexistent", "job-1", "cmd-1", "log line", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

func TestBaseRunner_PushOutLine_JobNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-po1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	err := r.PushOutLine("build-po1", "nonexistent-job", "cmd-1", "log line", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobNotFound))
}

func TestBaseRunner_PushOutLine_Success(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	job := &jobSync{
		step:  &runtime.Step{Id: "step-po", Name: "test-step", BuildId: "build-po2"},
		cmdmp: make(map[string]*cmdSync),
	}
	jobs := map[string]*jobSync{"step-po": job}
	bt, cleanup := setupTestBuildEngine(t, "build-po2", make(map[string]*taskStage), jobs)
	defer cleanup()
	bt.buildPath = filepath.Join(tmpDir, "build", "build-po2")

	r := &baseRunner{}
	err := r.PushOutLine("build-po2", "step-po", "cmd-1", "hello log output", false)
	require.NoError(t, err)

	// Verify log file was created
	logPath := filepath.Join(tmpDir, common.PathBuild, "build-po2", common.PathJobs, "step-po", "build.log")
	data, err := os.ReadFile(logPath) //nolint:gosec
	require.NoError(t, err)
	assert.Contains(t, string(data), "hello log output")
}

func TestBaseRunner_PushOutLine_ErrorFlag(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	job := &jobSync{
		step:  &runtime.Step{Id: "step-po2", Name: "error-step", BuildId: "build-po3"},
		cmdmp: make(map[string]*cmdSync),
	}
	jobs := map[string]*jobSync{"step-po2": job}
	bt, cleanup := setupTestBuildEngine(t, "build-po3", make(map[string]*taskStage), jobs)
	defer cleanup()
	bt.buildPath = filepath.Join(tmpDir, "build", "build-po3")

	r := &baseRunner{}
	err := r.PushOutLine("build-po3", "step-po2", "cmd-2", "error output", true)
	require.NoError(t, err)

	logPath := filepath.Join(tmpDir, common.PathBuild, "build-po3", common.PathJobs, "step-po2", "build.log")
	data, err := os.ReadFile(logPath) //nolint:gosec
	require.NoError(t, err)
	// The JSON should have errs:true
	assert.Contains(t, string(data), `"errs":true`)
}

func TestBaseRunner_PushOutLine_MultipleLines(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	job := &jobSync{
		step:  &runtime.Step{Id: "step-po3", Name: "multi-step", BuildId: "build-po4"},
		cmdmp: make(map[string]*cmdSync),
	}
	jobs := map[string]*jobSync{"step-po3": job}
	bt, cleanup := setupTestBuildEngine(t, "build-po4", make(map[string]*taskStage), jobs)
	defer cleanup()
	bt.buildPath = filepath.Join(tmpDir, "build", "build-po4")

	r := &baseRunner{}
	require.NoError(t, r.PushOutLine("build-po4", "step-po3", "cmd-a", "line 1", false))
	require.NoError(t, r.PushOutLine("build-po4", "step-po3", "cmd-b", "line 2", false))

	logPath := filepath.Join(tmpDir, common.PathBuild, "build-po4", common.PathJobs, "step-po3", "build.log")
	data, err := os.ReadFile(logPath) //nolint:gosec
	require.NoError(t, err)
	assert.Contains(t, string(data), "line 1")
	assert.Contains(t, string(data), "line 2")
}

// --- baseRunner.GetEnv ---

func TestBaseRunner_GetEnv_BuildNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	val, ok := r.GetEnv("nonexistent", "job-1", "key")
	assert.False(t, ok)
	assert.Empty(t, val)
}

func TestBaseRunner_GetEnv_JobNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-ge1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	val, ok := r.GetEnv("build-ge1", "nonexistent-job", "key")
	assert.False(t, ok)
	assert.Empty(t, val)
}

func TestBaseRunner_GetEnv_NoEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	job := &jobSync{
		step:  &runtime.Step{Id: "step-ge", Name: "test-step", BuildId: "build-ge2"},
		cmdmp: make(map[string]*cmdSync),
	}
	jobs := map[string]*jobSync{"step-ge": job}
	_, cleanup := setupTestBuildEngine(t, "build-ge2", make(map[string]*taskStage), jobs)
	defer cleanup()

	r := &baseRunner{}
	// No env file exists → should return false
	val, ok := r.GetEnv("build-ge2", "step-ge", "MY_VAR")
	assert.False(t, ok)
	assert.Empty(t, val)
}

func TestBaseRunner_GetEnv_Success(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	job := &jobSync{
		step:  &runtime.Step{Id: "step-ge2", Name: "test-step", BuildId: "build-ge3"},
		cmdmp: make(map[string]*cmdSync),
	}
	jobs := map[string]*jobSync{"step-ge2": job}
	_, cleanup := setupTestBuildEngine(t, "build-ge3", make(map[string]*taskStage), jobs)
	defer cleanup()

	// Create the env file manually
	dir := filepath.Join(tmpDir, common.PathBuild, "build-ge3", common.PathJobs, "step-ge2")
	require.NoError(t, os.MkdirAll(dir, 0750))
	mp := hbtp.NewMaps(nil)
	mp.Set("MY_KEY", "my_value")
	mp.Set("NUM_KEY", 42)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "build.env"), mp.ToBytes(), 0600))

	r := &baseRunner{}
	val, ok := r.GetEnv("build-ge3", "step-ge2", "MY_KEY")
	assert.True(t, ok)
	assert.Equal(t, "my_value", val)
}

func TestBaseRunner_GetEnv_KeyNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	job := &jobSync{
		step:  &runtime.Step{Id: "step-ge3", Name: "test-step", BuildId: "build-ge4"},
		cmdmp: make(map[string]*cmdSync),
	}
	jobs := map[string]*jobSync{"step-ge3": job}
	_, cleanup := setupTestBuildEngine(t, "build-ge4", make(map[string]*taskStage), jobs)
	defer cleanup()

	// Create the env file
	dir := filepath.Join(tmpDir, common.PathBuild, "build-ge4", common.PathJobs, "step-ge3")
	require.NoError(t, os.MkdirAll(dir, 0750))
	mp := hbtp.NewMaps(nil)
	mp.Set("EXISTING_KEY", "value")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "build.env"), mp.ToBytes(), 0600))

	r := &baseRunner{}
	val, ok := r.GetEnv("build-ge4", "step-ge3", "NONEXISTENT_KEY")
	assert.False(t, ok)
	assert.Empty(t, val)
}

// --- baseRunner.GenEnv ---

func TestBaseRunner_GenEnv_BuildNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	err := r.GenEnv("nonexistent", "job-1", utils.EnvVal{"KEY": "val"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBuildNotFound))
}

func TestBaseRunner_GenEnv_JobNotFound(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-gen1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	err := r.GenEnv("build-gen1", "nonexistent-job", utils.EnvVal{"KEY": "val"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobNotFound))
}

func TestBaseRunner_GenEnv_Success(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	job := &jobSync{
		step:  &runtime.Step{Id: "step-gen", Name: "test-step", BuildId: "build-gen2"},
		cmdmp: make(map[string]*cmdSync),
	}
	jobs := map[string]*jobSync{"step-gen": job}
	_, cleanup := setupTestBuildEngine(t, "build-gen2", make(map[string]*taskStage), jobs)
	defer cleanup()

	// Pre-create the directory that GenEnv expects
	dir := filepath.Join(tmpDir, common.PathBuild, "build-gen2", common.PathJobs, "step-gen")
	require.NoError(t, os.MkdirAll(dir, 0750))

	r := &baseRunner{}
	env := utils.EnvVal{
		"MY_VAR":    "hello",
		"BUILD_NUM": "42",
	}
	err := r.GenEnv("build-gen2", "step-gen", env)
	require.NoError(t, err)

	// Verify the env file was created
	envPath := filepath.Join(dir, "build.env")
	data, err := os.ReadFile(envPath) //nolint:gosec
	require.NoError(t, err)
	assert.Contains(t, string(data), "MY_VAR")
	assert.Contains(t, string(data), "hello")
}

// --- baseRunner.PullJob ---

func TestBaseRunner_PullJob_Timeout(t *testing.T) {
	// Set up an empty job engine so PullJob times out
	origJobEgn := Mgr.jobEgn
	Mgr.jobEgn = &JobEngine{
		execs: make(map[string]*executer),
		jobs:  make(map[string]*jobSync),
	}
	defer func() { Mgr.jobEgn = origJobEgn }()

	r := &baseRunner{}
	start := time.Now()
	_, err := r.PullJob("test-runner", []string{"plugin-x"})
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrJobNotFound))
	// Should have waited ~5 seconds
	assert.GreaterOrEqual(t, elapsed.Seconds(), 4.5)
}

// --- Concurrent access tests ---

func TestBaseRunner_ConcurrentReadDir(t *testing.T) {
	tmpDir := t.TempDir()
	bt, cleanup := setupTestBuildEngine(t, "build-cc1", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()
	bt.repoPaths = tmpDir
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f1.txt"), []byte("a"), 0600))

	r := &baseRunner{}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			entries, err := r.ReadDir(1, "build-cc1", ".")
			if err == nil {
				_ = len(entries)
			}
		}()
	}
	wg.Wait()
}

func TestBaseRunner_ConcurrentCheckCancel(t *testing.T) {
	_, cleanup := setupTestBuildEngine(t, "build-cc2", make(map[string]*taskStage), make(map[string]*jobSync))
	defer cleanup()

	r := &baseRunner{}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.CheckCancel("build-cc2")
			_ = r.CheckCancel("nonexistent")
		}()
	}
	wg.Wait()
}
