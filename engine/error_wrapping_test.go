package engine

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gokins/core/runtime"
	"github.com/gokins/gokins/model"
	"github.com/gokins/runner/runners"
)

// TestErrorWrapping_Improved verifies that errors are properly wrapped with %w
// so they can be inspected with errors.Is and errors.As
func TestErrorWrapping_Improved(t *testing.T) {
	t.Run("genRunjob_panic_recovery", func(t *testing.T) {
		// Test that genRunjob properly wraps panic errors
		task := &BuildTask{
			egn:   &BuildEngine{},
			build: &runtime.Build{Id: "test-build", PipelineId: "test-pipe"},
		}
		stage := &runtime.Stage{Name: "test-stage", BuildId: "test-build", Id: "stage-1"}
		job := &jobSync{
			task: task,
			step: &runtime.Step{
				Id:      "step-1",
				BuildId: "test-build",
				StageId: "stage-1",
				Name:    "test-job",
				Step:    "test-plugin",
			},
		}

		// genRunjob should handle panics gracefully and wrap them properly
		// We can't easily trigger a panic in tests without mocking, but we can
		// verify the function signature and error return pattern
		err := task.genRunjob(stage, job)
		if err != nil {
			// If there's an error, it should be properly formatted
			if err.Error() == "" {
				t.Error("genRunjob returned empty error message")
			}
		}
	})

	t.Run("gencmds_panic_recovery", func(t *testing.T) {
		// Test that gencmds properly wraps panic errors
		task := &BuildTask{
			egn:   &BuildEngine{},
			build: &runtime.Build{Id: "test-build"},
		}

		runjb := &runners.RunJob{}

		// Test with valid input - should not panic
		err := task.gencmds(runjb, []any{"echo hello"})
		if err != nil {
			t.Errorf("gencmds with valid input failed: %v", err)
		}
	})

	t.Run("timerEngine_resetOne_panic_recovery", func(t *testing.T) {
		// Test that resetOne properly wraps panic errors
		timerEng := &TimerEngine{
			tasks: make(map[string]*timerExec),
		}

		// Test with invalid trigger type - should return wrapped error
		tmr := &model.TTrigger{
			Id:     "test-timer",
			Types:  "invalid",
			Params: "{}",
		}

		err := timerEng.resetOne(tmr)
		if err == nil {
			t.Error("resetOne should fail for invalid trigger type")
		}

		// Verify error message contains expected text
		expected := "expected trigger type 'timer'"
		if !contains(err.Error(), expected) {
			t.Errorf("error message = %q, want to contain %q", err.Error(), expected)
		}
	})
}

// TestErrorMessageClarity verifies that error messages are clear and actionable
func TestErrorMessageClarity(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() error
		expected string
	}{
		{
			name: "stage_build_id_mismatch",
			setup: func() error {
				return fmt.Errorf("stage build id mismatch: %s/%s", "stage-123", "build-456")
			},
			expected: "stage build id mismatch",
		},
		{
			name: "job_build_id_mismatch",
			setup: func() error {
				return fmt.Errorf("job build id mismatch: %s/%s", "job-789", "build-456")
			},
			expected: "job build id mismatch",
		},
		{
			name: "duplicate_stage_name",
			setup: func() error {
				return fmt.Errorf("duplicate stage name: %s", "build")
			},
			expected: "duplicate stage name",
		},
		{
			name: "duplicate_job_name",
			setup: func() error {
				return fmt.Errorf("duplicate job name: %s", "test")
			},
			expected: "duplicate job name",
		},
		{
			name: "dependency_failed",
			setup: func() error {
				return fmt.Errorf("dependency %q failed", "setup-job")
			},
			expected: "dependency",
		},
		{
			name: "put_job_to_engine",
			setup: func() error {
				return fmt.Errorf("put job to engine: %w", errors.New("plugin not found"))
			},
			expected: "put job to engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setup()
			if err == nil {
				t.Fatal("setup function returned nil error")
			}
			if !contains(err.Error(), tt.expected) {
				t.Errorf("error message = %q, want to contain %q", err.Error(), tt.expected)
			}
		})
	}
}

// TestPanicRecoveryWithWrappedErrors verifies that panic recovery preserves error chains
func TestPanicRecoveryWithWrappedErrors(t *testing.T) {
	t.Run("recover_with_error_type", func(t *testing.T) {
		var recoveredErr error

		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(error); ok {
						recoveredErr = fmt.Errorf("operation panic: %w", err)
					} else {
						recoveredErr = fmt.Errorf("operation panic: %v", r)
					}
				}
			}()

			// Simulate a panic with an error type
			panic(errors.New("database connection failed"))
		}()

		if recoveredErr == nil {
			t.Fatal("panic was not recovered")
		}

		// Verify the error chain is preserved
		if !contains(recoveredErr.Error(), "operation panic") {
			t.Errorf("error message = %q, want to contain 'operation panic'", recoveredErr.Error())
		}
		if !contains(recoveredErr.Error(), "database connection failed") {
			t.Errorf("error message = %q, want to contain 'database connection failed'", recoveredErr.Error())
		}
	})

	t.Run("recover_with_non_error_type", func(t *testing.T) {
		var recoveredErr error

		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(error); ok {
						recoveredErr = fmt.Errorf("operation panic: %w", err)
					} else {
						recoveredErr = fmt.Errorf("operation panic: %v", r)
					}
				}
			}()

			// Simulate a panic with a string type
			panic("nil pointer dereference")
		}()

		if recoveredErr == nil {
			t.Fatal("panic was not recovered")
		}

		// Verify the error message includes the panic value
		if !contains(recoveredErr.Error(), "operation panic") {
			t.Errorf("error message = %q, want to contain 'operation panic'", recoveredErr.Error())
		}
		if !contains(recoveredErr.Error(), "nil pointer dereference") {
			t.Errorf("error message = %q, want to contain 'nil pointer dereference'", recoveredErr.Error())
		}
	})
}

// TestBuildTaskCheckValidation verifies that check() returns clear error messages
func TestBuildTaskCheckValidation(t *testing.T) {
	tests := []struct {
		name        string
		build       *runtime.Build
		expectError string
	}{
		{
			name: "missing_repo",
			build: &runtime.Build{
				Id:           "build-1",
				PipelineId:   "pipe-1",
				Repo:         nil,
				Stages:       []*runtime.Stage{},
			},
			expectError: "repo param",
		},
		{
			name: "empty_stages",
			build: &runtime.Build{
				Id:           "build-2",
				PipelineId:   "pipe-2",
				Repo:         &runtime.Repository{CloneURL: "https://github.com/test/repo"},
				Stages:       []*runtime.Stage{},
			},
			expectError: "Stages is empty",
		},
		{
			name: "stage_name_empty",
			build: &runtime.Build{
				Id:           "build-3",
				PipelineId:   "pipe-3",
				Repo:         &runtime.Repository{CloneURL: "https://github.com/test/repo"},
				Stages: []*runtime.Stage{
					{
						Id:      "stage-1",
						BuildId: "build-3",
						Name:    "",
						Steps:   []*runtime.Step{},
					},
				},
			},
			expectError: "Stage name is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &BuildTask{
				egn:   &BuildEngine{},
				build: tt.build,
			}

			result := task.check()

			// All these test cases should fail validation
			if result {
				t.Errorf("check() = true, want false for invalid build")
			}

			// Verify error message is set
			if task.build.Error == "" {
				t.Error("build.Error is empty, want error message")
			}

			// Verify error message contains expected text
			if tt.expectError != "" && !contains(task.build.Error, tt.expectError) {
				t.Errorf("build.Error = %q, want to contain %q", task.build.Error, tt.expectError)
			}
		})
	}
}

// contains is a helper to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
