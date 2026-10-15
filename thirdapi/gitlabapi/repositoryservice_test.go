package gitlabapi

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
			uri:     "https://gitlab.com/api/v4",
			wantErr: false,
		},
		{
			name:    "invalid URL",
			uri:     "://invalid",
			wantErr: true,
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
	if client.BaseURL.String() != BaseApiGitlab {
		t.Errorf("NewDefault() BaseURL = %v, want %v", client.BaseURL.String(), BaseApiGitlab)
	}
}

func TestRepositoryService_GetRepos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		repos := []*thirdbean.ResultGitlabRepo{
			{
				Id:                1,
				Name:              "repo1",
				Path:              "repo1",
				PathWithNamespace: "owner/repo1",
				WebUrl:            "https://gitlab.com/owner/repo1",
			},
			{
				Id:                2,
				Name:              "repo2",
				Path:              "repo2",
				PathWithNamespace: "owner/repo2",
				WebUrl:            "https://gitlab.com/owner/repo2",
			},
		}
		repos[0].Owner.Username = "owner"
		repos[0].Namespace.Path = "owner"
		repos[1].Owner.Username = "owner"
		repos[1].Namespace.Path = "owner"
		w.Header().Set("X-Total-Pages", "3")
		json.NewEncoder(w).Encode(repos)
	}))
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	page, err := client.Repositories.GetRepos(ctx, "test-token", "user", "all", "updated", "desc", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos() error = %v", err)
	}

	if len(page.Ropes) != 2 {
		t.Errorf("GetRepos() returned %d repos, want 2", len(page.Ropes))
	}

	if page.TotalPages != 3 {
		t.Errorf("GetRepos() TotalPages = %d, want 3", page.TotalPages)
	}

	if page.Ropes[0].Name != "repo1" {
		t.Errorf("GetRepos() first repo name = %v, want repo1", page.Ropes[0].Name)
	}

	if page.Ropes[0].RepoType != "gitlab" {
		t.Errorf("GetRepos() RepoType = %v, want gitlab", page.Ropes[0].RepoType)
	}
}

func TestRepositoryService_GetRepos_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer server.Close()

	client, _ := New(server.URL)
	ctx := context.Background()
	_, err := client.Repositories.GetRepos(ctx, "token", "user", "all", "updated", "desc", 1, 10)
	if err == nil {
		t.Error("GetRepos() expected error for 403 response")
	}
	t.Logf("Error: %v", err)
}

func TestRepositoryService_DeleteHooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
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

		hook := thirdbean.ResultGetGitlabHook{
			Id:        789,
			Url:       "https://example.com/webhook",
			CreatedAt: time.Now(),
		}
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

	if hook.Id != 789 {
		t.Errorf("CreateWebHooks() hook ID = %d, want 789", hook.Id)
	}

	if hook.Url != "https://example.com/webhook" {
		t.Errorf("CreateWebHooks() hook URL = %v, want https://example.com/webhook", hook.Url)
	}
}

func TestRepositoryService_CreateWebHooks_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("invalid URL"))
	}))
	defer server.Close()

	client, _ := New(server.URL)
	ctx := context.Background()
	_, err := client.Repositories.CreateWebHooks(ctx, "token", "owner", "repo", "invalid-url", "secret")
	if err == nil {
		t.Error("CreateWebHooks() expected error for 422 response")
	}
}

func TestRepositoryService_GetRepoBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		branches := []*thirdbean.ResultGitlabRepoBranch{
			{Name: "main", Default: true, Protected: true},
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

func TestRepositoryService_GetRepoBranches_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer server.Close()

	client, _ := New(server.URL)
	ctx := context.Background()
	_, err := client.Repositories.GetRepoBranches(ctx, "bad-token", "owner", "repo")
	if err == nil {
		t.Error("GetRepoBranches() expected error for 401 response")
	}
}

func TestRepositoryService_GetWebHooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hooks := []*thirdbean.ResultGetGitlabHook{
			{
				Id:        1,
				Url:       "https://example.com/hook1",
				CreatedAt: time.Now(),
			},
			{
				Id:        2,
				Url:       "https://example.com/hook2",
				CreatedAt: time.Now(),
			},
		}
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

	if hooks[0].Url != "https://example.com/hook1" {
		t.Errorf("GetWebHooks() first hook URL = %v, want https://example.com/hook1", hooks[0].Url)
	}
}

func TestRepositoryService_GetWebHooks_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client, _ := New(server.URL)
	ctx := context.Background()
	_, err := client.Repositories.GetWebHooks(ctx, "token", "owner", "repo", 1, 10)
	if err == nil {
		t.Error("GetWebHooks() expected error for 500 response")
	}
}

func TestConvertFunctions(t *testing.T) {
	t.Run("convertRepository", func(t *testing.T) {
		from := &thirdbean.ResultGitlabRepo{
			Id:                456,
			Name:              "test-repo",
			Path:              "test-repo",
			PathWithNamespace: "group/test-repo",
			WebUrl:            "https://gitlab.com/group/test-repo",
		}
		from.Owner.Username = "testuser"
		from.Namespace.Path = "group"
		repo := convertRepository(from)
		if repo.Id != "456" {
			t.Errorf("convertRepository() Id = %v, want 456", repo.Id)
		}
		if repo.Name != "test-repo" {
			t.Errorf("convertRepository() Name = %v, want test-repo", repo.Name)
		}
		if repo.Path != "test-repo" {
			t.Errorf("convertRepository() Path = %v, want test-repo", repo.Path)
		}
		if repo.Namespace != "group" {
			t.Errorf("convertRepository() Namespace = %v, want group", repo.Namespace)
		}
		if repo.FullName != "group/test-repo" {
			t.Errorf("convertRepository() FullName = %v, want group/test-repo", repo.FullName)
		}
		if repo.RepoType != "gitlab" {
			t.Errorf("convertRepository() RepoType = %v, want gitlab", repo.RepoType)
		}
	})

	t.Run("convertBranch", func(t *testing.T) {
		from := &thirdbean.ResultGitlabRepoBranch{Name: "main"}
		branch := convertBranch(from)
		if branch.Name != "main" {
			t.Errorf("convertBranch() Name = %v, want main", branch.Name)
		}
	})

	t.Run("convertHook", func(t *testing.T) {
		now := time.Now()
		from := &thirdbean.ResultGetGitlabHook{
			Id:        999,
			Url:       "https://example.com/hook",
			CreatedAt: now,
		}
		hook := convertHook(from)
		if hook.Id != 999 {
			t.Errorf("convertHook() Id = %d, want 999", hook.Id)
		}
		if hook.Url != "https://example.com/hook" {
			t.Errorf("convertHook() Url = %v, want https://example.com/hook", hook.Url)
		}
		if !hook.CreatedAt.Equal(now) {
			t.Errorf("convertHook() CreatedAt = %v, want %v", hook.CreatedAt, now)
		}
	})

	t.Run("convertRepositoryList", func(t *testing.T) {
		from := []*thirdbean.ResultGitlabRepo{
			{Id: 1, Name: "repo1", Path: "repo1"},
			{Id: 2, Name: "repo2", Path: "repo2"},
		}
		repos := convertRepositoryList(from)
		if len(repos) != 2 {
			t.Errorf("convertRepositoryList() returned %d repos, want 2", len(repos))
		}
		if repos[0].Name != "repo1" {
			t.Errorf("convertRepositoryList() first repo = %v, want repo1", repos[0].Name)
		}
	})

	t.Run("convertBranchList", func(t *testing.T) {
		from := []*thirdbean.ResultGitlabRepoBranch{
			{Name: "main"},
			{Name: "develop"},
		}
		branches := convertBranchList(from)
		if len(branches) != 2 {
			t.Errorf("convertBranchList() returned %d branches, want 2", len(branches))
		}
	})

	t.Run("convertHookList", func(t *testing.T) {
		from := []*thirdbean.ResultGetGitlabHook{
			{Id: 1, Url: "https://example.com/1"},
			{Id: 2, Url: "https://example.com/2"},
		}
		hooks := convertHookList(from)
		if len(hooks) != 2 {
			t.Errorf("convertHookList() returned %d hooks, want 2", len(hooks))
		}
		if hooks[0].Url != "https://example.com/1" {
			t.Errorf("convertHookList() first hook URL = %v", hooks[0].Url)
		}
	})

	t.Run("convertRepositoryList empty", func(t *testing.T) {
		repos := convertRepositoryList(nil)
		if len(repos) != 0 {
			t.Errorf("convertRepositoryList(nil) returned %d repos, want 0", len(repos))
		}
	})

	t.Run("convertBranchList empty", func(t *testing.T) {
		branches := convertBranchList(nil)
		if len(branches) != 0 {
			t.Errorf("convertBranchList(nil) returned %d branches, want 0", len(branches))
		}
	})

	t.Run("convertHookList empty", func(t *testing.T) {
		hooks := convertHookList(nil)
		if len(hooks) != 0 {
			t.Errorf("convertHookList(nil) returned %d hooks, want 0", len(hooks))
		}
	})
}
