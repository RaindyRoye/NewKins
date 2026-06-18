package github

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

const testSecret = "testsecret"

func computeSignature(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func newGithubRequest(event string, body []byte, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GithubEvent, event)
	if secret != "" {
		sig := computeSignature([]byte(secret), body)
		req.Header.Set("X-Hub-Signature", sig)
	}
	return req
}

func TestValidate(t *testing.T) {
	message := []byte("test message")
	key := []byte("secret")

	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	sig := hex.EncodeToString(mac.Sum(nil))

	if !Validate(sha256.New, message, key, sig) {
		t.Error("Validate should return true for valid HMAC-SHA256 signature")
	}

	if Validate(sha256.New, message, key, "invalidsignature") {
		t.Error("Validate should return false for invalid signature")
	}

	if Validate(sha256.New, message, []byte("wrongkey"), sig) {
		t.Error("Validate should return false for wrong key")
	}
}

func TestValidatePrefix(t *testing.T) {
	message := []byte("test body")
	key := []byte("mysecret")

	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	sig256 := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !validatePrefix(message, key, sig256) {
		t.Error("validatePrefix should accept valid sha256 signature")
	}

	// Invalid prefix format should fail
	if validatePrefix(message, key, "deadbeef") {
		t.Error("validatePrefix should reject signature without prefix")
	}

	// Unknown prefix should fail
	if validatePrefix(message, key, "md5=deadbeef") {
		t.Error("validatePrefix should reject unknown hash prefix")
	}
}

func TestParsePushHook(t *testing.T) {
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
			"id":          1,
			"name":        "test-repo",
			"full_name":   "user/test-repo",
			"clone_url":   "https://github.com/user/test-repo.git",
			"html_url":    "https://github.com/user/test-repo",
			"ssh_url":     "git@github.com:user/test-repo.git",
			"svn_url":     "https://github.com/user/test-repo",
			"git_url":     "git://github.com/user/test-repo.git",
			"description": "A test repository",
			"private":     false,
			"created_at":  1609459200,
			"owner": map[string]any{
				"login": "testuser",
			},
		},
		"sender": map[string]any{
			"login": "testuser",
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
		t.Fatal("expected *hook.PushHook type")
	}

	if pushHook.Ref != "refs/heads/main" {
		t.Errorf("expected ref 'refs/heads/main', got '%s'", pushHook.Ref)
	}
	if pushHook.Before != "abc123" {
		t.Errorf("expected before 'abc123', got '%s'", pushHook.Before)
	}
	if pushHook.After != "def456" {
		t.Errorf("expected after 'def456', got '%s'", pushHook.After)
	}
	if pushHook.Repo.Name != "test-repo" {
		t.Errorf("expected repo name 'test-repo', got '%s'", pushHook.Repo.Name)
	}
	if pushHook.Repo.Branch != "main" {
		t.Errorf("expected branch 'main', got '%s'", pushHook.Repo.Branch)
	}
	if pushHook.Repo.RepoType != "github" {
		t.Errorf("expected repoType 'github', got '%s'", pushHook.Repo.RepoType)
	}
	if pushHook.Repo.Owner != "testuser" {
		t.Errorf("expected owner 'testuser', got '%s'", pushHook.Repo.Owner)
	}
	if pushHook.Sender.UserName != "testuser" {
		t.Errorf("expected sender 'testuser', got '%s'", pushHook.Sender.UserName)
	}
	if pushHook.Commit.Message != "test commit" {
		t.Errorf("expected commit message 'test commit', got '%s'", pushHook.Commit.Message)
	}
}

func TestParsePullRequestHook(t *testing.T) {
	payload := map[string]any{
		"action": "opened",
		"number": 42,
		"pull_request": map[string]any{
			"title": "Test PR",
			"body":  "PR body",
			"head": map[string]any{
				"ref": "feature-branch",
				"sha": "head123",
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
				"sha": "base123",
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
					"owner": map[string]any{
						"login": "user",
					},
				},
			},
			"user": map[string]any{
				"login": "prauthor",
			},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "test-repo",
			"full_name": "user/test-repo",
		},
		"sender": map[string]any{
			"login": "prauthor",
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventPR, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse PR hook failed: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestHook type")
	}

	// "opened" should be mapped to "open"
	if prHook.Action != hook.ActionOpen {
		t.Errorf("expected action '%s', got '%s'", hook.ActionOpen, prHook.Action)
	}
	if prHook.Repo.RepoType != "github" {
		t.Errorf("expected repoType 'github', got '%s'", prHook.Repo.RepoType)
	}
}

func TestParseUnknownEvent(t *testing.T) {
	body := []byte(`{}`)
	secret := testSecret
	req := newGithubRequest("unknown_event", body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestParseInvalidSignature(t *testing.T) {
	body := []byte(`{"ref":"refs/heads/main","commits":[{}]}`)
	req := newGithubRequest(hook.GithubEventPush, body, "correctsecret")

	// Parse with wrong secret
	_, err := Parse(req, "wrongsecret")
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	req := newGithubRequest(hook.GithubEventPush, body, "")

	_, err := Parse(req, "")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParsePRUnsupportedAction(t *testing.T) {
	payload := map[string]any{
		"action": "closed",
		"pull_request": map[string]any{
			"head": map[string]any{
				"repo": map[string]any{},
			},
			"base": map[string]any{
				"repo": map[string]any{},
			},
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGithubRequest(hook.GithubEventPR, body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for unsupported PR action 'closed'")
	}
}
