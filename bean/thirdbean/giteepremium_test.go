package thirdbean

import (
	"encoding/json"
	"testing"
)

func TestResultGiteePremiumRepo_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": 54321,
		"full_name": "premium-org/premium-repo",
		"human_name": "Premium Org / Premium Repo",
		"url": "https://gitee.com/enterprise/api/v5/repos/premium-org/premium-repo",
		"namespace": {
			"id": 500,
			"type": "enterprise",
			"name": "Premium Org",
			"path": "premium-org",
			"html_url": "https://gitee.com/premium-org"
		},
		"path": "premium-repo",
		"name": "premium-repo",
		"owner": {
			"id": 600,
			"login": "premium-admin",
			"name": "Premium Admin",
			"avatar_url": "https://gitee.com/premium-admin/avatar",
			"html_url": "https://gitee.com/premium-admin",
			"type": "User"
		},
		"description": "A premium enterprise repository",
		"private": true,
		"fork": false,
		"default_branch": "main",
		"stargazers_count": 100,
		"watchers_count": 50,
		"forks_count": 20,
		"open_issues_count": 5,
		"has_issues": true,
		"has_wiki": true,
		"created_at": "2023-01-01T00:00:00Z",
		"updated_at": "2024-01-01T00:00:00Z",
		"pushed_at": "2024-01-01T00:00:00Z"
	}`

	var repo ResultGiteePremiumRepo
	if err := json.Unmarshal([]byte(data), &repo); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if repo.Id != 54321 {
		t.Errorf("Id: got %d, want 54321", repo.Id)
	}
	if repo.FullName != "premium-org/premium-repo" {
		t.Errorf("FullName: got %q, want %q", repo.FullName, "premium-org/premium-repo")
	}
	if repo.Namespace.Type != "enterprise" {
		t.Errorf("Namespace.Type: got %q, want %q", repo.Namespace.Type, "enterprise")
	}
	if repo.Name != "premium-repo" {
		t.Errorf("Name: got %q, want %q", repo.Name, "premium-repo")
	}
	if repo.Owner.Login != "premium-admin" {
		t.Errorf("Owner.Login: got %q, want %q", repo.Owner.Login, "premium-admin")
	}
	if !repo.Private {
		t.Error("Private: got false, want true")
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("DefaultBranch: got %q, want %q", repo.DefaultBranch, "main")
	}
	if repo.StargazersCount != 100 {
		t.Errorf("StargazersCount: got %d, want 100", repo.StargazersCount)
	}
}

func TestResultGiteePremiumCreateHooks_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": 888,
		"url": "https://gitee.com/enterprise/api/v5/repos/org/repo/hooks/888",
		"password": "enterprise-secret",
		"project_id": 7777,
		"result": "success",
		"result_code": 200,
		"push_events": true,
		"tag_push_events": true,
		"issues_events": true,
		"note_events": false,
		"merge_requests_events": true,
		"created_at": "2024-03-15T10:00:00Z"
	}`

	var hook ResultGiteePremiumCreateHooks
	if err := json.Unmarshal([]byte(data), &hook); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if hook.Id != 888 {
		t.Errorf("Id: got %d, want 888", hook.Id)
	}
	if hook.Password != "enterprise-secret" {
		t.Errorf("Password: got %q, want %q", hook.Password, "enterprise-secret")
	}
	if hook.ProjectId != 7777 {
		t.Errorf("ProjectId: got %d, want 7777", hook.ProjectId)
	}
	if hook.Result != "success" {
		t.Errorf("Result: got %q, want %q", hook.Result, "success")
	}
	if !hook.PushEvents {
		t.Error("PushEvents: got false, want true")
	}
	if !hook.IssuesEvents {
		t.Error("IssuesEvents: got false, want true")
	}
	if !hook.MergeRequestsEvents {
		t.Error("MergeRequestsEvents: got false, want true")
	}
	if hook.NoteEvents {
		t.Error("NoteEvents: got true, want false")
	}
}

func TestResultGiteePremiumRepoBranch_UnmarshalJSON(t *testing.T) {
	data := `{
		"name": "release/v1.0",
		"protected": true,
		"commit": {
			"sha": "def456abc123",
			"url": "https://gitee.com/enterprise/api/v5/repos/org/repo/commits/def456abc123"
		},
		"protection_url": "https://gitee.com/enterprise/api/v5/repos/org/repo/branches/release%2Fv1.0/protection"
	}`

	var branch ResultGiteePremiumRepoBranch
	if err := json.Unmarshal([]byte(data), &branch); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if branch.Name != "release/v1.0" {
		t.Errorf("Name: got %q, want %q", branch.Name, "release/v1.0")
	}
	if !branch.Protected {
		t.Error("Protected: got false, want true")
	}
	if branch.Commit.Sha != "def456abc123" {
		t.Errorf("Commit.Sha: got %q, want %q", branch.Commit.Sha, "def456abc123")
	}
}

func TestResultGetGiteePremiumHook_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": 999,
		"url": "https://gitee.com/enterprise/api/v5/repos/org/repo/hooks/999",
		"password": "hook-secret",
		"project_id": 8888,
		"result": "active",
		"push_events": true,
		"tag_push_events": true,
		"issues_events": false,
		"note_events": false,
		"merge_requests_events": false,
		"created_at": "2024-06-01T08:00:00Z"
	}`

	var hook ResultGetGiteePremiumHook
	if err := json.Unmarshal([]byte(data), &hook); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if hook.Id != 999 {
		t.Errorf("Id: got %d, want 999", hook.Id)
	}
	if hook.Password != "hook-secret" {
		t.Errorf("Password: got %q, want %q", hook.Password, "hook-secret")
	}
	if hook.ProjectId != 8888 {
		t.Errorf("ProjectId: got %d, want 8888", hook.ProjectId)
	}
	if !hook.PushEvents {
		t.Error("PushEvents: got false, want true")
	}
	if !hook.TagPushEvents {
		t.Error("TagPushEvents: got false, want true")
	}
	if hook.IssuesEvents {
		t.Error("IssuesEvents: got true, want false")
	}
	if hook.MergeRequestsEvents {
		t.Error("MergeRequestsEvents: got true, want false")
	}
}

func TestResultGiteePremiumRepo_EmptyJSON(t *testing.T) {
	var repo ResultGiteePremiumRepo
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
