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

// --- parseCommentHook ---

func TestParseCommentHook_Valid(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"event_type":  "note",
		"user": map[string]any{
			"id":       1,
			"name":     "Commenter",
			"username": "commenter",
		},
		"project_id": 1,
		"project": map[string]any{
			"id":                  1,
			"name":                "repo",
			"path_with_namespace": "group/repo",
			"web_url":             "https://gitlab.com/group/repo",
			"git_http_url":        "https://gitlab.com/group/repo.git",
			"git_ssh_url":         "git@gitlab.com:group/repo.git",
			"ssh_url":             "git@gitlab.com:group/repo.git",
			"http_url":            "https://gitlab.com/group/repo.git",
		},
		"object_attributes": map[string]any{
			"id":            100,
			"note":          "LGTM",
			"noteable_type": "MergeRequest", //nolint:misspell // GitLab API field name
			"noteable_id":   42,             //nolint:misspell // GitLab API field name
			"discussion_id": "abc123",
			"author_id":     1,
			"created_at":    "2024-01-01T00:00:00Z",
			"updated_at":    "2024-01-01T00:00:00Z",
		},
		"merge_request": map[string]any{
			"id":            42,
			"iid":           10,
			"title":         "Test MR",
			"source_branch": "feature",
			"target_branch": "main",
			"source": map[string]any{
				"id":                  2,
				"name":                "fork",
				"path_with_namespace": "user/fork",
				"web_url":             "https://gitlab.com/user/fork",
				"git_http_url":        "https://gitlab.com/user/fork.git",
				"git_ssh_url":         "git@gitlab.com:user/fork.git",
				"ssh_url":             "git@gitlab.com:user/fork.git",
				"http_url":            "https://gitlab.com/user/fork.git",
				"url":                 "git@gitlab.com:user/fork.git",
				"description":         "forked repo",
			},
			"target": map[string]any{
				"id":                  1,
				"name":                "repo",
				"path_with_namespace": "group/repo",
				"web_url":             "https://gitlab.com/group/repo",
				"git_http_url":        "https://gitlab.com/group/repo.git",
				"git_ssh_url":         "git@gitlab.com:group/repo.git",
				"ssh_url":             "git@gitlab.com:group/repo.git",
				"http_url":            "https://gitlab.com/group/repo.git",
				"url":                 "git@gitlab.com:group/repo.git",
				"description":         "main repo",
			},
			"last_commit": map[string]any{
				"id":      "commit123",
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
		t.Errorf("expected action %q, got %q", hook.EventsTypeComment, commentHook.Action)
	}
	if commentHook.Comment.Body != "LGTM" {
		t.Errorf("expected comment body 'LGTM', got %q", commentHook.Comment.Body)
	}
	if commentHook.Comment.Author.UserName != "commenter" {
		t.Errorf("expected commenter 'commenter', got %q", commentHook.Comment.Author.UserName)
	}
	if commentHook.PullRequest.Number != 10 {
		t.Errorf("expected MR number 10, got %d", commentHook.PullRequest.Number)
	}
	if commentHook.Repo.Branch != "feature" {
		t.Errorf("expected branch 'feature', got %q", commentHook.Repo.Branch)
	}
	if commentHook.TargetRepo.Branch != "main" {
		t.Errorf("expected target branch 'main', got %q", commentHook.TargetRepo.Branch)
	}
}

func TestParseCommentHook_InvalidJSON(t *testing.T) {
	body := []byte(`{broken json`)
	req := newGitlabRequest(hook.GitlabEventNote, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment hook")
	}
}

func TestParseCommentHook_SecretValidationFailure(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"user": map[string]any{
			"id":       1,
			"username": "user",
		},
		"project_id": 1,
		"project": map[string]any{
			"id": 1,
		},
		"object_attributes": map[string]any{
			"id":   1,
			"note": "comment",
		},
		"merge_request": map[string]any{
			"iid":   5,
			"title": "MR",
			"source": map[string]any{
				"id": 2,
			},
			"target": map[string]any{
				"id": 1,
			},
			"last_commit": map[string]any{
				"id": "sha",
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventNote, body, "correctsecret")

	_, err := Parse(req, "wrongsecret")
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
	if !errors.Is(err, hook.ErrSecretValidationFailed) {
		t.Errorf("expected ErrSecretValidationFailed, got: %v", err)
	}
}

// --- Validate and validate functions ---

func TestValidate_ValidSignature(t *testing.T) {
	message := []byte("test message")
	key := []byte("secret")

	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	sig := hex.EncodeToString(mac.Sum(nil))

	if !Validate(sha256.New, message, key, sig) {
		t.Error("Validate should return true for valid signature")
	}
}

func TestValidate_InvalidSignature(t *testing.T) {
	message := []byte("test message")
	key := []byte("secret")

	if Validate(sha256.New, message, key, "invalidsig") {
		t.Error("Validate should return false for invalid signature")
	}
}

func TestValidate_InvalidHex(t *testing.T) {
	message := []byte("test message")
	key := []byte("secret")

	// Non-hex string should cause hex.DecodeString to fail
	if Validate(sha256.New, message, key, "ZZZZ-not-hex") {
		t.Error("Validate should return false for non-hex signature")
	}
}

func TestValidate_WrongKey(t *testing.T) {
	message := []byte("test message")
	key1 := []byte("secret1")
	key2 := []byte("secret2")

	mac := hmac.New(sha256.New, key1)
	mac.Write(message)
	sig := hex.EncodeToString(mac.Sum(nil))

	if Validate(sha256.New, message, key2, sig) {
		t.Error("Validate should return false for wrong key")
	}
}

func TestValidate_EmptySignature(t *testing.T) {
	message := []byte("test message")
	key := []byte("secret")

	if Validate(sha256.New, message, key, "") {
		t.Error("Validate should return false for empty signature")
	}
}

// --- convertCommentHook ---

func TestConvertCommentHook_FullFieldMapping(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"user": map[string]any{
			"id":       1,
			"name":     "Test User",
			"username": "testuser",
		},
		"project_id": 10,
		"project": map[string]any{
			"id":                  10,
			"name":                "myrepo",
			"path_with_namespace": "group/myrepo",
			"web_url":             "https://gitlab.com/group/myrepo",
			"git_http_url":        "https://gitlab.com/group/myrepo.git",
			"git_ssh_url":         "git@gitlab.com:group/myrepo.git",
			"ssh_url":             "git@gitlab.com:group/myrepo.git",
			"http_url":            "https://gitlab.com/group/myrepo.git",
			"url":                 "git@gitlab.com:group/myrepo.git",
			"description":         "test repo",
		},
		"object_attributes": map[string]any{
			"id":            100,
			"note":          "Great work!",
			"noteable_type": "MergeRequest", //nolint:misspell // GitLab API field name
			"noteable_id":   42,             //nolint:misspell // GitLab API field name
			"discussion_id": "disc123",
			"author_id":     1,
			"created_at":    "2024-01-01T00:00:00Z",
			"updated_at":    "2024-01-01T00:00:00Z",
		},
		"merge_request": map[string]any{
			"id":            42,
			"iid":           15,
			"title":         "Feature MR",
			"source_branch": "feature-branch",
			"target_branch": "main",
			"source": map[string]any{
				"id":                  20,
				"name":                "fork",
				"path_with_namespace": "user/fork",
				"web_url":             "https://gitlab.com/user/fork",
				"git_http_url":        "https://gitlab.com/user/fork.git",
				"git_ssh_url":         "git@gitlab.com:user/fork.git",
				"ssh_url":             "git@gitlab.com:user/fork.git",
				"http_url":            "https://gitlab.com/user/fork.git",
				"url":                 "git@gitlab.com:user/fork.git",
				"description":         "forked repo",
			},
			"target": map[string]any{
				"id":                  10,
				"name":                "myrepo",
				"path_with_namespace": "group/myrepo",
				"web_url":             "https://gitlab.com/group/myrepo",
				"git_http_url":        "https://gitlab.com/group/myrepo.git",
				"git_ssh_url":         "git@gitlab.com:group/myrepo.git",
				"ssh_url":             "git@gitlab.com:group/myrepo.git",
				"http_url":            "https://gitlab.com/group/myrepo.git",
				"url":                 "git@gitlab.com:group/myrepo.git",
				"description":         "main repo",
			},
			"last_commit": map[string]any{
				"id":      "abc123",
				"message": "latest commit",
			},
		},
	}

	body, _ := json.Marshal(payload)
	var gp gitlabCommentHook
	if err := json.Unmarshal(body, &gp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	result, err := convertCommentHook(&gp)
	if err != nil {
		t.Fatalf("convertCommentHook failed: %v", err)
	}

	// Verify action
	if result.Action != hook.EventsTypeComment {
		t.Errorf("expected action %q, got %q", hook.EventsTypeComment, result.Action)
	}

	// Verify comment fields
	if result.Comment.Body != "Great work!" {
		t.Errorf("expected comment body 'Great work!', got %q", result.Comment.Body)
	}
	if result.Comment.Author.UserName != "testuser" {
		t.Errorf("expected commenter 'testuser', got %q", result.Comment.Author.UserName)
	}

	// Verify pull request fields
	if result.PullRequest.Number != 15 {
		t.Errorf("expected MR number 15, got %d", result.PullRequest.Number)
	}
	if result.PullRequest.Title != "Feature MR" {
		t.Errorf("expected title 'Feature MR', got %q", result.PullRequest.Title)
	}
	if result.PullRequest.Author.UserName != "testuser" {
		t.Errorf("expected author 'testuser', got %q", result.PullRequest.Author.UserName)
	}

	// Verify repo fields
	if result.Repo.Branch != "feature-branch" {
		t.Errorf("expected branch 'feature-branch', got %q", result.Repo.Branch)
	}
	if result.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got %q", result.Repo.RepoType)
	}
	if result.Repo.RepoOpenid != "10" {
		t.Errorf("expected RepoOpenid '10', got %q", result.Repo.RepoOpenid)
	}
	if result.Repo.FullName != "user/fork" {
		t.Errorf("expected FullName 'user/fork', got %q", result.Repo.FullName)
	}

	// Verify target repo fields
	if result.TargetRepo.Branch != "main" {
		t.Errorf("expected target branch 'main', got %q", result.TargetRepo.Branch)
	}
	if result.TargetRepo.FullName != "group/myrepo" {
		t.Errorf("expected target FullName 'group/myrepo', got %q", result.TargetRepo.FullName)
	}

	// Verify sender
	if result.Sender.UserName != "testuser" {
		t.Errorf("expected sender 'testuser', got %q", result.Sender.UserName)
	}
}

// --- parsePullRequestHook edge cases ---

func TestParsePullRequestHook_UpdateAction(t *testing.T) {
	payload := map[string]any{
		"object_kind": "merge_request",
		"event_type":  "merge_request",
		"user": map[string]any{
			"id":       1,
			"username": "user",
		},
		"project": map[string]any{
			"id": 1,
		},
		"object_attributes": map[string]any{
			"action":        "update",
			"iid":           20,
			"title":         "Updated MR",
			"source_branch": "feat",
			"target_branch": "main",
			"source": map[string]any{
				"id": 2,
			},
			"target": map[string]any{
				"id": 1,
			},
			"last_commit": map[string]any{
				"id": "sha123",
			},
		},
	}

	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPR, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse MR update hook failed: %v", err)
	}

	prHook := wh.(*hook.PullRequestHook)
	if prHook.Action != hook.ActionUpdate {
		t.Errorf("expected action %q, got %q", hook.ActionUpdate, prHook.Action)
	}
}

func TestParsePullRequestHook_EmptyAction(t *testing.T) {
	// When action is empty, it should pass through unchanged
	payload := map[string]any{
		"object_kind": "merge_request",
		"user": map[string]any{
			"username": "user",
		},
		"project": map[string]any{
			"id": 1,
		},
		"object_attributes": map[string]any{
			"action":        "",
			"iid":           5,
			"source_branch": "feat",
			"target_branch": "main",
			"source": map[string]any{
				"id": 2,
			},
			"target": map[string]any{
				"id": 1,
			},
			"last_commit": map[string]any{
				"id": "sha",
			},
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
		t.Errorf("expected empty action, got %q", prHook.Action)
	}
}

func TestParsePullRequestHook_InvalidJSON(t *testing.T) {
	body := []byte(`{not valid`)
	req := newGitlabRequest(hook.GitlabEventPR, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in PR hook")
	}
}

// --- convertPullRequestHook full field mapping ---

func TestConvertPullRequestHook_FullFieldMapping(t *testing.T) {
	payload := map[string]any{
		"object_kind": "merge_request",
		"user": map[string]any{
			"id":       1,
			"name":     "Author",
			"username": "author",
		},
		"project": map[string]any{
			"id":                  100,
			"name":                "project",
			"path_with_namespace": "group/project",
			"web_url":             "https://gitlab.com/group/project",
			"git_http_url":        "https://gitlab.com/group/project.git",
			"git_ssh_url":         "git@gitlab.com:group/project.git",
			"ssh_url":             "git@gitlab.com:group/project.git",
			"http_url":            "https://gitlab.com/group/project.git",
		},
		"object_attributes": map[string]any{
			"action":        "open",
			"iid":           25,
			"title":         "Big Feature",
			"source_branch": "feature/big",
			"target_branch": "develop",
			"source": map[string]any{
				"id":                  200,
				"name":                "fork",
				"path_with_namespace": "user/fork",
				"web_url":             "https://gitlab.com/user/fork",
				"git_http_url":        "https://gitlab.com/user/fork.git",
				"git_ssh_url":         "git@gitlab.com:user/fork.git",
				"ssh_url":             "git@gitlab.com:user/fork.git",
				"http_url":            "https://gitlab.com/user/fork.git",
				"url":                 "git@gitlab.com:user/fork.git",
				"description":         "forked project",
			},
			"target": map[string]any{
				"id":                  100,
				"name":                "project",
				"path_with_namespace": "group/project",
				"web_url":             "https://gitlab.com/group/project",
				"git_http_url":        "https://gitlab.com/group/project.git",
				"git_ssh_url":         "git@gitlab.com:group/project.git",
				"ssh_url":             "git@gitlab.com:group/project.git",
				"http_url":            "https://gitlab.com/group/project.git",
				"url":                 "git@gitlab.com:group/project.git",
				"description":         "main project",
			},
			"last_commit": map[string]any{
				"id":      "commit456",
				"message": "final commit",
			},
		},
	}

	body, _ := json.Marshal(payload)
	var gp gitlabPRHook
	if err := json.Unmarshal(body, &gp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	result := convertPullRequestHook(&gp)

	// Verify action
	if result.Action != hook.ActionOpen {
		t.Errorf("expected action %q, got %q", hook.ActionOpen, result.Action)
	}

	// Verify pull request fields
	if result.PullRequest.Number != 25 {
		t.Errorf("expected MR number 25, got %d", result.PullRequest.Number)
	}
	if result.PullRequest.Title != "Big Feature" {
		t.Errorf("expected title 'Big Feature', got %q", result.PullRequest.Title)
	}
	if result.PullRequest.Author.UserName != "author" {
		t.Errorf("expected author 'author', got %q", result.PullRequest.Author.UserName)
	}

	// Verify repo fields
	if result.Repo.Branch != "feature/big" {
		t.Errorf("expected branch 'feature/big', got %q", result.Repo.Branch)
	}
	if result.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got %q", result.Repo.RepoType)
	}
	if result.Repo.RepoOpenid != "100" {
		t.Errorf("expected RepoOpenid '100', got %q", result.Repo.RepoOpenid)
	}
	if result.Repo.FullName != "user/fork" {
		t.Errorf("expected FullName 'user/fork', got %q", result.Repo.FullName)
	}
	if result.Repo.Description != "forked project" {
		t.Errorf("expected description 'forked project', got %q", result.Repo.Description)
	}

	// Verify target repo fields
	if result.TargetRepo.Branch != "develop" {
		t.Errorf("expected target branch 'develop', got %q", result.TargetRepo.Branch)
	}
	if result.TargetRepo.FullName != "group/project" {
		t.Errorf("expected target FullName 'group/project', got %q", result.TargetRepo.FullName)
	}

	// Verify sender
	if result.Sender.UserName != "author" {
		t.Errorf("expected sender 'author', got %q", result.Sender.UserName)
	}
}

// --- convertPushHook edge cases ---

func TestConvertPushHook_EmptyRef(t *testing.T) {
	payload := map[string]any{
		"ref":    "",
		"before": "b",
		"after":  "a",
		"project": map[string]any{
			"id": 1,
		},
		"repository": map[string]any{},
	}

	body, _ := json.Marshal(payload)
	var gp gitlabPushHook
	if err := json.Unmarshal(body, &gp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	result := convertPushHook(&gp)
	if result.Repo.Branch != "" {
		t.Errorf("expected empty branch for empty ref, got %q", result.Repo.Branch)
	}
}

func TestConvertPushHook_ShortRef(t *testing.T) {
	// A ref like "refs/tags/v1.0" has only 3 segments
	payload := map[string]any{
		"ref":    "refs/tags/v1.0",
		"before": "b",
		"after":  "a",
		"project": map[string]any{
			"id": 1,
		},
		"repository": map[string]any{},
	}

	body, _ := json.Marshal(payload)
	var gp gitlabPushHook
	if err := json.Unmarshal(body, &gp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	result := convertPushHook(&gp)
	if result.Repo.Branch != "v1.0" {
		t.Errorf("expected branch 'v1.0' for refs/tags/v1.0, got %q", result.Repo.Branch)
	}
}

func TestConvertPushHook_TwoSegmentRef(t *testing.T) {
	// A ref with only 2 segments should yield empty branch
	payload := map[string]any{
		"ref":    "refs/heads",
		"before": "b",
		"after":  "a",
		"project": map[string]any{
			"id": 1,
		},
		"repository": map[string]any{},
	}

	body, _ := json.Marshal(payload)
	var gp gitlabPushHook
	if err := json.Unmarshal(body, &gp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	result := convertPushHook(&gp)
	if result.Repo.Branch != "" {
		t.Errorf("expected empty branch for 2-segment ref, got %q", result.Repo.Branch)
	}
}

func TestConvertPushHook_FullFieldMapping(t *testing.T) {
	payload := map[string]any{
		"ref":           "refs/heads/main",
		"before":        "before123",
		"after":         "after456",
		"user_username": "pusher",
		"project_id":    42,
		"project": map[string]any{
			"id":                  42,
			"name":                "myrepo",
			"path_with_namespace": "group/myrepo",
			"web_url":             "https://gitlab.com/group/myrepo",
			"git_http_url":        "https://gitlab.com/group/myrepo.git",
			"git_ssh_url":         "git@gitlab.com:group/myrepo.git",
			"ssh_url":             "git@gitlab.com:group/myrepo.git",
			"http_url":            "https://gitlab.com/group/myrepo.git",
		},
		"repository": map[string]any{
			"name":             "myrepo",
			"url":              "git@gitlab.com:group/myrepo.git",
			"description":      "test repo",
			"homepage":         "https://gitlab.com/group/myrepo",
			"git_http_url":     "https://gitlab.com/group/myrepo.git",
			"git_ssh_url":      "git@gitlab.com:group/myrepo.git",
			"visibility_level": 20,
		},
	}

	body, _ := json.Marshal(payload)
	var gp gitlabPushHook
	if err := json.Unmarshal(body, &gp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	result := convertPushHook(&gp)

	// Verify ref
	if result.Ref != "refs/heads/main" {
		t.Errorf("expected ref 'refs/heads/main', got %q", result.Ref)
	}
	if result.Repo.Ref != "refs/heads/main" {
		t.Errorf("expected Repo.Ref 'refs/heads/main', got %q", result.Repo.Ref)
	}

	// Verify branch extraction
	if result.Repo.Branch != "main" {
		t.Errorf("expected branch 'main', got %q", result.Repo.Branch)
	}

	// Verify before/after
	if result.Before != "before123" {
		t.Errorf("expected before 'before123', got %q", result.Before)
	}
	if result.After != "after456" {
		t.Errorf("expected after 'after456', got %q", result.After)
	}
	if result.Repo.Sha != "after456" {
		t.Errorf("expected Repo.Sha 'after456', got %q", result.Repo.Sha)
	}

	// Verify repo fields
	if result.Repo.Name != "myrepo" {
		t.Errorf("expected name 'myrepo', got %q", result.Repo.Name)
	}
	if result.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got %q", result.Repo.RepoType)
	}
	if result.Repo.RepoOpenid != "42" {
		t.Errorf("expected RepoOpenid '42', got %q", result.Repo.RepoOpenid)
	}
	if result.Repo.FullName != "group/myrepo" {
		t.Errorf("expected FullName 'group/myrepo', got %q", result.Repo.FullName)
	}
	if result.Repo.Description != "test repo" {
		t.Errorf("expected description 'test repo', got %q", result.Repo.Description)
	}
	if result.Repo.Owner != "pusher" {
		t.Errorf("expected owner 'pusher', got %q", result.Repo.Owner)
	}

	// Verify sender
	if result.Sender.UserName != "pusher" {
		t.Errorf("expected sender 'pusher', got %q", result.Sender.UserName)
	}
}

// --- Parse error paths ---

func TestParse_ReadBodyError(t *testing.T) {
	// Create a request with a body that returns error on read
	errReader := &errorReader{err: errors.New("read error")}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", errReader)
	req.Header.Set(hook.GitlabEvent, hook.GitlabEventPush)

	_, err := Parse(req, "secret")
	if err == nil {
		t.Fatal("expected error when body read fails")
	}
}

// errorReader is an io.Reader that always returns an error.
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, e.err
}

// Ensure unused import doesn't cause issues
var _ = bytes.NewReader
