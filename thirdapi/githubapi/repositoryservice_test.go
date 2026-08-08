package githubapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gokins/gokins/bean/thirdbean"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{
			name:    "valid URL",
			uri:     "https://api.github.com",
			wantErr: false,
		},
		{
			name:    "invalid URL",
			uri:     "://invalid",
			wantErr: true,
		},
		{
			name:    "empty URL",
			uri:     "",
			wantErr: false, // Go's url.Parse accepts empty strings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("New() returned nil client")
			}
		})
	}
}

func TestNewDefault(t *testing.T) {
	client := NewDefault()
	if client == nil {
		t.Fatal("NewDefault() returned nil")
	}
	if client.BaseURL.String() != BaseApiGithub {
		t.Errorf("NewDefault() BaseURL = %v, want %v", client.BaseURL.String(), BaseApiGithub)
	}
}

func TestRepositoryService_GetRepos(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if auth != "token test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return mock repos
		repos := []*thirdbean.ResultGithubRepo{
			{
				Id:       1,
				Name:     "repo1",
				FullName: "owner/repo1",
				HtmlUrl:  "https://github.com/owner/repo1",
			},
			{
				Id:       2,
				Name:     "repo2",
				FullName: "owner/repo2",
				HtmlUrl:  "https://github.com/owner/repo2",
			},
		}
		repos[0].Owner.Login = "owner"
		repos[1].Owner.Login = "owner"
		// Omit Link header; TotalPages defaults to 1 when absent.
		json.NewEncoder(w).Encode(repos)
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	page, err := client.Repositories.GetRepos(ctx, "test-token", "user", "all", "full_name", "desc", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos() error = %v", err)
	}

	if len(page.Ropes) != 2 {
		t.Errorf("GetRepos() returned %d repos, want 2", len(page.Ropes))
	}

	if page.TotalPages != 1 {
		t.Errorf("GetRepos() TotalPages = %d, want 1 (no Link header)", page.TotalPages)
	}

	if page.Ropes[0].Name != "repo1" {
		t.Errorf("GetRepos() first repo name = %v, want repo1", page.Ropes[0].Name)
	}

	if page.Ropes[0].RepoType != "github" {
		t.Errorf("GetRepos() RepoType = %v, want github", page.Ropes[0].RepoType)
	}
}

func TestRepositoryService_GetRepos_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client, _ := New(server.URL)
	ctx := context.Background()
	_, err := client.Repositories.GetRepos(ctx, "token", "user", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Error("GetRepos() expected error for 500 response")
	}

	if err != nil {
		// Check if it's an APIError (may be wrapped)
		t.Logf("Error type: %T, value: %v", err, err)
	}
}

func TestRepositoryService_DeleteHooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "token test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, _ := New(server.URL)
	ctx := context.Background()
	err := client.Repositories.DeleteHooks(ctx, "test-token", "owner", "repo", "123")
	if err != nil {
		t.Errorf("DeleteHooks() error = %v", err)
	}
}

func TestRepositoryService_DeleteHooks_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("hook not found"))
	}))
	defer server.Close()

	client, _ := New(server.URL)
	ctx := context.Background()
	err := client.Repositories.DeleteHooks(ctx, "token", "owner", "repo", "123")
	if err == nil {
		t.Error("DeleteHooks() expected error for 404 response")
	}
}

func TestRepositoryService_CreateWebHooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json")
		}

		hook := thirdbean.ResultGetGithubHook{
			Id:        456,
			Url:       "https://example.com/webhook",
			CreatedAt: time.Now(),
		}
		hook.Config.Url = "https://example.com/webhook"
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(hook)
	}))
	defer server.Close()

	client, _ := New(server.URL)
	ctx := context.Background()
	hook, err := client.Repositories.CreateWebHooks(ctx, "token", "owner", "repo", "https://example.com/webhook", "secret")
	if err != nil {
		t.Fatalf("CreateWebHooks() error = %v", err)
	}

	if hook.Id != 456 {
		t.Errorf("CreateWebHooks() hook ID = %d, want 456", hook.Id)
	}

	if hook.Url != "https://example.com/webhook" {
		t.Errorf("CreateWebHooks() hook URL = %v, want https://example.com/webhook", hook.Url)
	}
}

func TestRepositoryService_CreateWebHooks_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer server.Close()

	client, _ := New(server.URL)
	ctx := context.Background()
	_, err := client.Repositories.CreateWebHooks(ctx, "token", "owner", "repo", "https://example.com", "secret")
	if err == nil {
		t.Error("CreateWebHooks() expected error for 403 response")
	}
}

func TestRepositoryService_GetRepoBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		branches := []*thirdbean.ResultGithubRepoBranch{
			{Name: "main"},
			{Name: "develop"},
			{Name: "feature/test"},
		}
		json.NewEncoder(w).Encode(branches)
	}))
	defer server.Close()

	client, _ := New(server.URL)
	ctx := context.Background()
	branches, err := client.Repositories.GetRepoBranches(ctx, "token", "owner", "repo")
	if err != nil {
		t.Fatalf("GetRepoBranches() error = %v", err)
	}

	if len(branches) != 3 {
		t.Errorf("GetRepoBranches() returned %d branches, want 3", len(branches))
	}

	if branches[0].Name != "main" {
		t.Errorf("GetRepoBranches() first branch = %v, want main", branches[0].Name)
	}
}

func TestRepositoryService_GetWebHooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hook1 := &thirdbean.ResultGetGithubHook{
			Id:        1,
			Url:       "https://example.com/hook1",
			CreatedAt: time.Now(),
		}
		hook1.Config.Url = "https://example.com/hook1"
		
		hook2 := &thirdbean.ResultGetGithubHook{
			Id:        2,
			Url:       "https://example.com/hook2",
			CreatedAt: time.Now(),
		}
		hook2.Config.Url = "https://example.com/hook2"
		
		hooks := []*thirdbean.ResultGetGithubHook{hook1, hook2}
		json.NewEncoder(w).Encode(hooks)
	}))
	defer server.Close()

	client, _ := New(server.URL)
	ctx := context.Background()
	hooks, err := client.Repositories.GetWebHooks(ctx, "token", "owner", "repo", 1, 10)
	if err != nil {
		t.Fatalf("GetWebHooks() error = %v", err)
	}

	if len(hooks) != 2 {
		t.Errorf("GetWebHooks() returned %d hooks, want 2", len(hooks))
	}

	if hooks[0].Id != 1 {
		t.Errorf("GetWebHooks() first hook ID = %d, want 1", hooks[0].Id)
	}
}

func TestConvertFunctions(t *testing.T) {
	t.Run("convertRepository", func(t *testing.T) {
		from := &thirdbean.ResultGithubRepo{
			Id:       123,
			Name:     "test-repo",
			FullName: "owner/test-repo",
			HtmlUrl:  "https://github.com/owner/test-repo",
		}
		from.Owner.Login = "owner"
		repo := convertRepository(from)
		if repo.Id != "123" {
			t.Errorf("convertRepository() Id = %v, want 123", repo.Id)
		}
		if repo.Name != "test-repo" {
			t.Errorf("convertRepository() Name = %v, want test-repo", repo.Name)
		}
		if repo.RepoType != "github" {
			t.Errorf("convertRepository() RepoType = %v, want github", repo.RepoType)
		}
	})

	t.Run("convertBranch", func(t *testing.T) {
		from := &thirdbean.ResultGithubRepoBranch{Name: "main"}
		branch := convertBranch(from)
		if branch.Name != "main" {
			t.Errorf("convertBranch() Name = %v, want main", branch.Name)
		}
	})

	t.Run("convertHook", func(t *testing.T) {
		now := time.Now()
		from := &thirdbean.ResultGetGithubHook{
			Id:        789,
			Url:       "https://example.com/hook",
			CreatedAt: now,
		}
		from.Config.Url = "https://example.com/hook"
		hook := convertHook(from)
		if hook.Id != 789 {
			t.Errorf("convertHook() Id = %d, want 789", hook.Id)
		}
		if hook.Url != "https://example.com/hook" {
			t.Errorf("convertHook() Url = %v, want https://example.com/hook", hook.Url)
		}
	})

	t.Run("convertRepositoryList", func(t *testing.T) {
		from := []*thirdbean.ResultGithubRepo{
			{Id: 1, Name: "repo1"},
			{Id: 2, Name: "repo2"},
		}
		repos := convertRepositoryList(from)
		if len(repos) != 2 {
			t.Errorf("convertRepositoryList() returned %d repos, want 2", len(repos))
		}
	})

	t.Run("convertBranchList", func(t *testing.T) {
		from := []*thirdbean.ResultGithubRepoBranch{
			{Name: "main"},
			{Name: "develop"},
		}
		branches := convertBranchList(from)
		if len(branches) != 2 {
			t.Errorf("convertBranchList() returned %d branches, want 2", len(branches))
		}
	})

	t.Run("convertHookList", func(t *testing.T) {
		from := []*thirdbean.ResultGetGithubHook{
			{Id: 1},
			{Id: 2},
		}
		hooks := convertHookList(from)
		if len(hooks) != 2 {
			t.Errorf("convertHookList() returned %d hooks, want 2", len(hooks))
		}
	})
}
