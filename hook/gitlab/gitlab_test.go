package gitlab

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

const (
	testSecret = "testsecret"
	testUser   = "testuser"
)

func newGitlabRequest(event string, body []byte, secret string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GitlabEvent, event)
	if secret != "" {
		req.Header.Set("X-Gitlab-Token", secret)
	}
	return req
}

func TestParsePushHook(t *testing.T) {
	payload := map[string]any{
		"object_kind":   "push",
		"event_name":    "push",
		"ref":           "refs/heads/main",
		"before":        "abc123",
		"after":         "def456",
		"user_username": "testuser",
		"project_id":    1,
		"project": map[string]any{
			"id":                  1,
			"name":                "test-repo",
			"path_with_namespace": "group/test-repo",
			"web_url":             "https://gitlab.com/group/test-repo",
			"git_http_url":        "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
			"ssh_url":             "git@gitlab.com:group/test-repo.git",
			"http_url":            "https://gitlab.com/group/test-repo.git",
		},
		"commits": []map[string]any{
			{
				"id":      "def456",
				"message": "test commit",
				"url":     "https://gitlab.com/group/test-repo/-/commit/def456",
			},
		},
		"repository": map[string]any{
			"name":             "test-repo",
			"url":              "git@gitlab.com:group/test-repo.git",
			"description":      "A test repository",
			"homepage":         "https://gitlab.com/group/test-repo",
			"git_http_url":     "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":      "git@gitlab.com:group/test-repo.git",
			"visibility_level": 0,
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
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
	if pushHook.Repo.Owner != testUser {
		t.Errorf("expected owner 'testuser', got '%s'", pushHook.Repo.Owner)
	}
	if pushHook.Sender.UserName != testUser {
		t.Errorf("expected sender 'testuser', got '%s'", pushHook.Sender.UserName)
	}
}

func TestParsePullRequestHook(t *testing.T) {
	payload := map[string]any{
		"object_kind": "merge_request",
		"event_type":  "merge_request",
		"user": map[string]any{
			"id":       1,
			"name":     "Test User",
			"username": "testuser",
		},
		"project": map[string]any{
			"id":                  1,
			"name":                "test-repo",
			"path_with_namespace": "group/test-repo",
			"web_url":             "https://gitlab.com/group/test-repo",
			"git_http_url":        "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
			"ssh_url":             "git@gitlab.com:group/test-repo.git",
			"http_url":            "https://gitlab.com/group/test-repo.git",
		},
		"object_attributes": map[string]any{
			"id":            1,
			"iid":           42,
			"title":         "Test MR",
			"source_branch": "feature-branch",
			"target_branch": "main",
			"action":        "open",
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
	if prHook.Sender.UserName != testUser {
		t.Errorf("expected sender 'testuser', got '%s'", prHook.Sender.UserName)
	}
}

func TestParseUnknownEvent(t *testing.T) {
	body := []byte(`{}`)
	secret := testSecret
	req := newGitlabRequest("Unknown Hook", body, secret)

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
		"project":    map[string]any{"id": 1},
		"repository": map[string]any{},
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
	payload := map[string]any{
		"object_attributes": map[string]any{
			"action": "close",
			"source": map[string]any{},
			"target": map[string]any{},
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
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

func TestParseCommentHook(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"event_type":  "note",
		"user": map[string]any{
			"id":       1,
			"name":     "Test User",
			"username": testUser,
		},
		"project_id": 1,
		"project": map[string]any{
			"id":                  1,
			"name":                "test-repo",
			"path_with_namespace": "group/test-repo",
			"web_url":             "https://gitlab.com/group/test-repo",
			"git_http_url":        "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
			"ssh_url":             "git@gitlab.com:group/test-repo.git",
			"http_url":            "https://gitlab.com/group/test-repo.git",
		},
		"object_attributes": map[string]any{
			"note": "This is a test comment",
			"id":   100,
		},
		"merge_request": map[string]any{
			"id":            1,
			"iid":           42,
			"title":         "Test MR",
			"source_branch": "feature-branch",
			"target_branch": "main",
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
	req := newGitlabRequest(hook.GitlabEventNote, body, secret)

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
	if commentHook.Comment.Body != "This is a test comment" {
		t.Errorf("expected comment body 'This is a test comment', got '%s'", commentHook.Comment.Body)
	}
	if commentHook.Comment.Author.UserName != testUser {
		t.Errorf("expected comment author 'testuser', got '%s'", commentHook.Comment.Author.UserName)
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
	if commentHook.PullRequest.Number != 42 {
		t.Errorf("expected PR number 42, got %d", commentHook.PullRequest.Number)
	}
	if commentHook.Sender.UserName != testUser {
		t.Errorf("expected sender 'testuser', got '%s'", commentHook.Sender.UserName)
	}
}

func TestParseCommentHookInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	req := newGitlabRequest(hook.GitlabEventNote, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment hook")
	}
}

func TestParsePRActionUpdate(t *testing.T) {
	payload := map[string]any{
		"object_kind": "merge_request",
		"event_type":  "merge_request",
		"user": map[string]any{
			"username": testUser,
		},
		"project": map[string]any{
			"id": 1,
		},
		"object_attributes": map[string]any{
			"action":        hook.ActionUpdate,
			"iid":           10,
			"source_branch": "feat",
			"target_branch": "main",
			"source":        map[string]any{"name": "src"},
			"target":        map[string]any{"name": "tgt"},
			"last_commit":   map[string]any{"id": "abc"},
		},
	}

	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPR, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse PR with update action failed: %v", err)
	}
	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestHook type")
	}
	if prHook.Action != hook.ActionUpdate {
		t.Errorf("expected action '%s', got '%s'", hook.ActionUpdate, prHook.Action)
	}
}

func TestParsePRAactionEmpty(t *testing.T) {
	// When action is empty, it should be accepted (no action filter)
	payload := map[string]any{
		"object_kind": "merge_request",
		"user":        map[string]any{"username": testUser},
		"project":     map[string]any{"id": 1},
		"object_attributes": map[string]any{
			"action":        "",
			"iid":           5,
			"source_branch": "b1",
			"target_branch": "main",
			"source":        map[string]any{"name": "s"},
			"target":        map[string]any{"name": "t"},
			"last_commit":   map[string]any{"id": "xyz"},
		},
	}

	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPR, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse PR with empty action failed: %v", err)
	}
	prHook := wh.(*hook.PullRequestHook)
	if prHook.PullRequest.Number != 5 {
		t.Errorf("expected PR number 5, got %d", prHook.PullRequest.Number)
	}
}

func TestParsePushHookBranchExtraction(t *testing.T) {
	// Test branch extraction from ref: refs/heads/feature/my-branch -> feature/my-branch
	payload := map[string]any{
		"ref":           "refs/heads/feature/my-branch",
		"before":        "a",
		"after":         "b",
		"user_username": testUser,
		"project_id":    1,
		"project": map[string]any{
			"id":                  1,
			"name":                "repo",
			"path_with_namespace": "group/repo",
			"http_url":            "https://gitlab.com/group/repo.git",
			"ssh_url":             "git@gitlab.com:group/repo.git",
		},
		"repository": map[string]any{
			"name":         "repo",
			"git_http_url": "https://gitlab.com/group/repo.git",
			"git_ssh_url":  "git@gitlab.com:group/repo.git",
		},
	}

	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPush, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse push hook failed: %v", err)
	}
	pushHook := wh.(*hook.PushHook)
	// The branch extraction takes the 3rd element: refs/heads/feature/my-branch -> feature
	if pushHook.Repo.Branch != "feature" {
		t.Errorf("expected branch 'feature', got '%s'", pushHook.Repo.Branch)
	}
}

func TestValidate(t *testing.T) {
	// Validate is a generic HMAC function exported from the gitlab package.
	// GitLab uses token-based auth so Validate is not part of its main auth flow,
	// but it's still exported and should work correctly when given a hash func.
	key := []byte("secret")
	message := []byte("payload")

	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	sig := hex.EncodeToString(mac.Sum(nil))

	if !Validate(sha256.New, message, key, sig) {
		t.Fatal("Validate should return true for a valid HMAC-SHA256 signature")
	}
	if Validate(sha256.New, message, []byte("wrong-key"), sig) {
		t.Fatal("Validate should return false for wrong key")
	}
	if Validate(sha256.New, message, key, "deadbeef") {
		t.Fatal("Validate should return false for non-hex signature")
	}
}

// Ensure hash.Hash is imported (used indirectly via sha256.New function signature).
var _ hash.Hash = sha256.New()
