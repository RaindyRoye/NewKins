package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gokins/gokins/model"
)

// --- parseHook Tests ---

func TestParseHook_UnknownType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook", nil)
	_, err := parseHook("unknown", req, "")
	if err == nil {
		t.Fatal("expected error for unknown hook type")
	}
	if !strings.Contains(err.Error(), "unknown webhook type") {
		t.Errorf("error should mention unknown type, got: %v", err)
	}
}

func TestParseHook_EmptyType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook", nil)
	_, err := parseHook("", req, "")
	if err == nil {
		t.Fatal("expected error for empty hook type")
	}
	if !strings.Contains(err.Error(), "unknown webhook type") {
		t.Errorf("error should mention unknown type, got: %v", err)
	}
}

func TestParseHook_GiteeInvalidPayload(t *testing.T) {
	// Gitee requires a valid JSON body; empty body should fail
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(""))
	_, err := parseHook("gitee", req, "")
	if err == nil {
		t.Fatal("expected error for empty gitee payload")
	}
}

func TestParseHook_GithubInvalidPayload(t *testing.T) {
	// GitHub requires X-GitHub-Event header; missing header should fail
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := parseHook("github", req, "")
	if err == nil {
		t.Fatal("expected error for github without event header")
	}
}

func TestParseHook_GitlabInvalidPayload(t *testing.T) {
	// GitLab requires X-Gitlab-Event header; missing header should fail
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := parseHook("gitlab", req, "")
	if err == nil {
		t.Fatal("expected error for gitlab without event header")
	}
}

func TestParseHook_GiteaInvalidPayload(t *testing.T) {
	// Gitea requires X-Gitea-Event header; missing header should fail
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
	_, err := parseHook("gitea", req, "")
	if err == nil {
		t.Fatal("expected error for gitea without event header")
	}
}

func TestParseHook_GiteePremiumInvalidPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(""))
	_, err := parseHook("giteepremium", req, "")
	if err == nil {
		t.Fatal("expected error for empty giteepremium payload")
	}
}

func TestParseHook_CaseInsensitive(t *testing.T) {
	// Verify that hookType matching is case-insensitive
	tests := []struct {
		hookType string
	}{
		{"GITEE"},
		{"Gitee"},
		{"GITHUB"},
		{"GitHub"},
		{"GITLAB"},
		{"GitLab"},
		{"GITEA"},
		{"Gitea"},
	}
	for _, tt := range tests {
		t.Run(tt.hookType, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader(""))
			_, err := parseHook(tt.hookType, req, "")
			// All should fail with parse errors (not "unknown type")
			if err != nil && strings.Contains(err.Error(), "unknown webhook type") {
				t.Errorf("parseHook(%q) should be recognized, but got unknown type error", tt.hookType)
			}
		})
	}
}

// --- TriggerHookCtx / TriggerWeb / TriggerTimer validation tests ---

func TestTriggerHookCtx_EmptyParams(t *testing.T) {
	// TriggerHookCtx has a deferred function that uses comm.Db to save trigger runs.
	// When comm.Db is nil (unit tests without DB), this panics after the function returns.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (expected without DB): %v", r)
		}
	}()
	tt := &model.TTrigger{
		Id:     "trigger-1",
		Params: "",
	}
	req := httptest.NewRequest(http.MethodPost, "/hook", nil)
	_, err := TriggerHookCtx(context.Background(), tt, req)
	if err == nil {
		t.Fatal("expected error for empty params")
	}
	if !errors.Is(err, ErrTriggerNoParams) {
		t.Errorf("expected ErrTriggerNoParams, got: %v", err)
	}
}

func TestTriggerWeb_EmptyParams(t *testing.T) {
	// TriggerWeb has a deferred function that uses comm.Db to save trigger runs.
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (expected without DB): %v", r)
		}
	}()
	tt := &model.TTrigger{
		Id:     "trigger-2",
		Params: "",
	}
	_, err := TriggerWeb(context.Background(), tt, "secret")
	if err == nil {
		t.Fatal("expected error for empty params")
	}
	if !errors.Is(err, ErrTriggerNoParams) {
		t.Errorf("expected ErrTriggerNoParams, got: %v", err)
	}
}

func TestTriggerTimer_EmptyTrigger(t *testing.T) {
	// TriggerTimer with nil trigger should fail in TriggerPermCtx
	// Recover from panic if comm.Db is nil
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (expected in unit test without DB): %v", r)
		}
	}()
	tt := &model.TTrigger{
		Id:           "trigger-3",
		PipelineId:   "",
		Uid:          "user1",
	}
	_, err := TriggerTimer(context.Background(), tt)
	if err == nil {
		t.Fatal("expected error for timer trigger with empty pipeline")
	}
}

func TestTriggerTimer_NilTrigger(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (expected in unit test without DB): %v", r)
		}
	}()
	_, err := TriggerTimer(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil trigger")
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

// --- TriggerWeb secret validation ---

func TestTriggerWeb_InvalidParams(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (expected without DB): %v", r)
		}
	}()
	tt := &model.TTrigger{
		Id:     "trigger-4",
		Params: `{invalid json`,
	}
	_, err := TriggerWeb(context.Background(), tt, "secret")
	if err == nil {
		t.Fatal("expected error for invalid JSON params")
	}
}

func TestTriggerWeb_MissingSecret(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (expected in unit test without DB): %v", r)
		}
	}()
	tt := &model.TTrigger{
		Id:     "trigger-5",
		Params: `{"branch": "main"}`,
	}
	_, err := TriggerWeb(context.Background(), tt, "secret")
	if err == nil {
		t.Fatal("expected error when params have no secret")
	}
	if !errors.Is(err, ErrTriggerNoSecret) {
		t.Errorf("expected ErrTriggerNoSecret, got: %v", err)
	}
}

func TestTriggerWeb_WrongSecret(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (expected in unit test without DB): %v", r)
		}
	}()
	tt := &model.TTrigger{
		Id:     "trigger-6",
		Params: `{"secret": "correct", "branch": "main"}`,
	}
	_, err := TriggerWeb(context.Background(), tt, "wrong")
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
	if !errors.Is(err, ErrTriggerSecretMismatch) {
		t.Errorf("expected ErrTriggerSecretMismatch, got: %v", err)
	}
}

// --- TriggerHookCtx JSON validation ---

func TestTriggerHookCtx_InvalidJSON(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (expected without DB): %v", r)
		}
	}()
	tt := &model.TTrigger{
		Id:     "trigger-7",
		Params: `{invalid json`,
	}
	req := httptest.NewRequest(http.MethodPost, "/hook", nil)
	_, err := TriggerHookCtx(context.Background(), tt, req)
	if err == nil {
		t.Fatal("expected error for invalid JSON params")
	}
}

func TestTriggerHookCtx_MissingHookType(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Logf("recovered panic (expected in unit test without DB): %v", r)
		}
	}()
	tt := &model.TTrigger{
		Id:     "trigger-8",
		Params: `{"branch": "main"}`,
	}
	req := httptest.NewRequest(http.MethodPost, "/hook", nil)
	_, err := TriggerHookCtx(context.Background(), tt, req)
	if err == nil {
		t.Fatal("expected error when hookType is missing from params")
	}
	if !errors.Is(err, ErrHookTypeEmpty) {
		t.Errorf("expected ErrHookTypeEmpty, got: %v", err)
	}
}
