package gitea

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
	return hex.EncodeToString(mac.Sum(nil))
}

func newGiteaRequest(event string, body []byte, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GiteaEvent, event)
	if secret != "" {
		sig := computeSignature([]byte(secret), body)
		req.Header.Set("X-Gitea-Signature", sig)
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

	// SHA256 signature should pass
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	sig256 := hex.EncodeToString(mac.Sum(nil))

	if !validatePrefix(message, key, sig256) {
		t.Error("validatePrefix should accept valid SHA256 signature")
	}

	// Invalid signature should fail
	if validatePrefix(message, key, "deadbeef") {
		t.Error("validatePrefix should reject invalid signature")
	}
}

func TestParsePushHook(t *testing.T) {
	payload := map[string]interface{}{
		"ref":    "refs/heads/main",
		"before": "abc123",
		"after":  "def456",
		"commits": []map[string]interface{}{
			{
				"id":      "def456",
				"message": "test commit",
				"url":     "https://gitea.example.com/commit/def456",
			},
		},
		"repository": map[string]interface{}{
			"id":          1,
			"name":        "test-repo",
			"full_name":   "user/test-repo",
			"clone_url":   "https://gitea.example.com/user/test-repo.git",
			"html_url":    "https://gitea.example.com/user/test-repo",
			"ssh_url":     "git@gitea.example.com:user/test-repo.git",
			"description": "A test repository",
			"private":     false,
			"owner": map[string]interface{}{
				"login": "testuser",
			},
		},
		"sender": map[string]interface{}{
			"login": "testuser",
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventPush, body, secret)

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
	if pushHook.Repo.RepoType != "gitea" {
		t.Errorf("expected repoType 'gitea', got '%s'", pushHook.Repo.RepoType)
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
	payload := map[string]interface{}{
		"action": "opened",
		"pull_request": map[string]interface{}{
			"id":    1,
			"title": "Test PR",
			"number": 42,
			"head": map[string]interface{}{
				"ref": "feature-branch",
				"sha": "head123",
				"repo": map[string]interface{}{
					"id":        2,
					"name":      "test-repo-fork",
					"full_name": "forker/test-repo",
					"clone_url": "https://gitea.example.com/forker/test-repo.git",
					"html_url":  "https://gitea.example.com/forker/test-repo",
				},
			},
			"base": map[string]interface{}{
				"ref": "main",
				"sha": "base123",
				"repo": map[string]interface{}{
					"id":        1,
					"name":      "test-repo",
					"full_name": "user/test-repo",
					"clone_url": "https://gitea.example.com/user/test-repo.git",
					"html_url":  "https://gitea.example.com/user/test-repo",
				},
			},
			"user": map[string]interface{}{
				"login": "prauthor",
			},
		},
		"repository": map[string]interface{}{
			"id":        1,
			"name":      "test-repo",
			"full_name": "user/test-repo",
			"clone_url": "https://gitea.example.com/user/test-repo.git",
			"html_url":  "https://gitea.example.com/user/test-repo",
		},
		"sender": map[string]interface{}{
			"login": "prauthor",
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventPR, body, secret)

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
}

func TestParseUnknownEvent(t *testing.T) {
	body := []byte(`{}`)
	secret := testSecret
	req := newGiteaRequest("unknown_event", body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestParseInvalidSignature(t *testing.T) {
	body := []byte(`{"ref":"refs/heads/main","commits":[{}]}`)
	req := newGiteaRequest(hook.GiteaEventPush, body, "correctsecret")

	// Parse with wrong secret
	_, err := Parse(req, "wrongsecret")
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	req := newGiteaRequest(hook.GiteaEventPush, body, "")

	_, err := Parse(req, "")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
