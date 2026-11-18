package gitlab

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

const (
	testSecret = "testsecret"
	testUser   = "testuser"
)

func newGitlabRequest(event string, body []byte, secret string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GitlabEvent, event)
	if secret != "" {
		req.Header.Set("X-Gitlab-Token", secret)
	}
	return req
}

func TestParsePushHook(t *testing.T) {
	payload := map[string]any{
		"object_kind":   "push",
		"event_name":    "push",
		"ref":           "refs/heads/main",
		"before":        "abc123",
		"after":         "def456",
		"user_username": "testuser",
		"project_id":    1,
		"project": map[string]any{
			"id":                  1,
			"name":                "test-repo",
			"path_with_namespace": "group/test-repo",
			"web_url":             "https://gitlab.com/group/test-repo",
			"git_http_url":        "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
			"ssh_url":             "git@gitlab.com:group/test-repo.git",
			"http_url":            "https://gitlab.com/group/test-repo.git",
		},
		"commits": []map[string]any{
			{
				"id":      "def456",
				"message": "test commit",
				"url":     "https://gitlab.com/group/test-repo/-/commit/def456",
			},
		},
		"repository": map[string]any{
			"name":             "test-repo",
			"url":              "git@gitlab.com:group/test-repo.git",
			"description":      "A test repository",
			"homepage":         "https://gitlab.com/group/test-repo",
			"git_http_url":     "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":      "git@gitlab.com:group/test-repo.git",
			"visibility_level": 0,
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGitlabRequest(hook.GitlabEventPush, body, secret)

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
	if pushHook.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got '%s'", pushHook.Repo.RepoType)
	}
	if pushHook.Repo.Owner != testUser {
		t.Errorf("expected owner 'testuser', got '%s'", pushHook.Repo.Owner)
	}
	if pushHook.Sender.UserName != testUser {
		t.Errorf("expected sender 'testuser', got '%s'", pushHook.Sender.UserName)
	}
}

func TestParsePullRequestHook(t *testing.T) {
	payload := map[string]any{
		"object_kind": "merge_request",
		"event_type":  "merge_request",
		"user": map[string]any{
			"id":       1,
			"name":     "Test User",
			"username": "testuser",
		},
		"project": map[string]any{
			"id":                  1,
			"name":                "test-repo",
			"path_with_namespace": "group/test-repo",
			"web_url":             "https://gitlab.com/group/test-repo",
			"git_http_url":        "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
			"ssh_url":             "git@gitlab.com:group/test-repo.git",
			"http_url":            "https://gitlab.com/group/test-repo.git",
		},
		"object_attributes": map[string]any{
			"id":            1,
			"iid":           42,
			"title":         "Test MR",
			"source_branch": "feature-branch",
			"target_branch": "main",
			"action":        "open",
			"state":         "opened",
			"source": map[string]any{
				"id":                  2,
				"name":                "test-repo-fork",
				"description":         "forked repo",
				"web_url":             "https://gitlab.com/forker/test-repo",
				"git_http_url":        "https://gitlab.com/forker/test-repo.git",
				"git_ssh_url":         "git@gitlab.com:forker/test-repo.git",
				"ssh_url":             "git@gitlab.com:forker/test-repo.git",
				"http_url":            "https://gitlab.com/forker/test-repo.git",
				"url":                 "git@gitlab.com:forker/test-repo.git",
				"path_with_namespace": "forker/test-repo",
			},
			"target": map[string]any{
				"id":                  1,
				"name":                "test-repo",
				"description":         "base repo",
				"web_url":             "https://gitlab.com/group/test-repo",
				"git_http_url":        "https://gitlab.com/group/test-repo.git",
				"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
				"ssh_url":             "git@gitlab.com:group/test-repo.git",
				"http_url":            "https://gitlab.com/group/test-repo.git",
				"url":                 "git@gitlab.com:group/test-repo.git",
				"path_with_namespace": "group/test-repo",
			},
			"last_commit": map[string]any{
				"id":      "commitsha123",
				"message": "latest commit",
			},
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGitlabRequest(hook.GitlabEventPR, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse MR hook failed: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestHook type")
	}

	if prHook.Action != hook.ActionOpen {
		t.Errorf("expected action '%s', got '%s'", hook.ActionOpen, prHook.Action)
	}
	if prHook.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got '%s'", prHook.Repo.RepoType)
	}
	if prHook.Sender.UserName != testUser {
		t.Errorf("expected sender 'testuser', got '%s'", prHook.Sender.UserName)
	}
}

func TestParseUnknownEvent(t *testing.T) {
	body := []byte(`{}`)
	secret := testSecret
	req := newGitlabRequest("Unknown Hook", body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestParseInvalidSecret(t *testing.T) {
	payload := map[string]any{
		"ref":    "refs/heads/main",
		"before": "abc",
		"after":  "def",
		"commits": []map[string]any{
			{"id": "def", "message": "test"},
		},
		"project":    map[string]any{"id": 1},
		"repository": map[string]any{},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPush, body, "correctsecret")

	_, err := Parse(req, "wrongsecret")
	if err == nil {
		t.Fatal("expected error for invalid secret")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	req := newGitlabRequest(hook.GitlabEventPush, body, "secret")

	_, err := Parse(req, "secret")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParsePRUnsupportedAction(t *testing.T) {
	payload := map[string]any{
		"object_attributes": map[string]any{
			"action": "close",
			"source": map[string]any{},
			"target": map[string]any{},
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGitlabEventPR(body, secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for unsupported MR action 'close'")
	}
}

// newGitlabEventPR is a helper for PR-specific requests.
func newGitlabEventPR(body []byte, secret string) *http.Request {
	return newGitlabRequest(hook.GitlabEventPR, body, secret)
}

func TestParseCommentHook(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"event_type":  "note",
		"user": map[string]any{
			"id":       1,
			"name":     "Test User",
			"username": testUser,
		},
		"project_id": 1,
		"project": map[string]any{
			"id":                  1,
			"name":                "test-repo",
			"path_with_namespace": "group/test-repo",
			"web_url":             "https://gitlab.com/group/test-repo",
			"git_http_url":        "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
			"ssh_url":             "git@gitlab.com:group/test-repo.git",
			"http_url":            "https://gitlab.com/group/test-repo.git",
		},
		"object_attributes": map[string]any{
			"id":             100,
			"note":           "LGTM, merging!",
			"noteable_type":  "MergeRequest",
			"noteable_id":    42,
		},
		"merge_request": map[string]any{
			"id":             1,
			"iid":            42,
			"title":          "Test MR",
			"source_branch":  "feature-branch",
			"target_branch":  "main",
			"state":          "opened",
			"source": map[string]any{
				"id":                  2,
				"name":                "test-repo-fork",
				"description":         "forked repo",
				"web_url":             "https://gitlab.com/forker/test-repo",
				"git_http_url":        "https://gitlab.com/forker/test-repo.git",
				"git_ssh_url":         "git@gitlab.com:forker/test-repo.git",
				"ssh_url":             "git@gitlab.com:forker/test-repo.git",
				"http_url":            "https://gitlab.com/forker/test-repo.git",
				"url":                 "git@gitlab.com:forker/test-repo.git",
				"path_with_namespace": "forker/test-repo",
			},
			"target": map[string]any{
				"id":                  1,
				"name":                "test-repo",
				"description":         "base repo",
				"web_url":             "https://gitlab.com/group/test-repo",
				"git_http_url":        "https://gitlab.com/group/test-repo.git",
				"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
				"ssh_url":             "git@gitlab.com:group/test-repo.git",
				"http_url":            "https://gitlab.com/group/test-repo.git",
				"url":                 "git@gitlab.com:group/test-repo.git",
				"path_with_namespace": "group/test-repo",
			},
			"last_commit": map[string]any{
				"id":      "commitsha123",
				"message": "latest commit",
			},
		},
	}

	body, _ := json.Marshal(payload)
	secret := testSecret
	req := newGitlabRequest(hook.GitlabEventNote, body, secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse comment hook failed: %v", err)
	}

	commentHook, ok := wh.(*hook.PullRequestCommentHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestCommentHook type")
	}

	if commentHook.Comment.Body != "LGTM, merging!" {
		t.Errorf("expected comment body 'LGTM, merging!', got '%s'", commentHook.Comment.Body)
	}
	if commentHook.Comment.Author.UserName != testUser {
		t.Errorf("expected comment author '%s', got '%s'", testUser, commentHook.Comment.Author.UserName)
	}
	if commentHook.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got '%s'", commentHook.Repo.RepoType)
	}
	if commentHook.Repo.Branch != "feature-branch" {
		t.Errorf("expected branch 'feature-branch', got '%s'", commentHook.Repo.Branch)
	}
	if commentHook.TargetRepo.Branch != "main" {
		t.Errorf("expected target branch 'main', got '%s'", commentHook.TargetRepo.Branch)
	}
	if commentHook.PullRequest.Number != 42 {
		t.Errorf("expected PR number 42, got %d", commentHook.PullRequest.Number)
	}
	if commentHook.PullRequest.Title != "Test MR" {
		t.Errorf("expected PR title 'Test MR', got '%s'", commentHook.PullRequest.Title)
	}
	if commentHook.Sender.UserName != testUser {
		t.Errorf("expected sender '%s', got '%s'", testUser, commentHook.Sender.UserName)
	}
}

func TestParseCommentHookInvalidJSON(t *testing.T) {
	body := []byte(`{bad json}`)
	req := newGitlabRequest(hook.GitlabEventNote, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment hook")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		message   string
		key       string
		signature string
		want      bool
	}{
		{
			name:    "valid signature",
			message: "test message",
			key:     "secret-key",
			// Pre-computed HMAC-SHA256 of "test message" with key "secret-key"
			signature: "",
			want:      false, // Will be replaced with computed value below
		},
		{
			name:      "invalid hex signature",
			message:   "test message",
			key:       "secret-key",
			signature: "not-valid-hex",
			want:      false,
		},
		{
			name:      "empty signature",
			message:   "test message",
			key:       "secret-key",
			signature: "",
			want:      false,
		},
	}

	// Compute a valid HMAC for the first test case
	import_crypto_hmac := func() string {
		mac := hmac.New(sha256.New, []byte("secret-key"))
		mac.Write([]byte("test message"))
		return hex.EncodeToString(mac.Sum(nil))
	}
	tests[0].signature = import_crypto_hmac()
	tests[0].want = true

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(sha256.New, []byte(tt.message), []byte(tt.key), tt.signature)
			if got != tt.want {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateMismatch(t *testing.T) {
	mac := hmac.New(sha256.New, []byte("key1"))
	mac.Write([]byte("message"))
	validSig := hex.EncodeToString(mac.Sum(nil))

	// Same signature but different message should fail
	if Validate(sha256.New, []byte("different message"), []byte("key1"), validSig) {
		t.Error("expected validation to fail for different message")
	}

	// Same signature but different key should fail
	if Validate(sha256.New, []byte("message"), []byte("key2"), validSig) {
		t.Error("expected validation to fail for different key")
	}
}

func TestParsePushHookNoRef(t *testing.T) {
	payload := map[string]any{
		"object_kind":   "push",
		"ref":           "",
		"before":        "abc",
		"after":         "def",
		"user_username": testUser,
		"project_id":    1,
		"project":       map[string]any{"id": 1, "name": "repo"},
		"repository":    map[string]any{},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPush, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse push hook with empty ref failed: %v", err)
	}

	pushHook, ok := wh.(*hook.PushHook)
	if !ok {
		t.Fatal("expected *hook.PushHook type")
	}
	if pushHook.Repo.Branch != "" {
		t.Errorf("expected empty branch for empty ref, got '%s'", pushHook.Repo.Branch)
	}
}

func TestParsePushHookRefWithNoSlash(t *testing.T) {
	payload := map[string]any{
		"object_kind":   "push",
		"ref":           "tag-v1.0",
		"before":        "abc",
		"after":         "def",
		"user_username": testUser,
		"project_id":    1,
		"project":       map[string]any{"id": 1, "name": "repo"},
		"repository":    map[string]any{},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPush, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse push hook with tag ref failed: %v", err)
	}

	pushHook := wh.(*hook.PushHook)
	// "tag-v1.0" has only one segment after split by "/", so branch should be empty
	if pushHook.Repo.Branch != "" {
		t.Errorf("expected empty branch for tag ref, got '%s'", pushHook.Repo.Branch)
	}
}

func TestParsePRWithEmptyAction(t *testing.T) {
	payload := map[string]any{
		"object_attributes": map[string]any{
			"action": "",
			"source": map[string]any{"name": "src"},
			"target": map[string]any{"name": "tgt"},
		},
		"user":    map[string]any{"username": testUser},
		"project": map[string]any{"id": 1},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabEventPR(body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse PR with empty action should not fail: %v", err)
	}

	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestHook type")
	}
	if prHook.Action != "" {
		t.Errorf("expected empty action, got '%s'", prHook.Action)
	}
}

func TestParsePRWithUpdateAction(t *testing.T) {
	payload := map[string]any{
		"object_attributes": map[string]any{
			"action":        hook.ActionUpdate,
			"source":        map[string]any{"name": "src"},
			"target":        map[string]any{"name": "tgt"},
		},
		"user":    map[string]any{"username": testUser},
		"project": map[string]any{"id": 1},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabEventPR(body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse PR with update action should not fail: %v", err)
	}

	prHook := wh.(*hook.PullRequestHook)
	if prHook.Action != hook.ActionUpdate {
		t.Errorf("expected action '%s', got '%s'", hook.ActionUpdate, prHook.Action)
	}
}
