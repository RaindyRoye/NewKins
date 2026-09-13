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
	"strings"
	"testing"

	"github.com/gokins/gokins/hook"
)

func TestValidateFunc(t *testing.T) {
	message := []byte("test message")
	key := []byte("secret")

	h := hmac.New(sha256.New, key)
	h.Write(message)
	sig := hex.EncodeToString(h.Sum(nil))

	if !Validate(sha256.New, message, key, sig) {
		t.Error("Validate should return true for valid HMAC-SHA256 signature")
	}

	if Validate(sha256.New, message, key, "invalidhex") {
		t.Error("Validate should return false for invalid hex signature")
	}

	if Validate(sha256.New, message, []byte("wrongkey"), sig) {
		t.Error("Validate should return false for wrong key")
	}
}

func TestValidateInvalidHex(t *testing.T) {
	if Validate(sha256.New, []byte("msg"), []byte("key"), "not-valid-hex") {
		t.Error("Validate should return false for non-hex signature")
	}
}

func TestValidateInternal(t *testing.T) {
	message := []byte("data")
	key := []byte("k")

	h := hmac.New(sha256.New, key)
	h.Write(message)
	sum := h.Sum(nil)

	if !validate(sha256.New, message, key, sum) {
		t.Error("validate should return true for valid HMAC")
	}

	if validate(sha256.New, message, []byte("wrong"), sum) {
		t.Error("validate should return false for wrong key")
	}
}

func TestParseNoteHook(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"event_type":  "note",
		"user": map[string]any{
			"id":       1,
			"name":     "Test User",
			"username": "testuser",
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
			"note":          "test comment",
			"noteable_id":   42,             //nolint:misspell // GitLab API field
			"noteable_type": "MergeRequest", //nolint:misspell // GitLab API field
		},
		"merge_request": map[string]any{
			"iid":           42,
			"title":         "Test MR",
			"source_branch": "feature",
			"target_branch": "main",
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
	secret := "testsecret"
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GitlabEvent, hook.GitlabEventNote)
	req.Header.Set("X-Gitlab-Token", secret)

	wh, err := Parse(req, secret)
	if err != nil {
		t.Fatalf("Parse note hook failed: %v", err)
	}

	commentHook, ok := wh.(*hook.PullRequestCommentHook)
	if !ok {
		t.Fatal("expected *hook.PullRequestCommentHook type")
	}

	if commentHook.Action != hook.EventsTypeComment {
		t.Errorf("expected action '%s', got '%s'", hook.EventsTypeComment, commentHook.Action)
	}
	if commentHook.Comment.Body != "test comment" {
		t.Errorf("expected comment body 'test comment', got '%s'", commentHook.Comment.Body)
	}
	if commentHook.Comment.Author.UserName != "testuser" {
		t.Errorf("expected author 'testuser', got '%s'", commentHook.Comment.Author.UserName)
	}
	if commentHook.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got '%s'", commentHook.Repo.RepoType)
	}
	if commentHook.PullRequest.Number != 42 {
		t.Errorf("expected PR number 42, got %d", commentHook.PullRequest.Number)
	}
}

func TestParseNoteHookInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	secret := "testsecret"
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hook.GitlabEvent, hook.GitlabEventNote)
	req.Header.Set("X-Gitlab-Token", secret)

	_, err := Parse(req, secret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in note hook")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("expected unmarshal error, got: %v", err)
	}
}

func TestConvertCommentHookDirect(t *testing.T) {
	// Test convertCommentHook directly with minimal struct
	gp := &gitlabCommentHook{
		User: struct {
			Id        int    `json:"id"`
			Name      string `json:"name"`
			Username  string `json:"username"`
			AvatarUrl string `json:"avatar_url"`
			Email     string `json:"email"`
		}{
			Username: "commenter",
		},
		Project: struct {
			Id                int    `json:"id"`
			Name              string `json:"name"`
			Description       string `json:"description"`
			WebUrl            string `json:"web_url"`
			AvatarUrl         any    `json:"avatar_url"`
			GitSshUrl         string `json:"git_ssh_url"`
			GitHttpUrl        string `json:"git_http_url"`
			Namespace         string `json:"namespace"`
			VisibilityLevel   int    `json:"visibility_level"`
			PathWithNamespace string `json:"path_with_namespace"`
			DefaultBranch     string `json:"default_branch"`
			CiConfigPath      string `json:"ci_config_path"`
			Homepage          string `json:"homepage"`
			Url               string `json:"url"`
			SshUrl            string `json:"ssh_url"`
			HttpUrl           string `json:"http_url"`
		}{
			Id: 5,
		},
		ObjectAttributes: struct {
			Attachment       any    `json:"attachment"`
			AuthorId         int    `json:"author_id"`
			ChangePosition   any    `json:"change_position"`
			CommitId         any    `json:"commit_id"`
			CreatedAt        string `json:"created_at"`
			DiscussionId     string `json:"discussion_id"`
			Id               int    `json:"id"`
			LineCode         any    `json:"line_code"`
			Note             string `json:"note"`
			NoteableId       int    `json:"noteable_id"`   //nolint:misspell // GitLab API field
			NoteableType     string `json:"noteable_type"` //nolint:misspell // GitLab API field
			OriginalPosition any    `json:"original_position"`
			Position         any    `json:"position"`
			ProjectId        int    `json:"project_id"`
			ResolvedAt       any    `json:"resolved_at"`
			ResolvedById     any    `json:"resolved_by_id"`
			ResolvedByPush   any    `json:"resolved_by_push"`
			StDiff           any    `json:"st_diff"`
			System           bool   `json:"system"`
			Type             any    `json:"type"`
			UpdatedAt        string `json:"updated_at"`
			UpdatedById      any    `json:"updated_by_id"`
			Description      string `json:"description"`
			Url              string `json:"url"`
		}{
			Note: "direct comment",
		},
	}
	gp.MergeRequest.Iid = 99
	gp.MergeRequest.Title = "Direct MR"
	gp.MergeRequest.SourceBranch = "feat"
	gp.MergeRequest.TargetBranch = "main"
	gp.MergeRequest.Source.Name = "src-repo"
	gp.MergeRequest.Target.Name = "tgt-repo"
	gp.MergeRequest.LastCommit.Id = "abc123"

	result, err := convertCommentHook(gp)
	if err != nil {
		t.Fatalf("convertCommentHook failed: %v", err)
	}

	if result.Comment.Body != "direct comment" {
		t.Errorf("expected comment body 'direct comment', got '%s'", result.Comment.Body)
	}
	if result.PullRequest.Number != 99 {
		t.Errorf("expected PR number 99, got %d", result.PullRequest.Number)
	}
	if result.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got '%s'", result.Repo.RepoType)
	}
	if result.PullRequest.Title != "Direct MR" {
		t.Errorf("expected PR title 'Direct MR', got '%s'", result.PullRequest.Title)
	}
	if result.Repo.Owner != "commenter" {
		t.Errorf("expected owner 'commenter', got '%s'", result.Repo.Owner)
	}
}

func TestConvertCommentHookFields(t *testing.T) {
	gp := &gitlabCommentHook{
		User: struct {
			Id        int    `json:"id"`
			Name      string `json:"name"`
			Username  string `json:"username"`
			AvatarUrl string `json:"avatar_url"`
			Email     string `json:"email"`
		}{
			Username: "u1",
		},
	}
	gp.Project.Id = 42
	gp.ObjectAttributes.Note = "hello"
	gp.MergeRequest.SourceBranch = "sb"
	gp.MergeRequest.TargetBranch = "tb"
	gp.MergeRequest.LastCommit.Id = "sha1"
	gp.MergeRequest.Source.HttpUrl = "https://src.git"
	gp.MergeRequest.Source.GitHttpUrl = "https://src.git"
	gp.MergeRequest.Source.SshUrl = "git@src"
	gp.MergeRequest.Source.Url = "git@src"
	gp.MergeRequest.Source.WebUrl = "https://src.web"
	gp.MergeRequest.Source.Description = "src desc"
	gp.MergeRequest.Source.PathWithNamespace = "group/src"
	gp.MergeRequest.Source.Name = "src"
	gp.MergeRequest.Target.HttpUrl = "https://tgt.git"
	gp.MergeRequest.Target.GitHttpUrl = "https://tgt.git"
	gp.MergeRequest.Target.SshUrl = "git@tgt"
	gp.MergeRequest.Target.Url = "git@tgt"
	gp.MergeRequest.Target.WebUrl = "https://tgt.web"
	gp.MergeRequest.Target.Description = "tgt desc"
	gp.MergeRequest.Target.PathWithNamespace = "group/tgt"
	gp.MergeRequest.Target.Name = "tgt"

	result, err := convertCommentHook(gp)
	if err != nil {
		t.Fatalf("convertCommentHook failed: %v", err)
	}

	// Repo checks
	if result.Repo.CloneURL != "https://src.git" {
		t.Errorf("expected Repo.CloneURL 'https://src.git', got '%s'", result.Repo.CloneURL)
	}
	if result.Repo.SshURL != "git@src" {
		t.Errorf("expected Repo.SshURL 'git@src', got '%s'", result.Repo.SshURL)
	}
	if result.Repo.FullName != "group/src" {
		t.Errorf("expected Repo.FullName 'group/src', got '%s'", result.Repo.FullName)
	}
	if result.Repo.RepoOpenid != "42" {
		t.Errorf("expected Repo.RepoOpenid '42', got '%s'", result.Repo.RepoOpenid)
	}

	// TargetRepo checks
	if result.TargetRepo.CloneURL != "https://tgt.git" {
		t.Errorf("expected TargetRepo.CloneURL 'https://tgt.git', got '%s'", result.TargetRepo.CloneURL)
	}
	if result.TargetRepo.FullName != "group/tgt" {
		t.Errorf("expected TargetRepo.FullName 'group/tgt', got '%s'", result.TargetRepo.FullName)
	}

	// Sender
	if result.Sender.UserName != "u1" {
		t.Errorf("expected Sender 'u1', got '%s'", result.Sender.UserName)
	}

	// Comment author
	if result.Comment.Author.UserName != "u1" {
		t.Errorf("expected comment author 'u1', got '%s'", result.Comment.Author.UserName)
	}
}
