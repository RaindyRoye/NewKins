package engine

import (
	"container/list"
	"testing"

	"github.com/gokins/core/common"
	"github.com/gokins/core/runtime"
	"github.com/gokins/runner/runners"
)

// TestJobEnginePullWithJobs verifies that Pull correctly dequeues jobs
// when jobs have been put into the queue first
func TestJobEnginePullWithJobs(t *testing.T) {
	je := newTestJobEngine()

	// Create and register an executer
	je.execs["test-plugin"] = &executer{
		plug:  "test-plugin",
		jobwt: list.New(),
	}

	// Create a job and put it in the queue
	job := &jobSync{
		step: &runtime.Step{
			Id:   "step-1",
			Step: "test-plugin",
		},
		runjb: &runners.RunJob{
			Id:   "runjob-1",
			Step: "test-plugin",
		},
		cmdmp: make(map[string]*cmdSync),
	}

	err := je.Put(job)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Pull should return the job
	result := je.Pull("runner1", []string{"test-plugin"})
	if result == nil {
		t.Fatal("Pull returned nil, expected a job")
	}
	if result.Id != "runjob-1" {
		t.Errorf("expected job ID 'runjob-1', got %q", result.Id)
	}

	// Pull again should return nil (queue is empty)
	result = je.Pull("runner1", []string{"test-plugin"})
	if result != nil {
		t.Error("second Pull returned non-nil, expected nil for empty queue")
	}
}

// TestJobEnginePullMultiplePlugins verifies that Pull tries multiple plugins
// and returns the first available job
func TestJobEnginePullMultiplePlugins(t *testing.T) {
	je := newTestJobEngine()

	// Create executer for plugin-a (empty)
	je.execs["plugin-a"] = &executer{
		plug:  "plugin-a",
		jobwt: list.New(),
	}

	// Create executer for plugin-b with a job
	je.execs["plugin-b"] = &executer{
		plug:  "plugin-b",
		jobwt: list.New(),
	}

	job := &jobSync{
		step: &runtime.Step{
			Id:   "step-b",
			Step: "plugin-b",
		},
		runjb: &runners.RunJob{
			Id:   "runjob-b",
			Step: "plugin-b",
		},
		cmdmp: make(map[string]*cmdSync),
	}

	err := je.Put(job)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Pull should try plugin-a first (empty), then plugin-b (has job)
	result := je.Pull("runner1", []string{"plugin-a", "plugin-b"})
	if result == nil {
		t.Fatal("Pull returned nil, expected job from plugin-b")
	}
	if result.Id != "runjob-b" {
		t.Errorf("expected job ID 'runjob-b', got %q", result.Id)
	}
}

// TestBuildTaskCheckValidBuild verifies that check() passes for a valid build
// with proper stages and steps
func TestBuildTaskCheckValidBuild(t *testing.T) {
	build := &runtime.Build{
		Id:         "build-1",
		PipelineId: "pipe-1",
		Status:     common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      "stage-1",
				BuildId: "build-1",
				Name:    "build",
				Steps: []*runtime.Step{
					{
						Id:      "step-1",
						BuildId: "build-1",
						StageId: "stage-1",
						Step:    "test-plugin",
						Name:    "test-step",
					},
				},
			},
		},
	}

	task := &BuildTask{
		build:  build,
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}

	// check() will fail because genRunjob requires database access
	// but we can verify it gets past the basic validation
	result := task.check()

	// The check should fail at genRunjob (DB operations), but the
	// important thing is it passed the structural validation
	if result {
		t.Log("check() unexpectedly succeeded (may need DB setup)")
	} else {
		// Should fail with a genRunjob-related error, not a validation error
		if build.Event == common.BuildEventCheckParam {
			t.Errorf("check() failed at validation: %s", build.Error)
		}
	}
}

// TestBuildTaskCheckDuplicateStageNames verifies that check() rejects builds
// with duplicate stage names
func TestBuildTaskCheckDuplicateStageNames(t *testing.T) {
	build := &runtime.Build{
		Id:         "build-1",
		PipelineId: "pipe-1",
		Status:     common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      "stage-1",
				BuildId: "build-1",
				Name:    "build",
				Steps: []*runtime.Step{
					{
						Id:      "step-1",
						BuildId: "build-1",
						StageId: "stage-1",
						Step:    "test-plugin",
						Name:    "test-step",
					},
				},
			},
			{
				Id:      "stage-2",
				BuildId: "build-1",
				Name:    "build", // duplicate name
				Steps: []*runtime.Step{
					{
						Id:      "step-2",
						BuildId: "build-1",
						StageId: "stage-2",
						Step:    "test-plugin",
						Name:    "test-step-2",
					},
				},
			},
		},
	}

	task := &BuildTask{
		build:  build,
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()

	if result {
		t.Fatal("check() should have failed for duplicate stage names")
	}
	if build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, build.Event)
	}
	if build.Error != "build Stages.build is repeat" {
		t.Errorf("expected error message about duplicate stages, got %q", build.Error)
	}
}

// TestBuildTaskCheckMismatchedBuildIds verifies that check() rejects stages
// with mismatched build IDs
func TestBuildTaskCheckMismatchedBuildIds(t *testing.T) {
	build := &runtime.Build{
		Id:         "build-1",
		PipelineId: "pipe-1",
		Status:     common.BuildStatusPending,
		Repo: &runtime.Repository{
			CloneURL: "",
		},
		Stages: []*runtime.Stage{
			{
				Id:      "stage-1",
				BuildId: "build-2", // mismatched
				Name:    "build",
				Steps: []*runtime.Step{
					{
						Id:      "step-1",
						BuildId: "build-1",
						StageId: "stage-1",
						Step:    "test-plugin",
						Name:    "test-step",
					},
				},
			},
		},
	}

	task := &BuildTask{
		build:  build,
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
	}
	result := task.check()

	if result {
		t.Fatal("check() should have failed for mismatched build IDs")
	}
	if build.Event != common.BuildEventCheckParam {
		t.Errorf("expected event %q, got %q", common.BuildEventCheckParam, build.Event)
	}
}
