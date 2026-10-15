package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

// TestParseCommentHook exercises the Note Hook (comment) parsing path.
// This covers parseCommentHook and convertCommentHook which were at 0%.
func TestParseCommentHook(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"event_type":  "note",
		"user": map[string]any{
			"id":       1,
			"name":     "Test User",
			"username": "testuser",
		},
		"project_id": 1,
		"project": map[string]any{
			"id":                  1,
			"name":                "test-repo",
			"path_with_namespace": "group/test-repo",
			"web_url":             "https://gitlab.com/group/test-repo",
			"git_http_url":        "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
			"http_url":            "https://gitlab.com/group/test-repo.git",
			"ssh_url":             "git@gitlab.com:group/test-repo.git",
		},
		"object_attributes": map[string]any{
			"id":            100,
			"note":          "This is a test comment on a merge request",
			"noteable_type": "MergeRequest",
			"noteable_id":   42,
		},
		"merge_request": map[string]any{
			"id":            42,
			"iid":           10,
			"title":         "Test Merge Request",
			"source_branch": "feature-branch",
			"target_branch": "main",
			"state":         "opened",
			"source": map[string]any{
				"id":                  2,
				"name":                "test-repo-fork",
				"description":         "forked repo",
				"web_url":             "https://gitlab.com/forker/test-repo",
				"git_http_url":        "https://gitlab.com/forker/test-repo.git",
				"git_ssh_url":         "git@gitlab.com:forker/test-repo.git",
				"ssh_url":             "git@gitlab.com:forker/test-repo.git",
				"http_url":            "https://gitlab.com/forker/test-repo.git",
				"url":                 "git@gitlab.com:forker/test-repo.git",
				"path_with_namespace": "forker/test-repo",
			},
			"target": map[string]any{
				"id":                  1,
				"name":                "test-repo",
				"description":         "base repo",
				"web_url":             "https://gitlab.com/group/test-repo",
				"git_http_url":        "https://gitlab.com/group/test-repo.git",
				"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
				"ssh_url":             "git@gitlab.com:group/test-repo.git",
				"http_url":            "https://gitlab.com/group/test-repo.git",
				"url":                 "git@gitlab.com:group/test-repo.git",
				"path_with_namespace": "group/test-repo",
			},
			"last_commit": map[string]any{
				"id":      "commitsha123",
				"message": "latest commit",
			},
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/webhook",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GitlabEvent, hook.GitlabEventNote)
	req.Header.Set("X-Gitlab-Token", secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse comment hook failed: %v", err)
	}

	commentHook, ok := wh.(*hook.PullRequestCommentHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestCommentHook type")
	}

	if commentHook.Action != hook.EventsTypeComment {
		t.Errorf("expected action '%s', got '%s'", hook.EventsTypeComment, commentHook.Action)
	}
	if commentHook.Comment.Body != "This is a test comment on a merge request" {
		t.Errorf("expected comment body, got '%s'", commentHook.Comment.Body)
	}
	if commentHook.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got '%s'", commentHook.Repo.RepoType)
	}
	if commentHook.Repo.Branch != "feature-branch" {
		t.Errorf("expected branch 'feature-branch', got '%s'", commentHook.Repo.Branch)
	}
	if commentHook.TargetRepo.Branch != "main" {
		t.Errorf("expected target branch 'main', got '%s'", commentHook.TargetRepo.Branch)
	}
	if commentHook.PullRequest.Title != "Test Merge Request" {
		t.Errorf("expected PR title 'Test Merge Request', got '%s'", commentHook.PullRequest.Title)
	}
	if commentHook.Sender.UserName != "testuser" {
		t.Errorf("expected sender 'testuser', got '%s'", commentHook.Sender.UserName)
	}
}

// TestParseCommentHookInvalidJSON ensures parseCommentHook returns an error
// for malformed JSON payloads.
func TestParseCommentHookInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/webhook",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GitlabEvent, hook.GitlabEventNote)
	req.Header.Set("X-Gitlab-Token", testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment hook")
	}
}

// TestParsePRUpdateAction verifies that the "update" action is accepted
// (not treated as unsupported) in GitLab MR hooks.
func TestParsePRUpdateAction(t *testing.T) {
	payload := map[string]any{
		"object_attributes": map[string]any{
			"action":        "update",
			"iid":           5,
			"title":         "Updated MR",
			"source_branch": "feat",
			"target_branch": "main",
			"source":        map[string]any{"name": "src"},
			"target":        map[string]any{"name": "tgt"},
			"last_commit":   map[string]any{"id": "abc123"},
		},
		"user": map[string]any{
			"username": "testuser",
		},
		"project": map[string]any{
			"id": 1,
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/webhook",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GitlabEvent, hook.GitlabEventPR)
	req.Header.Set("X-Gitlab-Token", testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse PR update action failed: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestHook type")
	}
	if prHook.Action != "update" {
		t.Errorf("expected action 'update', got '%s'", prHook.Action)
	}
}

// TestParsePREmptyAction verifies that an empty action field doesn't trigger
// the unsupported action error.
func TestParsePREmptyAction(t *testing.T) {
	payload := map[string]any{
		"object_attributes": map[string]any{
			"iid":           7,
			"title":         "No Action MR",
			"source_branch": "feat",
			"target_branch": "main",
			"source":        map[string]any{"name": "src"},
			"target":        map[string]any{"name": "tgt"},
			"last_commit":   map[string]any{"id": "def456"},
		},
		"user": map[string]any{
			"username": "testuser",
		},
		"project": map[string]any{
			"id": 1,
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/webhook",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GitlabEvent, hook.GitlabEventPR)
	req.Header.Set("X-Gitlab-Token", testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse PR with empty action failed: %v", err)
	}
	if wh == nil {
		t.Fatal("expected non-nil webhook result")
	}
}

// TestValidateInvalidHex verifies that Validate returns false for non-hex signatures.
func TestValidateInvalidHex(t *testing.T) {
	if Validate(nil, []byte("msg"), []byte("key"), "zzzz-not-hex") {
		t.Error("Validate should return false for non-hex signature")
	}
}
