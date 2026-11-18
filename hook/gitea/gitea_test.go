package gitea

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // G505: SHA1 required for Gitea webhook signature verification
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
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", bytes.NewReader(body))
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
	payload := map[string]any{
		"ref":    "refs/heads/main",
		"before": "abc123",
		"after":  "def456",
		"commits": []map[string]any{
			{
				"id":      "def456",
				"message": "test commit",
				"url":     "https://gitea.example.com/commit/def456",
			},
		},
		"repository": map[string]any{
			"id":          1,
			"name":        "test-repo",
			"full_name":   "user/test-repo",
			"clone_url":   "https://gitea.example.com/user/test-repo.git",
			"html_url":    "https://gitea.example.com/user/test-repo",
			"ssh_url":     "git@gitea.example.com:user/test-repo.git",
			"description": "A test repository",
			"private":     false,
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
	payload := map[string]any{
		"action": "opened",
		"pull_request": map[string]any{
			"id":     1,
			"title":  "Test PR",
			"number": 42,
			"head": map[string]any{
				"ref": "feature-branch",
				"sha": "head123",
				"repo": map[string]any{
					"id":        2,
					"name":      "test-repo-fork",
					"full_name": "forker/test-repo",
					"clone_url": "https://gitea.example.com/forker/test-repo.git",
					"html_url":  "https://gitea.example.com/forker/test-repo",
				},
			},
			"base": map[string]any{
				"ref": "main",
				"sha": "base123",
				"repo": map[string]any{
					"id":        1,
					"name":      "test-repo",
					"full_name": "user/test-repo",
					"clone_url": "https://gitea.example.com/user/test-repo.git",
					"html_url":  "https://gitea.example.com/user/test-repo",
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
			"clone_url": "https://gitea.example.com/user/test-repo.git",
			"html_url":  "https://gitea.example.com/user/test-repo",
		},
		"sender": map[string]any{
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

func TestParseCommentHookNotPullRequest(t *testing.T) {
	payload := map[string]any{
		"is_pull": false,
		"comment": map[string]any{
			"id":   1,
			"body": "nice work",
		},
		"repository": map[string]any{"id": 1},
		"sender":     map[string]any{"login": "testuser"},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventNote, body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for comment not associated with PR")
	}
}

func TestParseCommentHookInvalidJSON(t *testing.T) {
	body := []byte(`{broken json`)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventNote, body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment hook")
	}
}

func TestParseCommentHookIsPull(t *testing.T) {
	// convertPullRequestURL returns nil,nil (TODO), so convertCommentHook
	// will panic on nil pointer access. The defer recover in Parse catches it.
	payload := map[string]any{
		"is_pull": true,
		"issue": map[string]any{
			"id":     10,
			"number": 5,
		},
		"comment": map[string]any{
			"id":   100,
			"body": "LGTM",
			"user": map[string]any{"login": "reviewer"},
		},
		"repository": map[string]any{
			"id":        1,
			"name":      "test-repo",
			"full_name": "user/test-repo",
			"clone_url": "https://gitea.example.com/user/test-repo.git",
			"html_url":  "https://gitea.example.com/user/test-repo",
			"ssh_url":   "git@gitea.example.com:user/test-repo.git",
			"owner":     map[string]any{"login": "testuser"},
		},
		"sender": map[string]any{"login": "reviewer"},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventNote, body, secret)

	// This will trigger parseCommentHook → convertCommentHook → nil pointer panic
	// caught by util.RecoverResult in Parse
	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error from convertCommentHook nil pointer (TODO not implemented)")
	}
}

func TestParsePRWithSynchronizeAction(t *testing.T) {
	payload := map[string]any{
		"action": "synchronize",
		"number": 42,
		"pull_request": map[string]any{
			"id":    1,
			"title": "Sync PR",
			"body":  "updated commits",
			"user":  map[string]any{"login": "prauthor"},
			"head": map[string]any{
				"ref": "feature",
				"sha": "headsha",
				"repo": map[string]any{
					"id":        2,
					"name":      "fork-repo",
					"full_name": "forker/test-repo",
					"clone_url": "https://gitea.example.com/forker/test-repo.git",
					"html_url":  "https://gitea.example.com/forker/test-repo",
					"ssh_url":   "git@gitea.example.com:forker/test-repo.git",
					"owner":     map[string]any{"login": "forker"},
				},
			},
			"base": map[string]any{
				"ref": "main",
				"sha": "basesha",
				"repo": map[string]any{
					"id":        1,
					"name":      "test-repo",
					"full_name": "user/test-repo",
					"clone_url": "https://gitea.example.com/user/test-repo.git",
					"html_url":  "https://gitea.example.com/user/test-repo",
					"ssh_url":   "git@gitea.example.com:user/test-repo.git",
					"owner":     map[string]any{"login": "testuser"},
				},
			},
		},
		"repository": map[string]any{"id": 1},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventPR, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse PR with synchronize action failed: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestHook type")
	}
	if prHook.Action != hook.ActionUpdate {
		t.Errorf("expected action '%s', got '%s'", hook.ActionUpdate, prHook.Action)
	}
	if prHook.PullRequest.Title != "Sync PR" {
		t.Errorf("expected title 'Sync PR', got '%s'", prHook.PullRequest.Title)
	}
	if prHook.Repo.Branch != "feature" {
		t.Errorf("expected branch 'feature', got '%s'", prHook.Repo.Branch)
	}
	if prHook.TargetRepo.Branch != "main" {
		t.Errorf("expected target branch 'main', got '%s'", prHook.TargetRepo.Branch)
	}
}

func TestParsePRUnsupportedAction(t *testing.T) {
	payload := map[string]any{
		"action": "closed",
		"pull_request": map[string]any{
			"head": map[string]any{"repo": map[string]any{}},
			"base": map[string]any{"repo": map[string]any{}},
		},
		"repository": map[string]any{"id": 1},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventPR, body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for unsupported action 'closed'")
	}
}

func TestParsePRInvalidJSON(t *testing.T) {
	body := []byte(`{invalid`)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventPR, body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in PR hook")
	}
}

func TestValidatePrefixSHA1(t *testing.T) {
	message := []byte("test body for sha1")
	key := []byte("mysecret")

	// SHA1 signature should be accepted when SHA256 fails
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	sig1 := hex.EncodeToString(mac.Sum(nil))

	if !validatePrefix(message, key, sig1) {
		t.Error("validatePrefix should accept valid SHA1 signature")
	}
}

func TestPushHookEmptyRef(t *testing.T) {
	payload := map[string]any{
		"ref":    "",
		"before": "abc",
		"after":  "def",
		"commits": []map[string]any{
			{"id": "def", "message": "msg", "url": "http://x"},
		},
		"repository": map[string]any{"id": 1, "name": "repo", "owner": map[string]any{"login": "u"}},
		"sender":     map[string]any{"login": "u"},
	}
	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventPush, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse push with empty ref failed: %v", err)
	}
	pushHook := wh.(*hook.PushHook)
	if pushHook.Repo.Branch != "" {
		t.Errorf("expected empty branch, got '%s'", pushHook.Repo.Branch)
	}
}

func TestPushHookTagRef(t *testing.T) {
	payload := map[string]any{
		"ref":    "refs/tags/v1.0",
		"before": "000",
		"after":  "aaa",
		"commits": []map[string]any{
			{"id": "aaa", "message": "tag", "url": "http://x"},
		},
		"repository": map[string]any{"id": 1, "name": "repo", "owner": map[string]any{"login": "u"}},
		"sender":     map[string]any{"login": "u"},
	}
	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventPush, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse push with tag ref failed: %v", err)
	}
	pushHook := wh.(*hook.PushHook)
	if pushHook.Repo.Branch != "v1.0" {
		t.Errorf("expected branch 'v1.0', got '%s'", pushHook.Repo.Branch)
	}
}

func TestValidateInvalidHex(t *testing.T) {
	if Validate(sha256.New, []byte("msg"), []byte("key"), "zzz-not-hex") {
		t.Error("Validate should return false for non-hex signature")
	}
}

func TestPREmptyAction(t *testing.T) {
	payload := map[string]any{
		"action": "",
		"number": 7,
		"pull_request": map[string]any{
			"id":    1,
			"title": "Empty Action PR",
			"user":  map[string]any{"login": "author"},
			"head": map[string]any{
				"ref": "feat",
				"sha": "hsha",
				"repo": map[string]any{
					"id":        2,
					"name":      "fork",
					"full_name": "forker/repo",
					"clone_url": "https://g.example.com/forker/repo.git",
					"html_url":  "https://g.example.com/forker/repo",
					"ssh_url":   "git@g.example.com:forker/repo.git",
					"owner":     map[string]any{"login": "forker"},
				},
			},
			"base": map[string]any{
				"ref": "main",
				"sha": "bsha",
				"repo": map[string]any{
					"id":        1,
					"name":      "repo",
					"full_name": "user/repo",
					"clone_url": "https://g.example.com/user/repo.git",
					"html_url":  "https://g.example.com/user/repo",
					"ssh_url":   "git@g.example.com:user/repo.git",
					"owner":     map[string]any{"login": "user"},
				},
			},
		},
		"repository": map[string]any{"id": 1},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGiteaRequest(hook.GiteaEventPR, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse PR with empty action failed: %v", err)
	}
	prHook := wh.(*hook.PullRequestHook)
	if prHook.Action != "" {
		t.Errorf("expected empty action, got '%s'", prHook.Action)
	}
}
