package engine

import (
	"errors"
	"testing"
)

// TestSentinelErrors verifies that all sentinel errors are properly defined
// and can be used with errors.Is for error matching.
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrBuildNotFound", ErrBuildNotFound},
		{"ErrJobNotFound", ErrJobNotFound},
		{"ErrCmdNotFound", ErrCmdNotFound},
		{"ErrInvalidFSType", ErrInvalidFSType},
		{"ErrEmptyParams", ErrEmptyParams},
		{"ErrArtifactoryNotFound", ErrArtifactoryNotFound},
		{"ErrArtifactNotFound", ErrArtifactNotFound},
		{"ErrPermissionDenied", ErrPermissionDenied},
		{"ErrPluginNotFound", ErrPluginNotFound},
		{"ErrInvalidTriggerType", ErrInvalidTriggerType},
		{"ErrArtifactoryDisabled", ErrArtifactoryDisabled},
		{"ErrUnknownWebhookType", ErrUnknownWebhookType},
		{"ErrAssetNotFound", ErrAssetNotFound},
		{"ErrInvalidConfig", ErrInvalidConfig},
		{"ErrDuplicateEntry", ErrDuplicateEntry},
		{"ErrRepositoryNil", ErrRepositoryNil},
		{"ErrRepoNil", ErrRepoNil},
		{"ErrStagesEmpty", ErrStagesEmpty},
		{"ErrStageBuildIdMismatch", ErrStageBuildIdMismatch},
		{"ErrStageNameEmpty", ErrStageNameEmpty},
		{"ErrStageDuplicate", ErrStageDuplicate},
		{"ErrJobBuildIdMismatch", ErrJobBuildIdMismatch},
		{"ErrJobStageIdMismatch", ErrJobStageIdMismatch},
		{"ErrStepPluginEmpty", ErrStepPluginEmpty},
		{"ErrStepNameEmpty", ErrStepNameEmpty},
		{"ErrJobDuplicate", ErrJobDuplicate},
		{"ErrJobCommands", ErrJobCommands},
		{"ErrStageNotFound", ErrStageNotFound},
		{"ErrWaitNotFound", ErrWaitNotFound},
		{"ErrWaitSelf", ErrWaitSelf},
		{"ErrWaitFailed", ErrWaitFailed},
		{"ErrJobPut", ErrJobPut},
		{"ErrCreateBuildPath", ErrCreateBuildPath},
		{"ErrCloneRepo", ErrCloneRepo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
			}

			// Verify that errors.Is works with the sentinel error
			wrapped := errors.Join(errors.New("context"), tt.err)
			if !errors.Is(wrapped, tt.err) {
				t.Errorf("errors.Is(%v, %v) = false, want true", wrapped, tt.err)
			}

			// Verify error message is not empty
			if tt.err.Error() == "" {
				t.Errorf("%s has empty error message", tt.name)
			}
		})
	}
}

// TestSentinelErrorWrapping verifies that wrapped errors can be unwrapped correctly.
func TestSentinelErrorWrapping(t *testing.T) {
	// Test wrapping with additional context
	baseErr := ErrBuildNotFound
	wrapped := errors.Join(errors.New("operation failed"), baseErr)

	if !errors.Is(wrapped, baseErr) {
		t.Errorf("wrapped error should match sentinel error")
	}

	// Test that error messages are preserved
	msg := wrapped.Error()
	if msg == "" {
		t.Errorf("wrapped error message should not be empty")
	}
}
