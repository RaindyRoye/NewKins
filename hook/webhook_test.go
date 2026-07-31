package hook

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"ErrSecretValidationFailed wraps itself", ErrSecretValidationFailed, ErrSecretValidationFailed},
		{"ErrCommentNotPullRequest wraps itself", ErrCommentNotPullRequest, ErrCommentNotPullRequest},
		{"ErrUnsupportedEvent wraps itself", ErrUnsupportedEvent, ErrUnsupportedEvent},
		{"ErrUnsupportedAction wraps itself", ErrUnsupportedAction, ErrUnsupportedAction},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.err, tt.want)
			}
		})
	}
}

func TestSentinelErrorsAreDistinct(t *testing.T) {
	sentinels := []error{
		ErrSecretValidationFailed,
		ErrCommentNotPullRequest,
		ErrUnsupportedEvent,
		ErrUnsupportedAction,
	}
	for i := 0; i < len(sentinels); i++ {
		for j := i + 1; j < len(sentinels); j++ {
			if errors.Is(sentinels[i], sentinels[j]) {
				t.Errorf("sentinel %d and %d should be distinct but errors.Is returns true", i, j)
			}
		}
	}
}

func TestSentinelErrorMessages(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrSecretValidationFailed, "webhook secret validation failed"},
		{ErrCommentNotPullRequest, "comment is not associated with a pull request"},
		{ErrUnsupportedEvent, "unsupported webhook event type"},
		{ErrUnsupportedAction, "unsupported webhook action"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("Error() = %q, want %q", tt.err.Error(), tt.want)
			}
		})
	}
}
