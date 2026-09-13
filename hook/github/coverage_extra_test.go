package github

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gokins/gokins/hook"
)

// mockPRServer creates a test server that returns mock PR data
func mockPRServer(prData map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(prData)
	}))
}

// createMockPRData generates minimal valid PR response data
func createMockPRData(number int, title, headRef, baseRef string) map[string]any {
	return map[string]any{
		"id":         123,
		"number":     number,
		"title":      title,
		"body":       "PR body",
		"state":      "open",
		"created_at": time.Now().Format(time.RFC3339),
		"user": map[string]any{
			"login": "prauthor",
		},
		"head": map[string]any{
			"ref": headRef,
			"sha": "headsha123",
			"repo": map[string]any{
				"id":         10,
				"name":       "fork-repo",
				"full_name":  "forker/fork-repo",
				"clone_url":  "https://github.com/forker/fork-repo.git",
				"html_url":   "https://github.com/forker/fork-repo",
				"git_url":    "git://github.com/forker/fork-repo.git",
				"ssh_url":    "git@github.com:forker/fork-repo.git",
				"svn_url":    "https://github.com/forker/fork-repo",
				"created_at": time.Now().Format(time.RFC3339),
				"owner":      map[string]any{"login": "forker"},
			},
		},
		"base": map[string]any{
			"ref": baseRef,
			"sha": "basesha456",
			"repo": map[string]any{
				"id":         1,
				"name":       "base-repo",
				"full_name":  "owner/base-repo",
				"clone_url":  "https://github.com/owner/base-repo.git",
				"html_url":   "https://github.com/owner/base-repo",
				"git_url":    "git://github.com/owner/base-repo.git",
				"ssh_url":    "git@github.com:owner/base-repo.git",
				"svn_url":    "https://github.com/owner/base-repo",
				"created_at": time.Now().Format(time.RFC3339),
				"owner":      map[string]any{"login": "owner"},
			},
		},
	}
}

func TestConvertPullRequestURLCoverage(t *testing.T) {
	prData := createMockPRData(42, "Test PR", "feature", "main")
	ts := mockPRServer(prData)
	defer ts.Close()

	result, err := convertPullRequestURL(ts.URL)
	if err != nil {
		t.Fatalf("convertPullRequestURL failed: %v", err)
	}

	if result.Number != 42 {
		t.Errorf("expected PR number 42, got %d", result.Number)
	}
	if result.Title != "Test PR" {
		t.Errorf("expected title 'Test PR', got '%s'", result.Title)
	}
	if result.Head.Ref != "feature" {
		t.Errorf("expected head ref 'feature', got '%s'", result.Head.Ref)
	}
	if result.Base.Ref != "main" {
		t.Errorf("expected base ref 'main', got '%s'", result.Base.Ref)
	}
}

func TestConvertPullRequestURLInvalidURLCoverage(t *testing.T) {
	_, err := convertPullRequestURL("://invalid")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestConvertCommentHook(t *testing.T) {
	prData := createMockPRData(7, "Commented PR", "feat", "main")
	ts := mockPRServer(prData)
	defer ts.Close()

	// Create comment hook with minimal data using JSON
	commentJSON := `{
		"action": "created",
		"issue": {
			"number": 7,
			"pull_request": {
				"url": "` + ts.URL + `"
			}
		},
		"comment": {
			"body": "looks good!",
			"user": {"login": "reviewer"}
		},
		"sender": {"login": "reviewer"},
		"repository": {"id": 5}
	}`

	var commentHook githubCommentHook
	if err := json.Unmarshal([]byte(commentJSON), &commentHook); err != nil {
		t.Fatalf("failed to unmarshal comment hook: %v", err)
	}

	result, err := convertCommentHook(&commentHook)
	if err != nil {
		t.Fatalf("convertCommentHook failed: %v", err)
	}

	if result.Comment.Body != "looks good!" {
		t.Errorf("expected comment body 'looks good!', got '%s'", result.Comment.Body)
	}
	if result.PullRequest.Number != 7 {
		t.Errorf("expected PR number 7, got %d", result.PullRequest.Number)
	}
	if result.Repo.RepoType != "github" {
		t.Errorf("expected repoType 'github', got '%s'", result.Repo.RepoType)
	}
	if result.Sender.UserName != "reviewer" {
		t.Errorf("expected sender 'reviewer', got '%s'", result.Sender.UserName)
	}
}

func TestValidateInternal(t *testing.T) {
	message := []byte("test data")
	key := []byte("secret")

	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	sum := mac.Sum(nil)

	if !validate(sha256.New, message, key, sum) {
		t.Error("validate should return true for valid HMAC")
	}

	if validate(sha256.New, message, []byte("wrong"), sum) {
		t.Error("validate should return false for wrong key")
	}
}

func TestParseIssueCommentHook(t *testing.T) {
	prData := createMockPRData(10, "Mock PR", "dev", "main")
	ts := mockPRServer(prData)
	defer ts.Close()

	payload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number": 10,
			"pull_request": map[string]any{
				"url": ts.URL,
			},
		},
		"comment": map[string]any{
			"body": "nice work",
			"user": map[string]any{"login": "reviewer1"},
		},
		"sender":     map[string]any{"login": "reviewer1"},
		"repository": map[string]any{"id": 5},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GithubEvent, hook.GithubEventIssueComment)
	sig := computeSignature([]byte(secret), body)
	req.Header.Set("X-Hub-Signature", sig)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse issue_comment hook failed: %v", err)
	}

	commentHook, ok := wh.(*hook.PullRequestCommentHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestCommentHook type")
	}

	if commentHook.Action != hook.EventsTypeComment {
		t.Errorf("expected action '%s', got '%s'", hook.EventsTypeComment, commentHook.Action)
	}
	if commentHook.Comment.Body != "nice work" {
		t.Errorf("expected comment body 'nice work', got '%s'", commentHook.Comment.Body)
	}
	if commentHook.PullRequest.Number != 10 {
		t.Errorf("expected PR number 10, got %d", commentHook.PullRequest.Number)
	}
}

func TestParseCommentHookInvalidJSON(t *testing.T) {
	_, err := parseCommentHook([]byte(`{bad`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestConvertPushHookBranchExtraction(t *testing.T) {
	tests := []struct {
		ref      string
		expected string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/feature/x", "feature"}, // Only takes 3rd segment
		{"refs/tags/v1.0", "v1.0"},
		{"", ""},
	}

	for _, tt := range tests {
		// Create minimal push hook via JSON
		pushJSON := `{"ref": "` + tt.ref + `", "commits": [{"message": "m", "url": "u"}]}`
		var gp githubPushHook
		if err := json.Unmarshal([]byte(pushJSON), &gp); err != nil {
			t.Fatalf("failed to unmarshal push hook: %v", err)
		}

		result := convertPushHook(&gp)
		if result.Repo.Branch != tt.expected {
			t.Errorf("ref=%q: expected branch=%q, got %q", tt.ref, tt.expected, result.Repo.Branch)
		}
	}
}
