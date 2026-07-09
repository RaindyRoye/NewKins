package thirdbean

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResultGiteaRepo_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": 42,
		"name": "my-repo",
		"full_name": "owner/my-repo",
		"description": "a test repo",
		"private": true,
		"fork": false,
		"mirror": false,
		"size": 1024,
		"html_url": "https://gitea.example.com/owner/my-repo",
		"ssh_url": "git@gitea.example.com:owner/my-repo.git",
		"clone_url": "https://gitea.example.com/owner/my-repo.git",
		"default_branch": "main",
		"stars_count": 10,
		"forks_count": 3,
		"watchers_count": 5,
		"open_issues_count": 2,
		"owner": {
			"id": 1,
			"login": "admin",
			"full_name": "Admin User",
			"email": "admin@example.com",
			"avatar_url": "https://gitea.example.com/avatars/1",
			"username": "admin",
			"is_admin": true,
			"active": true,
			"created": "2023-01-15T10:30:00Z"
		},
		"permissions": {
			"admin": true,
			"push": true,
			"pull": true
		},
		"has_issues": true,
		"has_wiki": false,
		"has_pull_requests": true,
		"default_merge_style": "merge",
		"internal": false,
		"created_at": "2023-06-01T12:00:00Z",
		"updated_at": "2024-01-15T08:30:00Z"
	}`

	var repo ResultGiteaRepo
	if err := json.Unmarshal([]byte(data), &repo); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if repo.Id != 42 {
		t.Errorf("Id: got %d, want 42", repo.Id)
	}
	if repo.Name != "my-repo" {
		t.Errorf("Name: got %q, want %q", repo.Name, "my-repo")
	}
	if repo.FullName != "owner/my-repo" {
		t.Errorf("FullName: got %q, want %q", repo.FullName, "owner/my-repo")
	}
	if !repo.Private {
		t.Error("Private: got false, want true")
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("DefaultBranch: got %q, want %q", repo.DefaultBranch, "main")
	}
	if repo.StarsCount != 10 {
		t.Errorf("StarsCount: got %d, want 10", repo.StarsCount)
	}
	if repo.Owner.Login != "admin" {
		t.Errorf("Owner.Login: got %q, want %q", repo.Owner.Login, "admin")
	}
	if !repo.Owner.IsAdmin {
		t.Error("Owner.IsAdmin: got false, want true")
	}
	if !repo.Permissions.Admin {
		t.Error("Permissions.Admin: got false, want true")
	}
	if !repo.Permissions.Push {
		t.Error("Permissions.Push: got false, want true")
	}
	expected := time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)
	if !repo.CreatedAt.Equal(expected) {
		t.Errorf("CreatedAt: got %v, want %v", repo.CreatedAt, expected)
	}
}

func TestResultGiteaRepoBranch_UnmarshalJSON(t *testing.T) {
	data := `{
		"name": "develop",
		"protected": true,
		"required_approvals": 2,
		"user_can_push": false,
		"user_can_merge": true,
		"commit": {
			"id": "abc123",
			"message": "fix: resolve bug",
			"url": "https://gitea.example.com/owner/repo/commit/abc123",
			"timestamp": "2024-03-10T14:22:00Z",
			"author": {
				"name": "Dev",
				"email": "dev@example.com",
				"username": "dev"
			},
			"committer": {
				"name": "Dev",
				"email": "dev@example.com",
				"username": "dev"
			}
		}
	}`

	var branch ResultGiteaRepoBranch
	if err := json.Unmarshal([]byte(data), &branch); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if branch.Name != "develop" {
		t.Errorf("Name: got %q, want %q", branch.Name, "develop")
	}
	if !branch.Protected {
		t.Error("Protected: got false, want true")
	}
	if branch.RequiredApprovals != 2 {
		t.Errorf("RequiredApprovals: got %d, want 2", branch.RequiredApprovals)
	}
	if branch.Commit.Id != "abc123" {
		t.Errorf("Commit.Id: got %q, want %q", branch.Commit.Id, "abc123")
	}
	if branch.Commit.Author.Username != "dev" {
		t.Errorf("Commit.Author.Username: got %q, want %q", branch.Commit.Author.Username, "dev")
	}
}

func TestResultGetGiteaHook_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": 7,
		"type": "gitea",
		"config": {
			"content_type": "json",
			"url": "https://ci.example.com/hook"
		},
		"events": ["push", "pull_request"],
		"active": true,
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-02-15T12:00:00Z"
	}`

	var hook ResultGetGiteaHook
	if err := json.Unmarshal([]byte(data), &hook); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if hook.Id != 7 {
		t.Errorf("Id: got %d, want 7", hook.Id)
	}
	if hook.Type != "gitea" {
		t.Errorf("Type: got %q, want %q", hook.Type, "gitea")
	}
	if hook.Config.ContentType != "json" {
		t.Errorf("Config.ContentType: got %q, want %q", hook.Config.ContentType, "json")
	}
	if hook.Config.Url != "https://ci.example.com/hook" {
		t.Errorf("Config.Url: got %q, want %q", hook.Config.Url, "https://ci.example.com/hook")
	}
	if len(hook.Events) != 2 {
		t.Errorf("Events: got %d, want 2", len(hook.Events))
	}
	if !hook.Active {
		t.Error("Active: got false, want true")
	}
}

func TestResultGiteaRepo_EmptyJSON(t *testing.T) {
	var repo ResultGiteaRepo
	if err := json.Unmarshal([]byte("{}"), &repo); err != nil {
		t.Fatalf("unmarshal empty object failed: %v", err)
	}
	if repo.Id != 0 {
		t.Errorf("Id: got %d, want 0", repo.Id)
	}
	if repo.Name != "" {
		t.Errorf("Name: got %q, want empty", repo.Name)
	}
}

func TestResultGiteaRepo_MarshalRoundTrip(t *testing.T) {
	original := ResultGiteaRepo{
		Id:            99,
		Name:          "roundtrip",
		FullName:      "org/roundtrip",
		Private:       true,
		DefaultBranch: "main",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded ResultGiteaRepo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Id != original.Id {
		t.Errorf("Id: got %d, want %d", decoded.Id, original.Id)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.FullName != original.FullName {
		t.Errorf("FullName: got %q, want %q", decoded.FullName, original.FullName)
	}
	if decoded.Private != original.Private {
		t.Errorf("Private: got %v, want %v", decoded.Private, original.Private)
	}
	if decoded.DefaultBranch != original.DefaultBranch {
		t.Errorf("DefaultBranch: got %q, want %q", decoded.DefaultBranch, original.DefaultBranch)
	}
}
