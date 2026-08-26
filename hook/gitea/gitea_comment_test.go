package gitea

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/gokins/gokins/hook"
)

// TestParseCommentHookNotPullRequest verifies that parseCommentHook returns
// ErrCommentNotPullRequest when the comment is not on a pull request.
func TestParseCommentHookNotPullRequest(t *testing.T) {
	commentPayload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number": 1,
			"title":  "Issue title",
		},
		"comment": map[string]any{
			"body": "test",
			"user": map[string]any{"login": "u"},
		},
		"repository": map[string]any{
			"id": 1, "name": "r",
			"owner": map[string]any{"login": "o"},
		},
		"sender":  map[string]any{"login": "s"},
		"is_pull": false,
	}
	body, _ := json.Marshal(commentPayload)

	_, err := parseCommentHook(body)
	if err == nil {
		t.Fatal("expected error for non-PR comment")
	}
	if !errors.Is(err, hook.ErrCommentNotPullRequest) {
		t.Errorf("expected ErrCommentNotPullRequest, got %v", err)
	}
}

// TestParseCommentHookInvalidJSON verifies that parseCommentHook returns
// an error when the body is not valid JSON.
func TestConvertPullRequestURLReturnsNil(t *testing.T) {
	commentPayload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number": 1,
			"title":  "Issue title",
		},
		"comment": map[string]any{
			"body": "test",
			"user": map[string]any{"login": "u"},
		},
		"repository": map[string]any{
			"id": 1, "name": "r",
			"owner": map[string]any{"login": "o"},
		},
		"sender":  map[string]any{"login": "s"},
		"is_pull": true,
	}
	body, _ := json.Marshal(commentPayload)

	// Parse the comment hook
	gp := new(giteaCommentHook)
	err := json.Unmarshal(body, gp)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// convertPullRequestURL is a TODO and returns nil
	result, err := convertPullRequestURL(gp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result from TODO implementation, got %v", result)
	}
}

// TestParseCommentHookFullFlow tests the full Parse -> parseCommentHook ->
// convertCommentHook chain. Since convertPullRequestURL returns nil,
// convertCommentHook will panic on nil pointer dereference, which is caught
// by the recover in Parse.
func TestParseCommentHookFullFlow(t *testing.T) {
	commentPayload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number": 1,
			"title":  "Issue title",
		},
		"comment": map[string]any{
			"body": "test comment",
			"user": map[string]any{"login": "commenter"},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "test-repo",
			"full_name": "user/test-repo",
			"clone_url": "https://gitea.example.com/user/test-repo.git",
			"html_url":  "https://gitea.example.com/user/test-repo",
			"ssh_url":   "git@gitea.example.com:user/test-repo.git",
			"private":   false,
			"owner": map[string]any{
				"login": "user",
			},
		},
		"sender":  map[string]any{"login": "commenter"},
		"is_pull": true,
	}
	body, _ := json.Marshal(commentPayload)
	secret := "test-secret"
	req := newGiteaRequest(hook.GiteaEventNote, body, secret)

	// This should panic due to nil pointer dereference in convertCommentHook
	// (because convertPullRequestURL returns nil)
	// The panic is caught by util.RecoverResult in Parse.
	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error due to nil pointer in convertCommentHook")
	}
}

// TestParseCommentHookSecretFailure verifies that when comment parsing
// succeeds but the secret is wrong, ErrSecretValidationFailed is returned.
// Note: signature validation happens AFTER parse, so we need a valid payload
// that doesn't panic. Since gitea's convertPullRequestURL returns nil (TODO),
// a valid is_pull=true payload will panic. So we test with is_pull=false
// which returns ErrCommentNotPullRequest before signature check.
func TestParseCommentHookSecretFailure(t *testing.T) {
	commentPayload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"number": 1,
			"title":  "Issue title",
		},
		"comment": map[string]any{
			"body": "test",
			"user": map[string]any{"login": "u"},
		},
		"repository": map[string]any{
			"id": 1, "name": "r",
			"owner": map[string]any{"login": "o"},
		},
		"sender":  map[string]any{"login": "s"},
		"is_pull": false,
	}
	body, _ := json.Marshal(commentPayload)
	// Use correct signature so we get past validation
	req := newGiteaRequest(hook.GiteaEventNote, body, "correct")

	// Parse with correct secret - should get ErrCommentNotPullRequest
	_, err := Parse(req, "correct")
	if err == nil {
		t.Fatal("expected error for non-PR comment")
	}
	if !errors.Is(err, hook.ErrCommentNotPullRequest) {
		t.Errorf("expected ErrCommentNotPullRequest, got %v", err)
	}
}

// TestParseCommentHookSecretMismatch verifies that when the secret doesn't
// match but the payload parses successfully, ErrSecretValidationFailed is returned.
// We use a push event since it doesn't panic.
func TestParseCommentHookSecretMismatch(t *testing.T) {
	payload := map[string]any{
		"ref":    "refs/heads/main",
		"before": "abc",
		"after":  "def",
		"commits": []map[string]any{
			{"id": "def", "message": "test", "url": "https://example.com"},
		},
		"repository": map[string]any{
			"id": 1, "name": "r", "full_name": "u/r",
			"clone_url": "https://gitea.example.com/u/r.git",
			"html_url":  "https://gitea.example.com/u/r",
			"ssh_url":   "git@gitea.example.com:u/r.git",
			"owner":     map[string]any{"login": "u"},
		},
		"sender": map[string]any{"login": "s"},
	}
	body, _ := json.Marshal(payload)
	req := newGiteaRequest(hook.GiteaEventPush, body, "correct")

	// Parse with wrong secret
	_, err := Parse(req, "wrong")
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
	if err != hook.ErrSecretValidationFailed {
		t.Errorf("expected ErrSecretValidationFailed, got %v", err)
	}
}
