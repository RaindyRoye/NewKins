package thirdbean

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResultGiteeRepo_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": 12345,
		"full_name": "owner/repo-name",
		"human_name": "owner / Repo Name",
		"url": "https://gitee.com/owner/repo-name",
		"namespace": {
			"id": 100,
			"type": "user",
			"name": "owner",
			"path": "owner",
			"html_url": "https://gitee.com/owner"
		},
		"path": "repo-name",
		"name": "repo-name",
		"owner": {
			"id": 200,
			"login": "owner",
			"name": "Owner User",
			"avatar_url": "https://gitee.com/owner/avatar",
			"html_url": "https://gitee.com/owner",
			"type": "User"
		},
		"description": "A test repository",
		"private": false,
		"fork": false,
		"default_branch": "master",
		"stargazers_count": 15,
		"watchers_count": 3,
		"forks_count": 2,
		"open_issues_count": 1,
		"has_issues": true,
		"has_wiki": false,
		"created_at": "2023-01-01T00:00:00Z",
		"updated_at": "2024-01-01T00:00:00Z",
		"pushed_at": "2024-01-01T00:00:00Z"
	}`

	var repo ResultGiteeRepo
	if err := json.Unmarshal([]byte(data), &repo); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if repo.Id != int64(12345) {
		t.Errorf("Id: got %d, want 12345", repo.Id)
	}
	if repo.FullName != "owner/repo-name" {
		t.Errorf("FullName: got %q, want %q", repo.FullName, "owner/repo-name")
	}
	if repo.HumanName != "owner / Repo Name" {
		t.Errorf("HumanName: got %q, want %q", repo.HumanName, "owner / Repo Name")
	}
	if repo.Namespace.Id != 100 {
		t.Errorf("Namespace.Id: got %d, want 100", repo.Namespace.Id)
	}
	if repo.Namespace.Type != "user" {
		t.Errorf("Namespace.Type: got %q, want %q", repo.Namespace.Type, "user")
	}
	if repo.Name != "repo-name" {
		t.Errorf("Name: got %q, want %q", repo.Name, "repo-name")
	}
	if repo.Owner.Login != "owner" {
		t.Errorf("Owner.Login: got %q, want %q", repo.Owner.Login, "owner")
	}
	if repo.Owner.Id != 200 {
		t.Errorf("Owner.Id: got %d, want 200", repo.Owner.Id)
	}
	if repo.DefaultBranch != "master" {
		t.Errorf("DefaultBranch: got %q, want %q", repo.DefaultBranch, "master")
	}
	if repo.StargazersCount != 15 {
		t.Errorf("StargazersCount: got %d, want 15", repo.StargazersCount)
	}
	if repo.Private {
		t.Error("Private: got true, want false")
	}
}

func TestResultGiteeCreateHooks_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": 555,
		"url": "https://gitee.com/api/v5/repos/owner/repo/hooks/555",
		"password": "secret123",
		"project_id": 999,
		"result": "success",
		"result_code": 200,
		"push_events": true,
		"tag_push_events": true,
		"issues_events": false,
		"note_events": false,
		"merge_requests_events": true,
		"created_at": "2024-01-01T00:00:00Z"
	}`

	var hook ResultGiteeCreateHooks
	if err := json.Unmarshal([]byte(data), &hook); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if hook.Id != 555 {
		t.Errorf("Id: got %d, want 555", hook.Id)
	}
	if hook.Password != "secret123" {
		t.Errorf("Password: got %q, want %q", hook.Password, "secret123")
	}
	if hook.ProjectId != 999 {
		t.Errorf("ProjectId: got %d, want 999", hook.ProjectId)
	}
	if hook.Result != "success" {
		t.Errorf("Result: got %q, want %q", hook.Result, "success")
	}
	if !hook.PushEvents {
		t.Error("PushEvents: got false, want true")
	}
	if !hook.TagPushEvents {
		t.Error("TagPushEvents: got false, want true")
	}
	if !hook.MergeRequestsEvents {
		t.Error("MergeRequestsEvents: got false, want true")
	}
	if hook.IssuesEvents {
		t.Error("IssuesEvents: got true, want false")
	}
	expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !hook.CreatedAt.Equal(expected) {
		t.Errorf("CreatedAt: got %v, want %v", hook.CreatedAt, expected)
	}
}

func TestResultGiteeRepoBranch_UnmarshalJSON(t *testing.T) {
	data := `{
		"name": "develop",
		"protected": true,
		"commit": {
			"sha": "abc123def456",
			"url": "https://gitee.com/api/v5/repos/owner/repo/commits/abc123def456"
		}
	}`

	var branch ResultGiteeRepoBranch
	if err := json.Unmarshal([]byte(data), &branch); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if branch.Name != "develop" {
		t.Errorf("Name: got %q, want %q", branch.Name, "develop")
	}
	if !branch.Protected {
		t.Error("Protected: got false, want true")
	}
	if branch.Commit.Sha != "abc123def456" {
		t.Errorf("Commit.Sha: got %q, want %q", branch.Commit.Sha, "abc123def456")
	}
}

func TestResultGetGiteeHook_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": 777,
		"url": "https://gitee.com/api/v5/repos/owner/repo/hooks/777",
		"password": "mypassword",
		"project_id": 1234,
		"result": "active",
		"push_events": true,
		"tag_push_events": false,
		"issues_events": true,
		"note_events": true,
		"merge_requests_events": false,
		"created_at": "2024-02-15T12:00:00Z"
	}`

	var hook ResultGetGiteeHook
	if err := json.Unmarshal([]byte(data), &hook); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if hook.Id != 777 {
		t.Errorf("Id: got %d, want 777", hook.Id)
	}
	if hook.Password != "mypassword" {
		t.Errorf("Password: got %q, want %q", hook.Password, "mypassword")
	}
	if hook.ProjectId != 1234 {
		t.Errorf("ProjectId: got %d, want 1234", hook.ProjectId)
	}
	if !hook.PushEvents {
		t.Error("PushEvents: got false, want true")
	}
	if !hook.IssuesEvents {
		t.Error("IssuesEvents: got false, want true")
	}
	if !hook.NoteEvents {
		t.Error("NoteEvents: got false, want true")
	}
	if hook.TagPushEvents {
		t.Error("TagPushEvents: got true, want false")
	}
}

func TestResultGiteeRepo_EmptyJSON(t *testing.T) {
	var repo ResultGiteeRepo
	if err := json.Unmarshal([]byte("{}"), &repo); err != nil {
		t.Fatalf("unmarshal empty object failed: %v", err)
	}
	if repo.Id != 0 {
		t.Errorf("Id: got %d, want 0", repo.Id)
	}
	if repo.FullName != "" {
		t.Errorf("FullName: got %q, want empty", repo.FullName)
	}
}
