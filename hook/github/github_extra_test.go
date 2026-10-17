package github

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // SHA1 is required for GitHub webhook signature verification
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gokins/gokins/hook"
)

// TestParsePushHookSHA1Signature verifies that sha1-based X-Hub-Signature is accepted.
func TestParsePushHookSHA1Signature(t *testing.T) {
	payload := map[string]any{
		"ref":    "refs/heads/main",
		"before": "aaa111",
		"after":  "bbb222",
		"commits": []map[string]any{
			{
				"id":      "bbb222",
				"message": "sha1 commit",
				"url":     "https://github.com/user/repo/commit/bbb222",
			},
		},
		"repository": map[string]any{
			"id":          10,
			"name":        "repo",
			"full_name":   "user/repo",
			"clone_url":   "https://github.com/user/repo.git",
			"html_url":    "https://github.com/user/repo",
			"ssh_url":     "git@github.com:user/repo.git",
			"svn_url":     "https://github.com/user/repo",
			"git_url":     "git://github.com/user/repo.git",
			"description": "",
			"private":     true,
			"created_at":  1609459200,
			"owner": map[string]any{
				"login": "user",
			},
		},
		"sender": map[string]any{
			"login": "user",
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret

	// Use SHA1 signature instead of SHA256
	mac := hmac.New(sha1.New, []byte(secret)) //nolint:gosec
	mac.Write(body)
	sig := "sha1=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GithubEvent, hook.GithubEventPush)
	req.Header.Set("X-Hub-Signature", sig)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse with sha1 signature failed: %v", err)
	}

	pushHook, ok := wh.(*hook.PushHook)
	if !ok {
		t.Fatal("expected *hook.PushHook type")
	}
	if pushHook.Repo.Private != true {
		t.Error("expected repo to be private")
	}
}

// TestValidatePrefixSHA1Algo verifies that validatePrefix handles sha1 signatures.
func TestValidatePrefixSHA1Algo(t *testing.T) {
	message := []byte("hello sha1")
	key := []byte("mysecret")

	mac := hmac.New(sha1.New, key) //nolint:gosec
	mac.Write(message)
	sig := "sha1=" + hex.EncodeToString(mac.Sum(nil))

	if !validatePrefix(message, key, sig) {
		t.Error("validatePrefix should accept valid sha1 signature")
	}
}

// TestParsePullRequestSynchronize verifies the "synchronize" action is mapped to "update".
func TestParsePullRequestSynchronize(t *testing.T) {
	payload := map[string]any{
		"action": "synchronize",
		"number": 7,
		"pull_request": map[string]any{
			"title": "Sync PR",
			"body":  "updated",
			"head": map[string]any{
				"ref": "feature",
				"sha": "newhead",
				"repo": map[string]any{
					"id":          3,
					"name":        "fork",
					"full_name":   "dev/fork",
					"clone_url":   "https://github.com/dev/fork.git",
					"html_url":    "https://github.com/dev/fork",
					"git_url":     "git://github.com/dev/fork.git",
					"ssh_url":     "git@github.com:dev/fork.git",
					"svn_url":     "https://github.com/dev/fork",
					"created_at":  "2022-06-01T12:00:00Z",
					"description": "dev fork",
					"private":     false,
					"owner": map[string]any{
						"login": "dev",
					},
				},
			},
			"base": map[string]any{
				"ref": "main",
				"sha": "basesha",
				"repo": map[string]any{
					"id":          1,
					"name":        "repo",
					"full_name":   "org/repo",
					"clone_url":   "https://github.com/org/repo.git",
					"html_url":    "https://github.com/org/repo",
					"git_url":     "git://github.com/org/repo.git",
					"ssh_url":     "git@github.com:org/repo.git",
					"svn_url":     "https://github.com/org/repo",
					"created_at":  "2021-01-01T00:00:00Z",
					"description": "base",
					"private":     false,
					"owner": map[string]any{
						"login": "org",
					},
				},
			},
			"user": map[string]any{
				"login": "dev",
			},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "repo",
			"full_name": "org/repo",
		},
	}

	body, _ := json.Marshal(payload)
	req := newGithubRequest(hook.GithubEventPR, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse synchronize PR failed: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestHook type")
	}
	if prHook.Action != hook.ActionUpdate {
		t.Errorf("expected action '%s', got '%s'", hook.ActionUpdate, prHook.Action)
	}
	if prHook.PullRequest.Title != "Sync PR" {
		t.Errorf("expected title 'Sync PR', got '%s'", prHook.PullRequest.Title)
	}
}

// TestParsePullRequestNoAction verifies that a PR payload without action still works.
func TestParsePullRequestNoAction(t *testing.T) {
	payload := map[string]any{
		"number": 1,
		"pull_request": map[string]any{
			"title": "No action PR",
			"body":  "body",
			"head": map[string]any{
				"ref":  "feat",
				"sha":  "h1",
				"repo": map[string]any{"id": 2, "name": "f", "full_name": "u/f"},
			},
			"base": map[string]any{
				"ref":  "main",
				"sha":  "b1",
				"repo": map[string]any{"id": 1, "name": "r", "full_name": "u/r"},
			},
			"user": map[string]any{"login": "u"},
		},
		"repository": map[string]any{"id": 1},
	}

	body, _ := json.Marshal(payload)
	req := newGithubRequest(hook.GithubEventPR, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse no-action PR failed: %v", err)
	}
	prHook := wh.(*hook.PullRequestHook)
	if prHook.Action != "" {
		t.Errorf("expected empty action, got '%s'", prHook.Action)
	}
}

// TestParseCommentHookWithMockServer verifies the full comment hook flow
// including the HTTP fetch of the PR URL via a mock server.
func TestParseCommentHookWithMockServer(t *testing.T) {
	// Mock server that responds with a PR payload
	prResponse := map[string]any{
		"number":  99,
		"title":   "Commented PR",
		"body":    "PR body text",
		"state":   "open",
		"locked":  false,
		"html_url": "https://github.com/org/repo/pull/99",
		"user": map[string]any{
			"login": "prauthor",
		},
		"head": map[string]any{
			"ref": "comment-branch",
			"sha": "headsha123",
			"repo": map[string]any{
				"id":          20,
				"name":        "head-repo",
				"full_name":   "forker/head-repo",
				"clone_url":   "https://github.com/forker/head-repo.git",
				"html_url":    "https://github.com/forker/head-repo",
				"git_url":     "git://github.com/forker/head-repo.git",
				"ssh_url":     "git@github.com:forker/head-repo.git",
				"svn_url":     "https://github.com/forker/head-repo",
				"created_at":  "2022-01-01T00:00:00Z",
				"description": "head repo desc",
				"private":     false,
				"owner": map[string]any{
					"login": "forker",
				},
			},
		},
		"base": map[string]any{
			"ref": "main",
			"sha": "basesha456",
			"repo": map[string]any{
				"id":          10,
				"name":        "base-repo",
				"full_name":   "org/base-repo",
				"clone_url":   "https://github.com/org/base-repo.git",
				"html_url":    "https://github.com/org/base-repo",
				"git_url":     "git://github.com/org/base-repo.git",
				"ssh_url":     "git@github.com:org/base-repo.git",
				"svn_url":     "https://github.com/org/base-repo",
				"created_at":  "2021-01-01T00:00:00Z",
				"description": "base repo desc",
				"private":     true,
				"owner": map[string]any{
					"login": "org",
				},
			},
		},
		"created_at": time.Now().Format(time.RFC3339),
		"updated_at": time.Now().Format(time.RFC3339),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prResponse)
	}))
	defer srv.Close()

	commentPayload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"url":    srv.URL,
			"number": 99,
			"title":  "Issue title",
			"pull_request": map[string]any{
				"url": srv.URL,
			},
		},
		"comment": map[string]any{
			"id":   12345,
			"body": "Great PR!",
			"user": map[string]any{
				"login": "commenter",
			},
		},
		"repository": map[string]any{
			"id":        10,
			"name":      "base-repo",
			"full_name": "org/base-repo",
		},
		"sender": map[string]any{
			"login": "commenter",
		},
	}

	body, _ := json.Marshal(commentPayload)
	req := newGithubRequest(hook.GithubEventIssueComment, body, testSecret)

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
	if commentHook.Comment.Body != "Great PR!" {
		t.Errorf("expected comment body 'Great PR!', got '%s'", commentHook.Comment.Body)
	}
	if commentHook.Comment.Author.UserName != "commenter" {
		t.Errorf("expected commenter 'commenter', got '%s'", commentHook.Comment.Author.UserName)
	}
	if commentHook.PullRequest.Number != 99 {
		t.Errorf("expected PR number 99, got %d", commentHook.PullRequest.Number)
	}
	if commentHook.PullRequest.Title != "Commented PR" {
		t.Errorf("expected PR title 'Commented PR', got '%s'", commentHook.PullRequest.Title)
	}
	if commentHook.Repo.Branch != "comment-branch" {
		t.Errorf("expected branch 'comment-branch', got '%s'", commentHook.Repo.Branch)
	}
	if commentHook.Repo.RepoType != "github" {
		t.Errorf("expected repoType 'github', got '%s'", commentHook.Repo.RepoType)
	}
	if commentHook.TargetRepo.Private != true {
		t.Error("expected target repo to be private")
	}
	if commentHook.Sender.UserName != "commenter" {
		t.Errorf("expected sender 'commenter', got '%s'", commentHook.Sender.UserName)
	}
}

// TestParseCommentHookInvalidJSON verifies that invalid JSON in comment payload returns an error.
func TestParseCommentHookInvalidJSON(t *testing.T) {
	body := []byte(`{invalid`)
	req := newGithubRequest(hook.GithubEventIssueComment, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment hook")
	}
}

// TestParseCommentHookBadPRURL verifies that a failing PR URL fetch returns an error.
func TestParseCommentHookBadPRURL(t *testing.T) {
	// Mock server that returns an error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	commentPayload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"pull_request": map[string]any{
				"url": srv.URL,
			},
		},
		"comment": map[string]any{
			"body": "comment",
			"user": map[string]any{"login": "u"},
		},
		"repository": map[string]any{"id": 1},
		"sender":     map[string]any{"login": "u"},
	}

	body, _ := json.Marshal(commentPayload)
	req := newGithubRequest(hook.GithubEventIssueComment, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error when PR URL returns invalid JSON")
	}
}

// TestConvertPullRequestURLBadURL verifies error handling for invalid URL.
func TestConvertPullRequestURLBadURL(t *testing.T) {
	_, err := convertPullRequestURL("://invalid-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// TestConvertPullRequestURLUnreachable verifies error handling for unreachable server.
func TestConvertPullRequestURLUnreachable(t *testing.T) {
	_, err := convertPullRequestURL("http://127.0.0.1:1/nonexistent")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}

// TestValidateInvalidHex verifies that Validate returns false for non-hex signatures.
func TestValidateInvalidHex(t *testing.T) {
	if Validate(sha1.New, []byte("msg"), []byte("key"), "zzzz") { //nolint:gosec
		t.Error("Validate should return false for non-hex signature")
	}
}

// TestParsePushHookEmptyRef verifies branch extraction with unusual ref values.
func TestParsePushHookEmptyRef(t *testing.T) {
	payload := map[string]any{
		"ref":    "",
		"before": "a",
		"after":  "b",
		"commits": []map[string]any{
			{"id": "b", "message": "msg", "url": "http://example.com"},
		},
		"repository": map[string]any{
			"id": 1, "name": "r", "full_name": "u/r",
			"clone_url": "http://x", "html_url": "http://x",
			"ssh_url": "http://x", "svn_url": "http://x",
			"git_url": "http://x", "owner": map[string]any{"login": "u"},
			"created_at": 0,
		},
		"sender": map[string]any{"login": "u"},
	}
	body, _ := json.Marshal(payload)
	req := newGithubRequest(hook.GithubEventPush, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse push hook with empty ref failed: %v", err)
	}
	pushHook := wh.(*hook.PushHook)
	if pushHook.Repo.Branch != "" {
		t.Errorf("expected empty branch, got '%s'", pushHook.Repo.Branch)
	}
}

// TestParsePushHookShortRef verifies branch extraction with a ref that has fewer than 3 segments.
func TestParsePushHookShortRef(t *testing.T) {
	payload := map[string]any{
		"ref":    "main",
		"before": "a",
		"after":  "b",
		"commits": []map[string]any{
			{"id": "b", "message": "msg", "url": "http://example.com"},
		},
		"repository": map[string]any{
			"id": 1, "name": "r", "full_name": "u/r",
			"clone_url": "http://x", "html_url": "http://x",
			"ssh_url": "http://x", "svn_url": "http://x",
			"git_url": "http://x", "owner": map[string]any{"login": "u"},
			"created_at": 0,
		},
		"sender": map[string]any{"login": "u"},
	}
	body, _ := json.Marshal(payload)
	req := newGithubRequest(hook.GithubEventPush, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse push hook with short ref failed: %v", err)
	}
	pushHook := wh.(*hook.PushHook)
	// "main" has only 1 segment, so branch stays as "main" (the full ref)
	if pushHook.Repo.Branch != "main" {
		t.Errorf("expected branch 'main', got '%s'", pushHook.Repo.Branch)
	}
}
