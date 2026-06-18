package gitee

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

const testSecret = "testsecret"

func newGiteeRequest(event string, body []byte, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GiteeEvent, event)
	if secret != "" {
		req.Header.Set("X-Gitee-Token", secret)
	}
	return req
}

func TestParsePushHook(t *testing.T) {
	payload := map[string]any{
		"ref":    "refs/heads/main",
		"before": "abc123",
		"after":  "def456",
		"commits": []map[string]any{
			{
				"id":      "def456",
				"message": "test commit",
				"url":     "https://gitee.com/user/test-repo/commit/def456",
			},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "test-repo",
			"full_name": "user/test-repo",
			"html_url":  "https://gitee.com/user/test-repo",
			"ssh_url":   "git@gitee.com:user/test-repo.git",
			"url":       "https://gitee.com/user/test-repo",
		},
		"sender": map[string]any{
			"login": "testuser",
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteeRequest(hook.GiteeEventPush, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse push hook failed: %v", err)
	}

	pushHook, ok := wh.(*hook.PushHook)
	if !ok {
		t.Fatal("expected *hook.PushHook type")
	}

	if pushHook.Ref != "refs/heads/main" {
		t.Errorf("expected ref 'refs/heads/main', got '%s'", pushHook.Ref)
	}
	if pushHook.Before != "abc123" {
		t.Errorf("expected before 'abc123', got '%s'", pushHook.Before)
	}
	if pushHook.After != "def456" {
		t.Errorf("expected after 'def456', got '%s'", pushHook.After)
	}
	if pushHook.Repo.Name != "test-repo" {
		t.Errorf("expected repo name 'test-repo', got '%s'", pushHook.Repo.Name)
	}
	if pushHook.Repo.RepoType != "gitee" {
		t.Errorf("expected repoType 'gitee', got '%s'", pushHook.Repo.RepoType)
	}
}

func TestParsePullRequestHook(t *testing.T) {
	payload := map[string]any{
		"action": "open",
		"iid":    42,
		"title":  "Test PR",
		"head": map[string]any{
			"ref": "feature-branch",
			"sha": "head123",
			"repo": map[string]any{
				"id":        2,
				"name":      "test-repo-fork",
				"full_name": "forker/test-repo",
				"html_url":  "https://gitee.com/forker/test-repo",
			},
		},
		"base": map[string]any{
			"ref": "main",
			"sha": "base123",
			"repo": map[string]any{
				"id":        1,
				"name":      "test-repo",
				"full_name": "user/test-repo",
				"html_url":  "https://gitee.com/user/test-repo",
			},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "test-repo",
			"full_name": "user/test-repo",
			"html_url":  "https://gitee.com/user/test-repo",
		},
		"sender": map[string]any{
			"login": "prauthor",
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteeRequest(hook.GiteeEventPR, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse PR hook failed: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestHook type")
	}

	if prHook.Action != hook.ActionOpen {
		t.Errorf("expected action '%s', got '%s'", hook.ActionOpen, prHook.Action)
	}
	if prHook.Repo.RepoType != "gitee" {
		t.Errorf("expected repoType 'gitee', got '%s'", prHook.Repo.RepoType)
	}
}

func TestParseUnknownEvent(t *testing.T) {
	body := []byte(`{}`)
	secret := testSecret
	req := newGiteeRequest("Unknown Hook", body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestParseInvalidSecret(t *testing.T) {
	payload := map[string]any{
		"ref":    "refs/heads/main",
		"before": "abc",
		"after":  "def",
		"commits": []map[string]any{
			{"id": "def", "message": "test"},
		},
		"repository": map[string]any{},
	}
	body, _ := json.Marshal(payload)
	req := newGiteeRequest(hook.GiteeEventPush, body, "correctsecret")

	_, err := Parse(req, "wrongsecret")
	if err == nil {
		t.Fatal("expected error for invalid secret")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	req := newGiteeRequest(hook.GiteeEventPush, body, "secret")

	_, err := Parse(req, "secret")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParsePRUnsupportedAction(t *testing.T) {
	payload := map[string]any{
		"action": "close",
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteeRequest(hook.GiteeEventPR, body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for unsupported PR action 'close'")
	}
}
