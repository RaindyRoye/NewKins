package github

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // SHA1 needed for GitHub webhook sig testing
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

// computeSHA1Signature returns a sha1=hex(hmac-sha1(key, body)) string.
func computeSHA1Signature(secret, body []byte) string {
	mac := hmac.New(sha1.New, secret)
	mac.Write(body)
	return "sha1=" + hex.EncodeToString(mac.Sum(nil))
}

// TestParseCommentHook covers parseCommentHook and convertCommentHook which were at 0%.
// It uses httptest.NewServer to mock the GitHub API endpoint that convertPullRequestURL hits.
func TestParseCommentHook(t *testing.T) {
	// Set up a fake GitHub API server that returns PR info
	prResp := map[string]any{
		"number": 42,
		"title":  "Test Pull Request",
		"body":   "PR body text",
		"head": map[string]any{
			"ref": "feature-branch",
			"sha": "headsha123",
			"repo": map[string]any{
				"id":        2,
				"name":      "fork-repo",
				"full_name": "forker/fork-repo",
				"clone_url": "https://github.com/forker/fork-repo.git",
				"html_url":  "https://github.com/forker/fork-repo",
				"git_url":   "git://github.com/forker/fork-repo.git",
				"ssh_url":   "git@github.com:forker/fork-repo.git",
				"svn_url":   "https://github.com/forker/fork-repo",
				"owner":     map[string]any{"login": "forker"},
			},
		},
		"base": map[string]any{
			"ref": "main",
			"sha": "basesha456",
			"repo": map[string]any{
				"id":        1,
				"name":      "base-repo",
				"full_name": "owner/base-repo",
				"clone_url": "https://github.com/owner/base-repo.git",
				"html_url":  "https://github.com/owner/base-repo",
				"git_url":   "git://github.com/owner/base-repo.git",
				"ssh_url":   "git@github.com:owner/base-repo.git",
				"svn_url":   "https://github.com/owner/base-repo",
				"owner":     map[string]any{"login": "owner"},
			},
		},
		"user": map[string]any{"login": "prauthor"},
	}
	prJSON, _ := json.Marshal(prResp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(prJSON)
	}))
	defer srv.Close()

	payload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"url":    srv.URL,
			"number": 42,
			"title":  "Test PR Issue",
			"user":   map[string]any{"login": "issueauthor"},
			"pull_request": map[string]any{
				"url": srv.URL,
			},
		},
		"comment": map[string]any{
			"id":   100,
			"body": "Nice PR, looks good!",
			"user": map[string]any{"login": "commenter"},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "base-repo",
			"full_name": "owner/base-repo",
			"private":   false,
			"owner":     map[string]any{"login": "owner"},
		},
		"sender": map[string]any{"login": "commenter"},
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
	req.Header.Set(hook.GithubEvent, hook.GithubEventIssueComment)
	req.Header.Set("X-Hub-Signature", computeSignature([]byte(secret), body))

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
	if commentHook.Comment.Body != "Nice PR, looks good!" {
		t.Errorf("expected comment body, got '%s'", commentHook.Comment.Body)
	}
	if commentHook.Comment.Author.UserName != "commenter" {
		t.Errorf("expected comment author 'commenter', got '%s'", commentHook.Comment.Author.UserName)
	}
	if commentHook.Repo.RepoType != "github" {
		t.Errorf("expected repoType 'github', got '%s'", commentHook.Repo.RepoType)
	}
	if commentHook.PullRequest.Title != "Test Pull Request" {
		t.Errorf("expected PR title 'Test Pull Request', got '%s'", commentHook.PullRequest.Title)
	}
	if commentHook.Sender.UserName != "commenter" {
		t.Errorf("expected sender 'commenter', got '%s'", commentHook.Sender.UserName)
	}
}

// TestParseCommentHookInvalidJSON ensures parseCommentHook returns an error for bad JSON.
func TestParseCommentHookInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/webhook",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GithubEvent, hook.GithubEventIssueComment)
	req.Header.Set("X-Hub-Signature", computeSignature([]byte(testSecret), body))

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment hook")
	}
}

// TestValidatePrefixSHA1 verifies that validatePrefix accepts sha1-prefixed signatures.
func TestValidatePrefixSHA1(t *testing.T) {
	message := []byte("test body")
	key := []byte("mysecret")

	mac := hmac.New(sha1.New, key) //nolint:gosec
	mac.Write(message)
	sig := "sha1=" + hex.EncodeToString(mac.Sum(nil))

	if !validatePrefix(message, key, sig) {
		t.Error("validatePrefix should accept valid sha1 signature")
	}
}

// TestParsePushHookSHA1Signature verifies that push hooks accept sha1 signatures.
func TestParsePushHookSHA1Signature(t *testing.T) {
	payload := map[string]any{
		"ref":    "refs/heads/main",
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
			"id":        1,
			"name":      "test-repo",
			"full_name": "user/test-repo",
			"clone_url": "https://github.com/user/test-repo.git",
			"html_url":  "https://github.com/user/test-repo",
			"owner":     map[string]any{"login": "testuser"},
		},
		"sender": map[string]any{"login": "testuser"},
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
	req.Header.Set(hook.GithubEvent, hook.GithubEventPush)
	// Use SHA1 signature instead of SHA256
	req.Header.Set("X-Hub-Signature", computeSHA1Signature([]byte(secret), body))

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse with SHA1 signature failed: %v", err)
	}
	if wh == nil {
		t.Fatal("expected non-nil webhook result")
	}
}

// TestParsePRSynchronizeAction verifies that "synchronize" action is mapped to "update".
func TestParsePRSynchronizeAction(t *testing.T) {
	payload := map[string]any{
		"action": "synchronize",
		"number": 42,
		"pull_request": map[string]any{
			"title": "Synced PR",
			"head": map[string]any{
				"ref":  "feature",
				"sha":  "newsha",
				"repo": map[string]any{"name": "fork-repo"},
			},
			"base": map[string]any{
				"ref":  "main",
				"sha":  "basesha",
				"repo": map[string]any{"name": "base-repo"},
			},
			"user": map[string]any{"login": "author"},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "base-repo",
			"full_name": "owner/base-repo",
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
	req.Header.Set(hook.GithubEvent, hook.GithubEventPR)
	req.Header.Set("X-Hub-Signature", computeSignature([]byte(secret), body))

	wh, err := Parse(req, secret)
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
}

// TestParsePREmptyAction verifies that a PR with no action field doesn't crash.
func TestParsePREmptyAction(t *testing.T) {
	payload := map[string]any{
		"number": 10,
		"pull_request": map[string]any{
			"title": "No Action PR",
			"head": map[string]any{
				"ref":  "feat",
				"sha":  "sha1",
				"repo": map[string]any{"name": "repo"},
			},
			"base": map[string]any{
				"ref":  "main",
				"sha":  "sha2",
				"repo": map[string]any{"name": "repo"},
			},
			"user": map[string]any{"login": "user"},
		},
		"repository": map[string]any{
			"id":   1,
			"name": "repo",
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
	req.Header.Set(hook.GithubEvent, hook.GithubEventPR)
	req.Header.Set("X-Hub-Signature", computeSignature([]byte(secret), body))

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse PR with empty action failed: %v", err)
	}
	if wh == nil {
		t.Fatal("expected non-nil webhook result")
	}
}

// TestValidateInvalidHex ensures Validate returns false for non-hex encoded signatures.
func TestValidateInvalidHex(t *testing.T) {
	if Validate(sha1.New, []byte("msg"), []byte("key"), "zz-not-hex-zz") { //nolint:gosec
		t.Error("Validate should return false for non-hex signature")
	}
}

// TestConvertPullRequestURLNetworkError verifies convertPullRequestURL handles
// unreachable servers gracefully.
func TestConvertPullRequestURLNetworkError(t *testing.T) {
	_, err := convertPullRequestURL("http://127.0.0.1:1/nonexistent")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}

// TestConvertPullRequestURLSuccess verifies convertPullRequestURL works with
// a valid HTTP endpoint.
func TestConvertPullRequestURLSuccess(t *testing.T) {
	prResp := map[string]any{
		"number": 7,
		"title":  "Test PR",
		"head": map[string]any{
			"ref":  "feat",
			"sha":  "abc",
			"repo": map[string]any{"name": "repo"},
		},
		"base": map[string]any{
			"ref":  "main",
			"sha":  "def",
			"repo": map[string]any{"name": "repo"},
		},
	}
	prJSON, _ := json.Marshal(prResp)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(prJSON)
	}))
	defer srv.Close()

	result, err := convertPullRequestURL(srv.URL)
	if err != nil {
		t.Fatalf("convertPullRequestURL failed: %v", err)
	}
	if result.Number != 7 {
		t.Errorf("expected number 7, got %d", result.Number)
	}
	if result.Title != "Test PR" {
		t.Errorf("expected title 'Test PR', got '%s'", result.Title)
	}
}

// TestConvertPullRequestURLInvalidJSON verifies convertPullRequestURL handles
// invalid JSON in the response.
func TestConvertPullRequestURLInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid`))
	}))
	defer srv.Close()

	_, err := convertPullRequestURL(srv.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}
