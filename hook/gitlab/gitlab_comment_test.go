package gitlab

import (
	"bytes"
	"context"
	"crypto/hmac"
	cryptoSha256 "crypto/sha256"
	"encoding/hex"
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
	req.Header.Set(hook.GitlabEvent, hook.GitlabEventNote)
	if secret != "" {
		req.Header.Set("X-Gitlab-Token", secret)
	}
	return req
}

// TestParseNote verifies that a GitLab note event on a merge request
// is parsed into a PullRequestCommentHook with the expected fields.
func TestParseNote(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"event_type":  "note",
		"user": map[string]any{
			"id":       1,
			"name":     "Reviewer",
			"username": testUser,
		},
		"project_id": 1,
		"project": map[string]any{
			"id":                  1,
			"name":                "repo",
			"web_url":             "https://gitlab.com/group/repo",
			"git_http_url":        "https://gitlab.com/group/repo.git",
			"git_ssh_url":         "git@gitlab.com:group/repo.git",
			"ssh_url":             "git@gitlab.com:group/repo.git",
			"http_url":            "https://gitlab.com/group/repo.git",
			"url":                 "git@gitlab.com:group/repo.git",
			"path_with_namespace": "group/repo",
		},
		"object_attributes": map[string]any{
			"id":            50,
			"note":          "looks good, please merge",
			"noteable_type": "MergeRequest", //nolint:misspell // GitLab API uses "noteable"
			"noteable_id":   99,             //nolint:misspell // GitLab API uses "noteable"
		},
		"merge_request": map[string]any{
			"id":            99,
			"iid":           11,
			"title":         "Add widgets",
			"source_branch": "feat/widgets",
			"target_branch": "main",
			"last_commit": map[string]any{
				"id":      "commitabc",
				"message": "final commit",
			},
			"source": map[string]any{
				"id":                  2,
				"name":                "fork-repo",
				"description":         "forked",
				"web_url":             "https://gitlab.com/forker/repo",
				"git_http_url":        "https://gitlab.com/forker/repo.git",
				"git_ssh_url":         "git@gitlab.com:forker/repo.git",
				"ssh_url":             "git@gitlab.com:forker/repo.git",
				"http_url":            "https://gitlab.com/forker/repo.git",
				"url":                 "git@gitlab.com:forker/repo.git",
				"path_with_namespace": "forker/repo",
			},
			"target": map[string]any{
				"id":                  1,
				"name":                "repo",
				"description":         "main repo",
				"web_url":             "https://gitlab.com/group/repo",
				"git_http_url":        "https://gitlab.com/group/repo.git",
				"git_ssh_url":         "git@gitlab.com:group/repo.git",
				"ssh_url":             "git@gitlab.com:group/repo.git",
				"http_url":            "https://gitlab.com/group/repo.git",
				"url":                 "git@gitlab.com:group/repo.git",
				"path_with_namespace": "group/repo",
			},
		},
		"repository": map[string]any{
			"name": "repo",
			"url":  "git@gitlab.com:group/repo.git",
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
	if cmt.Repo.RepoType != "gitlab" {
		t.Errorf("expected RepoType 'gitlab', got %q", cmt.Repo.RepoType)
	}
	if cmt.Repo.Branch != "feat/widgets" {
		t.Errorf("expected source branch 'feat/widgets', got %q", cmt.Repo.Branch)
	}
	if cmt.TargetRepo.Branch != "main" {
		t.Errorf("expected target branch 'main', got %q", cmt.TargetRepo.Branch)
	}
	if cmt.PullRequest.Number != 11 {
		t.Errorf("expected MR iid 11, got %d", cmt.PullRequest.Number)
	}
	if cmt.PullRequest.Title != "Add widgets" {
		t.Errorf("expected title 'Add widgets', got %q", cmt.PullRequest.Title)
	}
	if cmt.Comment.Body != "looks good, please merge" {
		t.Errorf("expected comment body, got %q", cmt.Comment.Body)
	}
	if cmt.Comment.Author.UserName != testUser {
		t.Errorf("expected comment author %q, got %q", testUser, cmt.Comment.Author.UserName)
	}
	if cmt.Sender.UserName != testUser {
		t.Errorf("expected sender %q, got %q", testUser, cmt.Sender.UserName)
	}
}

// TestParseNoteSecretMismatch ensures the GitLab comment hook enforces
// the X-Gitlab-Token secret.
func TestParseNoteSecretMismatch(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"merge_request": map[string]any{
			"source": map[string]any{},
			"target": map[string]any{},
		},
	}
	body, _ := json.Marshal(payload)
	req := newNoteRequest("wrongsecret", body)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for secret mismatch")
	}
}

// TestParseNoteInvalidJSON verifies that malformed JSON in a note
// event produces a parse error rather than a panic.
func TestParseNoteInvalidJSON(t *testing.T) {
	body := []byte(`{"object_kind": "note", "invalid`)
	req := newNoteRequest(testSecret, body)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestParseNoteErrorWrapping verifies that errors from Parse use %w for
// proper error chain propagation on the note event path.
func TestParseNoteErrorWrapping(t *testing.T) {
	body := []byte(`{invalid json`)
	req := newNoteRequest(testSecret, body)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "gitlab:") {
		t.Errorf("expected 'gitlab:' prefix, got: %v", err)
	}
	if errors.Unwrap(err) == nil {
		t.Error("expected %w wrapping via errors.Unwrap")
	}
}

// TestValidateHMAC verifies the HMAC-SHA256 signature validator.
func TestValidateHMAC(t *testing.T) {
	key := []byte("secret")
	msg := []byte("body")

	mac := hmac.New(cryptoSha256.New, key)
	mac.Write(msg)
	sig := hex.EncodeToString(mac.Sum(nil))

	if !Validate(cryptoSha256.New, msg, key, sig) {
		t.Error("Validate should accept correct signature")
	}
	if Validate(cryptoSha256.New, msg, key, "deadbeef") {
		t.Error("Validate should reject malformed hex")
	}
	if Validate(cryptoSha256.New, msg, []byte("different-key"), sig) {
		t.Error("Validate should reject wrong key")
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
		gp := &gitlabPushHook{Ref: tc.ref}
		gp.Project.Id = 1
		h := convertPushHook(gp)
		if h.Repo.Branch != tc.expected {
			t.Errorf("ref=%q: expected branch %q, got %q", tc.ref, tc.expected, h.Repo.Branch)
		}
	}
}
