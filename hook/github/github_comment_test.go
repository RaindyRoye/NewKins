package github

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // SHA1 test for GitHub webhook signature
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

// TestParseCommentHookWithServer exercises the full Parse → parseCommentHook →
// convertCommentHook → convertPullRequestURL path using an httptest server
// that returns the pull request metadata GitHub would provide.
func TestParseCommentHookWithServer(t *testing.T) {
	// Mock server that returns PR metadata (what GitHub would return)
	prPayload := map[string]any{
		"id":     100,
		"number": 42,
		"title":  "Fix bug",
		"body":   "Fixes #41",
		"head": map[string]any{
			"ref": "feature-branch",
			"sha": "headsha123",
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
				"owner":       map[string]any{"login": "forker"},
			},
		},
		"base": map[string]any{
			"ref": "main",
			"sha": "basesha123",
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
				"owner":       map[string]any{"login": "user"},
			},
		},
		"user": map[string]any{"login": "prauthor"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(prPayload)
	}))
	defer srv.Close()

	// Build comment payload pointing to the mock server
	commentPayload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"url":    srv.URL,
			"id":     1,
			"number": 42,
			"title":  "Test issue",
			"pull_request": map[string]any{
				"url": srv.URL,
			},
			"user": map[string]any{"login": "issueauthor"},
		},
		"comment": map[string]any{
			"id":   200,
			"body": "LGTM!",
			"user": map[string]any{"login": "reviewer"},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "test-repo",
			"full_name": "user/test-repo",
		},
		"sender": map[string]any{
			"login": "reviewer",
		},
	}

	body, _ := json.Marshal(commentPayload)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventIssueComment, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse comment hook failed: %v", err)
	}

	cmtHook, ok := wh.(*hook.PullRequestCommentHook)
	if !ok {
		t.Fatalf("expected *hook.PullRequestCommentHook, got %T", wh)
	}

	if cmtHook.Action != hook.EventsTypeComment {
		t.Errorf("expected action %q, got %q", hook.EventsTypeComment, cmtHook.Action)
	}
	if cmtHook.Repo.Name != "test-repo-fork" {
		t.Errorf("expected head repo name %q, got %q", "test-repo-fork", cmtHook.Repo.Name)
	}
	if cmtHook.Repo.RepoType != "github" {
		t.Errorf("expected repo type %q, got %q", "github", cmtHook.Repo.RepoType)
	}
	if cmtHook.TargetRepo.Name != "test-repo" {
		t.Errorf("expected target repo name %q, got %q", "test-repo", cmtHook.TargetRepo.Name)
	}
	if cmtHook.Comment.Body != "LGTM!" {
		t.Errorf("expected comment body %q, got %q", "LGTM!", cmtHook.Comment.Body)
	}
	if cmtHook.Comment.Author.UserName != "reviewer" {
		t.Errorf("expected comment author %q, got %q", "reviewer", cmtHook.Comment.Author.UserName)
	}
	if cmtHook.Sender.UserName != "reviewer" {
		t.Errorf("expected sender %q, got %q", "reviewer", cmtHook.Sender.UserName)
	}
	if cmtHook.PullRequest.Number != 42 {
		t.Errorf("expected PR number 42, got %d", cmtHook.PullRequest.Number)
	}
	if cmtHook.PullRequest.Title != "Fix bug" {
		t.Errorf("expected PR title %q, got %q", "Fix bug", cmtHook.PullRequest.Title)
	}
}

// TestParseCommentHookServerDown verifies that a network failure when fetching
// the PR metadata surfaces an error rather than panicking.
func TestParseCommentHookServerDown(t *testing.T) {
	commentPayload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"pull_request": map[string]any{
				"url": "http://127.0.0.1:1/invalid",
			},
		},
		"comment": map[string]any{
			"body": "comment",
			"user": map[string]any{"login": "u"},
		},
		"repository": map[string]any{"id": 1, "name": "r", "full_name": "o/r"},
		"sender":     map[string]any{"login": "u"},
	}

	body, _ := json.Marshal(commentPayload)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventIssueComment, body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error when PR metadata server is unreachable")
	}
}

// TestParseCommentHookInvalidJSON exercises the JSON unmarshal failure path.
func TestParseCommentHookInvalidJSON(t *testing.T) {
	body := []byte(`{broken json`)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventIssueComment, body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment payload")
	}
}

// TestValidatePrefixSHA1 exercises the SHA1 signature verification branch.
func TestValidatePrefixSHA1(t *testing.T) {
	message := []byte("sha1 test body")
	key := []byte("mysecret")

	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	sig := "sha1=" + hex.EncodeToString(mac.Sum(nil))

	if !validatePrefix(message, key, sig) {
		t.Error("validatePrefix should accept valid sha1 signature")
	}
}

// TestValidateSHA1 tests the Validate function with SHA1.
func TestValidateSHA1(t *testing.T) {
	message := []byte("test sha1")
	key := []byte("secret")

	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	sig := hex.EncodeToString(mac.Sum(nil))

	if !Validate(sha1.New, message, key, sig) {
		t.Error("Validate should return true for valid HMAC-SHA1 signature")
	}
}

// TestParseSignatureValidationFailure exercises the path where signature
// validation fails and the hook is returned along with the sentinel error.
func TestParseSignatureValidationFailure(t *testing.T) {
	payload := map[string]any{
		"ref": "refs/heads/main",
		"commits": []map[string]any{
			{"message": "test", "url": "http://example.com"},
		},
		"repository": map[string]any{
			"id":         1,
			"name":       "r",
			"full_name":  "o/r",
			"created_at": 0,
			"owner":      map[string]any{"login": "o"},
		},
		"sender": map[string]any{"login": "o"},
	}
	body, _ := json.Marshal(payload)

	// Sign with one secret, parse with another
	req := newGithubRequest(hook.GithubEventPush, body, "rightsecret")

	_, err := Parse(req, "wrongsecret")
	if err == nil {
		t.Fatal("expected error for mismatched secret")
	}
	if !errors.Is(err, hook.ErrSecretValidationFailed) {
		t.Fatalf("expected ErrSecretValidationFailed, got: %v", err)
	}
}

// TestPRSynchronizeAction verifies that the "synchronize" action is mapped to "update".
func TestPRSynchronizeAction(t *testing.T) {
	payload := map[string]any{
		"action": "synchronize",
		"number": 7,
		"pull_request": map[string]any{
			"title": "Sync PR",
			"body":  "body",
			"head": map[string]any{
				"ref": "feature",
				"sha": "abc",
				"repo": map[string]any{
					"id": 1, "name": "r", "full_name": "o/r",
					"clone_url": "http://c", "html_url": "http://h",
					"git_url": "git://g", "ssh_url": "git@g", "svn_url": "svn://s",
					"owner": map[string]any{"login": "o"},
				},
			},
			"base": map[string]any{
				"ref": "main",
				"sha": "def",
				"repo": map[string]any{
					"id": 2, "name": "b", "full_name": "o/b",
					"clone_url": "http://c", "html_url": "http://h",
					"git_url": "git://g", "ssh_url": "git@g", "svn_url": "svn://s",
					"owner": map[string]any{"login": "o"},
				},
			},
			"user": map[string]any{"login": "u"},
		},
		"repository": map[string]any{"id": 1},
		"sender":     map[string]any{"login": "u"},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventPR, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse synchronize PR hook failed: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatalf("expected *hook.PullRequestHook, got %T", wh)
	}
	if prHook.Action != hook.ActionUpdate {
		t.Errorf("expected action %q (mapped from synchronize), got %q", hook.ActionUpdate, prHook.Action)
	}
}

// TestPREmptyAction verifies that a PR with an empty action field still parses.
func TestPREmptyAction(t *testing.T) {
	payload := map[string]any{
		"action": "",
		"number": 1,
		"pull_request": map[string]any{
			"title": "NoAction PR",
			"head": map[string]any{
				"ref":  "branch",
				"sha":  "sha",
				"repo": map[string]any{"id": 1, "name": "r", "owner": map[string]any{"login": "o"}},
			},
			"base": map[string]any{
				"ref":  "main",
				"sha":  "sha",
				"repo": map[string]any{"id": 2, "name": "b", "owner": map[string]any{"login": "o"}},
			},
			"user": map[string]any{"login": "u"},
		},
		"repository": map[string]any{"id": 1},
		"sender":     map[string]any{"login": "u"},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventPR, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse PR with empty action failed: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatalf("expected *hook.PullRequestHook, got %T", wh)
	}
	// Empty action should pass through without mapping
	if prHook.Action != "" {
		t.Errorf("expected empty action, got %q", prHook.Action)
	}
}

// TestValidatePrefixEdgeCases exercises various malformed signature formats.
func TestValidatePrefixEdgeCases(t *testing.T) {
	msg := []byte("data")
	key := []byte("key")

	tests := []struct {
		name string
		sig  string
	}{
		{"empty string", ""},
		{"no equals sign", "sha256"},
		{"empty value", "sha256="},
		{"invalid hex", "sha256=zzzz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if validatePrefix(msg, key, tt.sig) {
				t.Errorf("validatePrefix(%q) should return false", tt.sig)
			}
		})
	}
}

// TestConvertPullRequestURLInvalidJSON exercises convertPullRequestURL with
// a URL that returns non-JSON content.
func TestConvertPullRequestURLInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := convertPullRequestURL(srv.URL)
	if err == nil {
		t.Fatal("expected error for non-JSON response")
	}
}
