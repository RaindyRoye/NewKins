package github

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

// TestConvertPullRequestURLSuccess verifies that convertPullRequestURL
// fetches a PR from a mock GitHub API server and parses the response.
func TestConvertPullRequestURLSuccess(t *testing.T) {
	prResp := map[string]any{
		"url":    "https://api.github.com/repos/user/repo/pulls/42",
		"id":     100,
		"number": 42,
		"title":  "Test PR via API",
		"body":   "PR body text",
		"state":  "open",
		"user":   map[string]any{"login": "prauthor"},
		"head": map[string]any{
			"ref": "feature-branch",
			"sha": "headsha123",
			"repo": map[string]any{
				"id": 2, "name": "repo-fork", "full_name": "forker/repo",
				"clone_url": "https://github.com/forker/repo.git",
				"html_url":  "https://github.com/forker/repo",
				"git_url":   "git://github.com/forker/repo.git",
				"ssh_url":   "git@github.com:forker/repo.git",
				"svn_url":   "https://github.com/forker/repo",
				"private":   false,
				"owner":     map[string]any{"login": "forker"},
			},
		},
		"base": map[string]any{
			"ref": "main",
			"sha": "basesha123",
			"repo": map[string]any{
				"id": 1, "name": "repo", "full_name": "user/repo",
				"clone_url": "https://github.com/user/repo.git",
				"html_url":  "https://github.com/user/repo",
				"git_url":   "git://github.com/user/repo.git",
				"ssh_url":   "git@github.com:user/repo.git",
				"svn_url":   "https://github.com/user/repo",
				"private":   false,
				"owner":     map[string]any{"login": "user"},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(prResp)
	}))
	defer srv.Close()

	result, err := convertPullRequestURL(srv.URL)
	if err != nil {
		t.Fatalf("convertPullRequestURL failed: %v", err)
	}
	if result.Number != 42 {
		t.Errorf("expected PR number 42, got %d", result.Number)
	}
	if result.Title != "Test PR via API" {
		t.Errorf("expected title 'Test PR via API', got %q", result.Title)
	}
	if result.Head.Ref != "feature-branch" {
		t.Errorf("expected head ref 'feature-branch', got %q", result.Head.Ref)
	}
	if result.Base.Ref != "main" {
		t.Errorf("expected base ref 'main', got %q", result.Base.Ref)
	}
}

// TestConvertPullRequestURLBadURL verifies error handling for an unreachable URL.
func TestConvertPullRequestURLBadURL(t *testing.T) {
	_, err := convertPullRequestURL("http://127.0.0.1:1/unreachable")
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
}

// TestConvertPullRequestURLBadJSON verifies error handling when the
// server returns invalid JSON.
func TestConvertPullRequestURLBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer srv.Close()

	_, err := convertPullRequestURL(srv.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

// TestConvertPullRequestURLServerError verifies behavior when the
// server returns a 500 status with a body that still parses.
func TestConvertPullRequestURLServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()

	result, err := convertPullRequestURL(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Number != 0 {
		t.Errorf("expected number 0, got %d", result.Number)
	}
}

// TestParseCommentHookInvalidJSON verifies that parseCommentHook returns
// an error when the body is not valid JSON.
func TestParseCommentHookInvalidJSON(t *testing.T) {
	_, err := parseCommentHook([]byte(`{bad json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestConvertCommentHookPRFetchError verifies error propagation when the
// PR URL fetch fails during convertCommentHook.
func TestConvertCommentHookPRFetchError(t *testing.T) {
	commentPayload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number": 1,
			"pull_request": map[string]any{
				"url": "http://127.0.0.1:1/unreachable",
			},
		},
		"comment": map[string]any{
			"body": "test",
			"user": map[string]any{"login": "u"},
		},
		"repository": map[string]any{
			"id": 1, "name": "r",
			"owner": map[string]any{"login": "o"},
		},
		"sender": map[string]any{"login": "s"},
	}
	body, _ := json.Marshal(commentPayload)

	_, err := parseCommentHook(body)
	if err == nil {
		t.Fatal("expected error when PR URL fetch fails")
	}
}

// TestParseCommentHookFullFlow tests the full Parse -> parseCommentHook ->
// convertCommentHook -> convertPullRequestURL chain with a mock server.
func TestParseCommentHookFullFlow(t *testing.T) {
	prResp := map[string]any{
		"url":    "https://api.github.com/repos/user/repo/pulls/7",
		"id":     200,
		"number": 7,
		"title":  "Comment PR",
		"body":   "PR body",
		"state":  "open",
		"user":   map[string]any{"login": "author"},
		"head": map[string]any{
			"ref": "fix-branch",
			"sha": "headabc",
			"repo": map[string]any{
				"id": 10, "name": "repo-fork", "full_name": "forker/repo",
				"clone_url": "https://github.com/forker/repo.git",
				"html_url":  "https://github.com/forker/repo",
				"git_url":   "git://github.com/forker/repo.git",
				"ssh_url":   "git@github.com:forker/repo.git",
				"svn_url":   "https://github.com/forker/repo",
				"private":   false,
				"owner":     map[string]any{"login": "forker"},
			},
		},
		"base": map[string]any{
			"ref": "main",
			"sha": "baseabc",
			"repo": map[string]any{
				"id": 5, "name": "repo", "full_name": "user/repo",
				"clone_url": "https://github.com/user/repo.git",
				"html_url":  "https://github.com/user/repo",
				"git_url":   "git://github.com/user/repo.git",
				"ssh_url":   "git@github.com:user/repo.git",
				"svn_url":   "https://github.com/user/repo",
				"private":   false,
				"owner":     map[string]any{"login": "user"},
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(prResp)
	}))
	defer srv.Close()

	commentPayload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number": 7,
			"title":  "Issue title",
			"pull_request": map[string]any{
				"url": srv.URL,
			},
		},
		"comment": map[string]any{
			"id":   300,
			"body": "LGTM!",
			"user": map[string]any{"login": "reviewer"},
		},
		"repository": map[string]any{
			"id": 5, "name": "repo", "full_name": "user/repo", "private": false,
			"owner": map[string]any{"login": "user"},
		},
		"sender": map[string]any{"login": "reviewer"},
	}
	body, _ := json.Marshal(commentPayload)
	secret := "comment-secret"
	req := newGithubRequest(hook.GithubEventIssueComment, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse comment hook failed: %v", err)
	}

	commentHook, ok := wh.(*hook.PullRequestCommentHook)
	if !ok {
		t.Fatalf("expected *hook.PullRequestCommentHook, got %T", wh)
	}
	if commentHook.Action != hook.EventsTypeComment {
		t.Errorf("expected action %q, got %q", hook.EventsTypeComment, commentHook.Action)
	}
	if commentHook.Comment.Body != "LGTM!" {
		t.Errorf("expected comment body 'LGTM!', got %q", commentHook.Comment.Body)
	}
	if commentHook.Comment.Author.UserName != "reviewer" {
		t.Errorf("expected comment author 'reviewer', got %q", commentHook.Comment.Author.UserName)
	}
	if commentHook.Sender.UserName != "reviewer" {
		t.Errorf("expected sender 'reviewer', got %q", commentHook.Sender.UserName)
	}
	if commentHook.PullRequest.Number != 7 {
		t.Errorf("expected PR number 7, got %d", commentHook.PullRequest.Number)
	}
	if commentHook.Repo.RepoType != "github" {
		t.Errorf("expected repoType 'github', got %q", commentHook.Repo.RepoType)
	}
}

// TestParseCommentHookSecretFailure verifies that comment parsing returns
// ErrSecretValidationFailed on bad signature.
func TestParseCommentHookSecretFailure(t *testing.T) {
	commentPayload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number":         1,
			"pull_request":   map[string]any{"url": "http://127.0.0.1:1/unreachable"},
		},
		"comment": map[string]any{
			"body": "test",
			"user": map[string]any{"login": "u"},
		},
		"repository": map[string]any{
			"id": 1, "name": "r",
			"owner": map[string]any{"login": "o"},
		},
		"sender": map[string]any{"login": "s"},
	}
	body, _ := json.Marshal(commentPayload)
	req := newGithubRequest(hook.GithubEventIssueComment, body, "correct")

	_, err := Parse(req, "wrong")
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

// TestValidateSHA1Prefix tests that validatePrefix handles SHA1 signatures.
func TestValidateSHA1Prefix(t *testing.T) {
	message := []byte("sha1 test body")
	key := []byte("sha1secret")

	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	sig := "sha1=" + hex.EncodeToString(mac.Sum(nil))

	if !validatePrefix(message, key, sig) {
		t.Error("validatePrefix should accept valid sha1 signature")
	}
}

// TestValidatePrefixSHA256 tests that validatePrefix handles SHA256 signatures.
func TestValidatePrefixSHA256(t *testing.T) {
	message := []byte("sha256 test body")
	key := []byte("sha256secret")

	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !validatePrefix(message, key, sig) {
		t.Error("validatePrefix should accept valid sha256 signature")
	}
}
