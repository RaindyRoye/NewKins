package gitlab

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gokins/gokins/hook"
)

// ---- Validate / validate (HMAC) ----

func computeHMACSHA256(message, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

// TestValidate_ValidSignature verifies that Validate accepts a correct hex-encoded
// HMAC-SHA256 signature.
func TestValidate_ValidSignature(t *testing.T) {
	message := []byte("test message body")
	key := []byte("secret-key")

	sig := hex.EncodeToString(computeHMACSHA256(message, key))

	if !Validate(sha256.New, message, key, sig) {
		t.Fatal("Validate should return true for correct signature")
	}
}

// TestValidate_InvalidSignature verifies that Validate rejects a wrong signature.
func TestValidate_InvalidSignature(t *testing.T) {
	message := []byte("test message body")
	key := []byte("secret-key")

	wrongSig := hex.EncodeToString(computeHMACSHA256(message, key))
	if wrongSig[0] == 'a' {
		wrongSig = "b" + wrongSig[1:]
	} else {
		wrongSig = "a" + wrongSig[1:]
	}

	if Validate(sha256.New, message, key, wrongSig) {
		t.Fatal("Validate should return false for wrong signature")
	}
}

// TestValidate_InvalidHex verifies that Validate returns false for malformed hex.
func TestValidate_InvalidHex(t *testing.T) {
	message := []byte("test")
	key := []byte("secret")

	if Validate(sha256.New, message, key, "not-valid-hex!!") {
		t.Fatal("Validate should return false for invalid hex string")
	}
}

// TestValidate_EmptyMessage ensures HMAC validation works with empty message.
func TestValidate_EmptyMessage(t *testing.T) {
	message := []byte("")
	key := []byte("secret")

	sig := hex.EncodeToString(computeHMACSHA256(message, key))

	if !Validate(sha256.New, message, key, sig) {
		t.Fatal("Validate should accept HMAC of empty message")
	}
}

// TestValidate_DifferentMessages ensures different messages produce different HMACs.
func TestValidate_DifferentMessages(t *testing.T) {
	key := []byte("secret")
	msg1 := []byte("message-one")
	msg2 := []byte("message-two")

	sig1 := hex.EncodeToString(computeHMACSHA256(msg1, key))

	if Validate(sha256.New, msg2, key, sig1) {
		t.Fatal("Validate should not accept signature of a different message")
	}
}

// TestValidateInternal_ValidDirect tests the lower-level validate function directly.
func TestValidateInternal_ValidDirect(t *testing.T) {
	message := []byte("payload")
	key := []byte("key")

	mac := computeHMACSHA256(message, key)
	if !validate(sha256.New, message, key, mac) {
		t.Fatal("validate should return true for correct raw signature")
	}
}

// TestValidateInternal_InvalidDirect tests that validate rejects wrong raw bytes.
func TestValidateInternal_InvalidDirect(t *testing.T) {
	message := []byte("payload")
	key := []byte("key")

	wrongSig := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if validate(sha256.New, message, key, wrongSig) {
		t.Fatal("validate should return false for wrong raw signature")
	}
}

// ---- Comment / Note Hook ----

func buildCommentPayload(t *testing.T) map[string]any {
	t.Helper()
	return map[string]any{
		"object_kind": "note",
		"event_type":  "note",
		"user": map[string]any{
			"id":       1,
			"name":     "Test User",
			"username": testUser,
		},
		"project_id": 42,
		"project": map[string]any{
			"id":                  42,
			"name":                "test-repo",
			"path_with_namespace": "group/test-repo",
			"web_url":             "https://gitlab.com/group/test-repo",
			"git_http_url":        "https://gitlab.com/group/test-repo.git",
			"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
			"ssh_url":             "git@gitlab.com:group/test-repo.git",
			"http_url":            "https://gitlab.com/group/test-repo.git",
		},
		"object_attributes": map[string]any{
			"id":            100,
			"note":          "LGTM, looks great!",
			"noteable_type": "MergeRequest",
			"noteable_id":   7,
			"author_id":     1,
			"discussion_id": "abc123",
			"created_at":    "2026-01-01T00:00:00Z",
			"updated_at":    "2026-01-01T00:00:00Z",
		},
		"merge_request": map[string]any{
			"id":            7,
			"iid":           7,
			"title":         "Fix bug",
			"source_branch": "feature",
			"target_branch": "main",
			"state":         "opened",
			"source": map[string]any{
				"id":                  2,
				"name":                "test-repo",
				"description":         "src desc",
				"web_url":             "https://gitlab.com/group/test-repo",
				"git_http_url":        "https://gitlab.com/group/test-repo.git",
				"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
				"ssh_url":             "git@gitlab.com:group/test-repo.git",
				"http_url":            "https://gitlab.com/group/test-repo.git",
				"url":                 "git@gitlab.com:group/test-repo.git",
				"path_with_namespace": "group/test-repo",
			},
			"target": map[string]any{
				"id":                  1,
				"name":                "test-repo",
				"description":         "tgt desc",
				"web_url":             "https://gitlab.com/group/test-repo",
				"git_http_url":        "https://gitlab.com/group/test-repo.git",
				"git_ssh_url":         "git@gitlab.com:group/test-repo.git",
				"ssh_url":             "git@gitlab.com:group/test-repo.git",
				"http_url":            "https://gitlab.com/group/test-repo.git",
				"url":                 "git@gitlab.com:group/test-repo.git",
				"path_with_namespace": "group/test-repo",
			},
			"last_commit": map[string]any{
				"id": "sha123",
			},
		},
	}
}

// TestParseCommentHook verifies parsing of a GitLab Note event (MR comment).
func TestParseCommentHook(t *testing.T) {
	payload := buildCommentPayload(t)
	body, _ := json.Marshal(payload)

	req := newGitlabRequest(hook.GitlabEventNote, body, testSecret)
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
	if cmt.Comment.Body != "LGTM, looks great!" {
		t.Errorf("expected comment body 'LGTM, looks great!', got %q", cmt.Comment.Body)
	}
	if cmt.Comment.Author.UserName != testUser {
		t.Errorf("expected comment author %q, got %q", testUser, cmt.Comment.Author.UserName)
	}
	if cmt.PullRequest.Number != 7 {
		t.Errorf("expected PR number 7, got %d", cmt.PullRequest.Number)
	}
	if cmt.PullRequest.Title != "Fix bug" {
		t.Errorf("expected PR title 'Fix bug', got %q", cmt.PullRequest.Title)
	}
	if cmt.Repo.RepoType != "gitlab" {
		t.Errorf("expected repo type 'gitlab', got %q", cmt.Repo.RepoType)
	}
	if cmt.Repo.Branch != "feature" {
		t.Errorf("expected source branch 'feature', got %q", cmt.Repo.Branch)
	}
	if cmt.TargetRepo.Branch != "main" {
		t.Errorf("expected target branch 'main', got %q", cmt.TargetRepo.Branch)
	}
	if cmt.Sender.UserName != testUser {
		t.Errorf("expected sender %q, got %q", testUser, cmt.Sender.UserName)
	}
}

// TestParseCommentHook_InvalidJSON verifies that malformed JSON is rejected.
func TestParseCommentHook_InvalidJSON(t *testing.T) {
	body := []byte(`{not valid json`)
	req := newGitlabRequest(hook.GitlabEventNote, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for malformed JSON in comment hook")
	}
}

// TestConvertCommentHook_Direct calls convertCommentHook directly and verifies
// field mapping, including the RepoOpenid from project ID.
func TestConvertCommentHook_Direct(t *testing.T) {
	inp := &gitlabCommentHook{}
	inp.User.Username = "alice"
	inp.Project.Id = 99
	inp.ObjectAttributes.Note = "review comment"
	inp.MergeRequest.Iid = 123
	inp.MergeRequest.Title = "Big refactor"
	inp.MergeRequest.SourceBranch = "refactor-branch"
	inp.MergeRequest.TargetBranch = "develop"
	inp.MergeRequest.LastCommit.Id = "deadbeef"
	inp.MergeRequest.Source.Name = "src-name"
	inp.MergeRequest.Source.HttpUrl = "https://gl/src"
	inp.MergeRequest.Source.SshUrl = "git@gl:src"
	inp.MergeRequest.Source.Url = "git@gl:src.git"
	inp.MergeRequest.Source.WebUrl = "https://gl/src/web"
	inp.MergeRequest.Source.GitHttpUrl = "https://gl/src.git"
	inp.MergeRequest.Source.GitSshUrl = "git@gl:src.git"
	inp.MergeRequest.Source.PathWithNamespace = "grp/src"
	inp.MergeRequest.Source.Description = "src desc"
	inp.MergeRequest.Target.Name = "tgt-name"
	inp.MergeRequest.Target.HttpUrl = "https://gl/tgt"
	inp.MergeRequest.Target.SshUrl = "git@gl:tgt"
	inp.MergeRequest.Target.Url = "git@gl:tgt.git"
	inp.MergeRequest.Target.WebUrl = "https://gl/tgt/web"
	inp.MergeRequest.Target.GitHttpUrl = "https://gl/tgt.git"
	inp.MergeRequest.Target.GitSshUrl = "git@gl:tgt.git"
	inp.MergeRequest.Target.PathWithNamespace = "grp/tgt"
	inp.MergeRequest.Target.Description = "tgt desc"

	out, err := convertCommentHook(inp)
	if err != nil {
		t.Fatalf("convertCommentHook error: %v", err)
	}
	if out.Action != hook.EventsTypeComment {
		t.Errorf("expected action %q, got %q", hook.EventsTypeComment, out.Action)
	}
	if out.Comment.Body != "review comment" {
		t.Errorf("expected comment 'review comment', got %q", out.Comment.Body)
	}
	if out.PullRequest.Number != 123 {
		t.Errorf("expected PR number 123, got %d", out.PullRequest.Number)
	}
	if out.PullRequest.Title != "Big refactor" {
		t.Errorf("expected PR title 'Big refactor', got %q", out.PullRequest.Title)
	}
	if out.Repo.RepoOpenid != "99" {
		t.Errorf("expected RepoOpenid '99', got %q", out.Repo.RepoOpenid)
	}
	if out.Repo.CloneURL != "https://gl/src" {
		t.Errorf("expected CloneURL 'https://gl/src', got %q", out.Repo.CloneURL)
	}
	if out.Repo.HtmlURL != "https://gl/src/web" {
		t.Errorf("expected HtmlURL 'https://gl/src/web', got %q", out.Repo.HtmlURL)
	}
	if out.TargetRepo.Branch != "develop" {
		t.Errorf("expected target branch 'develop', got %q", out.TargetRepo.Branch)
	}
	if out.TargetRepo.Name != "tgt-name" {
		t.Errorf("expected target name 'tgt-name', got %q", out.TargetRepo.Name)
	}
	if out.Sender.UserName != "alice" {
		t.Errorf("expected sender 'alice', got %q", out.Sender.UserName)
	}
	if out.PullRequest.Base.Sha != "deadbeef" {
		t.Errorf("expected base sha 'deadbeef', got %q", out.PullRequest.Base.Sha)
	}
}

// ---- Pull Request edge cases ----

// TestParsePullRequestHook_UpdateAction verifies that action=update is accepted.
func TestParsePullRequestHook_UpdateAction(t *testing.T) {
	payload := map[string]any{
		"user": map[string]any{
			"username": "updater",
		},
		"project": map[string]any{
			"id":                  1,
			"web_url":             "https://gl/repo",
			"http_url":            "https://gl/repo.git",
			"ssh_url":             "git@gl:repo",
			"path_with_namespace": "grp/repo",
		},
		"object_attributes": map[string]any{
			"iid":           10,
			"title":         "Updated MR",
			"action":        "update",
			"source_branch": "feat",
			"target_branch": "main",
			"source": map[string]any{
				"name":                "repo",
				"web_url":             "https://gl/repo",
				"http_url":            "https://gl/repo.git",
				"ssh_url":             "git@gl:repo",
				"url":                 "git@gl:repo.git",
				"git_http_url":        "https://gl/repo.git",
				"git_ssh_url":         "git@gl:repo.git",
				"path_with_namespace": "grp/repo",
			},
			"target": map[string]any{
				"name":                "repo",
				"web_url":             "https://gl/repo",
				"http_url":            "https://gl/repo.git",
				"ssh_url":             "git@gl:repo",
				"url":                 "git@gl:repo.git",
				"git_http_url":        "https://gl/repo.git",
				"git_ssh_url":         "git@gl:repo.git",
				"path_with_namespace": "grp/repo",
			},
			"last_commit": map[string]any{
				"id": "abc123",
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPR, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse update MR hook failed: %v", err)
	}
	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatalf("expected *hook.PullRequestHook, got %T", wh)
	}
	if prHook.Action != "update" {
		t.Errorf("expected action 'update', got %q", prHook.Action)
	}
	if prHook.PullRequest.Title != "Updated MR" {
		t.Errorf("expected title 'Updated MR', got %q", prHook.PullRequest.Title)
	}
}

// TestParsePullRequestHook_EmptyAction verifies that an empty action is accepted.
func TestParsePullRequestHook_EmptyAction(t *testing.T) {
	payload := map[string]any{
		"user": map[string]any{
			"username": "opener",
		},
		"project": map[string]any{"id": 1},
		"object_attributes": map[string]any{
			"iid":           5,
			"title":         "No action MR",
			"source_branch": "feat",
			"target_branch": "main",
			"source": map[string]any{
				"name":                "repo",
				"web_url":             "https://gl/repo",
				"http_url":            "https://gl/repo.git",
				"ssh_url":             "git@gl:repo",
				"url":                 "git@gl:repo.git",
				"git_http_url":        "https://gl/repo.git",
				"git_ssh_url":         "git@gl:repo.git",
				"path_with_namespace": "grp/repo",
			},
			"target": map[string]any{
				"name":                "repo",
				"web_url":             "https://gl/repo",
				"http_url":            "https://gl/repo.git",
				"ssh_url":             "git@gl:repo",
				"url":                 "git@gl:repo.git",
				"git_http_url":        "https://gl/repo.git",
				"git_ssh_url":         "git@gl:repo.git",
				"path_with_namespace": "grp/repo",
			},
			"last_commit": map[string]any{"id": "deadbeef"},
		},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPR, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse empty-action MR hook failed: %v", err)
	}
	prHook, ok := wh.(*hook.PullRequestHook)
	if !ok {
		t.Fatalf("expected *hook.PullRequestHook, got %T", wh)
	}
	if prHook.Action != "" {
		t.Errorf("expected empty action, got %q", prHook.Action)
	}
}

// TestParsePullRequestHook_InvalidJSON verifies malformed JSON is rejected.
func TestParsePullRequestHook_InvalidJSON(t *testing.T) {
	body := []byte(`{not valid json`)
	req := newGitlabRequest(hook.GitlabEventPR, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for malformed JSON in PR hook")
	}
}

// TestParsePushHook_InvalidJSON verifies malformed JSON is rejected for push events.
func TestParsePushHook_InvalidJSON(t *testing.T) {
	body := []byte(`{not valid json`)
	req := newGitlabRequest(hook.GitlabEventPush, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for malformed JSON in push hook")
	}
}

// TestParsePushHook_EmptyRef verifies that a push with an empty ref does not panic.
func TestParsePushHook_EmptyRef(t *testing.T) {
	payload := map[string]any{
		"ref":           "",
		"before":        "a",
		"after":         "b",
		"user_username": "u",
		"project_id":    1,
		"project":       map[string]any{"id": 1},
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
		t.Fatalf("expected *hook.PushHook, got %T", wh)
	}
	if pushHook.Repo.Branch != "" {
		t.Errorf("expected empty branch for empty ref, got %q", pushHook.Repo.Branch)
	}
}

// TestParsePushHook_ShortRef verifies a ref with only 3 segments (tag ref).
func TestParsePushHook_ShortRef(t *testing.T) {
	payload := map[string]any{
		"ref":           "refs/tags/v1.0",
		"before":        "a",
		"after":         "b",
		"user_username": "u",
		"project_id":    1,
		"project":       map[string]any{"id": 1},
		"repository":    map[string]any{},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPush, body, testSecret)

	wh, err := Parse(req, testSecret)
	if err != nil {
		t.Fatalf("Parse push hook with tag ref failed: %v", err)
	}
	pushHook, ok := wh.(*hook.PushHook)
	if !ok {
		t.Fatalf("expected *hook.PushHook, got %T", wh)
	}
	if pushHook.Repo.Branch != "v1.0" {
		t.Errorf("expected branch 'v1.0' for tag ref, got %q", pushHook.Repo.Branch)
	}
}

// TestParse_SecretMismatch verifies that a valid hook with wrong secret returns
// both the parsed hook AND the secret validation error.
func TestParse_SecretMismatch(t *testing.T) {
	payload := map[string]any{
		"ref":           "refs/heads/main",
		"before":        "a",
		"after":         "b",
		"user_username": "u",
		"project_id":    1,
		"project":       map[string]any{"id": 1},
		"repository":    map[string]any{},
	}
	body, _ := json.Marshal(payload)
	req := newGitlabRequest(hook.GitlabEventPush, body, "correct-secret")

	wh, err := Parse(req, "wrong-secret")
	if !errors.Is(err, hook.ErrSecretValidationFailed) {
		t.Fatalf("expected ErrSecretValidationFailed, got: %v", err)
	}
	if wh == nil {
		t.Error("expected parsed hook to be returned even with secret mismatch")
	}
}

// TestParse_NilBody verifies that Parse handles nil request body gracefully.
func TestParse_NilBody(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set(hook.GitlabEvent, hook.GitlabEventPush)
	req.Header.Set("X-Gitlab-Token", testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error when parsing nil body push event")
	}
}
