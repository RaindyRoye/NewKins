package gitlab

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/gokins/gokins/hook"
)

func TestParseCommentHook(t *testing.T) {
	payload := map[string]any{
		"object_kind": "note",
		"event_type":  "note",
		"user": map[string]any{
			"id":       1,
			"name":     "Commenter",
			"username": "commenter",
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
			"url":                 "git@gitlab.com:group/test-repo.git",
		},
		"object_attributes": map[string]any{
			"note":       "This looks good!",
			"notable_id": 42,
		},
		"merge_request": map[string]any{
			"iid":           7,
			"title":         "Fix bug",
			"source_branch": "feature-x",
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
				"id":      "commitsha456",
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

	if commentHook.Action != hook.EventsTypeComment {
		t.Errorf("expected action '%s', got '%s'", hook.EventsTypeComment, commentHook.Action)
	}
	if commentHook.Comment.Body != "This looks good!" {
		t.Errorf("expected comment body 'This looks good!', got '%s'", commentHook.Comment.Body)
	}
	if commentHook.Repo.RepoType != "gitlab" {
		t.Errorf("expected repoType 'gitlab', got '%s'", commentHook.Repo.RepoType)
	}
	if commentHook.PullRequest.Title != "Fix bug" {
		t.Errorf("expected PR title 'Fix bug', got '%s'", commentHook.PullRequest.Title)
	}
	if commentHook.Sender.UserName != "commenter" {
		t.Errorf("expected sender 'commenter', got '%s'", commentHook.Sender.UserName)
	}
}

func TestParseCommentHookInvalidJSON(t *testing.T) {
	body := []byte(`{invalid json`)
	req := newGitlabRequest(hook.GitlabEventNote, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON in comment hook")
	}
}

func TestValidateGitlabInvalidHex(t *testing.T) {
	// Invalid hex string should return false
	result := Validate(nil, []byte("msg"), []byte("key"), "zzzz-not-hex")
	if result {
		t.Error("Validate should return false for invalid hex signature")
	}
}

func TestParseCommentHookErrorWrapping(t *testing.T) {
	// Invalid JSON should produce a wrapped error
	body := []byte(`{invalid`)
	req := newGitlabRequest(hook.GitlabEventNote, body, testSecret)

	_, err := Parse(req, testSecret)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	// The error should be unwrappable (proving %w was used)
	unwrapped := errors.Unwrap(err)
	if unwrapped == nil {
		t.Error("expected errors.Unwrap to return non-nil error")
	}
}

func TestConvertCommentHookFields(t *testing.T) {
	gp := &gitlabCommentHook{}
	gp.User.Username = "user1"
	gp.Project.Id = 100
	gp.ObjectAttributes.Note = "nice work"
	gp.MergeRequest.Iid = 5
	gp.MergeRequest.Title = "Add feature"
	gp.MergeRequest.SourceBranch = "feat"
	gp.MergeRequest.TargetBranch = "main"
	gp.MergeRequest.Source.Name = "src-repo"
	gp.MergeRequest.Source.HttpUrl = "https://gitlab.com/src.git"
	gp.MergeRequest.Source.GitHttpUrl = "https://gitlab.com/src.git"
	gp.MergeRequest.Source.SshUrl = "git@gitlab.com:src.git"
	gp.MergeRequest.Source.Url = "git@gitlab.com:src.git"
	gp.MergeRequest.Source.WebUrl = "https://gitlab.com/src"
	gp.MergeRequest.Source.PathWithNamespace = "group/src"
	gp.MergeRequest.Target.Name = "tgt-repo"
	gp.MergeRequest.Target.HttpUrl = "https://gitlab.com/tgt.git"
	gp.MergeRequest.Target.GitHttpUrl = "https://gitlab.com/tgt.git"
	gp.MergeRequest.Target.SshUrl = "git@gitlab.com:tgt.git"
	gp.MergeRequest.Target.Url = "git@gitlab.com:tgt.git"
	gp.MergeRequest.Target.WebUrl = "https://gitlab.com/tgt"
	gp.MergeRequest.Target.PathWithNamespace = "group/tgt"
	gp.MergeRequest.LastCommit.Id = "abc123"

	result, err := convertCommentHook(gp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Action != hook.EventsTypeComment {
		t.Errorf("expected action '%s', got '%s'", hook.EventsTypeComment, result.Action)
	}
	if result.Comment.Body != "nice work" {
		t.Errorf("expected comment body 'nice work', got '%s'", result.Comment.Body)
	}
	if result.Comment.Author.UserName != "user1" {
		t.Errorf("expected comment author 'user1', got '%s'", result.Comment.Author.UserName)
	}
	if result.PullRequest.Number != 5 {
		t.Errorf("expected PR number 5, got %d", result.PullRequest.Number)
	}
	if result.Repo.Name != "src-repo" {
		t.Errorf("expected repo name 'src-repo', got '%s'", result.Repo.Name)
	}
	if result.TargetRepo.Name != "tgt-repo" {
		t.Errorf("expected target repo 'tgt-repo', got '%s'", result.TargetRepo.Name)
	}
	if result.Repo.RepoOpenid != "100" {
		t.Errorf("expected RepoOpenid '100', got '%s'", result.Repo.RepoOpenid)
	}
}
