package thirdbean

import (
	"encoding/json"
	"testing"
)

func TestResultGitlabRepo_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": 50,
		"name": "my-project",
		"name_with_namespace": "Group / My Project",
		"path": "my-project",
		"path_with_namespace": "group/my-project",
		"description": "A GitLab project",
		"default_branch": "main",
		"ssh_url_to_repo": "git@gitlab.com:group/my-project.git",
		"http_url_to_repo": "https://gitlab.com/group/my-project.git",
		"web_url": "https://gitlab.com/group/my-project",
		"forks_count": 5,
		"star_count": 42,
		"visibility": "private",
		"archived": false,
		"empty_repo": false,
		"namespace": {
			"id": 10,
			"name": "Group",
			"path": "group",
			"kind": "group",
			"full_path": "group",
			"web_url": "https://gitlab.com/groups/group"
		},
		"owner": {
			"id": 3,
			"name": "Admin",
			"username": "admin",
			"state": "active",
			"web_url": "https://gitlab.com/admin"
		},
		"issues_enabled": true,
		"merge_requests_enabled": true,
		"wiki_enabled": false,
		"jobs_enabled": true,
		"snippets_enabled": false,
		"lfs_enabled": true,
		"open_issues_count": 3,
		"ci_config_path": ".gitlab-ci.yml",
		"created_at": "2023-06-01T12:00:00Z",
		"last_activity_at": "2024-01-15T08:30:00Z"
	}`

	var repo ResultGitlabRepo
	if err := json.Unmarshal([]byte(data), &repo); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if repo.Id != 50 {
		t.Errorf("Id: got %d, want 50", repo.Id)
	}
	if repo.Name != "my-project" {
		t.Errorf("Name: got %q, want %q", repo.Name, "my-project")
	}
	if repo.PathWithNamespace != "group/my-project" {
		t.Errorf("PathWithNamespace: got %q, want %q", repo.PathWithNamespace, "group/my-project")
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("DefaultBranch: got %q, want %q", repo.DefaultBranch, "main")
	}
	if repo.SshUrlToRepo != "git@gitlab.com:group/my-project.git" {
		t.Errorf("SshUrlToRepo: got %q", repo.SshUrlToRepo)
	}
	if repo.HttpUrlToRepo != "https://gitlab.com/group/my-project.git" {
		t.Errorf("HttpUrlToRepo: got %q", repo.HttpUrlToRepo)
	}
	if repo.StarCount != 42 {
		t.Errorf("StarCount: got %d, want 42", repo.StarCount)
	}
	if repo.ForksCount != 5 {
		t.Errorf("ForksCount: got %d, want 5", repo.ForksCount)
	}
	if repo.Visibility != "private" {
		t.Errorf("Visibility: got %q, want %q", repo.Visibility, "private")
	}
	if repo.Namespace.Kind != "group" {
		t.Errorf("Namespace.Kind: got %q, want %q", repo.Namespace.Kind, "group")
	}
	if repo.Owner.Username != "admin" {
		t.Errorf("Owner.Username: got %q, want %q", repo.Owner.Username, "admin")
	}
	if !repo.IssuesEnabled {
		t.Error("IssuesEnabled: got false, want true")
	}
	if !repo.MergeRequestsEnabled {
		t.Error("MergeRequestsEnabled: got false, want true")
	}
	if !repo.LfsEnabled {
		t.Error("LfsEnabled: got false, want true")
	}
	if repo.CiConfigPath != ".gitlab-ci.yml" {
		t.Errorf("CiConfigPath: got %q, want %q", repo.CiConfigPath, ".gitlab-ci.yml")
	}
}

func TestResultGitlabRepoBranch_UnmarshalJSON(t *testing.T) {
	data := `{
		"name": "main",
		"merged": false,
		"protected": true,
		"default": true,
		"developers_can_push": false,
		"developers_can_merge": true,
		"can_push": true,
		"web_url": "https://gitlab.com/group/my-project/-/tree/main",
		"commit": {
			"id": "abc123def456",
			"short_id": "abc123de",
			"title": "Initial commit",
			"message": "Initial commit",
			"author_name": "Admin",
			"author_email": "admin@example.com",
			"committer_name": "Admin",
			"committer_email": "admin@example.com",
			"authored_date": "2024-01-01T00:00:00Z",
			"committed_date": "2024-01-01T00:00:00Z",
			"parent_ids": []
		}
	}`

	var branch ResultGitlabRepoBranch
	if err := json.Unmarshal([]byte(data), &branch); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if branch.Name != "main" {
		t.Errorf("Name: got %q, want %q", branch.Name, "main")
	}
	if !branch.Protected {
		t.Error("Protected: got false, want true")
	}
	if !branch.Default {
		t.Error("Default: got false, want true")
	}
	if branch.Merged {
		t.Error("Merged: got true, want false")
	}
	if !branch.DevelopersCanMerge {
		t.Error("DevelopersCanMerge: got false, want true")
	}
	if !branch.CanPush {
		t.Error("CanPush: got false, want true")
	}
	if branch.Commit.Id != "abc123def456" {
		t.Errorf("Commit.Id: got %q, want %q", branch.Commit.Id, "abc123def456")
	}
	if branch.Commit.ShortId != "abc123de" {
		t.Errorf("Commit.ShortId: got %q, want %q", branch.Commit.ShortId, "abc123de")
	}
	if branch.Commit.AuthorName != "Admin" {
		t.Errorf("Commit.AuthorName: got %q, want %q", branch.Commit.AuthorName, "Admin")
	}
}

func TestResultGetGitlabHook_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": 33,
		"url": "https://ci.example.com/gitlab-hook",
		"push_events": true,
		"tag_push_events": true,
		"merge_requests_events": true,
		"repository_update_events": false,
		"enable_ssl_verification": true,
		"project_id": 50,
		"issues_events": false,
		"note_events": true,
		"pipeline_events": true,
		"wiki_page_events": false,
		"deployment_events": false,
		"job_events": false,
		"releases_events": false,
		"push_events_branch_filter": "main",
		"created_at": "2024-01-01T00:00:00Z"
	}`

	var hook ResultGetGitlabHook
	if err := json.Unmarshal([]byte(data), &hook); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if hook.Id != 33 {
		t.Errorf("Id: got %d, want 33", hook.Id)
	}
	if hook.Url != "https://ci.example.com/gitlab-hook" {
		t.Errorf("Url: got %q", hook.Url)
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
	if hook.RepositoryUpdateEvents {
		t.Error("RepositoryUpdateEvents: got true, want false")
	}
	if !hook.EnableSslVerification {
		t.Error("EnableSslVerification: got false, want true")
	}
	if hook.ProjectId != 50 {
		t.Errorf("ProjectId: got %d, want 50", hook.ProjectId)
	}
	if !hook.NoteEvents {
		t.Error("NoteEvents: got false, want true")
	}
	if !hook.PipelineEvents {
		t.Error("PipelineEvents: got false, want true")
	}
	if hook.PushEventsBranchFilter != "main" {
		t.Errorf("PushEventsBranchFilter: got %q, want %q", hook.PushEventsBranchFilter, "main")
	}
}

func TestResultGitlabRepo_EmptyJSON(t *testing.T) {
	var repo ResultGitlabRepo
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

func TestResultGitlabRepo_MarshalRoundTrip(t *testing.T) {
	original := ResultGitlabRepo{
		Id:                100,
		Name:              "test-project",
		PathWithNamespace: "group/test-project",
		DefaultBranch:     "main",
		Visibility:        "private",
		StarCount:         99,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded ResultGitlabRepo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Id != original.Id {
		t.Errorf("Id: got %d, want %d", decoded.Id, original.Id)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.PathWithNamespace != original.PathWithNamespace {
		t.Errorf("PathWithNamespace: got %q, want %q", decoded.PathWithNamespace, original.PathWithNamespace)
	}
	if decoded.StarCount != original.StarCount {
		t.Errorf("StarCount: got %d, want %d", decoded.StarCount, original.StarCount)
	}
}
