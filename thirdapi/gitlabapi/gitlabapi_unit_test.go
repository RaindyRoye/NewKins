package gitlabapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/bean/thirdbean"
)

func TestGitlabNew(t *testing.T) {
	client, err := New("https://gitlab.com/api/v4")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client == nil {
		t.Fatal("New() returned nil client")
	}
	if client.BaseURL.String() != "https://gitlab.com/api/v4" {
		t.Errorf("BaseURL = %v, want %v", client.BaseURL.String(), "https://gitlab.com/api/v4")
	}
}

func TestGitlabNewDefault(t *testing.T) {
	client := NewDefault()
	if client == nil {
		t.Fatal("NewDefault() returned nil client")
	}
	if client.BaseURL.String() != "https://gitlab.com/api/v4" {
		t.Errorf("BaseURL = %v, want %v", client.BaseURL.String(), "https://gitlab.com/api/v4")
	}
}

func TestGitlabGetRepos(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %v, want GET", r.Method)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("Authorization header is empty")
		}

		repos := []*thirdbean.ResultGitlabRepo{
			{
				Id:                1,
				Name:              "test-repo",
				Path:              "test-repo",
				PathWithNamespace: "user/test-repo",
				WebUrl:            "https://gitlab.com/user/test-repo",
				Owner: struct {
					Id        int    `json:"id"`
					Name      string `json:"name"`
					Username  string `json:"username"`
					State     string `json:"state"`
					AvatarUrl string `json:"avatar_url"`
					WebUrl    string `json:"web_url"`
				}{
					Id:       1,
					Username: "user",
				},
				Namespace: struct {
					Id        int    `json:"id"`
					Name      string `json:"name"`
					Path      string `json:"path"`
					Kind      string `json:"kind"`
					FullPath  string `json:"full_path"`
					ParentId  any    `json:"parent_id"`
					AvatarUrl string `json:"avatar_url"`
					WebUrl    string `json:"web_url"`
				}{
					Path: "user",
				},
			},
		}

		w.Header().Set("X-Total-Pages", "1")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(repos)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := client.Repositories.GetRepos(context.Background(), "test-token", "user", "all", "created_at", "desc", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos() error = %v", err)
	}
	if result == nil {
		t.Fatal("GetRepos() returned nil result")
	}
	if len(result.Ropes) != 1 {
		t.Errorf("GetRepos() returned %d repos, want 1", len(result.Ropes))
	}
	if result.Ropes[0].Name != "test-repo" {
		t.Errorf("Repo name = %v, want test-repo", result.Ropes[0].Name)
	}
	if result.TotalPages != 1 {
		t.Errorf("TotalPages = %v, want 1", result.TotalPages)
	}
}

func TestGitlabGetRepoBranches(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %v, want GET", r.Method)
		}

		branches := []*thirdbean.ResultGitlabRepoBranch{
			{
				Name:      "main",
				Protected: true,
			},
			{
				Name:      "develop",
				Protected: false,
			},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(branches)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := client.Repositories.GetRepoBranches(context.Background(), "test-token", "user", "repo")
	if err != nil {
		t.Fatalf("GetRepoBranches() error = %v", err)
	}
	if len(result) != 2 {
		t.Errorf("GetRepoBranches() returned %d branches, want 2", len(result))
	}
	if result[0].Name != "main" {
		t.Errorf("Branch name = %v, want main", result[0].Name)
	}
}

func TestGitlabGetWebHooks(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %v, want GET", r.Method)
		}

		hooks := []*thirdbean.ResultGetGitlabHook{
			{
				Id:  1,
				Url: "https://example.com/hook",
			},
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hooks)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := client.Repositories.GetWebHooks(context.Background(), "test-token", "user", "repo", 1, 10)
	if err != nil {
		t.Fatalf("GetWebHooks() error = %v", err)
	}
	if len(result) != 1 {
		t.Errorf("GetWebHooks() returned %d hooks, want 1", len(result))
	}
	if result[0].Url != "https://example.com/hook" {
		t.Errorf("Hook URL = %v, want https://example.com/hook", result[0].Url)
	}
}

func TestGitlabCreateWebHooks(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %v, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %v, want application/json", r.Header.Get("Content-Type"))
		}

		hook := &thirdbean.ResultGetGitlabHook{
			Id:  1,
			Url: "https://example.com/hook",
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(hook)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := client.Repositories.CreateWebHooks(context.Background(), "test-token", "user", "repo", "https://example.com/hook", "secret")
	if err != nil {
		t.Fatalf("CreateWebHooks() error = %v", err)
	}
	if result == nil {
		t.Fatal("CreateWebHooks() returned nil result")
	}
	if result.Url != "https://example.com/hook" {
		t.Errorf("Hook URL = %v, want https://example.com/hook", result.Url)
	}
}

func TestGitlabDeleteHooks(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Method = %v, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client, err := New(server.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.Repositories.DeleteHooks(context.Background(), "test-token", "user", "repo", "1")
	if err != nil {
		t.Fatalf("DeleteHooks() error = %v", err)
	}
}

func TestGitlabConvertRepository(t *testing.T) {
	from := &thirdbean.ResultGitlabRepo{
		Id:                1,
		Name:              "test-repo",
		Path:              "test-repo",
		PathWithNamespace: "group/test-repo",
		WebUrl:            "https://gitlab.com/group/test-repo",
		Owner: struct {
			Id        int    `json:"id"`
			Name      string `json:"name"`
			Username  string `json:"username"`
			State     string `json:"state"`
			AvatarUrl string `json:"avatar_url"`
			WebUrl    string `json:"web_url"`
		}{
			Username: "user",
		},
		Namespace: struct {
			Id        int    `json:"id"`
			Name      string `json:"name"`
			Path      string `json:"path"`
			Kind      string `json:"kind"`
			FullPath  string `json:"full_path"`
			ParentId  any    `json:"parent_id"`
			AvatarUrl string `json:"avatar_url"`
			WebUrl    string `json:"web_url"`
		}{
			Path: "group",
		},
	}

	result := convertRepository(from)
	if result == nil {
		t.Fatal("convertRepository() returned nil")
	}
	if result.Id != "1" {
		t.Errorf("Id = %v, want 1", result.Id)
	}
	if result.Name != "test-repo" {
		t.Errorf("Name = %v, want test-repo", result.Name)
	}
	if result.Path != "test-repo" {
		t.Errorf("Path = %v, want test-repo", result.Path)
	}
	if result.FullName != "group/test-repo" {
		t.Errorf("FullName = %v, want group/test-repo", result.FullName)
	}
	if result.RepoType != "gitlab" {
		t.Errorf("RepoType = %v, want gitlab", result.RepoType)
	}
}

func TestGitlabConvertBranch(t *testing.T) {
	from := &thirdbean.ResultGitlabRepoBranch{
		Name:      "main",
		Protected: true,
	}

	result := convertBranch(from)
	if result == nil {
		t.Fatal("convertBranch() returned nil")
	}
	if result.Name != "main" {
		t.Errorf("Name = %v, want main", result.Name)
	}
}

func TestGitlabConvertHook(t *testing.T) {
	from := &thirdbean.ResultGetGitlabHook{
		Id:  1,
		Url: "https://example.com/hook",
	}

	result := convertHook(from)
	if result == nil {
		t.Fatal("convertHook() returned nil")
	}
	if result.Id != 1 {
		t.Errorf("Id = %v, want 1", result.Id)
	}
	if result.Url != "https://example.com/hook" {
		t.Errorf("Url = %v, want https://example.com/hook", result.Url)
	}
}
