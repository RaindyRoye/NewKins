package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/comm"
)

func TestBuildTaskClears_NonClone(t *testing.T) {
	tmpDir := t.TempDir()
	bt := &BuildTask{
		build:     &runtime.Build{Id: "build-clear-1"},
		buildPath: tmpDir,
		isClone:   false,
		repoPaths: filepath.Join(tmpDir, "repo"),
		jobs:      make(map[string]*jobSync),
	}
	// Should not panic
	bt.clears()
}

func TestBuildTaskClears_Clone(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoPath, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	bt := &BuildTask{
		build:     &runtime.Build{Id: "build-clear-2"},
		buildPath: tmpDir,
		isClone:   true,
		repoPaths: repoPath,
		jobs:      make(map[string]*jobSync),
	}
	bt.clears()

	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		t.Errorf("repo path should be removed, stat err: %v", err)
	}
}

func TestBuildTaskClears_WithJobs(t *testing.T) {
	tmpDir := t.TempDir()
	jobsPath := filepath.Join(tmpDir, common.PathJobs)
	jobId := "step-123"
	artsPath := filepath.Join(jobsPath, jobId, common.PathArts)
	if err := os.MkdirAll(artsPath, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	testFile := filepath.Join(artsPath, "artifact.tar.gz")
	if err := os.WriteFile(testFile, []byte("data"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	job := &jobSync{
		step: &runtime.Step{Id: jobId, Name: "test-step"},
	}

	bt := &BuildTask{
		build:     &runtime.Build{Id: "build-clear-3"},
		buildPath: tmpDir,
		isClone:   false,
		jobs: map[string]*jobSync{
			jobId: job,
		},
	}
	bt.clears()

	if _, err := os.Stat(artsPath); !os.IsNotExist(err) {
		t.Errorf("artifacts path should be removed, stat err: %v", err)
	}
}

func TestBuildTaskRun_NilRepo(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	bt := &BuildTask{
		build: &runtime.Build{
			Id:     "build-run-1",
			Repo:   nil,
			Stages: []*runtime.Stage{},
		},
	}
	bt.run()

	if bt.build.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", bt.build.Status, common.BuildStatusError)
	}
	// Should have set Finished time
	if bt.build.Finished.IsZero() {
		t.Error("Finished should be set")
	}
	// buildPath should have been created and then cleared
	// (clears is called in defer)
}

func TestBuildTaskRun_EmptyStages(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	bt := &BuildTask{
		build: &runtime.Build{
			Id: "build-run-2",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{},
		},
	}
	bt.run()

	if bt.build.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", bt.build.Status, common.BuildStatusError)
	}
	if bt.build.Error != "build Stages is empty" {
		t.Errorf("error = %q, want %q", bt.build.Error, "build Stages is empty")
	}
}

func TestBuildTaskRun_PathCreationError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping as root (permissions tests won't work)")
	}
	tmpDir := t.TempDir()
	// Create a file where the build path would go
	buildPath := filepath.Join(tmpDir, common.PathBuild, "build-run-3")
	if err := os.MkdirAll(filepath.Dir(buildPath), 0750); err != nil {
		t.Fatalf("mkdir parent: %v", err)
	}
	if err := os.WriteFile(buildPath, []byte("block"), 0644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	bt := &BuildTask{
		build: &runtime.Build{
			Id: "build-run-3",
			Repo: &runtime.Repository{
				CloneURL: "",
			},
			Stages: []*runtime.Stage{
				{Id: "s1", Name: "build", BuildId: "build-run-3"},
			},
		},
	}
	bt.run()

	if bt.build.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", bt.build.Status, common.BuildStatusError)
	}
	if bt.build.Event != common.BuildEventPath {
		t.Errorf("event = %q, want %q", bt.build.Event, common.BuildEventPath)
	}
}

func TestBuildTaskRun_SuccessEmptyClone(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	// A build with a valid local path as CloneURL (existing dir), no stages
	// should succeed through check but have no stages to run.
	localRepo := t.TempDir()

	bt := &BuildTask{
		build: &runtime.Build{
			Id: "build-run-4",
			Repo: &runtime.Repository{
				CloneURL: localRepo,
			},
			Stages: []*runtime.Stage{},
		},
	}
	// This will fail at check() because Stages is empty
	bt.run()

	if bt.build.Status != common.BuildStatusError {
		t.Errorf("status = %q, want %q", bt.build.Status, common.BuildStatusError)
	}
	// Started should be set before check fails
	if bt.build.Started.IsZero() {
		t.Error("Started should be set")
	}
}

func TestBuildTaskRun_WithValidStage(t *testing.T) {
	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	defer func() { comm.WorkPath = origWorkPath }()

	// Use existing dir as clone path
	localRepo := t.TempDir()

	stage := &runtime.Stage{
		Id:      "stage-1",
		BuildId: "build-run-5",
		Name:    "build",
		Steps:   []*runtime.Step{},
	}

	bt := &BuildTask{
		build: &runtime.Build{
			Id: "build-run-5",
			Repo: &runtime.Repository{
				CloneURL: localRepo,
			},
			Stages: []*runtime.Stage{stage},
		},
	}
	// This should get past check (local path exists), but runStage will
	// complete quickly with no steps.
	before := time.Now()
	bt.run()
	after := time.Now()

	// run should complete (may succeed or error depending on stage handling)
	if bt.build.Finished.IsZero() {
		t.Error("Finished should be set")
	}
	if bt.build.Finished.Before(before) || bt.build.Finished.After(after.Add(time.Second)) {
		t.Errorf("Finished time %v is outside expected range", bt.build.Finished)
	}
}
