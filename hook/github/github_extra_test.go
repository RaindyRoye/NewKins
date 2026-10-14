package github

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // SHA1 is required for GitHub webhook signature verification compatibility
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

// TestParseCommentHook tests the comment hook parsing with mocked HTTP server
func TestParseCommentHook(t *testing.T) {
	// Mock server for PR info fetch
	prInfo := map[string]any{
		"url":        "https://api.github.com/repos/user/test-repo/pulls/42",
		"id":         42,
		"number":     42,
		"title":      "Test PR",
		"body":       "PR body",
		"html_url":   "https://github.com/user/test-repo/pull/42",
		"diff_url":   "https://github.com/user/test-repo/pull/42.diff",
		"patch_url":  "https://github.com/user/test-repo/pull/42.patch",
		"issue_url":  "https://api.github.com/repos/user/test-repo/issues/42",
		"state":      "open",
		"locked":     false,
		"created_at": "2021-01-01T00:00:00Z",
		"updated_at": "2021-01-01T00:00:00Z",
		"user": map[string]any{
			"login": "prauthor",
			"id":    1,
		},
		"head": map[string]any{
			"label": "user:feature-branch",
			"ref":   "feature-branch",
			"sha":   "head123",
			"user": map[string]any{
				"login": "user",
				"id":    1,
			},
			"repo": map[string]any{
				"id":          2,
				"name":        "test-repo-fork",
				"full_name":   "forker/test-repo",
				"clone_url":   "https://github.com/forker/test-repo.git",
				"html_url":    "https://github.com/forker/test-repo",
				"git_url":     "git://github.com/forker/test-repo.git",
				"ssh_url":     "git@github.com:forker/test-repo.git",
				"svn_url":     "https://github.com/forker/test-repo",
				"created_at":  "2021-01-01T00:00:00Z",
				"description": "forked repo",
				"private":     false,
				"owner": map[string]any{
					"login": "forker",
				},
			},
		},
		"base": map[string]any{
			"label": "user:main",
			"ref":   "main",
			"sha":   "base123",
			"user": map[string]any{
				"login": "user",
				"id":    1,
			},
			"repo": map[string]any{
				"id":          1,
				"name":        "test-repo",
				"full_name":   "user/test-repo",
				"clone_url":   "https://github.com/user/test-repo.git",
				"html_url":    "https://github.com/user/test-repo",
				"git_url":     "git://github.com/user/test-repo.git",
				"ssh_url":     "git@github.com:user/test-repo.git",
				"svn_url":     "https://github.com/user/test-repo",
				"created_at":  "2021-01-01T00:00:00Z",
				"description": "base repo",
				"private":     false,
				"owner": map[string]any{
					"login": "user",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(prInfo); err != nil {
			t.Logf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	payload := map[string]any{
		"action": "created",
		"comment": map[string]any{
			"id":   1,
			"body": "This is a test comment",
			"user": map[string]any{
				"login": "commenter",
			},
		},
		"issue": map[string]any{
			"id":     1,
			"number": 42,
			"title":  "Test Issue",
			"pull_request": map[string]any{
				"url":       server.URL,
				"html_url":  "https://github.com/user/test-repo/pull/42",
				"diff_url":  "https://github.com/user/test-repo/pull/42.diff",
				"patch_url": "https://github.com/user/test-repo/pull/42.patch",
			},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "test-repo",
			"full_name": "user/test-repo",
		},
		"sender": map[string]any{
			"login": "commenter",
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventIssueComment, body, secret)

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
	if commentHook.Comment.Author.UserName != "commenter" {
		t.Errorf("expected comment author 'commenter', got '%s'", commentHook.Comment.Author.UserName)
	}
	if commentHook.Repo.RepoType != "github" {
		t.Errorf("expected repoType 'github', got '%s'", commentHook.Repo.RepoType)
	}
	if commentHook.PullRequest.Number != 42 {
		t.Errorf("expected PR number 42, got %d", commentHook.PullRequest.Number)
	}
}

// TestParseCommentHookHTTPError tests comment hook when PR info fetch fails
func TestParseCommentHookHTTPError(t *testing.T) {
	// Mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	payload := map[string]any{
		"action": "created",
		"comment": map[string]any{
			"id":   1,
			"body": "Test comment",
			"user": map[string]any{
				"login": "commenter",
			},
		},
		"issue": map[string]any{
			"id":     1,
			"number": 42,
			"title":  "Test Issue",
			"pull_request": map[string]any{
				"url": server.URL,
			},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "test-repo",
			"full_name": "user/test-repo",
		},
		"sender": map[string]any{
			"login": "commenter",
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventIssueComment, body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error when PR info fetch fails")
	}
}

// TestParseCommentHookInvalidJSON tests comment hook with invalid JSON
func TestParseCommentHookInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	req := newGithubRequest(hook.GithubEventIssueComment, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestParsePullRequestHookSynchronizeAction tests PR hook with synchronize action
func TestParsePullRequestHookSynchronizeAction(t *testing.T) {
	payload := map[string]any{
		"action": "synchronize",
		"number": 42,
		"pull_request": map[string]any{
			"title": "Test PR",
			"body":  "PR body",
			"head": map[string]any{
				"ref": "feature-branch",
				"sha": "head123",
				"repo": map[string]any{
					"id":          2,
					"name":        "test-repo-fork",
					"full_name":   "forker/test-repo",
					"clone_url":   "https://github.com/forker/test-repo.git",
					"html_url":    "https://github.com/forker/test-repo",
					"git_url":     "git://github.com/forker/test-repo.git",
					"ssh_url":     "git@github.com:forker/test-repo.git",
					"svn_url":     "https://github.com/forker/test-repo",
					"created_at":  "2021-01-01T00:00:00Z",
					"description": "forked repo",
					"private":     false,
					"owner": map[string]any{
						"login": "forker",
					},
				},
			},
			"base": map[string]any{
				"ref": "main",
				"sha": "base123",
				"repo": map[string]any{
					"id":          1,
					"name":        "test-repo",
					"full_name":   "user/test-repo",
					"clone_url":   "https://github.com/user/test-repo.git",
					"html_url":    "https://github.com/user/test-repo",
					"git_url":     "git://github.com/user/test-repo.git",
					"ssh_url":     "git@github.com:user/test-repo.git",
					"svn_url":     "https://github.com/user/test-repo",
					"created_at":  "2021-01-01T00:00:00Z",
					"description": "base repo",
					"private":     false,
					"owner": map[string]any{
						"login": "user",
					},
				},
			},
			"user": map[string]any{
				"login": "prauthor",
			},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "test-repo",
			"full_name": "user/test-repo",
		},
		"sender": map[string]any{
			"login": "prauthor",
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventPR, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse PR hook with synchronize action failed: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestHook type")
	}

	// "synchronize" should be mapped to "update"
	if prHook.Action != hook.ActionUpdate {
		t.Errorf("expected action '%s', got '%s'", hook.ActionUpdate, prHook.Action)
	}
}

// TestParsePushHookEmptyRef tests push hook with empty ref
func TestParsePushHookEmptyRef(t *testing.T) {
	payload := map[string]any{
		"ref":    "",
		"before": "abc123",
		"after":  "def456",
		"commits": []map[string]any{
			{
				"id":      "def456",
				"message": "test commit",
				"url":     "https://github.com/user/test-repo/commit/def456",
			},
		},
		"repository": map[string]any{
			"id":          1,
			"name":        "test-repo",
			"full_name":   "user/test-repo",
			"clone_url":   "https://github.com/user/test-repo.git",
			"html_url":    "https://github.com/user/test-repo",
			"ssh_url":     "git@github.com:user/test-repo.git",
			"svn_url":     "https://github.com/user/test-repo",
			"git_url":     "git://github.com/user/test-repo.git",
			"description": "A test repository",
			"private":     false,
			"created_at":  1609459200,
			"owner": map[string]any{
				"login": "testuser",
			},
		},
		"sender": map[string]any{
			"login": "testuser",
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventPush, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse push hook with empty ref failed: %v", err)
	}

	pushHook, ok := wh.(*hook.PushHook)
	if !ok {
		t.Fatal("expected *hook.PushHook type")
	}

	if pushHook.Repo.Branch != "" {
		t.Errorf("expected empty branch, got '%s'", pushHook.Repo.Branch)
	}
}

// TestValidatePrefixSHA1Extra tests validatePrefix with SHA1 signature
func TestValidatePrefixSHA1Extra(t *testing.T) {
	message := []byte("test body")
	key := []byte("mysecret")

	// Create SHA1 signature
	mac := newHMACSHA1(key, message)
	sig := "sha1=" + mac

	if !validatePrefix(message, key, sig) {
		t.Error("validatePrefix should accept valid sha1 signature")
	}
}

// Helper function to create SHA1 HMAC
func newHMACSHA1(key, message []byte) string {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}
