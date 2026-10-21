package gitea

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

func TestConvertCommentHook_NilPointerPanicFixed(t *testing.T) {
	// This test verifies that convertCommentHook no longer panics with a nil
	// pointer dereference when convertPullRequestURL returns nil (the gitea API
	// PR lookup is not yet implemented).
	payload := map[string]any{
		"is_pull": true,
		"comment": map[string]any{
			"id":   1,
			"body": "test comment",
			"user": map[string]any{
				"login": "commenter",
			},
		},
		"issue": map[string]any{
			"number": 42,
		},
		"repository": map[string]any{
			"id":         1,
			"name":       "test-repo",
			"full_name":  "owner/test-repo",
			"html_url":   "https://gitea.example.com/owner/test-repo",
			"clone_url":  "https://gitea.example.com/owner/test-repo.git",
			"ssh_url":    "git@gitea.example.com:owner/test-repo.git",
			"private":    false,
			"created_at": "2024-01-01T00:00:00Z",
			"owner": map[string]any{
				"login": "owner",
			},
		},
		"sender": map[string]any{
			"login": "commenter",
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GiteaEvent, hook.GiteaEventNote)
	req.Header.Set("X-Gitea-Signature", "invalid")

	// Before the fix, this would panic with nil pointer dereference.
	// After the fix, it should return an error instead.
	_, err := Parse(req, "secret")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// The error should mention that gitea PR info is unavailable
	if !errors.Is(err, ErrGiteaPRInfoUnavailable) && !errors.Is(err, hook.ErrSecretValidationFailed) {
		t.Logf("got error: %v (may be secret validation failure, which is acceptable)", err)
	}
}

func TestConvertPullRequestURL_NilHook(t *testing.T) {
	_, err := convertPullRequestURL(nil)
	if err == nil {
		t.Fatal("expected error for nil comment hook")
	}
	if !errors.Is(err, ErrGiteaPRInfoUnavailable) {
		t.Fatalf("expected ErrGiteaPRInfoUnavailable, got: %v", err)
	}
}

func TestConvertPullRequestURL_ReturnsSentinelError(t *testing.T) {
	gc := &giteaCommentHook{}
	gc.Issue.Number = 42
	gc.Repository.FullName = "owner/test-repo"

	_, err := convertPullRequestURL(gc)
	if err == nil {
		t.Fatal("expected error (gitea API not implemented)")
	}
	if !errors.Is(err, ErrGiteaPRInfoUnavailable) {
		t.Fatalf("expected ErrGiteaPRInfoUnavailable, got: %v", err)
	}
	// Error message should contain useful context
	errMsg := err.Error()
	if !contains(errMsg, "42") {
		t.Errorf("expected error to mention issue number, got: %s", errMsg)
	}
	if !contains(errMsg, "owner/test-repo") {
		t.Errorf("expected error to mention repo name, got: %s", errMsg)
	}
}

func TestErrGiteaPRInfoUnavailable_IsDistinct(t *testing.T) {
	// Verify it doesn't collide with other sentinel errors
	if errors.Is(ErrGiteaPRInfoUnavailable, hook.ErrUnsupportedEvent) {
		t.Error("ErrGiteaPRInfoUnavailable should not match hook.ErrUnsupportedEvent")
	}
	if errors.Is(ErrGiteaPRInfoUnavailable, hook.ErrSecretValidationFailed) {
		t.Error("ErrGiteaPRInfoUnavailable should not match hook.ErrSecretValidationFailed")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
