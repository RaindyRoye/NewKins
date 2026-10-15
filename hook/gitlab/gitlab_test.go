package gitlab

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

func TestValidate(t *testing.T) {
	message := []byte("test message")
	key := []byte("secret")

	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	sig := hex.EncodeToString(mac.Sum(nil))

	if !Validate(sha256.New, message, key, sig) {
		t.Error("Validate should return true for valid HMAC-SHA256 signature")
	}

	if Validate(sha256.New, message, key, "invalidsignature") {
		t.Error("Validate should return false for invalid signature")
	}

	if Validate(sha256.New, message, []byte("wrongkey"), sig) {
		t.Error("Validate should return false for wrong key")
	}
}

func TestValidateInvalidHex(t *testing.T) {
	message := []byte("test body")
	key := []byte("secret")

	// Invalid hex characters should return false
	if Validate(sha256.New, message, key, "not-hex!!!") {
		t.Error("Validate should return false for non-hex signature")
	}
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
	if pushHook.Repo.FullName != "group/test-repo" {
		t.Errorf("expected fullName 'group/test-repo', got '%s'", pushHook.Repo.FullName)
	}
}

func TestParsePushHookEmptyRef(t *testing.T) {
	payload := map[string]any{
		"ref":           "",
		"before":        "aaa",
		"after":         "bbb",
		"user_username": "user",
		"project_id":    1,
		"project":       map[string]any{"id": 1},
		"commits":       []map[string]any{},
		"repository":    map[string]any{},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPush, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse push with empty ref failed: %v", err)
	}
	pushHook := wh.(*hook.PushHook)
	if pushHook.Repo.Branch != "" {
		t.Errorf("expected empty branch for empty ref, got '%s'", pushHook.Repo.Branch)
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
	if prHook.PullRequest.Number != 42 {
		t.Errorf("expected MR number 42, got %d", prHook.PullRequest.Number)
	}
	if prHook.PullRequest.Title != "Test MR" {
		t.Errorf("expected MR title 'Test MR', got '%s'", prHook.PullRequest.Title)
	}
	if prHook.Repo.Name != "test-repo-fork" {
		t.Errorf("expected source repo name 'test-repo-fork', got '%s'", prHook.Repo.Name)
	}
	if prHook.TargetRepo.Name != "test-repo" {
		t.Errorf("expected target repo name 'test-repo', got '%s'", prHook.TargetRepo.Name)
	}
}

func TestParsePullRequestUpdateAction(t *testing.T) {
	payload := map[string]any{
		"object_kind": "merge_request",
		"user":        map[string]any{"username": "user"},
		"project":     map[string]any{"id": 1},
		"object_attributes": map[string]any{
			"iid":           7,
			"title":         "Update MR",
			"source_branch": "feat",
			"target_branch": "main",
			"action":        "update",
			"source":        map[string]any{"id": 1, "name": "r"},
			"target":        map[string]any{"id": 1, "name": "r"},
			"last_commit":   map[string]any{"id": "sha"},
		},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPR, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse update MR failed: %v", err)
	}
	prHook := wh.(*hook.PullRequestHook)
	if prHook.Action != hook.ActionUpdate {
		t.Errorf("expected action '%s', got '%s'", hook.ActionUpdate, prHook.Action)
	}
}

func TestParsePREmptyAction(t *testing.T) {
	payload := map[string]any{
		"user":    map[string]any{"username": "u"},
		"project": map[string]any{"id": 1},
		"object_attributes": map[string]any{
			"iid":           5,
			"title":         "Empty action",
			"source_branch": "feat",
			"target_branch": "main",
			"action":        "",
			"source":        map[string]any{"id": 1, "name": "r"},
			"target":        map[string]any{"id": 1, "name": "r"},
			"last_commit":   map[string]any{"id": "sha"},
		},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPR, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse MR with empty action failed: %v", err)
	}
	prHook := wh.(*hook.PullRequestHook)
	if prHook.Action != "" {
		t.Errorf("expected empty action, got '%s'", prHook.Action)
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

// TestParseCommentHook tests the GitLab comment/note hook parsing.
func TestParseCommentHook(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"event_type":  "note",
		"user": map[string]any{
			"id":       1,
			"name":     "Comment User",
			"username": "commenter",
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
			"note":       "This is a test comment on MR",
			"created_at": "2021-06-01T00:00:00Z",
		},
		"merge_request": map[string]any{
			"id":            100,
			"iid":           42,
			"title":         "Test MR for comment",
			"source_branch": "feature-branch",
			"target_branch": "main",
			"source": map[string]any{
				"id":                  2,
				"name":                "source-repo",
				"description":         "source description",
				"web_url":             "https://gitlab.com/forker/source-repo",
				"git_http_url":        "https://gitlab.com/forker/source-repo.git",
				"git_ssh_url":         "git@gitlab.com:forker/source-repo.git",
				"ssh_url":             "git@gitlab.com:forker/source-repo.git",
				"http_url":            "https://gitlab.com/forker/source-repo.git",
				"url":                 "git@gitlab.com:forker/source-repo.git",
				"path_with_namespace": "forker/source-repo",
			},
			"target": map[string]any{
				"id":                  1,
				"name":                "target-repo",
				"description":         "target description",
				"web_url":             "https://gitlab.com/group/target-repo",
				"git_http_url":        "https://gitlab.com/group/target-repo.git",
				"git_ssh_url":         "git@gitlab.com:group/target-repo.git",
				"ssh_url":             "git@gitlab.com:group/target-repo.git",
				"http_url":            "https://gitlab.com/group/target-repo.git",
				"url":                 "git@gitlab.com:group/target-repo.git",
				"path_with_namespace": "group/target-repo",
			},
			"last_commit": map[string]any{
				"id":      "commitsha789",
				"message": "latest commit",
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventNote, body, testSecret)

	wh, err := Parse(req, testSecret)
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
	if commentHook.Comment.Body != "This is a test comment on MR" {
		t.Errorf("expected comment body 'This is a test comment on MR', got '%s'", commentHook.Comment.Body)
	}
	if commentHook.Comment.Author.UserName != "commenter" {
		t.Errorf("expected comment author 'commenter', got '%s'", commentHook.Comment.Author.UserName)
	}
	if commentHook.PullRequest.Number != 42 {
		t.Errorf("expected MR number 42, got %d", commentHook.PullRequest.Number)
	}
	if commentHook.PullRequest.Title != "Test MR for comment" {
		t.Errorf("expected MR title 'Test MR for comment', got '%s'", commentHook.PullRequest.Title)
	}
	if commentHook.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got '%s'", commentHook.Repo.RepoType)
	}
	if commentHook.Repo.Name != "source-repo" {
		t.Errorf("expected source repo name 'source-repo', got '%s'", commentHook.Repo.Name)
	}
	if commentHook.TargetRepo.Name != "target-repo" {
		t.Errorf("expected target repo name 'target-repo', got '%s'", commentHook.TargetRepo.Name)
	}
	if commentHook.Sender.UserName != "commenter" {
		t.Errorf("expected sender 'commenter', got '%s'", commentHook.Sender.UserName)
	}
}

// TestParseCommentHookInvalidJSON tests that parseCommentHook returns an error for invalid JSON.
func TestParseCommentHookInvalidJSON(t *testing.T) {
	body := []byte(`{bad json`)
	req := newGitlabRequest(hook.GitlabEventNote, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid comment JSON")
	}
}

// TestParsePRInvalidJSON tests that parsePullRequestHook returns an error for invalid JSON.
func TestParsePRInvalidJSON(t *testing.T) {
	body := []byte(`{not valid json`)
	req := newGitlabRequest(hook.GitlabEventPR, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid PR JSON")
	}
}

// TestParsePushHookInvalidJSON tests that parsePushHook returns an error for invalid JSON.
func TestParsePushHookInvalidJSON(t *testing.T) {
	body := []byte(`{invalid`)
	req := newGitlabRequest(hook.GitlabEventPush, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid push JSON")
	}
}

// TestParseNoSecret tests Parse with no X-Gitlab-Token header.
func TestParseNoSecret(t *testing.T) {
	payload := map[string]any{
		"ref":    "refs/heads/main",
		"before": "a",
		"after":  "b",
		"commits": []map[string]any{
			{"id": "b", "message": "m"},
		},
		"project":    map[string]any{"id": 1},
		"repository": map[string]any{},
	}
	body, _ := json.Marshal(payload)

	// Request without any X-Gitlab-Token header
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(hook.GitlabEvent, hook.GitlabEventPush)

	// With empty secret, validation should pass (empty == empty)
	_, err := Parse(req, "")
	if err != nil {
		t.Fatalf("expected no error with empty secret, got: %v", err)
	}
}

// TestParseSecretMismatch tests Parse with mismatched secrets.
func TestParseSecretMismatch(t *testing.T) {
	payload := map[string]any{
		"ref":    "refs/heads/main",
		"before": "a",
		"after":  "b",
		"commits": []map[string]any{
			{"id": "b", "message": "m"},
		},
		"project":    map[string]any{"id": 1},
		"repository": map[string]any{},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPush, body, "secret1")

	_, err := Parse(req, "secret2")
	if err == nil {
		t.Fatal("expected error for secret mismatch")
	}
	if !errors.Is(err, hook.ErrSecretValidationFailed) {
		t.Errorf("expected ErrSecretValidationFailed, got: %v", err)
	}
}

// newGitlabEventPR is a helper for PR-specific requests.
func newGitlabEventPR(body []byte, secret string) *http.Request {
	return newGitlabRequest(hook.GitlabEventPR, body, secret)
}
