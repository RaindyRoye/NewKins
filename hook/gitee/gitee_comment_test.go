package gitee

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gokins/gokins/hook"
)

func newNoteRequest(secret string, body []byte) *http.Request {
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/webhook",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GiteeEvent, hook.GiteeEventNote)
	if secret != "" {
		req.Header.Set("X-Gitee-Token", secret)
	}
	return req
}

// TestParseCommentHook verifies that a note event on a pull request is
// parsed into a PullRequestCommentHook with the expected fields.
func TestParseCommentHook(t *testing.T) {
	payload := map[string]any{
		"action": hook.EventsTypeComment,
		"note":   "please squash commits",
		"comment": map[string]any{
			"id":   10,
			"body": "inline feedback",
		},
		"author": map[string]any{
			"user_name": "commenter",
		},
		"pull_request": map[string]any{
			"id":     100,
			"number": 7,
			"title":  "Add feature",
			"body":   "PR body",
			"user": map[string]any{
				"user_name": "prauthor",
			},
			"head": map[string]any{
				"ref": "feature",
				"sha": "abc123",
				"repo": map[string]any{
					"id":           2,
					"name":         "fork",
					"full_name":    "forker/repo",
					"html_url":     "https://gitee.com/forker/repo",
					"clone_url":    "https://gitee.com/forker/repo.git",
					"git_http_url": "https://gitee.com/forker/repo.git",
					"git_ssh_url":  "git@gitee.com:forker/repo.git",
					"git_svn_url":  "svn://gitee.com/forker/repo",
					"git_url":      "git://gitee.com/forker/repo.git",
					"ssh_url":      "git@gitee.com:forker/repo.git",
					"svn_url":      "svn://gitee.com/forker/repo",
					"url":          "https://gitee.com/forker/repo",
					"description":  "forked repo",
					"private":      false,
					"created_at":   "2024-01-01T00:00:00Z",
					"owner": map[string]any{
						"user_name": "forker",
					},
				},
			},
			"base": map[string]any{
				"ref": "main",
				"sha": "def456",
				"repo": map[string]any{
					"id":           1,
					"name":         "repo",
					"full_name":    "owner/repo",
					"html_url":     "https://gitee.com/owner/repo",
					"clone_url":    "https://gitee.com/owner/repo.git",
					"git_http_url": "https://gitee.com/owner/repo.git",
					"git_ssh_url":  "git@gitee.com:owner/repo.git",
					"git_svn_url":  "svn://gitee.com/owner/repo",
					"git_url":      "git://gitee.com/owner/repo.git",
					"ssh_url":      "git@gitee.com:owner/repo.git",
					"svn_url":      "svn://gitee.com/owner/repo",
					"url":          "https://gitee.com/owner/repo",
					"description":  "base repo",
					"private":      false,
					"created_at":   "2023-01-01T00:00:00Z",
					"owner": map[string]any{
						"user_name": "owner",
					},
				},
			},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "repo",
			"full_name": "owner/repo",
		},
		"sender": map[string]any{
			"login": "prauthor",
		},
	}
	body, _ := json.Marshal(payload)
	req := newNoteRequest(testSecret, body)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse comment hook failed: %v", err)
	}
	cmt, ok := wh.(*hook.PullRequestCommentHook)
	if !ok {
		t.Fatalf("expected *hook.PullRequestCommentHook, got %T", wh)
	}
	if cmt.Action != hook.EventsTypeComment {
		t.Errorf("expected action %q, got %q", hook.EventsTypeComment, cmt.Action)
	}
	if cmt.Repo.RepoType != "gitee" {
		t.Errorf("expected RepoType 'gitee', got %q", cmt.Repo.RepoType)
	}
	if cmt.Repo.Branch != "feature" {
		t.Errorf("expected branch 'feature', got %q", cmt.Repo.Branch)
	}
	if cmt.TargetRepo.Branch != "main" {
		t.Errorf("expected target branch 'main', got %q", cmt.TargetRepo.Branch)
	}
	if cmt.PullRequest.Number != 7 {
		t.Errorf("expected PR number 7, got %d", cmt.PullRequest.Number)
	}
	// Gitee uses the "note" field for the comment body, not "comment.body".
	if cmt.Comment.Body != "please squash commits" {
		t.Errorf("expected comment body from 'note' field, got %q", cmt.Comment.Body)
	}
	if cmt.Comment.Author.UserName != "commenter" {
		t.Errorf("expected comment author 'commenter', got %q", cmt.Comment.Author.UserName)
	}
}

// TestParseCommentHookSecretMismatch ensures the Gitee comment hook enforces
// secret validation like the other event types.
func TestParseCommentHookSecretMismatch(t *testing.T) {
	payload := map[string]any{
		"action": hook.EventsTypeComment,
		"note":   "x",
	}
	body, _ := json.Marshal(payload)
	req := newNoteRequest("wrongsecret", body)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for secret mismatch")
	}
}

// TestParseCommentHookInvalidJSON verifies that malformed JSON in a note
// event produces a parse error rather than a panic.
func TestParseCommentHookInvalidJSON(t *testing.T) {
	body := []byte(`{"action": "comment", "note": "x", "invalid`)
	req := newNoteRequest(testSecret, body)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestParseErrorWrapping verifies that errors from Parse use %w for proper
// error chain propagation on the note event path as well.
func TestParseNoteErrorWrapping(t *testing.T) {
	body := []byte(`{invalid json`)
	req := newNoteRequest(testSecret, body)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "gitee:") {
		t.Errorf("expected 'gitee:' prefix, got: %v", err)
	}
	if errors.Unwrap(err) == nil {
		t.Error("expected %w wrapping via errors.Unwrap")
	}
}

// TestConvertPushHookBranchExtraction verifies branch parsing from ref.
func TestConvertPushHookBranchExtraction(t *testing.T) {
	cases := []struct {
		ref      string
		expected string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/feature/x", "feature/x"},
		{"refs/tags/v1", ""},
		{"", ""},
	}
	for _, tc := range cases {
		gp := &giteePushHook{Ref: tc.ref}
		gp.Repository.Name = "r"
		h := convertPushHook(gp)
		if h.Repo.Branch != tc.expected {
			t.Errorf("ref=%q: expected branch %q, got %q", tc.ref, tc.expected, h.Repo.Branch)
		}
	}
}
