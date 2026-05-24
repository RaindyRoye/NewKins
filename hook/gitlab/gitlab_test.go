package gitlab

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

func newGitlabRequest(event string, body []byte, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GitlabEvent, event)
	if secret != "" {
		req.Header.Set("X-Gitlab-Token", secret)
	}
	return req
}

func TestParsePushHook(t *testing.T) {
	payload := map[string]interface{}{
		"object_kind":    "push",
		"event_name":     "push",
		"ref":            "refs/heads/main",
		"before":         "abc123",
		"after":          "def456",
		"user_username":  "testuser",
		"project_id":     1,
		"project": map[string]interface{}{
			"id":                  1,
			"name":                "test-repo",
			"path_with_namespace": "group/test-repo",
			"web_url":             "https://gitlab.com/group/test-repo",
			"git_http_url":        "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
			"ssh_url":             "git@gitlab.com:group/test-repo.git",
			"http_url":            "https://gitlab.com/group/test-repo.git",
		},
		"commits": []map[string]interface{}{
			{
				"id":      "def456",
				"message": "test commit",
				"url":     "https://gitlab.com/group/test-repo/-/commit/def456",
			},
		},
		"repository": map[string]interface{}{
			"name":            "test-repo",
			"url":             "git@gitlab.com:group/test-repo.git",
			"description":     "A test repository",
			"homepage":        "https://gitlab.com/group/test-repo",
			"git_http_url":    "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":     "git@gitlab.com:group/test-repo.git",
			"visibility_level": 0,
		},
	}

	body, _ := json.Marshal(payload)
	secret := "testsecret"
	req := newGitlabRequest(hook.GitlabEventPush, body, secret)

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
	if pushHook.Repo.Branch != "main" {
		t.Errorf("expected branch 'main', got '%s'", pushHook.Repo.Branch)
	}
	if pushHook.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got '%s'", pushHook.Repo.RepoType)
	}
	if pushHook.Repo.Owner != "testuser" {
		t.Errorf("expected owner 'testuser', got '%s'", pushHook.Repo.Owner)
	}
	if pushHook.Sender.UserName != "testuser" {
		t.Errorf("expected sender 'testuser', got '%s'", pushHook.Sender.UserName)
	}
}

func TestParsePullRequestHook(t *testing.T) {
	payload := map[string]interface{}{
		"object_kind": "merge_request",
		"event_type":  "merge_request",
		"user": map[string]interface{}{
			"id":       1,
			"name":     "Test User",
			"username": "testuser",
		},
		"project": map[string]interface{}{
			"id":                  1,
			"name":                "test-repo",
			"path_with_namespace": "group/test-repo",
			"web_url":             "https://gitlab.com/group/test-repo",
			"git_http_url":        "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
			"ssh_url":             "git@gitlab.com:group/test-repo.git",
			"http_url":            "https://gitlab.com/group/test-repo.git",
		},
		"object_attributes": map[string]interface{}{
			"id":             1,
			"iid":            42,
			"title":          "Test MR",
			"source_branch":  "feature-branch",
			"target_branch":  "main",
			"action":         "open",
			"state":          "opened",
			"source": map[string]interface{}{
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
			"target": map[string]interface{}{
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
			"last_commit": map[string]interface{}{
				"id":      "commitsha123",
				"message": "latest commit",
			},
		},
	}

	body, _ := json.Marshal(payload)
	secret := "testsecret"
	req := newGitlabRequest(hook.GitlabEventPR, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse MR hook failed: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestHook type")
	}

	if prHook.Action != hook.ActionOpen {
		t.Errorf("expected action '%s', got '%s'", hook.ActionOpen, prHook.Action)
	}
	if prHook.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got '%s'", prHook.Repo.RepoType)
	}
	if prHook.Sender.UserName != "testuser" {
		t.Errorf("expected sender 'testuser', got '%s'", prHook.Sender.UserName)
	}
}

func TestParseUnknownEvent(t *testing.T) {
	body := []byte(`{}`)
	secret := "testsecret"
	req := newGitlabRequest("Unknown Hook", body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestParseInvalidSecret(t *testing.T) {
	payload := map[string]interface{}{
		"ref":    "refs/heads/main",
		"before": "abc",
		"after":  "def",
		"commits": []map[string]interface{}{
			{"id": "def", "message": "test"},
		},
		"project":    map[string]interface{}{"id": 1},
		"repository": map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPush, body, "correctsecret")

	_, err := Parse(req, "wrongsecret")
	if err == nil {
		t.Fatal("expected error for invalid secret")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	req := newGitlabRequest(hook.GitlabEventPush, body, "secret")

	_, err := Parse(req, "secret")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParsePRUnsupportedAction(t *testing.T) {
	payload := map[string]interface{}{
		"object_attributes": map[string]interface{}{
			"action": "close",
			"source": map[string]interface{}{},
			"target": map[string]interface{}{},
		},
	}

	body, _ := json.Marshal(payload)
	secret := "testsecret"
	req := newGitlabEventPR(body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for unsupported MR action 'close'")
	}
}

// newGitlabEventPR is a helper for PR-specific requests.
func newGitlabEventPR(body []byte, secret string) *http.Request {
	return newGitlabRequest(hook.GitlabEventPR, body, secret)
}
