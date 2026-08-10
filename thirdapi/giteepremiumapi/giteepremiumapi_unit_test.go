package giteepremiumapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gokins/gokins/bean/thirdbean"
)

func TestGiteePremiumNew(t *testing.T) {
	client, err := New("https://gitee.com/api/v5")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client == nil {
		t.Fatal("New() returned nil client")
	}
	if client.BaseURL.String() != "https://gitee.com/api/v5/" {
		t.Errorf("BaseURL = %v, want %v", client.BaseURL.String(), "https://gitee.com/api/v5/")
	}
}

func TestGiteePremiumNewDefault(t *testing.T) {
	client := NewDefault()
	if client == nil {
		t.Fatal("NewDefault() returned nil client")
	}
	if client.BaseURL.String() != "https://gitee.com/api/v5/" {
		t.Errorf("BaseURL = %v, want %v", client.BaseURL.String(), "https://gitee.com/api/v5/")
	}
}

func TestGiteePremiumGetRepos(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %v, want GET", r.Method)
		}

		repos := []*thirdbean.ResultGiteePremiumRepo{
			{
				Id:       1,
				FullName: "user/test-repo",
				Name:     "test-repo",
				Path:     "test-repo",
				HtmlUrl:  "https://gitee.com/user/test-repo",
				Owner: struct {
					Id                int    `json:"id"`
					Login             string `json:"login"`
					Name              string `json:"name"`
					AvatarUrl         string `json:"avatar_url"`
					Url               string `json:"url"`
					HtmlUrl           string `json:"html_url"`
					FollowersUrl      string `json:"followers_url"`
					FollowingUrl      string `json:"following_url"`
					GistsUrl          string `json:"gists_url"`
					StarredUrl        string `json:"starred_url"`
					SubscriptionsUrl  string `json:"subscriptions_url"`
					OrganizationsUrl  string `json:"organizations_url"`
					ReposUrl          string `json:"repos_url"`
					EventsUrl         string `json:"events_url"`
					ReceivedEventsUrl string `json:"received_events_url"`
					Type              string `json:"type"`
				}{
					Login: "user",
				},
				Namespace: struct {
					Id      int    `json:"id"`
					Type    string `json:"type"`
					Name    string `json:"name"`
					Path    string `json:"path"`
					HtmlUrl string `json:"html_url"`
				}{
					Path: "user",
				},
			},
		}

		w.Header().Set("total_page", "1")
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

func TestGiteePremiumGetRepoBranches(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %v, want GET", r.Method)
		}

		branches := []*thirdbean.ResultGiteePremiumRepoBranch{
			{
				Name: "master",
			},
			{
				Name: "develop",
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
	if result[0].Name != "master" {
		t.Errorf("Branch name = %v, want master", result[0].Name)
	}
}

func TestGiteePremiumGetWebHooks(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %v, want GET", r.Method)
		}

		hooks := []*thirdbean.ResultGetGiteePremiumHook{
			{
				Id:        1,
				Url:       "https://example.com/hook",
				CreatedAt: time.Now(),
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

func TestGiteePremiumCreateWebHooks(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Method = %v, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %v, want application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		}

		hook := &thirdbean.ResultGetGiteePremiumHook{
			Id:        1,
			Url:       "https://example.com/hook",
			CreatedAt: time.Now(),
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

func TestGiteePremiumDeleteHooks(t *testing.T) {
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

func TestGiteePremiumConvertRepository(t *testing.T) {
	from := &thirdbean.ResultGiteePremiumRepo{
		Id:       1,
		FullName: "user/test-repo",
		Name:     "test-repo",
		Path:     "test-repo",
		HtmlUrl:  "https://gitee.com/user/test-repo",
		Owner: struct {
			Id                int    `json:"id"`
			Login             string `json:"login"`
			Name              string `json:"name"`
			AvatarUrl         string `json:"avatar_url"`
			Url               string `json:"url"`
			HtmlUrl           string `json:"html_url"`
			FollowersUrl      string `json:"followers_url"`
			FollowingUrl      string `json:"following_url"`
			GistsUrl          string `json:"gists_url"`
			StarredUrl        string `json:"starred_url"`
			SubscriptionsUrl  string `json:"subscriptions_url"`
			OrganizationsUrl  string `json:"organizations_url"`
			ReposUrl          string `json:"repos_url"`
			EventsUrl         string `json:"events_url"`
			ReceivedEventsUrl string `json:"received_events_url"`
			Type              string `json:"type"`
		}{
			Login: "user",
		},
		Namespace: struct {
			Id      int    `json:"id"`
			Type    string `json:"type"`
			Name    string `json:"name"`
			Path    string `json:"path"`
			HtmlUrl string `json:"html_url"`
		}{
			Path: "user",
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
	if result.FullName != "user/test-repo" {
		t.Errorf("FullName = %v, want user/test-repo", result.FullName)
	}
	if result.RepoType != "gitee" {
		t.Errorf("RepoType = %v, want gitee", result.RepoType)
	}
}

func TestGiteePremiumConvertBranch(t *testing.T) {
	from := &thirdbean.ResultGiteePremiumRepoBranch{
		Name: "master",
	}

	result := convertBranch(from)
	if result == nil {
		t.Fatal("convertBranch() returned nil")
	}
	if result.Name != "master" {
		t.Errorf("Name = %v, want master", result.Name)
	}
}

func TestGiteePremiumConvertHook(t *testing.T) {
	now := time.Now()
	from := &thirdbean.ResultGetGiteePremiumHook{
		Id:        1,
		Url:       "https://example.com/hook",
		CreatedAt: now,
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
	if !result.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", result.CreatedAt, now)
	}
}
