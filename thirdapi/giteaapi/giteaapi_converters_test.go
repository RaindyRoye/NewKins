package giteaapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gokins/gokins/bean/thirdbean"
	"github.com/gokins/gokins/thirdapi"
)

// TestConvertRepositoryList tests the convertRepositoryList function.
func TestConvertRepositoryList(t *testing.T) {
	tests := []struct {
		name    string
		input   []*thirdbean.ResultGiteaRepo
		wantLen int
	}{
		{
			name:    "empty list",
			input:   []*thirdbean.ResultGiteaRepo{},
			wantLen: 0,
		},
		{
			name:    "nil list",
			input:   nil,
			wantLen: 0,
		},
		{
			name: "single repo",
			input: []*thirdbean.ResultGiteaRepo{
				newTestRepo(1, "owner", "test-repo", "owner/test-repo"),
			},
			wantLen: 1,
		},
		{
			name: "multiple repos",
			input: []*thirdbean.ResultGiteaRepo{
				newTestRepo(1, "owner", "repo1", "owner/repo1"),
				newTestRepo(2, "owner", "repo2", "owner/repo2"),
				newTestRepo(3, "owner", "repo3", "owner/repo3"),
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertRepositoryList(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("convertRepositoryList() length = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

// TestConvertRepository tests the convertRepository function.
func TestConvertRepository(t *testing.T) {
	input := newTestRepo(42, "myorg", "my-project", "myorg/my-project")

	result := convertRepository(input)

	if result.Id != "42" {
		t.Errorf("Id = %q, want %q", result.Id, "42")
	}
	if result.Owner != "myorg" {
		t.Errorf("Owner = %q, want %q", result.Owner, "myorg")
	}
	if result.Name != "my-project" {
		t.Errorf("Name = %q, want %q", result.Name, "my-project")
	}
	if result.Path != "my-project" {
		t.Errorf("Path = %q, want %q", result.Path, "my-project")
	}
	if result.Namespace != "myorg" {
		t.Errorf("Namespace = %q, want %q", result.Namespace, "myorg")
	}
	if result.FullName != "myorg/my-project" {
		t.Errorf("FullName = %q, want %q", result.FullName, "myorg/my-project")
	}
	if result.HtmlURL != "https://gitea.com/myorg/my-project" {
		t.Errorf("HtmlURL = %q, want %q", result.HtmlURL, "https://gitea.com/myorg/my-project")
	}
	if result.RepoType != "gitea" {
		t.Errorf("RepoType = %q, want %q", result.RepoType, "gitea")
	}
}

// TestConvertBranchList tests the convertBranchList function.
func TestConvertBranchList(t *testing.T) {
	tests := []struct {
		name    string
		input   []*thirdbean.ResultGiteaRepoBranch
		wantLen int
	}{
		{"empty", []*thirdbean.ResultGiteaRepoBranch{}, 0},
		{"nil", nil, 0},
		{
			"multiple branches",
			[]*thirdbean.ResultGiteaRepoBranch{
				{Name: "main"},
				{Name: "develop"},
				{Name: "feature/x"},
			},
			3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertBranchList(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("convertBranchList() length = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

// TestConvertBranch tests the convertBranch function.
func TestConvertBranch(t *testing.T) {
	input := &thirdbean.ResultGiteaRepoBranch{Name: "release/v1.0"}
	result := convertBranch(input)
	if result.Name != "release/v1.0" {
		t.Errorf("Name = %q, want %q", result.Name, "release/v1.0")
	}
}

// TestConvertHookList tests the convertHookList function.
func TestConvertHookList(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		input   []*thirdbean.ResultGetGiteaHook
		wantLen int
	}{
		{"empty", []*thirdbean.ResultGetGiteaHook{}, 0},
		{"nil", nil, 0},
		{
			"multiple hooks",
			[]*thirdbean.ResultGetGiteaHook{
				newTestHook(1, "https://a.com", now),
				newTestHook(2, "https://b.com", now),
			},
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertHookList(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("convertHookList() length = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

// TestConvertHook tests the convertHook function.
func TestConvertHook(t *testing.T) {
	createdAt := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	input := newTestHook(99, "https://ci.example.com/hook", createdAt)
	result := convertHook(input)
	if result.Id != 99 {
		t.Errorf("Id = %d, want %d", result.Id, 99)
	}
	if result.Url != "https://ci.example.com/hook" {
		t.Errorf("Url = %q, want %q", result.Url, "https://ci.example.com/hook")
	}
	if !result.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt = %v, want %v", result.CreatedAt, createdAt)
	}
}

// TestNewDefault tests the NewDefault function.
func TestNewDefault(t *testing.T) {
	client := NewDefault()
	if client == nil {
		t.Fatal("NewDefault() returned nil")
	}
	if client.BaseURL == nil {
		t.Error("NewDefault() BaseURL is nil")
	}
	if client.BaseURL.String() != BaseApiGitea {
		t.Errorf("BaseURL = %q, want %q", client.BaseURL.String(), BaseApiGitea)
	}
	if client.HttpClient == nil {
		t.Error("NewDefault() HttpClient is nil")
	}
	if client.Repositories == nil {
		t.Error("NewDefault() Repositories is nil")
	}
}

// TestNew_InvalidURL tests New() with an invalid URL.
func TestNew_InvalidURL(t *testing.T) {
	_, err := New("://invalid-url")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

// TestNew_ValidURL tests New() with a valid URL.
func TestNew_ValidURL(t *testing.T) {
	client, err := New("https://gitea.example.com")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client == nil {
		t.Fatal("New() returned nil client")
	}
	if client.BaseURL.String() != "https://gitea.example.com" {
		t.Errorf("BaseURL = %q, want %q", client.BaseURL.String(), "https://gitea.example.com")
	}
}

// TestGetRepos_Success tests GetRepos with a mock server returning valid data.
func TestGetRepos_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if auth != "token test-token" {
			t.Errorf("Authorization header = %q, want %q", auth, "token test-token")
		}

		repos := []*thirdbean.ResultGiteaRepo{
			newTestRepo(1, "user", "repo1", "user/repo1"),
			newTestRepo(2, "user", "repo2", "user/repo2"),
		}
		w.Header().Set("x-total-count", "2")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(repos)
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := client.Repositories.GetRepos(context.Background(), "test-token", "user", "all", "full_name", "desc", 1, 10)
	if err != nil {
		t.Fatalf("GetRepos() error = %v", err)
	}
	if result == nil {
		t.Fatal("GetRepos() returned nil result")
	}
	if len(result.Ropes) != 2 {
		t.Errorf("result.Ropes length = %d, want 2", len(result.Ropes))
	}
	if result.TotalPages != 1 {
		t.Errorf("result.TotalPages = %d, want 1", result.TotalPages)
	}
}

// TestGetRepos_PaginationCalculation tests pagination math in GetRepos.
func TestGetRepos_PaginationCalculation(t *testing.T) {
	tests := []struct {
		name       string
		totalCount string
		perPage    int
		wantPages  int64
	}{
		{"exact division", "20", 10, 2},
		{"with remainder", "21", 10, 3},
		{"single page", "5", 10, 1},
		{"one item", "1", 10, 1},
		{"zero total", "0", 10, 1},
		{"no header", "", 10, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.totalCount != "" {
					w.Header().Set("x-total-count", tt.totalCount)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("[]"))
			}))
			defer ts.Close()

			client, err := New(ts.URL)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			result, err := client.Repositories.GetRepos(context.Background(), "token", "user", "", "", "", 1, tt.perPage)
			if err != nil {
				t.Fatalf("GetRepos() error = %v", err)
			}
			if result.TotalPages != tt.wantPages {
				t.Errorf("TotalPages = %d, want %d (totalCount=%s, perPage=%d)", result.TotalPages, tt.wantPages, tt.totalCount, tt.perPage)
			}
		})
	}
}

// TestGetRepos_InvalidTotalCount tests GetRepos with an invalid x-total-count header.
func TestGetRepos_InvalidTotalCount(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-total-count", "not-a-number")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Repositories.GetRepos(context.Background(), "token", "user", "", "", "", 1, 10)
	if err == nil {
		t.Fatal("expected error for invalid total count, got nil")
	}
}

// TestGetRepos_InvalidJSON tests GetRepos with invalid JSON response.
func TestGetRepos_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Repositories.GetRepos(context.Background(), "token", "user", "", "", "", 1, 10)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestGetPullQuest_Success tests GetPullQuest with a successful response.
func TestGetPullQuest_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "token test-token" {
			t.Errorf("Authorization header = %q, want %q", auth, "token test-token")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":1,"title":"Test PR","state":"open"}`))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// GetPullQuest is not part of the interface, so we need to cast
	repoService, ok := client.Repositories.(*RepositoryService)
	if !ok {
		t.Fatal("could not cast to *RepositoryService")
	}

	result, err := repoService.GetPullQuest(context.Background(), "test-token", "owner", "repo", 1)
	if err != nil {
		t.Fatalf("GetPullQuest() error = %v", err)
	}
	if result == nil {
		t.Fatal("GetPullQuest() returned nil")
	}
	if len(result) == 0 {
		t.Error("GetPullQuest() returned empty body")
	}
}

// TestGetPullQuest_APIError tests GetPullQuest with a server error.
func TestGetPullQuest_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	repoService, ok := client.Repositories.(*RepositoryService)
	if !ok {
		t.Fatal("could not cast to *RepositoryService")
	}

	_, err = repoService.GetPullQuest(context.Background(), "token", "owner", "repo", 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestGetWebHooks_Success tests GetWebHooks with a successful response.
func TestGetWebHooks_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hooks := []*thirdbean.ResultGetGiteaHook{
			newTestHook(1, "https://ci.example.com/hook", time.Now()),
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(hooks)
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	hooks, err := client.Repositories.GetWebHooks(context.Background(), "token", "owner", "repo", 1, 10)
	if err != nil {
		t.Fatalf("GetWebHooks() error = %v", err)
	}
	if len(hooks) != 1 {
		t.Errorf("hooks length = %d, want 1", len(hooks))
	}
}

// TestGetWebHooks_InvalidJSON tests GetWebHooks with invalid JSON response.
func TestGetWebHooks_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid"))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Repositories.GetWebHooks(context.Background(), "token", "owner", "repo", 1, 10)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestGetRepoBranches_Success tests GetRepoBranches with valid data.
func TestGetRepoBranches_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		branches := []*thirdbean.ResultGiteaRepoBranch{
			{Name: "main"},
			{Name: "develop"},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(branches)
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	branches, err := client.Repositories.GetRepoBranches(context.Background(), "token", "owner", "repo")
	if err != nil {
		t.Fatalf("GetRepoBranches() error = %v", err)
	}
	if len(branches) != 2 {
		t.Errorf("branches length = %d, want 2", len(branches))
	}
}

// TestGetRepoBranches_InvalidJSON tests GetRepoBranches with invalid JSON.
func TestGetRepoBranches_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{bad json"))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Repositories.GetRepoBranches(context.Background(), "token", "owner", "repo")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestCreateWebHooks_Success tests CreateWebHooks with a successful response.
func TestCreateWebHooks_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want %q", r.Header.Get("Content-Type"), "application/json")
		}
		hook := newTestHook(42, "https://ci.example.com/hook", time.Now())
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(hook)
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	hook, err := client.Repositories.CreateWebHooks(context.Background(), "token", "owner", "repo", "https://ci.example.com/hook", "secret")
	if err != nil {
		t.Fatalf("CreateWebHooks() error = %v", err)
	}
	if hook == nil {
		t.Fatal("CreateWebHooks() returned nil hook")
	}
	if hook.Id != 42 {
		t.Errorf("hook.Id = %d, want 42", hook.Id)
	}
}

// TestCreateWebHooks_InvalidJSON tests CreateWebHooks with invalid JSON response.
func TestCreateWebHooks_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("not json"))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Repositories.CreateWebHooks(context.Background(), "token", "owner", "repo", "https://ci.example.com", "secret")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestDeleteHooks_Success tests DeleteHooks with a successful 204 response.
func TestDeleteHooks_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.Repositories.DeleteHooks(context.Background(), "token", "owner", "repo", "123")
	if err != nil {
		t.Fatalf("DeleteHooks() error = %v", err)
	}
}

// TestAPIConstants verifies that API constants are defined correctly.
func TestAPIConstants(t *testing.T) {
	if BaseApiGitea == "" {
		t.Error("BaseApiGitea is empty")
	}
	if ApiGiteaGetRepos == "" {
		t.Error("ApiGiteaGetRepos is empty")
	}
	if ApiGiteaCreateHooks == "" {
		t.Error("ApiGiteaCreateHooks is empty")
	}
	if ApiGiteaGetHooks == "" {
		t.Error("ApiGiteaGetHooks is empty")
	}
	if ApiGiteaDeleteHooks == "" {
		t.Error("ApiGiteaDeleteHooks is empty")
	}
	if ApiGiteaGetRepoBranches == "" {
		t.Error("ApiGiteaGetRepoBranches is empty")
	}
	if ApiGiteaGetPullRequest == "" {
		t.Error("ApiGiteaGetPullRequest is empty")
	}
	if ApiGiteaCreateFile == "" {
		t.Error("ApiGiteaCreateFile is empty")
	}

	// Verify format strings work
	_ = fmt.Sprintf(ApiGiteaGetRepos, 1, 10)
	_ = fmt.Sprintf(ApiGiteaCreateHooks, "owner", "repo")
	_ = fmt.Sprintf(ApiGiteaGetHooks, "owner", "repo", 1, 10)
	_ = fmt.Sprintf(ApiGiteaDeleteHooks, "owner", "repo", 123)
	_ = fmt.Sprintf(ApiGiteaGetRepoBranches, "owner", "repo")
	_ = fmt.Sprintf(ApiGiteaGetPullRequest, "owner", "repo", 1)
	_ = fmt.Sprintf(ApiGiteaCreateFile, "owner", "repo", "path/to/file")
}

// TestConvertRepository_ZeroValues tests convertRepository with zero-value input.
func TestConvertRepository_ZeroValues(t *testing.T) {
	input := &thirdbean.ResultGiteaRepo{}
	result := convertRepository(input)
	if result == nil {
		t.Fatal("convertRepository returned nil")
	}
	if result.Id != "0" {
		t.Errorf("Id = %q, want %q", result.Id, "0")
	}
	if result.RepoType != "gitea" {
		t.Errorf("RepoType = %q, want %q", result.RepoType, "gitea")
	}
}

// TestGetRepos_WithContextCancellation tests GetRepos with a canceled context.
func TestGetRepos_WithContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[]"))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.Repositories.GetRepos(ctx, "token", "user", "", "", "", 1, 10)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

// Helper functions

func newTestRepo(id int, owner, name, fullName string) *thirdbean.ResultGiteaRepo {
	repo := &thirdbean.ResultGiteaRepo{
		Id:       id,
		Name:     name,
		FullName: fullName,
		HtmlUrl:  fmt.Sprintf("https://gitea.com/%s", fullName),
	}
	repo.Owner.Login = owner
	return repo
}

func newTestHook(id int, url string, createdAt time.Time) *thirdbean.ResultGetGiteaHook {
	hook := &thirdbean.ResultGetGiteaHook{
		Id:        id,
		CreatedAt: createdAt,
	}
	hook.Config.Url = url
	return hook
}

// Ensure unused import is used
var _ = thirdapi.NewAPIError
