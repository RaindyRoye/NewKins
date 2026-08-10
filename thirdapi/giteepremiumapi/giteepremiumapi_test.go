package giteepremiumapi

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
	c, err := New("https://gitee.com/api/v5")
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	if c.BaseURL == nil {
		t.Fatal("BaseURL is nil")
	}
	if c.BaseURL.String() != "https://gitee.com/api/v5/" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL.String(), "https://gitee.com/api/v5/")
	}
	if c.Repositories == nil {
		t.Error("Repositories service is nil")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New("://bad url")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
	if !strings.Contains(err.Error(), "gitee premium") {
		t.Errorf("error = %q, want it to contain 'gitee premium'", err.Error())
	}
}

func TestNewDefault(t *testing.T) {
	c := NewDefault()
	if c == nil {
		t.Fatal("NewDefault returned nil")
	}
	if !strings.Contains(c.BaseURL.String(), "gitee.com") {
		t.Errorf("BaseURL = %q, want it to contain 'gitee.com'", c.BaseURL.String())
	}
}

// --- API path constant tests ---

func TestAPIPathConstants(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"BaseApiGiteePremium", BaseApiGiteePremium, "https://gitee.com/api/v5"},
		{"ApiGiteePremiumCreateFile", ApiGiteePremiumCreateFile, "/repos/%s/%s/contents/%v"},
		{"ApiGiteePremiumGetHooks", ApiGiteePremiumGetHooks, "/repos/%s/%s/hooks?access_token=%s&page=%v&perPage=%v"},
		{"ApiGiteePremiumDeleteHooks", ApiGiteePremiumDeleteHooks, "/repos/%s/%s/hooks/%v?access_token=%s"},
		{"ApiGiteePremiumGetRepoBranches", ApiGiteePremiumGetRepoBranches, "/repos/%s/%s/branches?access_token=%s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.path != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.path, tt.want)
			}
		})
	}
}

// --- RepositoryService.GetRepos tests ---

func TestGetRepos_Success(t *testing.T) {
	repos := []*thirdbean.ResultGiteePremiumRepo{
		{
			Id:       100,
			Name:     "test-repo",
			Path:     "test-repo",
			FullName: "user/test-repo",
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
			}{Login: "giteeuser"},
			Namespace: struct {
				Id      int    `json:"id"`
				Type    string `json:"type"`
				Name    string `json:"name"`
				Path    string `json:"path"`
				HtmlUrl string `json:"html_url"`
			}{Path: "user"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("total_page", "3")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(repos)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	rp, err := c.Repositories.GetRepos(context.Background(), "tok", "giteeuser", "all", "full_name", "desc", 1, 20)
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
	if rp.Ropes[0].Owner != "giteeuser" {
		t.Errorf("Owner = %q, want %q", rp.Ropes[0].Owner, "giteeuser")
	}
	if rp.Ropes[0].RepoType != "gitee" {
		t.Errorf("RepoType = %q, want %q", rp.Ropes[0].RepoType, "gitee")
	}
}

func TestGetRepos_NoTotalPageHeader(t *testing.T) {
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
	if rp.TotalPages != 0 {
		t.Errorf("TotalPages = %d, want 0", rp.TotalPages)
	}
}

func TestGetRepos_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`service unavailable`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Repositories.GetRepos(context.Background(), "tok", "u", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error for 503 response")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T", err)
	}
	if !apiErr.IsRetryable() {
		t.Error("503 should be retryable")
	}
}

func TestGetRepos_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{{{bad`))
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
	branches := []*thirdbean.ResultGiteePremiumRepoBranch{
		{Name: "master"},
		{Name: "develop"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	if bs[0].Name != "master" {
		t.Errorf("branches[0].Name = %q, want %q", bs[0].Name, "master")
	}
}

func TestGetRepoBranches_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Repositories.GetRepoBranches(context.Background(), "tok", "owner", "repo")
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

// --- RepositoryService.CreateWebHooks tests ---

func TestCreateWebHooks_Success(t *testing.T) {
	hook := &thirdbean.ResultGetGiteePremiumHook{
		Id:        55,
		Url:       "https://example.com/webhook",
		CreatedAt: time.Now().UTC(),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want 'application/x-www-form-urlencoded'", ct)
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
	if h.Id != 55 {
		t.Errorf("hook.Id = %d, want 55", h.Id)
	}
	if h.Url != "https://example.com/webhook" {
		t.Errorf("hook.Url = %q, want %q", h.Url, "https://example.com/webhook")
	}
}

func TestCreateWebHooks_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad request"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Repositories.CreateWebHooks(context.Background(), "tok", "owner", "repo", "https://example.com/webhook", "secret")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
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

func TestDeleteHooks_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Repositories.DeleteHooks(context.Background(), "tok", "owner", "repo", "42")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Error("404 should be a not-found error")
	}
}

// --- RepositoryService.GetWebHooks tests ---

func TestGetWebHooks_Success(t *testing.T) {
	hooks := []*thirdbean.ResultGetGiteePremiumHook{
		{Id: 1, Url: "https://example.com/h1", CreatedAt: time.Now().UTC()},
		{Id: 2, Url: "https://example.com/h2", CreatedAt: time.Now().UTC()},
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
	if len(hs) != 2 {
		t.Fatalf("len(hooks) = %d, want 2", len(hs))
	}
}

func TestGetWebHooks_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Repositories.GetWebHooks(context.Background(), "bad", "owner", "repo", 1, 20)
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
	from := &thirdbean.ResultGiteePremiumRepo{
		Id:       42,
		Name:     "myrepo",
		Path:     "myrepo",
		FullName: "user/myrepo",
		HtmlUrl:  "https://gitee.com/user/myrepo",
	}
	from.Owner.Login = "user"
	from.Namespace.Path = "user"

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
	if r.RepoType != "gitee" {
		t.Errorf("RepoType = %q, want %q", r.RepoType, "gitee")
	}
}

func TestConvertBranch(t *testing.T) {
	from := &thirdbean.ResultGiteePremiumRepoBranch{Name: "master"}
	b := convertBranch(from)
	if b.Name != "master" {
		t.Errorf("Name = %q, want %q", b.Name, "master")
	}
}

func TestConvertHook(t *testing.T) {
	now := time.Now().UTC()
	from := &thirdbean.ResultGetGiteePremiumHook{Id: 7, Url: "https://example.com/h", CreatedAt: now}
	h := convertHook(from)
	if h.Id != 7 {
		t.Errorf("Id = %d, want 7", h.Id)
	}
	if h.Url != "https://example.com/h" {
		t.Errorf("Url = %q, want %q", h.Url, "https://example.com/h")
	}
}

func TestConvertRepositoryList_Empty(t *testing.T) {
	result := convertRepositoryList(nil)
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestConvertBranchList(t *testing.T) {
	result := convertBranchList([]*thirdbean.ResultGiteePremiumRepoBranch{{Name: "a"}, {Name: "b"}})
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
}

func TestConvertHookList(t *testing.T) {
	result := convertHookList([]*thirdbean.ResultGetGiteePremiumHook{{Id: 1}, {Id: 2}})
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
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
	cancel()
	_, err := c.Repositories.GetRepos(ctx, "tok", "u", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}
