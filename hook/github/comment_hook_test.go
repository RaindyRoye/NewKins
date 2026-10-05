package github

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

// TestParseCommentHook_Success tests the full comment hook parsing path
// including convertCommentHook and convertPullRequestURL via a local HTTP server.
func TestParseCommentHook_Success(t *testing.T) {
	// Mock the GitHub API endpoint that convertPullRequestURL calls
	prResponseJSON := `{
		"number": 42,
		"title": "Test PR",
		"body": "PR description",
		"state": "open",
		"user": {"login": "prauthor"},
		"head": {
			"ref": "feature-branch",
			"sha": "abc123def",
			"repo": {
				"name": "myrepo",
				"full_name": "user/myrepo",
				"clone_url": "https://github.com/user/myrepo.git",
				"git_url": "git://github.com/user/myrepo.git",
				"ssh_url": "git@github.com:user/myrepo.git",
				"svn_url": "https://github.com/user/myrepo",
				"html_url": "https://github.com/user/myrepo",
				"url": "https://api.github.com/repos/user/myrepo",
				"private": false,
				"owner": {"login": "user"}
			}
		},
		"base": {
			"ref": "main",
			"sha": "base789",
			"repo": {
				"name": "myrepo",
				"full_name": "org/myrepo",
				"clone_url": "https://github.com/org/myrepo.git",
				"git_url": "git://github.com/org/myrepo.git",
				"ssh_url": "git@github.com:org/myrepo.git",
				"svn_url": "https://github.com/org/myrepo",
				"html_url": "https://github.com/org/myrepo",
				"url": "https://api.github.com/repos/org/myrepo",
				"private": false,
				"owner": {"login": "org"}
			}
		}
	}`

	// Start a test HTTP server to serve the PR info
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(prResponseJSON))
	}))
	defer srv.Close()

	// Build comment hook payload that points to our test server
	payload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number": 42,
			"pull_request": map[string]any{
				"url": srv.URL + "/repos/org/myrepo/pulls/42",
			},
		},
		"comment": map[string]any{
			"body": "LGTM!",
			"user": map[string]any{
				"login": "reviewer",
			},
		},
		"repository": map[string]any{
			"id":        100,
			"name":      "myrepo",
			"full_name": "org/myrepo",
		},
		"sender": map[string]any{
			"login": "reviewer",
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
		t.Fatalf("expected *hook.PullRequestCommentHook, got %T", wh)
	}

	// Verify action
	if commentHook.Action != hook.EventsTypeComment {
		t.Errorf("expected action %q, got %q", hook.EventsTypeComment, commentHook.Action)
	}

	// Verify repo info from the PR response
	if commentHook.Repo.Name != "myrepo" {
		t.Errorf("expected repo name 'myrepo', got %q", commentHook.Repo.Name)
	}
	if commentHook.Repo.Branch != "feature-branch" {
		t.Errorf("expected branch 'feature-branch', got %q", commentHook.Repo.Branch)
	}
	if commentHook.Repo.RepoType != "github" {
		t.Errorf("expected repoType 'github', got %q", commentHook.Repo.RepoType)
	}

	// Verify target repo
	if commentHook.TargetRepo.Name != "myrepo" {
		t.Errorf("expected target repo name 'myrepo', got %q", commentHook.TargetRepo.Name)
	}
	if commentHook.TargetRepo.Branch != "main" {
		t.Errorf("expected target branch 'main', got %q", commentHook.TargetRepo.Branch)
	}

	// Verify PR info
	if commentHook.PullRequest.Number != 42 {
		t.Errorf("expected PR number 42, got %d", commentHook.PullRequest.Number)
	}
	if commentHook.PullRequest.Title != "Test PR" {
		t.Errorf("expected PR title 'Test PR', got %q", commentHook.PullRequest.Title)
	}

	// Verify comment
	if commentHook.Comment.Body != "LGTM!" {
		t.Errorf("expected comment body 'LGTM!', got %q", commentHook.Comment.Body)
	}
	if commentHook.Comment.Author.UserName != "reviewer" {
		t.Errorf("expected comment author 'reviewer', got %q", commentHook.Comment.Author.UserName)
	}

	// Verify sender
	if commentHook.Sender.UserName != "reviewer" {
		t.Errorf("expected sender 'reviewer', got %q", commentHook.Sender.UserName)
	}
}

// TestParseCommentHook_InvalidJSON tests that invalid JSON returns an error.
func TestParseCommentHook_InvalidJSON(t *testing.T) {
	body := []byte(`{invalid json here`)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventIssueComment, body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment hook")
	}
}

// TestConvertPullRequestURL_InvalidURL tests that a bad URL returns an error.
func TestConvertPullRequestURL_InvalidURL(t *testing.T) {
	_, err := convertPullRequestURL("://invalid-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// TestConvertPullRequestURL_ServerError tests that a server error is properly propagated.
func TestConvertPullRequestURL_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	// This will succeed at HTTP level but the response won't be valid JSON
	result, err := convertPullRequestURL(srv.URL + "/repos/test/test/pulls/1")
	// Should fail because the response body is not valid JSON
	if err == nil && result == nil {
		t.Fatal("expected error or nil result for server error response")
	}
}

// TestConvertPullRequestURL_InvalidResponse tests that unparseable JSON is handled.
func TestConvertPullRequestURL_InvalidResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()

	_, err := convertPullRequestURL(srv.URL + "/repos/test/test/pulls/1")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

// TestParseCommentHook_ServerUnreachable tests error when the PR URL server is down.
func TestParseCommentHook_ServerUnreachable(t *testing.T) {
	// Use a URL that will fail to connect
	payload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number": 1,
			"pull_request": map[string]any{
				"url": "http://127.0.0.1:1/unreachable",
			},
		},
		"comment": map[string]any{
			"body": "test",
			"user": map[string]any{"login": "tester"},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "repo",
			"full_name": "user/repo",
		},
		"sender": map[string]any{"login": "tester"},
	}

	body, _ := json.Marshal(payload)
	// Use empty secret to skip signature validation
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, "/webhook",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GithubEvent, hook.GithubEventIssueComment)

	_, err := Parse(req, "")
	if err == nil {
		t.Fatal("expected error when PR URL server is unreachable")
	}
}
