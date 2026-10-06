package github

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // G505: SHA1 is required for GitHub webhook signature verification (X-Hub-Signature)
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

// mockPRServer creates an httptest server that serves a fake PR JSON response.
// It returns the server and the URL that mimics a GitHub PR API endpoint.
func mockPRServer() *httptest.Server {
	prResp := map[string]any{
		"url":        "https://api.github.com/repos/user/test-repo/pulls/42",
		"id":         12345,
		"number":     42,
		"title":      "Test PR for comment",
		"body":       "PR body text",
		"html_url":   "https://github.com/user/test-repo/pull/42",
		"diff_url":   "https://github.com/user/test-repo/pull/42.diff",
		"patch_url":  "https://github.com/user/test-repo/pull/42.patch",
		"issue_url":  "https://api.github.com/repos/user/test-repo/issues/42",
		"state":      "open",
		"locked":     false,
		"created_at": "2021-06-01T00:00:00Z",
		"updated_at": "2021-06-02T00:00:00Z",
		"user": map[string]any{
			"login": "prauthor",
			"id":    100,
		},
		"head": map[string]any{
			"ref": "feature-branch",
			"sha": "abc123head",
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
			"sha": "def456base",
			"repo": map[string]any{
				"id":          1,
				"name":        "test-repo",
				"full_name":   "user/test-repo",
				"clone_url":   "https://github.com/user/test-repo.git",
				"html_url":    "https://github.com/user/test-repo",
				"git_url":     "git://github.com/user/test-repo.git",
				"ssh_url":     "git@github.com:user/test-repo.git",
				"svn_url":     "https://github.com/user/test-repo",
				"created_at":  "2020-01-01T00:00:00Z",
				"description": "base repo",
				"private":     false,
				"owner": map[string]any{
					"login": "user",
				},
			},
		},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(prResp)
	}))
}

func buildCommentPayload(prURL string) map[string]any {
	return map[string]any{
		"action": "created",
		"issue": map[string]any{
			"url":    "https://api.github.com/repos/user/test-repo/issues/42",
			"id":     1,
			"number": 42,
			"title":  "Test Issue",
			"user": map[string]any{
				"login": "issueauthor",
			},
			"pull_request": map[string]any{
				"url":       prURL,
				"html_url":  "https://github.com/user/test-repo/pull/42",
				"diff_url":  "https://github.com/user/test-repo/pull/42.diff",
				"patch_url": "https://github.com/user/test-repo/pull/42.patch",
			},
		},
		"comment": map[string]any{
			"id":   999,
			"body": "This is a test comment on the PR",
			"user": map[string]any{
				"login": "commenter",
			},
		},
		"repository": map[string]any{
			"id":          1,
			"name":        "test-repo",
			"full_name":   "user/test-repo",
			"clone_url":   "https://github.com/user/test-repo.git",
			"html_url":    "https://github.com/user/test-repo",
			"git_url":     "git://github.com/user/test-repo.git",
			"ssh_url":     "git@github.com:user/test-repo.git",
			"svn_url":     "https://github.com/user/test-repo",
			"description": "test repo",
			"private":     false,
			"owner": map[string]any{
				"login": "user",
			},
		},
		"sender": map[string]any{
			"login": "commenter",
		},
	}
}

func TestParseCommentHook_Success(t *testing.T) {
	srv := mockPRServer()
	defer srv.Close()

	payload := buildCommentPayload(srv.URL)
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

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

	if commentHook.Action != hook.EventsTypeComment {
		t.Errorf("expected action %q, got %q", hook.EventsTypeComment, commentHook.Action)
	}
	if commentHook.Comment.Body != "This is a test comment on the PR" {
		t.Errorf("expected comment body %q, got %q", "This is a test comment on the PR", commentHook.Comment.Body)
	}
	if commentHook.Comment.Author.UserName != "commenter" {
		t.Errorf("expected comment author %q, got %q", "commenter", commentHook.Comment.Author.UserName)
	}
	if commentHook.Repo.RepoType != "github" {
		t.Errorf("expected repo type %q, got %q", "github", commentHook.Repo.RepoType)
	}
	if commentHook.Repo.Name != "test-repo-fork" {
		t.Errorf("expected repo name %q, got %q", "test-repo-fork", commentHook.Repo.Name)
	}
	if commentHook.TargetRepo.Name != "test-repo" {
		t.Errorf("expected target repo name %q, got %q", "test-repo", commentHook.TargetRepo.Name)
	}
	if commentHook.PullRequest.Number != 42 {
		t.Errorf("expected PR number 42, got %d", commentHook.PullRequest.Number)
	}
	if commentHook.PullRequest.Title != "Test PR for comment" {
		t.Errorf("expected PR title %q, got %q", "Test PR for comment", commentHook.PullRequest.Title)
	}
	if commentHook.Sender.UserName != "commenter" {
		t.Errorf("expected sender %q, got %q", "commenter", commentHook.Sender.UserName)
	}
}

func TestParseCommentHook_InvalidJSON(t *testing.T) {
	body := []byte(`{not valid json}`)
	req := newGithubRequest(hook.GithubEventIssueComment, body, "")

	_, err := Parse(req, "")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseCommentHook_PRFetchFails(t *testing.T) {
	// Use a URL that will fail (no server running)
	badURL := "http://127.0.0.1:1/nonexistent"
	payload := buildCommentPayload(badURL)
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	secret := testSecret
	req := newGithubRequest(hook.GithubEventIssueComment, body, secret)

	_, err = Parse(req, secret)
	if err == nil {
		t.Fatal("expected error when PR URL fetch fails, got nil")
	}
}

func TestParseCommentHook_SecretValidationFails(t *testing.T) {
	srv := mockPRServer()
	defer srv.Close()

	payload := buildCommentPayload(srv.URL)
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := newGithubRequest(hook.GithubEventIssueComment, body, "correctsecret")

	_, err = Parse(req, "wrongsecret")
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestConvertPullRequestURL_InvalidURL(t *testing.T) {
	_, err := convertPullRequestURL("http://127.0.0.1:1/no-such-server")
	if err == nil {
		t.Fatal("expected error for unreachable URL, got nil")
	}
}

func TestConvertPullRequestURL_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := convertPullRequestURL(srv.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON response, got nil")
	}
}

func TestConvertPullRequestURL_Success(t *testing.T) {
	srv := mockPRServer()
	defer srv.Close()

	pr, err := convertPullRequestURL(srv.URL)
	if err != nil {
		t.Fatalf("convertPullRequestURL failed: %v", err)
	}
	if pr.Number != 42 {
		t.Errorf("expected PR number 42, got %d", pr.Number)
	}
	if pr.Head.Ref != "feature-branch" {
		t.Errorf("expected head ref %q, got %q", "feature-branch", pr.Head.Ref)
	}
}

func TestValidatePrefix_SHA1(t *testing.T) {
	message := []byte("test body for sha1")
	key := []byte("mysecret")

	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	sig := "sha1=" + hex.EncodeToString(mac.Sum(nil))

	if !validatePrefix(message, key, sig) {
		t.Error("validatePrefix should accept valid sha1 signature")
	}
}

func TestValidatePrefixMalformedSignature(t *testing.T) {
	message := []byte("body")
	key := []byte("key")

	// No = separator
	if validatePrefix(message, key, "sha256noequalssign") {
		t.Error("should reject malformed signature without = separator")
	}

	// Empty string
	if validatePrefix(message, key, "") {
		t.Error("should reject empty signature")
	}
}

func TestValidate_InvalidHex(t *testing.T) {
	// Non-hex signature should return false
	if Validate(sha1.New, []byte("msg"), []byte("key"), "nothex!!") {
		t.Error("Validate should return false for non-hex signature")
	}
}

func TestParsePushHook_BranchExtraction(t *testing.T) {
	// Test that branch extraction from refs/heads/xxx works
	payload := map[string]any{
		"ref":    "refs/heads/my-branch",
		"before": "aaa",
		"after":  "bbb",
		"commits": []map[string]any{
			{
				"id":      "bbb",
				"message": "nested branch commit",
				"url":     "https://github.com/user/repo/commit/bbb",
			},
		},
		"repository": map[string]any{
			"id":          5,
			"name":        "repo",
			"full_name":   "user/repo",
			"clone_url":   "https://github.com/user/repo.git",
			"html_url":    "https://github.com/user/repo",
			"ssh_url":     "git@github.com:user/repo.git",
			"svn_url":     "https://github.com/user/repo",
			"git_url":     "git://github.com/user/repo.git",
			"description": "desc",
			"private":     true,
			"created_at":  1609459200,
			"owner": map[string]any{
				"login": "user",
			},
		},
		"sender": map[string]any{
			"login": "pusher",
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventPush, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse push hook failed: %v", err)
	}

	pushHook, ok := wh.(*hook.PushHook)
	if !ok {
		t.Fatalf("expected *hook.PushHook, got %T", wh)
	}

	// Branch should be extracted as "my-branch" (third part of refs/heads/my-branch)
	if pushHook.Repo.Branch != "my-branch" {
		t.Errorf("expected branch %q, got %q", "my-branch", pushHook.Repo.Branch)
	}
	if !pushHook.Repo.Private {
		t.Error("expected repo to be private")
	}
}

func TestParsePullRequestHook_Synchronize(t *testing.T) {
	payload := map[string]any{
		"action": "synchronize",
		"number": 7,
		"pull_request": map[string]any{
			"title": "Sync PR",
			"body":  "synced",
			"head": map[string]any{
				"ref": "sync-branch",
				"sha": "syncsha",
				"repo": map[string]any{
					"id":          10,
					"name":        "sync-repo",
					"full_name":   "syncer/sync-repo",
					"clone_url":   "https://github.com/syncer/sync-repo.git",
					"html_url":    "https://github.com/syncer/sync-repo",
					"git_url":     "git://github.com/syncer/sync-repo.git",
					"ssh_url":     "git@github.com:syncer/sync-repo.git",
					"svn_url":     "https://github.com/syncer/sync-repo",
					"created_at":  "2022-01-01T00:00:00Z",
					"description": "sync desc",
					"private":     false,
					"owner":       map[string]any{"login": "syncer"},
				},
			},
			"base": map[string]any{
				"ref": "main",
				"sha": "basesha",
				"repo": map[string]any{
					"id":          11,
					"name":        "base-repo",
					"full_name":   "org/base-repo",
					"clone_url":   "https://github.com/org/base-repo.git",
					"html_url":    "https://github.com/org/base-repo",
					"git_url":     "git://github.com/org/base-repo.git",
					"ssh_url":     "git@github.com:org/base-repo.git",
					"svn_url":     "https://github.com/org/base-repo",
					"created_at":  "2022-01-01T00:00:00Z",
					"description": "base desc",
					"private":     false,
					"owner":       map[string]any{"login": "org"},
				},
			},
			"user": map[string]any{
				"login": "syncauthor",
			},
		},
		"repository": map[string]any{
			"id":        11,
			"name":      "base-repo",
			"full_name": "org/base-repo",
		},
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

	// "synchronize" should be mapped to "update"
	if prHook.Action != hook.ActionUpdate {
		t.Errorf("expected action %q, got %q", hook.ActionUpdate, prHook.Action)
	}
}
