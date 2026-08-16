package giteeapi

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

// newTestClient creates a giteeapi Client pointed at the given httptest server.
func newTestClient(t *testing.T, serverURL string) *thirdapi.Client {
	t.Helper()
	c, err := New(serverURL)
	if err != nil {
		t.Fatalf("New(%q): %v", serverURL, err)
	}
	return c
}

// --- GetRepos ---

func TestGetRepos_Success(t *testing.T) {
	repos := []*thirdbean.ResultGiteeRepo{
		{
			Id:       1,
			FullName: "owner/test-repo",
			Name:     "test-repo",
			Path:     "test-repo",
			HtmlUrl:  "https://gitee.com/owner/test-repo",
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
			}{Login: "owner"},
			Namespace: struct {
				Id      int    `json:"id"`
				Type    string `json:"type"`
				Name    string `json:"name"`
				Path    string `json:"path"`
				HtmlUrl string `json:"html_url"`
			}{Path: "owner"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("total_page", "5")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(repos)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	page, err := c.Repositories.GetRepos(context.Background(), "tok", "owner", "all", "pushed", "desc", 1, 20)
	if err != nil {
		t.Fatalf("GetRepos: %v", err)
	}
	if page.TotalPages != 5 {
		t.Errorf("TotalPages = %d, want 5", page.TotalPages)
	}
	if len(page.Ropes) != 1 {
		t.Fatalf("len(Ropes) = %d, want 1", len(page.Ropes))
	}
	rp := page.Ropes[0]
	if rp.Id != "1" {
		t.Errorf("Id = %q, want %q", rp.Id, "1")
	}
	if rp.Owner != "owner" {
		t.Errorf("Owner = %q, want %q", rp.Owner, "owner")
	}
	if rp.RepoType != "gitee" {
		t.Errorf("RepoType = %q, want %q", rp.RepoType, "gitee")
	}
}

func TestGetRepos_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Repositories.GetRepos(context.Background(), "tok", "u", "all", "pushed", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
	if !apiErr.IsRetryable() {
		t.Error("500 should be retryable")
	}
}

func TestGetRepos_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Repositories.GetRepos(context.Background(), "tok", "u", "all", "pushed", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestGetRepos_InvalidTotalPageHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("total_page", "not-a-number")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Repositories.GetRepos(context.Background(), "tok", "u", "all", "pushed", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected parse error for invalid total_page")
	}
}

func TestGetRepos_EmptyTotalPageHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	page, err := c.Repositories.GetRepos(context.Background(), "tok", "u", "all", "pushed", "desc", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos: %v", err)
	}
	if page.TotalPages != 0 {
		t.Errorf("TotalPages = %d, want 0", page.TotalPages)
	}
	if len(page.Ropes) != 0 {
		t.Errorf("len(Ropes) = %d, want 0", len(page.Ropes))
	}
}

func TestGetRepos_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	c := newTestClient(t, srv.URL)
	_, err := c.Repositories.GetRepos(ctx, "tok", "u", "all", "pushed", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

// --- DeleteHooks ---

func TestDeleteHooks_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.Repositories.DeleteHooks(context.Background(), "tok", "owner", "repo", "123")
	if err != nil {
		t.Fatalf("DeleteHooks: %v", err)
	}
}

func TestDeleteHooks_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.Repositories.DeleteHooks(context.Background(), "tok", "owner", "repo", "999")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() == true for 404")
	}
}

// --- CreateWebHooks ---

func TestCreateWebHooks_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want form-urlencoded", ct)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(thirdbean.ResultGetGiteeHook{
			Id:        42,
			Url:       "https://example.com/hook",
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	hook, err := c.Repositories.CreateWebHooks(context.Background(), "tok", "owner", "repo", "https://cb.example.com", "secret")
	if err != nil {
		t.Fatalf("CreateWebHooks: %v", err)
	}
	if hook.Id != 42 {
		t.Errorf("hook.Id = %d, want 42", hook.Id)
	}
	if hook.Url != "https://example.com/hook" {
		t.Errorf("hook.Url = %q", hook.Url)
	}
}

func TestCreateWebHooks_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Repositories.CreateWebHooks(context.Background(), "bad-tok", "owner", "repo", "https://cb.example.com", "")
	if err == nil {
		t.Fatal("expected error for 401")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsAuthError() {
		t.Error("401 should be an auth error")
	}
}

func TestCreateWebHooks_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{bad"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Repositories.CreateWebHooks(context.Background(), "tok", "owner", "repo", "https://cb.example.com", "")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

// --- GetRepoBranches ---

func TestGetRepoBranches_Success(t *testing.T) {
	branches := []*thirdbean.ResultGiteeRepoBranch{
		{Name: "main"},
		{Name: "develop"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(branches)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	bs, err := c.Repositories.GetRepoBranches(context.Background(), "tok", "owner", "repo")
	if err != nil {
		t.Fatalf("GetRepoBranches: %v", err)
	}
	if len(bs) != 2 {
		t.Fatalf("len = %d, want 2", len(bs))
	}
	if bs[0].Name != "main" {
		t.Errorf("bs[0].Name = %q, want %q", bs[0].Name, "main")
	}
	if bs[1].Name != "develop" {
		t.Errorf("bs[1].Name = %q, want %q", bs[1].Name, "develop")
	}
}

func TestGetRepoBranches_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Repositories.GetRepoBranches(context.Background(), "tok", "owner", "private-repo")
	if err == nil {
		t.Fatal("expected error for 403")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsAuthError() {
		t.Error("403 should be an auth error")
	}
}

func TestGetRepoBranches_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Repositories.GetRepoBranches(context.Background(), "tok", "owner", "repo")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

// --- GetWebHooks ---

func TestGetWebHooks_Success(t *testing.T) {
	hooks := []*thirdbean.ResultGetGiteeHook{
		{Id: 1, Url: "https://a.com/hook", CreatedAt: time.Now()},
		{Id: 2, Url: "https://b.com/hook", CreatedAt: time.Now()},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hooks)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	hs, err := c.Repositories.GetWebHooks(context.Background(), "tok", "owner", "repo", 1, 20)
	if err != nil {
		t.Fatalf("GetWebHooks: %v", err)
	}
	if len(hs) != 2 {
		t.Fatalf("len = %d, want 2", len(hs))
	}
	if hs[0].Id != 1 || hs[1].Id != 2 {
		t.Errorf("unexpected hook ids: %v", hs)
	}
}

func TestGetWebHooks_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Repositories.GetWebHooks(context.Background(), "tok", "owner", "repo", 1, 20)
	if err == nil {
		t.Fatal("expected error for 503")
	}
	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsRetryable() {
		t.Error("503 should be retryable")
	}
}

func TestGetWebHooks_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[{bad"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.Repositories.GetWebHooks(context.Background(), "tok", "owner", "repo", 1, 20)
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

// --- Converter unit tests ---

func TestConvertRepositoryList_Empty(t *testing.T) {
	out := convertRepositoryList(nil)
	if len(out) != 0 {
		t.Errorf("expected empty slice, got %d", len(out))
	}
}

func TestConvertRepository_Fields(t *testing.T) {
	from := &thirdbean.ResultGiteeRepo{
		Id:       99,
		FullName: "alice/hello",
		Name:     "hello",
		Path:     "hello",
		HtmlUrl:  "https://gitee.com/alice/hello",
	}
	from.Owner.Login = "alice"
	from.Namespace.Path = "alice"

	r := convertRepository(from)
	if r.Id != "99" {
		t.Errorf("Id = %q, want %q", r.Id, "99")
	}
	if r.Owner != "alice" {
		t.Errorf("Owner = %q", r.Owner)
	}
	if r.FullName != "alice/hello" {
		t.Errorf("FullName = %q", r.FullName)
	}
	if r.RepoType != "gitee" {
		t.Errorf("RepoType = %q", r.RepoType)
	}
}

func TestConvertBranchList_Empty(t *testing.T) {
	out := convertBranchList(nil)
	if len(out) != 0 {
		t.Errorf("expected empty, got %d", len(out))
	}
}

func TestConvertHookList_Empty(t *testing.T) {
	out := convertHookList(nil)
	if len(out) != 0 {
		t.Errorf("expected empty, got %d", len(out))
	}
}

func TestConvertHook_Fields(t *testing.T) {
	ts := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	from := &thirdbean.ResultGetGiteeHook{
		Id:        7,
		Url:       "https://hook.example.com",
		CreatedAt: ts,
	}
	h := convertHook(from)
	if h.Id != 7 {
		t.Errorf("Id = %d, want 7", h.Id)
	}
	if h.Url != "https://hook.example.com" {
		t.Errorf("Url = %q", h.Url)
	}
	if !h.CreatedAt.Equal(ts) {
		t.Errorf("CreatedAt = %v, want %v", h.CreatedAt, ts)
	}
}
