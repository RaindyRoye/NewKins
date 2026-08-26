package gitlab

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gokins/gokins/hook"
)

func TestParseCommentNoteEvent(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"event_type":  "note",
		"user": map[string]any{
			"id":       1,
			"name":     "Test User",
			"username": "testuser",
		},
		"project_id": 42,
		"project": map[string]any{
			"id":                  42,
			"name":                "test-project",
			"description":         "project desc",
			"web_url":             "https://gitlab.com/group/test-project",
			"git_ssh_url":         "git@gitlab.com:group/test-project.git",
			"git_http_url":        "https://gitlab.com/group/test-project.git",
			"path_with_namespace": "group/test-project",
			"homepage":            "https://gitlab.com/group/test-project",
			"url":                 "git@gitlab.com:group/test-project.git",
			"ssh_url":             "git@gitlab.com:group/test-project.git",
			"http_url":            "https://gitlab.com/group/test-project.git",
		},
		"object_attributes": map[string]any{
			"note":          "This is a test comment",
			"noteable_type": "MergeRequest", //nolint:misspell // GitLab API field name
			"noteable_id":   10,             //nolint:misspell // GitLab API field name
		},
		"merge_request": map[string]any{
			"iid":               int64(99),
			"title":             "Test MR",
			"source_branch":     "feature-x",
			"target_branch":     "main",
			"source_project_id": 42,
			"target_project_id": 42,
			"source": map[string]any{
				"id":                  42,
				"name":                "test-project",
				"description":         "src desc",
				"web_url":             "https://gitlab.com/group/test-project",
				"git_ssh_url":         "git@gitlab.com:group/test-project.git",
				"git_http_url":        "https://gitlab.com/group/test-project.git",
				"path_with_namespace": "group/test-project",
				"url":                 "git@gitlab.com:group/test-project.git",
				"ssh_url":             "git@gitlab.com:group/test-project.git",
				"http_url":            "https://gitlab.com/group/test-project.git",
			},
			"target": map[string]any{
				"id":                  42,
				"name":                "test-project",
				"description":         "tgt desc",
				"web_url":             "https://gitlab.com/group/test-project",
				"git_ssh_url":         "git@gitlab.com:group/test-project.git",
				"git_http_url":        "https://gitlab.com/group/test-project.git",
				"path_with_namespace": "group/test-project",
				"url":                 "git@gitlab.com:group/test-project.git",
				"ssh_url":             "git@gitlab.com:group/test-project.git",
				"http_url":            "https://gitlab.com/group/test-project.git",
			},
			"last_commit": map[string]any{
				"id":      "sha123",
				"message": "last commit msg",
			},
		},
		"repository": map[string]any{
			"name":        "test-project",
			"url":         "git@gitlab.com:group/test-project.git",
			"description": "repo desc",
			"homepage":    "https://gitlab.com/group/test-project",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	secret := testSecret
	req := newGitlabRequest(hook.GitlabEventNote, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse comment hook failed: %v", err)
	}

	commentHook, ok := wh.(*hook.PullRequestCommentHook)
	if !ok {
		t.Fatalf("expected *hook.PullRequestCommentHook type, got %T", wh)
	}

	if commentHook.Action != hook.EventsTypeComment {
		t.Errorf("action = %q, want %q", commentHook.Action, hook.EventsTypeComment)
	}
	if commentHook.Comment.Body != "This is a test comment" {
		t.Errorf("comment body = %q, want %q", commentHook.Comment.Body, "This is a test comment")
	}
	if commentHook.Comment.Author.UserName != "testuser" {
		t.Errorf("comment author = %q, want %q", commentHook.Comment.Author.UserName, "testuser")
	}
	if commentHook.Repo.RepoType != "gitlab" {
		t.Errorf("repo type = %q, want %q", commentHook.Repo.RepoType, "gitlab")
	}
	if commentHook.Repo.Name != "test-project" {
		t.Errorf("repo name = %q, want %q", commentHook.Repo.Name, "test-project")
	}
	if commentHook.Repo.Branch != "feature-x" {
		t.Errorf("repo branch = %q, want %q", commentHook.Repo.Branch, "feature-x")
	}
	if commentHook.Repo.Sha != "sha123" {
		t.Errorf("repo sha = %q, want %q", commentHook.Repo.Sha, "sha123")
	}
	if commentHook.TargetRepo.Branch != "main" {
		t.Errorf("target branch = %q, want %q", commentHook.TargetRepo.Branch, "main")
	}
	if commentHook.PullRequest.Title != "Test MR" {
		t.Errorf("PR title = %q, want %q", commentHook.PullRequest.Title, "Test MR")
	}
	if commentHook.PullRequest.Number != 99 {
		t.Errorf("PR number = %d, want %d", commentHook.PullRequest.Number, 99)
	}
	if commentHook.Sender.UserName != "testuser" {
		t.Errorf("sender = %q, want %q", commentHook.Sender.UserName, "testuser")
	}
	if commentHook.Repo.RepoOpenid != "42" {
		t.Errorf("repo openid = %q, want %q", commentHook.Repo.RepoOpenid, "42")
	}
}

func TestParseCommentNoteInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	req := newGitlabRequest(hook.GitlabEventNote, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment hook")
	}
}

func TestParseCommentNoteWrongSecret(t *testing.T) {
	payload := map[string]any{
		"user":              map[string]any{"username": "u"},
		"object_attributes": map[string]any{"note": "test"},
		"merge_request":     map[string]any{"source": map[string]any{}, "target": map[string]any{}},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventNote, body, "correctsecret")

	_, err := Parse(req, "wrongsecret")
	if err == nil {
		t.Fatal("expected error for wrong secret on comment hook")
	}
	if !errors.Is(err, hook.ErrSecretValidationFailed) {
		t.Errorf("err = %v, want %v", err, hook.ErrSecretValidationFailed)
	}
}

func TestValidateHMACSHA256(t *testing.T) {
	message := []byte("test message body")
	key := []byte("mysecretkey")

	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	sig := hex.EncodeToString(mac.Sum(nil))

	if !Validate(sha256.New, message, key, sig) {
		t.Error("Validate should return true for valid HMAC-SHA256")
	}

	if Validate(sha256.New, message, key, "notahexstring") {
		t.Error("Validate should return false for invalid hex signature")
	}

	if Validate(sha256.New, message, []byte("wrongkey"), sig) {
		t.Error("Validate should return false for wrong key")
	}

	if Validate(sha256.New, []byte("different message"), key, sig) {
		t.Error("Validate should return false for different message")
	}
}

func TestValidateInternal(t *testing.T) {
	message := []byte("hello world")
	key := []byte("secret")

	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	sum := mac.Sum(nil)

	if !validate(sha256.New, message, key, sum) {
		t.Error("validate should return true for matching HMAC")
	}

	if validate(sha256.New, message, key, []byte("wrong")) {
		t.Error("validate should return false for wrong signature bytes")
	}
}
