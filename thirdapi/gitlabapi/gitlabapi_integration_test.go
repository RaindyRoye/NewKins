package gitlabapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gokins/gokins/bean/thirdbean"
	"github.com/gokins/gokins/thirdapi"
)

// newTestServer creates an httptest.Server with the given handler and returns
// a configured thirdapi.Client pointed at the test server URL.
func newTestServer(t *testing.T, handler http.Handler) (*thirdapi.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	cli, err := New(srv.URL)
	if err != nil {
		srv.Close()
		t.Fatalf("New(%q): %v", srv.URL, err)
	}
	return cli, srv
}

func TestNew_ValidURL(t *testing.T) {
	cli, err := New("https://gitlab.com/api/v4")
	if err != nil {
		t.Fatalf("New with valid URL: %v", err)
	}
	if cli == nil {
		t.Fatal("expected non-nil client")
	}
	if cli.BaseURL.String() != "https://gitlab.com/api/v4" {
		t.Errorf("BaseURL = %q, want %q", cli.BaseURL.String(), "https://gitlab.com/api/v4")
	}
	if cli.Repositories == nil {
		t.Error("expected Repositories service to be set")
	}
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New("://bad-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNewDefault(t *testing.T) {
	cli := NewDefault()
	if cli == nil {
		t.Fatal("NewDefault returned nil")
	}
	if cli.BaseURL.String() != BaseApiGitlab {
		t.Errorf("BaseURL = %q, want %q", cli.BaseURL.String(), BaseApiGitlab)
	}
}

func TestGetRepos_Success(t *testing.T) {
	repos := []*thirdbean.ResultGitlabRepo{
		{
			Id:                1,
			Name:              "repo1",
			Path:              "repo1",
			PathWithNamespace: "group/repo1",
			WebUrl:            "https://gitlab.com/group/repo1",
			Owner: struct {
				Id        int    `json:"id"`
				Name      string `json:"name"`
				Username  string `json:"username"`
				State     string `json:"state"`
				AvatarUrl string `json:"avatar_url"`
				WebUrl    string `json:"web_url"`
			}{Username: "group"},
			Namespace: struct {
				Id        int    `json:"id"`
				Name      string `json:"name"`
				Path      string `json:"path"`
				Kind      string `json:"kind"`
				FullPath  string `json:"full_path"`
				ParentId  any    `json:"parent_id"`
				AvatarUrl string `json:"avatar_url"`
				WebUrl    string `json:"web_url"`
			}{Path: "group"},
		},
		{
			Id:                2,
			Name:              "repo2",
			Path:              "repo2",
			PathWithNamespace: "group/repo2",
			WebUrl:            "https://gitlab.com/group/repo2",
			Owner: struct {
				Id        int    `json:"id"`
				Name      string `json:"name"`
				Username  string `json:"username"`
				State     string `json:"state"`
				AvatarUrl string `json:"avatar_url"`
				WebUrl    string `json:"web_url"`
			}{Username: "group"},
			Namespace: struct {
				Id        int    `json:"id"`
				Name      string `json:"name"`
				Path      string `json:"path"`
				Kind      string `json:"kind"`
				FullPath  string `json:"full_path"`
				ParentId  any    `json:"parent_id"`
				AvatarUrl string `json:"avatar_url"`
				WebUrl    string `json:"web_url"`
			}{Path: "group"},
		},
	}

	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		w.Header().Set("X-Total-Pages", "5")
		_ = json.NewEncoder(w).Encode(repos)
	}))
	defer srv.Close()

	page, err := cli.Repositories.GetRepos(context.Background(), "test-token", "user", "all", "full_name", "desc", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos: %v", err)
	}
	if page.TotalPages != 5 {
		t.Errorf("TotalPages = %d, want 5", page.TotalPages)
	}
	if len(page.Ropes) != 2 {
		t.Fatalf("len(Ropes) = %d, want 2", len(page.Ropes))
	}
	if page.Ropes[0].Name != "repo1" {
		t.Errorf("Ropes[0].Name = %q, want %q", page.Ropes[0].Name, "repo1")
	}
	if page.Ropes[1].Owner != "group" {
		t.Errorf("Ropes[1].Owner = %q, want %q", page.Ropes[1].Owner, "group")
	}
	if page.Ropes[0].RepoType != "gitlab" {
		t.Errorf("Ropes[0].RepoType = %q, want %q", page.Ropes[0].RepoType, "gitlab")
	}
}

func TestGetRepos_NoPaginationHeader(t *testing.T) {
	repos := []*thirdbean.ResultGitlabRepo{
		{Id: 1, Name: "only", Path: "only", PathWithNamespace: "me/only"},
	}
	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No X-Total-Pages header → totalPages should default to 0
		_ = json.NewEncoder(w).Encode(repos)
	}))
	defer srv.Close()

	page, err := cli.Repositories.GetRepos(context.Background(), "tok", "me", "all", "full_name", "desc", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos: %v", err)
	}
	if page.TotalPages != 0 {
		t.Errorf("TotalPages = %d, want 0 (default)", page.TotalPages)
	}
}

func TestGetRepos_ServerError(t *testing.T) {
	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := cli.Repositories.GetRepos(context.Background(), "tok", "me", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error on 500")
	}
	var apiErr *thirdapi.APIError
	if errors.As(err, &apiErr) {
		if !apiErr.IsRetryable() {
			t.Error("500 should be retryable")
		}
	}
}

func TestGetRepos_CanceledContext(t *testing.T) {
	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_ = json.NewEncoder(w).Encode([]*thirdbean.ResultGitlabRepo{})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := cli.Repositories.GetRepos(ctx, "tok", "me", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error on canceled context")
	}
}

func TestCreateWebHooks_Success(t *testing.T) {
	hookResp := &thirdbean.ResultGetGitlabHook{
		Id:        99,
		Url:       "https://ci.example.com/hook",
		CreatedAt: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(hookResp)
	}))
	defer srv.Close()

	hook, err := cli.Repositories.CreateWebHooks(context.Background(), "tok", "owner", "repo", "https://ci.example.com/hook", "secret123")
	if err != nil {
		t.Fatalf("CreateWebHooks: %v", err)
	}
	if hook.Id != 99 {
		t.Errorf("hook.Id = %d, want 99", hook.Id)
	}
	if hook.Url != "https://ci.example.com/hook" {
		t.Errorf("hook.Url = %q, want %q", hook.Url, "https://ci.example.com/hook")
	}
}

func TestCreateWebHooks_ServerError(t *testing.T) {
	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":"invalid_url"}`))
	}))
	defer srv.Close()

	_, err := cli.Repositories.CreateWebHooks(context.Background(), "tok", "owner", "repo", "https://ci.example.com/hook", "secret")
	if err == nil {
		t.Fatal("expected error on 422")
	}
}

func TestDeleteHooks_Success(t *testing.T) {
	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	err := cli.Repositories.DeleteHooks(context.Background(), "tok", "owner", "repo", "42")
	if err != nil {
		t.Fatalf("DeleteHooks: %v", err)
	}
}

func TestDeleteHooks_NotFound(t *testing.T) {
	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"404 Hook Not Found"}`))
	}))
	defer srv.Close()

	err := cli.Repositories.DeleteHooks(context.Background(), "tok", "owner", "repo", "999")
	if err == nil {
		t.Fatal("expected error on 404")
	}
	var apiErr *thirdapi.APIError
	if errors.As(err, &apiErr) {
		if !apiErr.IsNotFound() {
			t.Error("404 should be IsNotFound()")
		}
	}
}

func TestGetWebHooks_Success(t *testing.T) {
	hooks := []*thirdbean.ResultGetGitlabHook{
		{
			Id:        1,
			Url:       "https://ci.example.com/hook1",
			CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Id:        2,
			Url:       "https://ci.example.com/hook2",
			CreatedAt: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(hooks)
	}))
	defer srv.Close()

	result, err := cli.Repositories.GetWebHooks(context.Background(), "tok", "owner", "repo", 1, 10)
	if err != nil {
		t.Fatalf("GetWebHooks: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result[0].Id != 1 {
		t.Errorf("result[0].Id = %d, want 1", result[0].Id)
	}
	if result[1].Url != "https://ci.example.com/hook2" {
		t.Errorf("result[1].Url = %q", result[1].Url)
	}
}

func TestGetWebHooks_AuthError(t *testing.T) {
	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"401 Unauthorized"}`))
	}))
	defer srv.Close()

	_, err := cli.Repositories.GetWebHooks(context.Background(), "bad-tok", "owner", "repo", 1, 10)
	if err == nil {
		t.Fatal("expected error on 401")
	}
	var apiErr *thirdapi.APIError
	if errors.As(err, &apiErr) {
		if !apiErr.IsAuthError() {
			t.Error("401 should be IsAuthError()")
		}
	}
}

func TestGetRepoBranches_Success(t *testing.T) {
	branches := []*thirdbean.ResultGitlabRepoBranch{
		{Name: "main"},
		{Name: "develop"},
		{Name: "feature/x"},
	}

	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(branches)
	}))
	defer srv.Close()

	result, err := cli.Repositories.GetRepoBranches(context.Background(), "tok", "owner", "repo")
	if err != nil {
		t.Fatalf("GetRepoBranches: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("len(result) = %d, want 3", len(result))
	}
	if result[0].Name != "main" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "main")
	}
	if result[2].Name != "feature/x" {
		t.Errorf("result[2].Name = %q, want %q", result[2].Name, "feature/x")
	}
}

func TestGetRepoBranches_Forbidden(t *testing.T) {
	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"403 Forbidden"}`))
	}))
	defer srv.Close()

	_, err := cli.Repositories.GetRepoBranches(context.Background(), "tok", "owner", "repo")
	if err == nil {
		t.Fatal("expected error on 403")
	}
	var apiErr *thirdapi.APIError
	if errors.As(err, &apiErr) {
		if !apiErr.IsAuthError() {
			t.Error("403 should be IsAuthError()")
		}
	}
}

func TestGetRepos_InvalidJSON(t *testing.T) {
	cli, srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	_, err := cli.Repositories.GetRepos(context.Background(), "tok", "me", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestConvertRepository(t *testing.T) {
	from := &thirdbean.ResultGitlabRepo{
		Id:                456,
		Name:              "myrepo",
		Path:              "myrepo",
		PathWithNamespace: "group/myrepo",
		WebUrl:            "https://gitlab.com/group/myrepo",
		Owner: struct {
			Id        int    `json:"id"`
			Name      string `json:"name"`
			Username  string `json:"username"`
			State     string `json:"state"`
			AvatarUrl string `json:"avatar_url"`
			WebUrl    string `json:"web_url"`
		}{Username: "group"},
		Namespace: struct {
			Id        int    `json:"id"`
			Name      string `json:"name"`
			Path      string `json:"path"`
			Kind      string `json:"kind"`
			FullPath  string `json:"full_path"`
			ParentId  any    `json:"parent_id"`
			AvatarUrl string `json:"avatar_url"`
			WebUrl    string `json:"web_url"`
		}{Path: "group"},
	}
	r := convertRepository(from)
	if r.Id != "456" {
		t.Errorf("Id = %q, want %q", r.Id, "456")
	}
	if r.Owner != "group" {
		t.Errorf("Owner = %q, want %q", r.Owner, "group")
	}
	if r.Name != "myrepo" {
		t.Errorf("Name = %q, want %q", r.Name, "myrepo")
	}
	if r.Path != "myrepo" {
		t.Errorf("Path = %q, want %q", r.Path, "myrepo")
	}
	if r.FullName != "group/myrepo" {
		t.Errorf("FullName = %q, want %q", r.FullName, "group/myrepo")
	}
	if r.HtmlURL != "https://gitlab.com/group/myrepo" {
		t.Errorf("HtmlURL = %q, want %q", r.HtmlURL, "https://gitlab.com/group/myrepo")
	}
	if r.RepoType != "gitlab" {
		t.Errorf("RepoType = %q, want %q", r.RepoType, "gitlab")
	}
}

func TestConvertBranchList(t *testing.T) {
	from := []*thirdbean.ResultGitlabRepoBranch{
		{Name: "main"},
		{Name: "dev"},
	}
	bs := convertBranchList(from)
	if len(bs) != 2 {
		t.Fatalf("len = %d, want 2", len(bs))
	}
	if bs[0].Name != "main" {
		t.Errorf("bs[0].Name = %q", bs[0].Name)
	}
}

func TestConvertHookList(t *testing.T) {
	from := []*thirdbean.ResultGetGitlabHook{
		{
			Id:        10,
			Url:       "https://example.com/hook",
			CreatedAt: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		},
	}
	hs := convertHookList(from)
	if len(hs) != 1 {
		t.Fatalf("len = %d, want 1", len(hs))
	}
	if hs[0].Id != 10 {
		t.Errorf("Id = %d, want 10", hs[0].Id)
	}
	if hs[0].Url != "https://example.com/hook" {
		t.Errorf("Url = %q", hs[0].Url)
	}
}

func TestConvertRepositoryList_Empty(t *testing.T) {
	rs := convertRepositoryList(nil)
	if len(rs) != 0 {
		t.Errorf("len = %d, want 0", len(rs))
	}
}
