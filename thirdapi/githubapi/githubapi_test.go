package githubapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gokins/gokins/bean/thirdbean"
	"github.com/gokins/gokins/thirdapi"
)

// helper that builds a client pointed at a test server.
func newTestClient(t *testing.T, srv *httptest.Server) *thirdapi.Client {
	t.Helper()
	c, err := New(srv.URL)
	if err != nil {
		t.Fatalf("New(%q) error: %v", srv.URL, err)
	}
	c.HttpClient = srv.Client()
	return c
}

// --- factory tests ---

func TestNew(t *testing.T) {
	c, err := New("https://api.github.com")
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	if c.BaseURL == nil {
		t.Fatal("BaseURL is nil")
	}
	if c.BaseURL.String() != "https://api.github.com" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL.String(), "https://api.github.com")
	}
	if c.Repositories == nil {
		t.Error("Repositories service is nil")
	}
	if c.HttpClient == nil {
		t.Error("HttpClient is nil")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New("://bad url")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
	if !strings.Contains(err.Error(), "github") {
		t.Errorf("error = %q, want it to contain 'github'", err.Error())
	}
}

func TestNewDefault(t *testing.T) {
	c := NewDefault()
	if c == nil {
		t.Fatal("NewDefault returned nil")
	}
	if c.BaseURL.String() != BaseApiGithub {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL.String(), BaseApiGithub)
	}
}

// --- API path constant tests ---

func TestAPIPathConstants(t *testing.T) {
	// Verify the constants contain expected substrings
	tests := []struct {
		name     string
		path     string
		contains string
	}{
		{"BaseApiGithub", BaseApiGithub, "api.github.com"},
		{"ApiGithubCreateFile", ApiGithubCreateFile, "/repos/%s/%s/contents/%s"},
		{"ApiGithubGetRepos", ApiGithubGetRepos, "/user/repos"},
		{"ApiGithubCreateHooks", ApiGithubCreateHooks, "/repos/%s/%s/hooks"},
		{"ApiGithubGetHooks", ApiGithubGetHooks, "/repos/%s/%s/hooks"},
		{"ApiGithubDeleteHooks", ApiGithubDeleteHooks, "/repos/%s/%s/hooks/%v"},
		{"ApiGithubGetRepoBranches", ApiGithubGetRepoBranches, "/repos/%s/%s/branches"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.path, tt.contains) {
				t.Errorf("%s = %q, want it to contain %q", tt.name, tt.path, tt.contains)
			}
		})
	}
}

// --- RepositoryService.GetRepos tests ---

func TestGetRepos_Success(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	repos := []*thirdbean.ResultGithubRepo{
		{
			Id:       1,
			Name:     "test-repo",
			FullName: "owner/test-repo",
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
			}{Login: "testuser"},
			HtmlUrl:   "https://github.com/owner/test-repo",
			CreatedAt: parseTime(t, now),
			UpdatedAt: parseTime(t, now),
			PushedAt:  parseTime(t, now),
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "token test-token" {
			t.Errorf("Authorization = %q, want 'token test-token'", auth)
		}
		w.Header().Set("Link", `<https://api.github.com/user/repos?page=3>; rel="last"`)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(repos)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	rp, err := c.Repositories.GetRepos(context.Background(), "test-token", "testuser", "all", "full_name", "desc", 1, 20)
	if err != nil {
		t.Fatalf("GetRepos error: %v", err)
	}
	if rp.TotalPages != 3 {
		t.Errorf("TotalPages = %d, want 3", rp.TotalPages)
	}
	if len(rp.Ropes) != 1 {
		t.Fatalf("len(Ropes) = %d, want 1", len(rp.Ropes))
	}
	if rp.Ropes[0].Name != "test-repo" {
		t.Errorf("Name = %q, want %q", rp.Ropes[0].Name, "test-repo")
	}
	if rp.Ropes[0].Owner != "testuser" {
		t.Errorf("Owner = %q, want %q", rp.Ropes[0].Owner, "testuser")
	}
	if rp.Ropes[0].RepoType != "github" {
		t.Errorf("RepoType = %q, want %q", rp.Ropes[0].RepoType, "github")
	}
}

func TestGetRepos_NoPaginationHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	rp, err := c.Repositories.GetRepos(context.Background(), "tok", "u", "all", "full_name", "desc", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos error: %v", err)
	}
	if rp.TotalPages != 1 {
		t.Errorf("TotalPages = %d, want 1 (default)", rp.TotalPages)
	}
}

func TestGetRepos_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Repositories.GetRepos(context.Background(), "tok", "u", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T: %v", err, err)
	}
	if !apiErr.IsRetryable() {
		t.Error("500 should be retryable")
	}
}

func TestGetRepos_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Repositories.GetRepos(context.Background(), "tok", "u", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error = %q, want it to contain 'unmarshal'", err.Error())
	}
}

// --- RepositoryService.GetRepoBranches tests ---

func TestGetRepoBranches_Success(t *testing.T) {
	branches := []*thirdbean.ResultGithubRepoBranch{
		{Name: "main"},
		{Name: "develop"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "token tok" {
			t.Errorf("Authorization = %q, want 'token tok'", auth)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(branches)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	bs, err := c.Repositories.GetRepoBranches(context.Background(), "tok", "owner", "repo")
	if err != nil {
		t.Fatalf("GetRepoBranches error: %v", err)
	}
	if len(bs) != 2 {
		t.Fatalf("len(branches) = %d, want 2", len(bs))
	}
	if bs[0].Name != "main" {
		t.Errorf("branches[0].Name = %q, want %q", bs[0].Name, "main")
	}
}

func TestGetRepoBranches_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Repositories.GetRepoBranches(context.Background(), "tok", "owner", "repo")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to be true for 404")
	}
}

// --- RepositoryService.CreateWebHooks tests ---

func TestCreateWebHooks_Success(t *testing.T) {
	hook := &thirdbean.ResultGetGithubHook{
		Id:        42,
		Name:      "web",
		Active:    true,
		CreatedAt: time.Now().UTC(),
	}
	hook.Config.Url = "https://example.com/webhook"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want 'application/json'", ct)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(hook)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	h, err := c.Repositories.CreateWebHooks(context.Background(), "tok", "owner", "repo", "https://example.com/webhook", "secret")
	if err != nil {
		t.Fatalf("CreateWebHooks error: %v", err)
	}
	if h.Id != 42 {
		t.Errorf("hook.Id = %d, want 42", h.Id)
	}
	if h.Url != "https://example.com/webhook" {
		t.Errorf("hook.Url = %q, want %q", h.Url, "https://example.com/webhook")
	}
}

func TestCreateWebHooks_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Validation Failed"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Repositories.CreateWebHooks(context.Background(), "tok", "owner", "repo", "https://example.com/webhook", "secret")
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusUnprocessableEntity)
	}
}

// --- RepositoryService.DeleteHooks tests ---

func TestDeleteHooks_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Method = %q, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Repositories.DeleteHooks(context.Background(), "tok", "owner", "repo", "42")
	if err != nil {
		t.Fatalf("DeleteHooks error: %v", err)
	}
}

func TestDeleteHooks_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Repositories.DeleteHooks(context.Background(), "tok", "owner", "repo", "42")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T", err)
	}
	if !apiErr.IsAuthError() {
		t.Error("403 should be an auth error")
	}
}

// --- RepositoryService.GetWebHooks tests ---

func TestGetWebHooks_Success(t *testing.T) {
	now := time.Now().UTC()
	hooks := []*thirdbean.ResultGetGithubHook{
		{
			Id:        1,
			Name:      "web",
			Active:    true,
			CreatedAt: now,
			Config: struct {
				ContentType string `json:"content_type"`
				InsecureSsl string `json:"insecure_ssl"`
				Url         string `json:"url"`
			}{Url: "https://example.com/hook"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hooks)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	hs, err := c.Repositories.GetWebHooks(context.Background(), "tok", "owner", "repo", 1, 20)
	if err != nil {
		t.Fatalf("GetWebHooks error: %v", err)
	}
	if len(hs) != 1 {
		t.Fatalf("len(hooks) = %d, want 1", len(hs))
	}
	if hs[0].Id != 1 {
		t.Errorf("hook.Id = %d, want 1", hs[0].Id)
	}
}

func TestGetWebHooks_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Repositories.GetWebHooks(context.Background(), "bad-tok", "owner", "repo", 1, 20)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T", err)
	}
	if !apiErr.IsAuthError() {
		t.Error("401 should be an auth error")
	}
}

// --- converter tests ---

func TestConvertRepository(t *testing.T) {
	from := &thirdbean.ResultGithubRepo{
		Id:       42,
		Name:     "myrepo",
		FullName: "user/myrepo",
		HtmlUrl:  "https://github.com/user/myrepo",
	}
	from.Owner.Login = "user"

	r := convertRepository(from)
	if r.Id != "42" {
		t.Errorf("Id = %q, want %q", r.Id, "42")
	}
	if r.Owner != "user" {
		t.Errorf("Owner = %q, want %q", r.Owner, "user")
	}
	if r.Name != "myrepo" {
		t.Errorf("Name = %q, want %q", r.Name, "myrepo")
	}
	if r.RepoType != "github" {
		t.Errorf("RepoType = %q, want %q", r.RepoType, "github")
	}
}

func TestConvertBranch(t *testing.T) {
	from := &thirdbean.ResultGithubRepoBranch{Name: "main"}
	b := convertBranch(from)
	if b.Name != "main" {
		t.Errorf("Name = %q, want %q", b.Name, "main")
	}
}

func TestConvertHook(t *testing.T) {
	now := time.Now().UTC()
	from := &thirdbean.ResultGetGithubHook{
		Id:        99,
		CreatedAt: now,
	}
	from.Config.Url = "https://example.com/hook"

	h := convertHook(from)
	if h.Id != 99 {
		t.Errorf("Id = %d, want 99", h.Id)
	}
	if h.Url != "https://example.com/hook" {
		t.Errorf("Url = %q, want %q", h.Url, "https://example.com/hook")
	}
}

func TestConvertRepositoryList(t *testing.T) {
	ls := []*thirdbean.ResultGithubRepo{
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b"},
	}
	result := convertRepositoryList(ls)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
}

func TestConvertBranchList(t *testing.T) {
	ls := []*thirdbean.ResultGithubRepoBranch{
		{Name: "main"},
		{Name: "dev"},
	}
	result := convertBranchList(ls)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
}

func TestConvertHookList(t *testing.T) {
	ls := []*thirdbean.ResultGetGithubHook{
		{Id: 1},
		{Id: 2},
	}
	result := convertHookList(ls)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
}

func TestConvertRepositoryList_Empty(t *testing.T) {
	result := convertRepositoryList(nil)
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

// --- context cancellation test ---

func TestGetRepos_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := c.Repositories.GetRepos(ctx, "tok", "u", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

// helper
func parseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parseTime error: %v", err)
	}
	return parsed
}
