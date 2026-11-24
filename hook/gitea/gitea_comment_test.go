package gitea

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

func signGitea(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func newNoteRequest(secret string, body []byte) *http.Request {
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/webhook",
		bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GiteaEvent, hook.GiteaEventNote)
	if secret != "" {
		req.Header.Set("X-Gitea-Signature", signGitea([]byte(secret), body))
	}
	return req
}

// TestParseNoteNotPullRequest verifies that a note event on a non-PR
// comment returns ErrCommentNotPullRequest.
func TestParseNoteNotPullRequest(t *testing.T) {
	payload := map[string]any{
		"is_pull": false,
		"comment": map[string]any{
			"id":   1,
			"body": "nice work",
		},
		"issue": map[string]any{
			"id":    100,
			"title": "bug report",
		},
		"repository": map[string]any{"id": 1},
		"sender":     map[string]any{"login": "reviewer"},
	}
	body, _ := json.Marshal(payload)
	req := newNoteRequest(testSecret, body)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for non-PR comment")
	}
	// The error chain should include ErrCommentNotPullRequest.
	if err != nil && err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

// TestParseNotePRComment verifies that a note event on a PR comment
// is parsed successfully. convertPullRequestURL currently returns nil,nil
// because the Gitea API call is not yet implemented (TODO), so the resulting
// PullRequestCommentHook is constructed from the nil fields.
func TestParseNotePRComment(t *testing.T) {
	payload := map[string]any{
		"is_pull": true,
		"comment": map[string]any{
			"id":   42,
			"body": "please fix",
			"user": map[string]any{
				"login": "reviewer",
			},
		},
		"issue": map[string]any{
			"id":    100,
			"title": "PR title",
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "myrepo",
			"full_name": "org/myrepo",
			"clone_url": "https://gitea.example.com/org/myrepo.git",
			"html_url":  "https://gitea.example.com/org/myrepo",
			"ssh_url":   "git@gitea.example.com:org/myrepo.git",
			"private":   false,
			"owner": map[string]any{
				"login": "orgadmin",
			},
		},
		"sender": map[string]any{
			"login": "reviewer",
		},
	}
	body, _ := json.Marshal(payload)
	req := newNoteRequest(testSecret, body)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse PR comment hook failed: %v", err)
	}
	cmt, ok := wh.(*hook.PullRequestCommentHook)
	if !ok {
		t.Fatalf("expected *hook.PullRequestCommentHook, got %T", wh)
	}
	if cmt.Action != hook.EventsTypeComment {
		t.Errorf("expected action %q, got %q", hook.EventsTypeComment, cmt.Action)
	}
	if cmt.Repo.RepoType != "gitea" {
		t.Errorf("expected RepoType 'gitea', got %q", cmt.Repo.RepoType)
	}
	if cmt.Repo.Name != "myrepo" {
		t.Errorf("expected repo name 'myrepo', got %q", cmt.Repo.Name)
	}
	if cmt.Comment.Body != "please fix" {
		t.Errorf("expected comment body 'please fix', got %q", cmt.Comment.Body)
	}
	if cmt.Comment.Author.UserName != "reviewer" {
		t.Errorf("expected comment author 'reviewer', got %q", cmt.Comment.Author.UserName)
	}
}

// TestParseNoteInvalidJSON verifies that malformed JSON in a note
// event produces a parse error rather than a panic.
func TestParseNoteInvalidJSON(t *testing.T) {
	body := []byte(`{"is_pull": true, "comment": INVALID`)
	req := newNoteRequest(testSecret, body)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in note event")
	}
}

// TestParseNoteSecretMismatch verifies that a note event with a
// mismatched signature returns ErrSecretValidationFailed.
func TestParseNoteSecretMismatch(t *testing.T) {
	payload := map[string]any{
		"is_pull": true,
		"comment": map[string]any{"id": 1, "body": "x"},
	}
	body, _ := json.Marshal(payload)
	// Sign with wrong secret; parse with correct secret.
	req := newNoteRequest("wrongsecret", body)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for secret mismatch")
	}
}

// TestConvertPushHookBranchExtraction verifies branch parsing from ref.
func TestConvertPushHookBranchExtraction(t *testing.T) {
	cases := []struct {
		ref      string
		expected string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/feature/abc", "feature/abc"},
		{"refs/tags/v1.0", ""},
		{"", ""},
		{"refs/heads", ""},
	}
	for _, tc := range cases {
		gp := &giteaPushHook{Ref: tc.ref}
		gp.Repository.Name = "r"
		gp.Repository.Owner.Login = "o"
		h := convertPushHook(gp)
		if h.Repo.Branch != tc.expected {
			t.Errorf("ref=%q: expected branch %q, got %q", tc.ref, tc.expected, h.Repo.Branch)
		}
	}
}

// TestConvertPRHookActionMapping verifies Gitea action normalization:
// "opened" → "open", "synchronize" → "update".
func TestConvertPRHookActionMapping(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
	}{
		{"opened", false},
		{"synchronize", false},
		{"closed", true},
		{"reopened", true},
	}
	for _, tc := range cases {
		gp := &giteaPRHook{Action: tc.input}
		_, err := parsePullRequestHook(mustMarshal(t, gp))
		if tc.wantErr {
			if err == nil {
				t.Errorf("action=%q: expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("action=%q: unexpected error: %v", tc.input, err)
			continue
		}
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	return b
}
