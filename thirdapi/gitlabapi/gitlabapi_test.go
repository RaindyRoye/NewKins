package gitlabapi

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
	c, err := New("https://gitlab.com/api/v4")
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	if c.BaseURL == nil {
		t.Fatal("BaseURL is nil")
	}
	if c.BaseURL.String() != "https://gitlab.com/api/v4" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL.String(), "https://gitlab.com/api/v4")
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
	if !strings.Contains(err.Error(), "gitlab") {
		t.Errorf("error = %q, want it to contain 'gitlab'", err.Error())
	}
}

func TestNewDefault(t *testing.T) {
	c := NewDefault()
	if c == nil {
		t.Fatal("NewDefault returned nil")
	}
	if c.BaseURL.String() != BaseApiGitlab {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL.String(), BaseApiGitlab)
	}
}

// --- API path constant tests ---

func TestAPIPathConstants(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		contains string
	}{
		{"BaseApiGitlab", BaseApiGitlab, "gitlab.com/api/v4"},
		{"ApiGitlabCreateFile", ApiGitlabCreateFile, "/repos/%s/%s/contents/%s"},
		{"ApiGitlabGetRepos", ApiGitlabGetRepos, "/users/%s/projects"},
		{"ApiGitlabCreateHooks", ApiGitlabCreateHooks, "/projects/%s/hooks"},
		{"ApiGitlabGetHooks", ApiGitlabGetHooks, "/projects/%s/hooks"},
		{"ApiGitlabDeleteHooks", ApiGitlabDeleteHooks, "/projects/%s/hooks/%v"},
		{"ApiGitlabGetRepoBranches", ApiGitlabGetRepoBranches, "/projects/%v/repository/branches"},
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
			}{Username: "testuser"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want 'Bearer test-token'", auth)
		}
		w.Header().Set("X-Total-Pages", "5")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(repos)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	rp, err := c.Repositories.GetRepos(context.Background(), "test-token", "testuser", "all", "full_name", "desc", 1, 20)
	if err != nil {
		t.Fatalf("GetRepos error: %v", err)
	}
	if rp.TotalPages != 5 {
		t.Errorf("TotalPages = %d, want 5", rp.TotalPages)
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
	if rp.Ropes[0].RepoType != "gitlab" {
		t.Errorf("RepoType = %q, want %q", rp.Ropes[0].RepoType, "gitlab")
	}
}

func TestGetRepos_NoTotalPagesHeader(t *testing.T) {
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
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal"}`))
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
		_, _ = w.Write([]byte(`{invalid`))
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
	branches := []*thirdbean.ResultGitlabRepoBranch{
		{Name: "main"},
		{Name: "feature"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer tok" {
			t.Errorf("Authorization = %q, want 'Bearer tok'", auth)
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

func TestGetRepoBranches_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"404 Project Not Found"}`))
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
	hook := &thirdbean.ResultGetGitlabHook{
		Id:        10,
		Url:       "https://example.com/webhook",
		CreatedAt: time.Now().UTC(),
	}
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
	if h.Id != 10 {
		t.Errorf("hook.Id = %d, want 10", h.Id)
	}
}

func TestCreateWebHooks_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message":"Hook already exists"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Repositories.CreateWebHooks(context.Background(), "tok", "owner", "repo", "https://example.com/webhook", "secret")
	if err == nil {
		t.Fatal("expected error for 409 response")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusConflict {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusConflict)
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

func TestDeleteHooks_Forbidden(t *testing.T) {
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
	hooks := []*thirdbean.ResultGetGitlabHook{
		{Id: 1, Url: "https://example.com/hook1", CreatedAt: time.Now().UTC()},
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

func TestGetWebHooks_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"401 Unauthorized"}`))
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
	from := &thirdbean.ResultGitlabRepo{
		Id:                42,
		Name:              "myrepo",
		Path:              "myrepo",
		PathWithNamespace: "group/myrepo",
		WebUrl:            "https://gitlab.com/group/myrepo",
	}
	from.Owner.Username = "user"
	from.Namespace.Path = "group"

	r := convertRepository(from)
	if r.Id != "42" {
		t.Errorf("Id = %q, want %q", r.Id, "42")
	}
	if r.Owner != "user" {
		t.Errorf("Owner = %q, want %q", r.Owner, "user")
	}
	if r.Namespace != "group" {
		t.Errorf("Namespace = %q, want %q", r.Namespace, "group")
	}
	if r.FullName != "group/myrepo" {
		t.Errorf("FullName = %q, want %q", r.FullName, "group/myrepo")
	}
	if r.RepoType != "gitlab" {
		t.Errorf("RepoType = %q, want %q", r.RepoType, "gitlab")
	}
}

func TestConvertBranch(t *testing.T) {
	from := &thirdbean.ResultGitlabRepoBranch{Name: "main"}
	b := convertBranch(from)
	if b.Name != "main" {
		t.Errorf("Name = %q, want %q", b.Name, "main")
	}
}

func TestConvertHook(t *testing.T) {
	now := time.Now().UTC()
	from := &thirdbean.ResultGetGitlabHook{Id: 7, Url: "https://example.com/h", CreatedAt: now}
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
	result := convertBranchList([]*thirdbean.ResultGitlabRepoBranch{{Name: "a"}, {Name: "b"}})
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
}

func TestConvertHookList(t *testing.T) {
	result := convertHookList([]*thirdbean.ResultGetGitlabHook{{Id: 1}, {Id: 2}, {Id: 3}})
	if len(result) != 3 {
		t.Fatalf("len = %d, want 3", len(result))
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
