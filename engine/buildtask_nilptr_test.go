package engine

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/core/runtime"
)

// TestBuildTask_RunStage_MissingStageNoPanic verifies that runStage doesn't panic
// when the stage name is not found in the stages map.
func TestBuildTask_RunStage_MissingStageNoPanic(t *testing.T) {
	bt := &BuildTask{
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
		ctx:    context.Background(),
	}

	// Create a stage that doesn't exist in the map
	stage := &runtime.Stage{
		Name: "non-existent-stage",
	}

	// This should not panic even though the stage is not in the map
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runStage panicked: %v", r)
		}
	}()

	bt.runStage(stage)

	// Verify the stage status was set to error
	if stage.Status != "error" {
		t.Errorf("expected status 'error', got %s", stage.Status)
	}
	if stage.Error == "" {
		t.Error("expected error message to be set")
	}
}

// TestBuildTask_RunStage_MissingJobNoPanic verifies that runStage doesn't panic
// when a job name is not found in the jobs map.
func TestBuildTask_RunStage_MissingJobNoPanic(t *testing.T) {
	bt := &BuildTask{
		stages: make(map[string]*taskStage),
		jobs:   make(map[string]*jobSync),
		ctx:    context.Background(),
	}

	// Create a stage with a job that doesn't exist
	stage := &runtime.Stage{
		Name: "test-stage",
		Steps: []*runtime.Step{
			{Name: "non-existent-job"},
		},
	}

	// Add the stage to the map
	stg := &taskStage{
		stage: &runtime.Stage{Name: "test-stage"},
		jobs:  make(map[string]*jobSync),
	}
	bt.stages["test-stage"] = stg

	// This should not panic even though the job is not in the map
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("runStage panicked: %v", r)
		}
	}()

	bt.runStage(stage)

	// Give a moment for any async operations
	time.Sleep(10 * time.Millisecond)

	// Verify the taskStage status was set to error
	stg.RLock()
	defer stg.RUnlock()
	if stg.stage.Status != "error" {
		t.Errorf("expected taskStage status 'error', got %s", stg.stage.Status)
	}
	if stg.stage.Error == "" {
		t.Error("expected error message to be set")
	}
}
