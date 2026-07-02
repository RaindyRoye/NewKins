package thirdbean

import (
	"encoding/json"
	"testing"
)

func TestResultGithubRepo_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": 100,
		"node_id": "MDEwOlJlcG9zaXRvcnkxMDA=",
		"name": "hello-world",
		"full_name": "octocat/hello-world",
		"private": false,
		"fork": false,
		"owner": {
			"login": "octocat",
			"id": 1,
			"node_id": "MDQ6VXNlcjE=",
			"avatar_url": "https://github.com/images/error/octocat_happy.gif",
			"gravatar_id": "",
			"url": "https://api.github.com/users/octocat",
			"html_url": "https://github.com/octocat",
			"type": "User",
			"site_admin": false
		},
		"html_url": "https://github.com/octocat/hello-world",
		"description": "This your first repo!",
		"clone_url": "https://github.com/octocat/hello-world.git",
		"ssh_url": "git@github.com:octocat/hello-world.git",
		"git_url": "git://github.com/octocat/hello-world.git",
		"default_branch": "main",
		"stargazers_count": 80,
		"watchers_count": 80,
		"forks_count": 9,
		"open_issues_count": 0,
		"has_issues": true,
		"has_wiki": true,
		"has_pages": false,
		"archived": false,
		"disabled": false,
		"permissions": {
			"admin": false,
			"push": true,
			"pull": true
		},
		"size": 1,
		"created_at": "2023-01-01T00:00:00Z",
		"updated_at": "2024-01-01T00:00:00Z",
		"pushed_at": "2024-01-01T00:00:00Z"
	}`

	var repo ResultGithubRepo
	if err := json.Unmarshal([]byte(data), &repo); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if repo.Id != 100 {
		t.Errorf("Id: got %d, want 100", repo.Id)
	}
	if repo.NodeId != "MDEwOlJlcG9zaXRvcnkxMDA=" {
		t.Errorf("NodeId: got %q, want %q", repo.NodeId, "MDEwOlJlcG9zaXRvcnkxMDA=")
	}
	if repo.Name != "hello-world" {
		t.Errorf("Name: got %q, want %q", repo.Name, "hello-world")
	}
	if repo.FullName != "octocat/hello-world" {
		t.Errorf("FullName: got %q, want %q", repo.FullName, "octocat/hello-world")
	}
	if repo.Private {
		t.Error("Private: got true, want false")
	}
	if repo.Owner.Login != "octocat" {
		t.Errorf("Owner.Login: got %q, want %q", repo.Owner.Login, "octocat")
	}
	if repo.Owner.Id != 1 {
		t.Errorf("Owner.Id: got %d, want 1", repo.Owner.Id)
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("DefaultBranch: got %q, want %q", repo.DefaultBranch, "main")
	}
	if repo.StargazersCount != 80 {
		t.Errorf("StargazersCount: got %d, want 80", repo.StargazersCount)
	}
	if !repo.HasIssues {
		t.Error("HasIssues: got false, want true")
	}
	if repo.Permissions.Admin {
		t.Error("Permissions.Admin: got true, want false")
	}
	if !repo.Permissions.Push {
		t.Error("Permissions.Push: got false, want true")
	}
	if repo.Archived {
		t.Error("Archived: got true, want false")
	}
}

func TestResultGithubRepoBranch_UnmarshalJSON(t *testing.T) {
	data := `{
		"name": "feature-branch",
		"commit": {
			"sha": "6dcb09b5b57875f334f61aebed695e2e4193db5e",
			"url": "https://api.github.com/repos/octocat/hello-world/commits/6dcb09b5"
		},
		"protected": true,
		"protection": {
			"required_status_checks": {
				"enforcement_level": "non_admins",
				"contexts": ["ci/build", "ci/test"]
			}
		},
		"protection_url": "https://api.github.com/repos/octocat/hello-world/branches/feature-branch/protection"
	}`

	var branch ResultGithubRepoBranch
	if err := json.Unmarshal([]byte(data), &branch); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if branch.Name != "feature-branch" {
		t.Errorf("Name: got %q, want %q", branch.Name, "feature-branch")
	}
	if branch.Commit.Sha != "6dcb09b5b57875f334f61aebed695e2e4193db5e" {
		t.Errorf("Commit.Sha: got %q", branch.Commit.Sha)
	}
	if !branch.Protected {
		t.Error("Protected: got false, want true")
	}
	if branch.Protection.RequiredStatusChecks.EnforcementLevel != "non_admins" {
		t.Errorf("EnforcementLevel: got %q, want %q", branch.Protection.RequiredStatusChecks.EnforcementLevel, "non_admins")
	}
	if len(branch.Protection.RequiredStatusChecks.Contexts) != 2 {
		t.Errorf("Contexts: got %d, want 2", len(branch.Protection.RequiredStatusChecks.Contexts))
	}
}

func TestResultGetGithubHook_UnmarshalJSON(t *testing.T) {
	data := `{
		"type": "Repository",
		"id": 1,
		"name": "web",
		"active": true,
		"events": ["push", "pull_request"],
		"config": {
			"content_type": "json",
			"insecure_ssl": "0",
			"url": "https://ci.example.com/webhook"
		},
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-06-15T12:00:00Z",
		"url": "https://api.github.com/repos/octocat/hello-world/hooks/1",
		"test_url": "https://api.github.com/repos/octocat/hello-world/hooks/1/test",
		"ping_url": "https://api.github.com/repos/octocat/hello-world/hooks/1/pings",
		"last_response": {
			"code": 200,
			"status": "active",
			"message": "OK"
		}
	}`

	var hook ResultGetGithubHook
	if err := json.Unmarshal([]byte(data), &hook); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if hook.Id != 1 {
		t.Errorf("Id: got %d, want 1", hook.Id)
	}
	if hook.Type != "Repository" {
		t.Errorf("Type: got %q, want %q", hook.Type, "Repository")
	}
	if hook.Name != "web" {
		t.Errorf("Name: got %q, want %q", hook.Name, "web")
	}
	if !hook.Active {
		t.Error("Active: got false, want true")
	}
	if len(hook.Events) != 2 {
		t.Errorf("Events: got %d, want 2", len(hook.Events))
	}
	if hook.Config.ContentType != "json" {
		t.Errorf("Config.ContentType: got %q, want %q", hook.Config.ContentType, "json")
	}
	if hook.Config.InsecureSsl != "0" {
		t.Errorf("Config.InsecureSsl: got %q, want %q", hook.Config.InsecureSsl, "0")
	}
	if hook.Config.Url != "https://ci.example.com/webhook" {
		t.Errorf("Config.Url: got %q", hook.Config.Url)
	}
}

func TestResultGithubRepo_EmptyJSON(t *testing.T) {
	var repo ResultGithubRepo
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

func TestResultGithubRepo_MarshalRoundTrip(t *testing.T) {
	desc := "test description"
	original := ResultGithubRepo{
		Id:       200,
		NodeId:   "node123",
		Name:     "test-repo",
		FullName: "org/test-repo",
		Description: &desc,
		DefaultBranch: "main",
		StargazersCount: 50,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded ResultGithubRepo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Id != original.Id {
		t.Errorf("Id: got %d, want %d", decoded.Id, original.Id)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if *decoded.Description != *original.Description {
		t.Errorf("Description: got %q, want %q", *decoded.Description, *original.Description)
	}
	if decoded.StargazersCount != original.StargazersCount {
		t.Errorf("StargazersCount: got %d, want %d", decoded.StargazersCount, original.StargazersCount)
	}
}
