package service

import (
	"errors"
	"testing"
)

func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrPipelineNotFound", ErrPipelineNotFound},
		{"ErrPipelineYmlEmpty", ErrPipelineYmlEmpty},
		{"ErrTriggerNoParams", ErrTriggerNoParams},
		{"ErrHookTypeEmpty", ErrHookTypeEmpty},
		{"ErrWebhookParseFailed", ErrWebhookParseFailed},
		{"ErrWebhookEventMismatch", ErrWebhookEventMismatch},
		{"ErrBranchMismatch", ErrBranchMismatch},
		{"ErrTriggerNoSecret", ErrTriggerNoSecret},
		{"ErrTriggerSecretMismatch", ErrTriggerSecretMismatch},
		{"ErrPermissionDenied", ErrPermissionDenied},
		{"ErrParamDataNil", ErrParamDataNil},
		{"ErrParamNotFound", ErrParamNotFound},
	}

	for i, s := range sentinels {
		if s.err == nil {
			t.Errorf("sentinel error %s is nil", s.name)
		}
		if s.err.Error() == "" {
			t.Errorf("sentinel error %s has empty message", s.name)
		}
		// Verify each error is distinct from all others
		for j, s2 := range sentinels {
			if i != j && errors.Is(s.err, s2.err) {
				t.Errorf("sentinel errors %s and %s should be distinct but errors.Is returns true", s.name, s2.name)
			}
		}
	}
}

func TestSentinelErrors_ErrorsIs(t *testing.T) {
	// Verify errors.Is works correctly with wrapped errors
	tests := []struct {
		name    string
		wrapped error
		target  error
	}{
		{"wrap PipelineNotFound", wrapErr("context: %w", ErrPipelineNotFound), ErrPipelineNotFound},
		{"wrap TriggerNoParams", wrapErr("trigger: %w", ErrTriggerNoParams), ErrTriggerNoParams},
		{"wrap HookTypeEmpty", wrapErr("hook: %w", ErrHookTypeEmpty), ErrHookTypeEmpty},
		{"wrap WebhookParseFailed", wrapErr("parse: %w", ErrWebhookParseFailed), ErrWebhookParseFailed},
		{"wrap BranchMismatch", wrapErr("branch: %w", ErrBranchMismatch), ErrBranchMismatch},
		{"wrap SecretMismatch", wrapErr("auth: %w", ErrTriggerSecretMismatch), ErrTriggerSecretMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.wrapped, tt.target) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.wrapped, tt.target)
			}
		})
	}
}

func TestSentinelErrors_ErrorMessages(t *testing.T) {
	// Verify error messages are preserved for backward compatibility
	tests := []struct {
		err  error
		want string
	}{
		{ErrPipelineNotFound, "流水线不存在"},
		{ErrPipelineYmlEmpty, "流水线Yaml为空"},
		{ErrTriggerNoParams, "触发器没有配置参数"},
		{ErrHookTypeEmpty, "hookType为空"},
		{ErrWebhookParseFailed, "webhook解析失败"},
		{ErrWebhookEventMismatch, "webhook事件不匹配"},
		{ErrBranchMismatch, "分支不匹配"},
		{ErrTriggerNoSecret, "触发器没有配置密钥"},
		{ErrTriggerSecretMismatch, "密钥不正确"},
		{ErrPermissionDenied, "触发器创建者没有权限"},
		{ErrParamDataNil, "data is nil"},
		{ErrParamNotFound, "not found param"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("error message = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSentinelErrors_DirectComparison(t *testing.T) {
	// Verify that sentinel errors can be compared directly with ==
	// Verify self-comparison with Errors.Is (gocritic: dupSubExpr)
	if !errors.Is(ErrPipelineNotFound, ErrPipelineNotFound) {
		t.Error("ErrPipelineNotFound should equal itself via errors.Is")
	}
	if ErrTriggerNoParams == ErrPipelineNotFound {
		t.Error("different sentinel errors should not be equal with ==")
	}
}

// wrapErr is a helper to create wrapped errors for testing.
func wrapErr(format string, err error) error {
	return &wrappedError{msg: format, cause: err}
}

type wrappedError struct {
	msg   string
	cause error
}

func (e *wrappedError) Error() string {
	return e.msg
}

func (e *wrappedError) Unwrap() error {
	return e.cause
}
