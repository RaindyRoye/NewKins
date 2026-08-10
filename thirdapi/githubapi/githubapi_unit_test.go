package githubapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gokins/gokins/bean/thirdbean"
	"github.com/gokins/gokins/thirdapi"
)

// ---------- helpers ----------

func setupGitHubTestServer(t *testing.T, handler http.Handler) (*thirdapi.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	client, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New(%q) failed: %v", srv.URL, err)
	}
	return client, srv
}

// ---------- New / NewDefault ----------

func TestNew_ValidURL(t *testing.T) {
	c, err := New("https://api.github.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.BaseURL.String() != "https://api.github.com" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL.String(), "https://api.github.com")
	}
	if c.Repositories == nil {
		t.Error("expected non-nil Repositories service")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New("://bad")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNewDefault(t *testing.T) {
	c := NewDefault()
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.BaseURL.String() != BaseApiGithub {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL.String(), BaseApiGithub)
	}
}

// ---------- GetRepos ----------

func TestGetRepos_Success(t *testing.T) {
	repos := []*thirdbean.ResultGithubRepo{
		{
			Id:       1,
			Name:     "repo1",
			FullName: "user/repo1",
			Owner:    struct {
				Login             string `json:"login"`
				Id                int    `json:"id"`
				NodeId            string `json:"node_id"`
				AvatarUrl         string `json:"avatar_url"`
				GravatarId        string `json:"gravatar_id"`
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
				SiteAdmin         bool   `json:"site_admin"`
			}{Login: "user"},
			HtmlUrl: "https://github.com/user/repo1",
		},
		{
			Id:       2,
			Name:     "repo2",
			FullName: "user/repo2",
			Owner: struct {
				Login             string `json:"login"`
				Id                int    `json:"id"`
				NodeId            string `json:"node_id"`
				AvatarUrl         string `json:"avatar_url"`
				GravatarId        string `json:"gravatar_id"`
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
				SiteAdmin         bool   `json:"site_admin"`
			}{Login: "user"},
			HtmlUrl: "https://github.com/user/repo2",
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token testtoken" {
			t.Errorf("Authorization header = %q, want %q", r.Header.Get("Authorization"), "token testtoken")
		}
		w.Header().Set("Link", `<https://api.github.com/user/repos?page=5>; rel="last", <https://api.github.com/user/repos?page=2>; rel="next"`)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(repos)
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	page, err := client.Repositories.GetRepos(context.Background(), "testtoken", "user", "all", "full_name", "desc", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos failed: %v", err)
	}
	if page.TotalPages != 5 {
		t.Errorf("TotalPages = %d, want 5", page.TotalPages)
	}
	if len(page.Ropes) != 2 {
		t.Fatalf("got %d repos, want 2", len(page.Ropes))
	}
	if page.Ropes[0].Name != "repo1" {
		t.Errorf("repo[0].Name = %q, want %q", page.Ropes[0].Name, "repo1")
	}
	if page.Ropes[0].RepoType != "github" {
		t.Errorf("repo[0].RepoType = %q, want %q", page.Ropes[0].RepoType, "github")
	}
	if page.Ropes[1].FullName != "user/repo2" {
		t.Errorf("repo[1].FullName = %q, want %q", page.Ropes[1].FullName, "user/repo2")
	}
}

func TestGetRepos_NoLinkHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]*thirdbean.ResultGithubRepo{})
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	page, err := client.Repositories.GetRepos(context.Background(), "tok", "u", "all", "full_name", "desc", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos failed: %v", err)
	}
	if page.TotalPages != 1 {
		t.Errorf("TotalPages = %d, want 1 (default when no Link header)", page.TotalPages)
	}
}

func TestGetRepos_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	_, err := client.Repositories.GetRepos(context.Background(), "tok", "u", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestGetRepos_InvalidJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	_, err := client.Repositories.GetRepos(context.Background(), "tok", "u", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestGetRepos_ContextCanceled(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Repositories.GetRepos(ctx, "tok", "u", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

// ---------- DeleteHooks ----------

func TestDeleteHooks_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	err := client.Repositories.DeleteHooks(context.Background(), "tok", "owner", "repo", "123")
	if err != nil {
		t.Fatalf("DeleteHooks failed: %v", err)
	}
}

func TestDeleteHooks_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	err := client.Repositories.DeleteHooks(context.Background(), "tok", "owner", "repo", "999")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

// ---------- CreateWebHooks ----------

func TestCreateWebHooks_Success(t *testing.T) {
	respHook := &thirdbean.ResultGetGithubHook{
		Id:   42,
		Url:  "https://example.com/hook",
		Config: struct {
			ContentType string `json:"content_type"`
			InsecureSsl string `json:"insecure_ssl"`
			Url         string `json:"url"`
		}{Url: "https://example.com/callback"},
		CreatedAt: time.Now(),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(respHook)
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	hook, err := client.Repositories.CreateWebHooks(context.Background(), "tok", "owner", "repo", "https://example.com/callback", "secret123")
	if err != nil {
		t.Fatalf("CreateWebHooks failed: %v", err)
	}
	if hook.Id != 42 {
		t.Errorf("hook.Id = %d, want 42", hook.Id)
	}
	if hook.Url != "https://example.com/callback" {
		t.Errorf("hook.Url = %q, want %q", hook.Url, "https://example.com/callback")
	}
}

func TestCreateWebHooks_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"validation failed"}`))
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	_, err := client.Repositories.CreateWebHooks(context.Background(), "tok", "owner", "repo", "https://example.com", "secret")
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
}

func TestCreateWebHooks_InvalidJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("not json"))
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	_, err := client.Repositories.CreateWebHooks(context.Background(), "tok", "owner", "repo", "https://example.com", "secret")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

// ---------- GetRepoBranches ----------

func TestGetRepoBranches_Success(t *testing.T) {
	branches := []*thirdbean.ResultGithubRepoBranch{
		{Name: "main"},
		{Name: "develop"},
		{Name: "feature/x"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(branches)
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	result, err := client.Repositories.GetRepoBranches(context.Background(), "tok", "owner", "repo")
	if err != nil {
		t.Fatalf("GetRepoBranches failed: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("got %d branches, want 3", len(result))
	}
	if result[0].Name != "main" {
		t.Errorf("branch[0].Name = %q, want %q", result[0].Name, "main")
	}
	if result[2].Name != "feature/x" {
		t.Errorf("branch[2].Name = %q, want %q", result[2].Name, "feature/x")
	}
}

func TestGetRepoBranches_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	_, err := client.Repositories.GetRepoBranches(context.Background(), "tok", "owner", "repo")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestGetRepoBranches_InvalidJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{bad json"))
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	_, err := client.Repositories.GetRepoBranches(context.Background(), "tok", "owner", "repo")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ---------- GetWebHooks ----------

func TestGetWebHooks_Success(t *testing.T) {
	hooks := []*thirdbean.ResultGetGithubHook{
		{
			Id:        1,
			Url:       "https://example.com/hook1",
			CreatedAt: time.Now(),
			Config: struct {
				ContentType string `json:"content_type"`
				InsecureSsl string `json:"insecure_ssl"`
				Url         string `json:"url"`
			}{Url: "https://example.com/callback1"},
		},
		{
			Id:        2,
			Url:       "https://example.com/hook2",
			CreatedAt: time.Now(),
			Config: struct {
				ContentType string `json:"content_type"`
				InsecureSsl string `json:"insecure_ssl"`
				Url         string `json:"url"`
			}{Url: "https://example.com/callback2"},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hooks)
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	result, err := client.Repositories.GetWebHooks(context.Background(), "tok", "owner", "repo", 1, 10)
	if err != nil {
		t.Fatalf("GetWebHooks failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d hooks, want 2", len(result))
	}
	if result[0].Id != 1 {
		t.Errorf("hook[0].Id = %d, want 1", result[0].Id)
	}
	if result[1].Url != "https://example.com/callback2" {
		t.Errorf("hook[1].Url = %q, want %q", result[1].Url, "https://example.com/callback2")
	}
}

func TestGetWebHooks_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	_, err := client.Repositories.GetWebHooks(context.Background(), "badtok", "owner", "repo", 1, 10)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestGetWebHooks_InvalidJSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[{invalid"))
	})

	client, srv := setupGitHubTestServer(t, handler)
	defer srv.Close()

	_, err := client.Repositories.GetWebHooks(context.Background(), "tok", "owner", "repo", 1, 10)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ---------- converter unit tests ----------

func TestConvertRepository(t *testing.T) {
	from := &thirdbean.ResultGithubRepo{
		Id:       42,
		Name:     "myrepo",
		FullName: "org/myrepo",
		HtmlUrl:  "https://github.com/org/myrepo",
	}
	from.Owner.Login = "org"

	r := convertRepository(from)
	if r.Id != "42" {
		t.Errorf("Id = %q, want %q", r.Id, "42")
	}
	if r.Owner != "org" {
		t.Errorf("Owner = %q, want %q", r.Owner, "org")
	}
	if r.Name != "myrepo" {
		t.Errorf("Name = %q, want %q", r.Name, "myrepo")
	}
	if r.Path != "myrepo" {
		t.Errorf("Path = %q, want %q", r.Path, "myrepo")
	}
	if r.Namespace != "org" {
		t.Errorf("Namespace = %q, want %q", r.Namespace, "org")
	}
	if r.RepoType != "github" {
		t.Errorf("RepoType = %q, want %q", r.RepoType, "github")
	}
}

func TestConvertRepositoryList(t *testing.T) {
	from := []*thirdbean.ResultGithubRepo{
		{Id: 1, Name: "a", FullName: "u/a", HtmlUrl: "https://github.com/u/a"},
		{Id: 2, Name: "b", FullName: "u/b", HtmlUrl: "https://github.com/u/b"},
	}
	from[0].Owner.Login = "u"
	from[1].Owner.Login = "u"

	result := convertRepositoryList(from)
	if len(result) != 2 {
		t.Fatalf("got %d repos, want 2", len(result))
	}
	if result[0].Name != "a" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "a")
	}
}

func TestConvertBranchList_Empty(t *testing.T) {
	result := convertBranchList(nil)
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d", len(result))
	}
}

func TestConvertHookList_Empty(t *testing.T) {
	result := convertHookList(nil)
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d", len(result))
	}
}

func TestConvertHook(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	from := &thirdbean.ResultGetGithubHook{
		Id:        99,
		CreatedAt: now,
		Config: struct {
			ContentType string `json:"content_type"`
			InsecureSsl string `json:"insecure_ssl"`
			Url         string `json:"url"`
		}{Url: "https://example.com/hook"},
	}

	h := convertHook(from)
	if h.Id != 99 {
		t.Errorf("Id = %d, want 99", h.Id)
	}
	if h.Url != "https://example.com/hook" {
		t.Errorf("Url = %q, want %q", h.Url, "https://example.com/hook")
	}
	if !h.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", h.CreatedAt, now)
	}
}

// ---------- API constants ----------

func TestAPIConstants(t *testing.T) {
	if BaseApiGithub == "" {
		t.Error("BaseApiGithub must not be empty")
	}
	if ApiGithubGetRepos == "" {
		t.Error("ApiGithubGetRepos must not be empty")
	}
	if ApiGithubCreateHooks == "" {
		t.Error("ApiGithubCreateHooks must not be empty")
	}
	if ApiGithubDeleteHooks == "" {
		t.Error("ApiGithubDeleteHooks must not be empty")
	}
	if ApiGithubGetRepoBranches == "" {
		t.Error("ApiGithubGetRepoBranches must not be empty")
	}
	if ApiGithubGetHooks == "" {
		t.Error("ApiGithubGetHooks must not be empty")
	}
}
