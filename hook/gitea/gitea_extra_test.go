package gitea

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // G505: SHA1 required for Gitea webhook signature verification
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gokins/gokins/hook"
)

// --- parseCommentHook ---

func TestParseCommentHook_NotPullRequest(t *testing.T) {
	payload := map[string]any{
		"is_pull": false,
		"comment": map[string]any{
			"id":   1,
			"body": "nice work",
			"user": map[string]any{"login": "commenter"},
		},
		"issue": map[string]any{
			"id":     10,
			"number": 5,
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "repo",
			"full_name": "owner/repo",
			"clone_url": "https://gitea.example.com/owner/repo.git",
			"html_url":  "https://gitea.example.com/owner/repo",
			"ssh_url":   "git@gitea.example.com:owner/repo.git",
		},
		"sender": map[string]any{"login": "commenter"},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventNote, body, secret)

	// The comment hook parser rejects non-PR comments before signature validation.
	// Since Parse validates signature after parsing, we expect the comment error.
	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for non-PR comment")
	}
	if !errors.Is(err, hook.ErrCommentNotPullRequest) {
		t.Errorf("expected ErrCommentNotPullRequest, got: %v", err)
	}
}

func TestParseCommentHook_InvalidJSON(t *testing.T) {
	body := []byte(`{broken json`)
	req := newGiteaRequest(hook.GiteaEventNote, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment hook")
	}
}

// TestParseCommentHook_ValidPRComment verifies that a valid PR comment is parsed
// but returns an error because convertPullRequestURL is not yet implemented.
func TestParseCommentHook_ValidPRComment(t *testing.T) {
	// A valid PR comment payload: is_pull=true, so parseCommentHook succeeds.
	// convertPullRequestURL returns nil, so convertCommentHook should return
	// an error instead of panicking.
	payload := map[string]any{
		"is_pull": true,
		"comment": map[string]any{
			"id":   1,
			"body": "lgtm",
			"user": map[string]any{"login": "reviewer"},
		},
		"issue": map[string]any{
			"id":     10,
			"number": 5,
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "repo",
			"full_name": "owner/repo",
			"clone_url": "https://gitea.example.com/owner/repo.git",
			"html_url":  "https://gitea.example.com/owner/repo",
			"ssh_url":   "git@gitea.example.com:owner/repo.git",
		},
		"sender": map[string]any{"login": "reviewer"},
	}

	body, _ := json.Marshal(payload)
	req := newGiteaRequest(hook.GiteaEventNote, body, testSecret)

	// Parse should return an error about unimplemented PR URL conversion
	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for unimplemented PR URL conversion")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("expected 'not yet implemented' error, got: %v", err)
	}
}

// --- parsePullRequestHook actions ---

func TestParsePullRequestHook_SynchronizeAction(t *testing.T) {
	payload := map[string]any{
		"action": "synchronize",
		"number": 7,
		"pull_request": map[string]any{
			"id":    1,
			"title": "Synced PR",
			"body":  "updated commits",
			"user":  map[string]any{"login": "dev"},
			"head": map[string]any{
				"ref": "feature",
				"sha": "newsha",
				"repo": map[string]any{
					"id":        2,
					"name":      "fork",
					"full_name": "dev/fork",
					"clone_url": "https://gitea.example.com/dev/fork.git",
					"html_url":  "https://gitea.example.com/dev/fork",
					"ssh_url":   "git@gitea.example.com:dev/fork.git",
					"owner":     map[string]any{"login": "dev"},
				},
			},
			"base": map[string]any{
				"ref": "main",
				"sha": "basesha",
				"repo": map[string]any{
					"id":        1,
					"name":      "origin",
					"full_name": "owner/origin",
					"clone_url": "https://gitea.example.com/owner/origin.git",
					"html_url":  "https://gitea.example.com/owner/origin",
					"ssh_url":   "git@gitea.example.com:owner/origin.git",
					"owner":     map[string]any{"login": "owner"},
				},
			},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "origin",
			"full_name": "owner/origin",
			"clone_url": "https://gitea.example.com/owner/origin.git",
			"html_url":  "https://gitea.example.com/owner/origin",
			"ssh_url":   "git@gitea.example.com:owner/origin.git",
		},
		"sender": map[string]any{"login": "dev"},
	}

	body, _ := json.Marshal(payload)
	req := newGiteaRequest(hook.GiteaEventPR, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse PR synchronize hook failed: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestHook type")
	}

	// "synchronize" should be mapped to "update"
	if prHook.Action != hook.ActionUpdate {
		t.Errorf("expected action %q, got %q", hook.ActionUpdate, prHook.Action)
	}
	if prHook.PullRequest.Number != 7 {
		t.Errorf("expected PR number 7, got %d", prHook.PullRequest.Number)
	}
	if prHook.Repo.Branch != "feature" {
		t.Errorf("expected branch 'feature', got %q", prHook.Repo.Branch)
	}
	if prHook.Repo.RepoType != "gitea" {
		t.Errorf("expected repoType 'gitea', got %q", prHook.Repo.RepoType)
	}
	if prHook.TargetRepo.Branch != "main" {
		t.Errorf("expected target branch 'main', got %q", prHook.TargetRepo.Branch)
	}
	if prHook.PullRequest.Title != "Synced PR" {
		t.Errorf("expected title 'Synced PR', got %q", prHook.PullRequest.Title)
	}
	if prHook.PullRequest.Author.UserName != "dev" {
		t.Errorf("expected author 'dev', got %q", prHook.PullRequest.Author.UserName)
	}
}

func TestParsePullRequestHook_UnsupportedAction(t *testing.T) {
	payload := map[string]any{
		"action":       "closed",
		"number":       1,
		"pull_request": map[string]any{},
		"repository":   map[string]any{},
		"sender":       map[string]any{},
	}

	body, _ := json.Marshal(payload)
	req := newGiteaRequest(hook.GiteaEventPR, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for unsupported action 'closed'")
	}
	if !errors.Is(err, hook.ErrUnsupportedAction) {
		t.Errorf("expected ErrUnsupportedAction, got: %v", err)
	}
}

func TestParsePullRequestHook_EmptyAction(t *testing.T) {
	// When action is empty, the switch case is skipped and conversion proceeds
	payload := map[string]any{
		"action": "",
		"number": 3,
		"pull_request": map[string]any{
			"id":    1,
			"title": "No Action PR",
			"user":  map[string]any{"login": "dev"},
			"head": map[string]any{
				"ref":  "feat",
				"sha":  "h1",
				"repo": map[string]any{"id": 2, "name": "f", "full_name": "d/f", "clone_url": "u", "html_url": "u", "ssh_url": "u", "owner": map[string]any{"login": "d"}},
			},
			"base": map[string]any{
				"ref":  "main",
				"sha":  "b1",
				"repo": map[string]any{"id": 1, "name": "o", "full_name": "x/o", "clone_url": "u", "html_url": "u", "ssh_url": "u", "owner": map[string]any{"login": "x"}},
			},
		},
		"repository": map[string]any{"id": 1},
		"sender":     map[string]any{},
	}

	body, _ := json.Marshal(payload)
	req := newGiteaRequest(hook.GiteaEventPR, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse PR with empty action failed: %v", err)
	}

	prHook := wh.(*hook.PullRequestHook)
	// Empty action should pass through unchanged
	if prHook.Action != "" {
		t.Errorf("expected empty action, got %q", prHook.Action)
	}
}

// --- convertPushHook edge cases ---

func TestConvertPushHook_EmptyRef(t *testing.T) {
	payload := map[string]any{
		"ref":    "",
		"before": "b",
		"after":  "a",
		"commits": []map[string]any{
			{"id": "a", "message": "msg", "url": "u"},
		},
		"repository": map[string]any{},
		"sender":     map[string]any{},
	}

	body, _ := json.Marshal(payload)
	var gp giteaPushHook
	if err := json.Unmarshal(body, &gp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	result := convertPushHook(&gp)
	if result.Repo.Branch != "" {
		t.Errorf("expected empty branch for empty ref, got %q", result.Repo.Branch)
	}
}

func TestConvertPushHook_ShortRef(t *testing.T) {
	// A ref like "refs/tags/v1.0" has only 3 segments: branch should be extracted
	payload := map[string]any{
		"ref":    "refs/tags/v1.0",
		"before": "b",
		"after":  "a",
		"commits": []map[string]any{
			{"id": "a", "message": "tag", "url": "u"},
		},
		"repository": map[string]any{},
		"sender":     map[string]any{},
	}

	body, _ := json.Marshal(payload)
	var gp giteaPushHook
	if err := json.Unmarshal(body, &gp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	result := convertPushHook(&gp)
	if result.Repo.Branch != "v1.0" {
		t.Errorf("expected branch 'v1.0' for refs/tags/v1.0, got %q", result.Repo.Branch)
	}
}

func TestConvertPushHook_TwoSegmentRef(t *testing.T) {
	// A ref with only 2 segments should yield empty branch
	payload := map[string]any{
		"ref":    "refs/heads",
		"before": "b",
		"after":  "a",
		"commits": []map[string]any{
			{"id": "a", "message": "msg", "url": "u"},
		},
		"repository": map[string]any{},
		"sender":     map[string]any{},
	}

	body, _ := json.Marshal(payload)
	var gp giteaPushHook
	if err := json.Unmarshal(body, &gp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	result := convertPushHook(&gp)
	if result.Repo.Branch != "" {
		t.Errorf("expected empty branch for 2-segment ref, got %q", result.Repo.Branch)
	}
}

// --- Validate edge cases ---

func TestValidate_InvalidHexSignature(t *testing.T) {
	// A non-hex string should cause hex.DecodeString to fail
	if Validate(sha1.New, []byte("msg"), []byte("key"), "ZZZZ-not-hex") {
		t.Error("Validate should return false for non-hex signature")
	}
}

func TestValidate_SHA1Fallback(t *testing.T) {
	message := []byte("test body")
	key := []byte("mysecret")

	// Compute SHA1 HMAC
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	sig := hex.EncodeToString(mac.Sum(nil))

	// validatePrefix should accept SHA1 (first check) and SHA256 (second check)
	if !validatePrefix(message, key, sig) {
		t.Error("validatePrefix should accept valid SHA1 signature")
	}
}

func TestValidatePrefix_EmptySignature(t *testing.T) {
	if validatePrefix([]byte("body"), []byte("key"), "") {
		t.Error("validatePrefix should reject empty signature")
	}
}

// --- Parse with invalid body for PR hook ---

func TestParsePullRequestHook_InvalidJSON(t *testing.T) {
	body := []byte(`{not valid`)
	req := newGiteaRequest(hook.GiteaEventPR, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in PR hook")
	}
}

// --- Parse push hook with full field coverage ---

func TestParsePushHook_FullFieldMapping(t *testing.T) {
	payload := map[string]any{
		"ref":    "refs/heads/develop",
		"before": "aaa",
		"after":  "bbb",
		"commits": []map[string]any{
			{
				"id":      "bbb",
				"message": "feat: add feature",
				"url":     "https://gitea.example.com/commit/bbb",
			},
		},
		"repository": map[string]any{
			"id":          42,
			"name":        "myrepo",
			"full_name":   "org/myrepo",
			"description": "desc",
			"private":     true,
			"clone_url":   "https://gitea.example.com/org/myrepo.git",
			"html_url":    "https://gitea.example.com/org/myrepo",
			"ssh_url":     "git@gitea.example.com:org/myrepo.git",
			"owner": map[string]any{
				"login": "org",
			},
		},
		"sender": map[string]any{
			"login": "pusher",
		},
	}

	body, _ := json.Marshal(payload)
	req := newGiteaRequest(hook.GiteaEventPush, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse push hook failed: %v", err)
	}

	p := wh.(*hook.PushHook)

	// Verify repo field mapping
	if p.Repo.RepoOpenid != "42" {
		t.Errorf("expected RepoOpenid '42', got %q", p.Repo.RepoOpenid)
	}
	if p.Repo.Private != true {
		t.Error("expected Private=true")
	}
	if p.Repo.Description != "desc" {
		t.Errorf("expected Description 'desc', got %q", p.Repo.Description)
	}
	if p.Repo.GitShhURL != "git@gitea.example.com:org/myrepo.git" {
		t.Errorf("expected GitShhURL, got %q", p.Repo.GitShhURL)
	}
	if p.Repo.FullName != "org/myrepo" {
		t.Errorf("expected FullName 'org/myrepo', got %q", p.Repo.FullName)
	}
	if p.Commit.Link != "https://gitea.example.com/commit/bbb" {
		t.Errorf("expected commit link, got %q", p.Commit.Link)
	}
}

// --- Test secret validation failure path in Parse ---

func TestParse_SecretValidationFailure(t *testing.T) {
	// Valid push payload but signed with different secret
	payload := map[string]any{
		"ref":    "refs/heads/main",
		"before": "a",
		"after":  "b",
		"commits": []map[string]any{
			{"id": "b", "message": "m", "url": "u"},
		},
		"repository": map[string]any{"id": 1, "name": "r", "full_name": "o/r", "clone_url": "u", "html_url": "u", "ssh_url": "u", "owner": map[string]any{"login": "o"}},
		"sender":     map[string]any{"login": "s"},
	}

	body, _ := json.Marshal(payload)
	// Create request signed with "secret1"
	req := newGiteaRequest(hook.GiteaEventPush, body, "secret1")

	// Parse with "secret2" → signature mismatch
	wh, err := Parse(req, "secret2")
	if err == nil {
		t.Fatal("expected error for secret mismatch")
	}
	if !errors.Is(err, hook.ErrSecretValidationFailed) {
		t.Errorf("expected ErrSecretValidationFailed, got: %v", err)
	}
	// Even on signature failure, the parsed hook should be returned
	if wh == nil {
		t.Error("expected parsed webhook to be returned even on signature failure")
	}
}

// --- convertPullRequestURL is currently a stub returning nil ---

func TestConvertPullRequestURL_ReturnsNil(t *testing.T) {
	gc := &giteaCommentHook{}
	gc.Sender.Login = "user"
	gc.Repository.Id = 1

	result, err := convertPullRequestURL(gc)
	if err != nil {
		t.Errorf("convertPullRequestURL should not return error, got: %v", err)
	}
	if result != nil {
		t.Error("convertPullRequestURL currently returns nil (stub)")
	}
}

// --- ReadAll error path ---

func TestParse_ReadBodyError(t *testing.T) {
	// Create a request with a body that returns error on read
	errReader := &errorReader{err: errors.New("read error")}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", errReader)
	req.Header.Set(hook.GiteaEvent, hook.GiteaEventPush)

	_, err := Parse(req, "secret")
	if err == nil {
		t.Fatal("expected error when body read fails")
	}
}

// errorReader is an io.Reader that always returns an error.
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, e.err
}

// Ensure unused import doesn't cause issues
var _ = bytes.NewReader
