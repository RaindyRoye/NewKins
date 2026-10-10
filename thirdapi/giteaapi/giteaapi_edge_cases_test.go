package giteaapi

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gokins/gokins/thirdapi"
)

// TestGetRepos_URLParseError tests GetRepos when URL parsing fails.
func TestGetRepos_URLParseError(t *testing.T) {
	client := &thirdapi.Client{
		BaseURL:    &url.URL{Scheme: "://", Opaque: "bad"},
		HttpClient: http.DefaultClient,
	}
	w := &wrapper{client}
	svc := &RepositoryService{client: w}

	_, err := svc.GetRepos(context.Background(), "token", "user", "", "", "", 1, 10)
	if err == nil {
		t.Fatal("expected error for bad URL, got nil")
	}
}

// TestDeleteHooks_URLParseError tests DeleteHooks when URL parsing fails.
func TestDeleteHooks_URLParseError(t *testing.T) {
	client := &thirdapi.Client{
		BaseURL:    &url.URL{Scheme: "://", Opaque: "bad"},
		HttpClient: http.DefaultClient,
	}
	w := &wrapper{client}
	svc := &RepositoryService{client: w}

	err := svc.DeleteHooks(context.Background(), "token", "owner", "repo", "123")
	if err == nil {
		t.Fatal("expected error for bad URL, got nil")
	}
}

// TestCreateWebHooks_URLParseError tests CreateWebHooks when URL parsing fails.
func TestCreateWebHooks_URLParseError(t *testing.T) {
	client := &thirdapi.Client{
		BaseURL:    &url.URL{Scheme: "://", Opaque: "bad"},
		HttpClient: http.DefaultClient,
	}
	w := &wrapper{client}
	svc := &RepositoryService{client: w}

	_, err := svc.CreateWebHooks(context.Background(), "token", "owner", "repo", "https://example.com", "secret")
	if err == nil {
		t.Fatal("expected error for bad URL, got nil")
	}
}

// TestGetRepoBranches_URLParseError tests GetRepoBranches when URL parsing fails.
func TestGetRepoBranches_URLParseError(t *testing.T) {
	client := &thirdapi.Client{
		BaseURL:    &url.URL{Scheme: "://", Opaque: "bad"},
		HttpClient: http.DefaultClient,
	}
	w := &wrapper{client}
	svc := &RepositoryService{client: w}

	_, err := svc.GetRepoBranches(context.Background(), "token", "owner", "repo")
	if err == nil {
		t.Fatal("expected error for bad URL, got nil")
	}
}

// TestGetPullQuest_URLParseError tests GetPullQuest when URL parsing fails.
func TestGetPullQuest_URLParseError(t *testing.T) {
	client := &thirdapi.Client{
		BaseURL:    &url.URL{Scheme: "://", Opaque: "bad"},
		HttpClient: http.DefaultClient,
	}
	w := &wrapper{client}
	svc := &RepositoryService{client: w}

	_, err := svc.GetPullQuest(context.Background(), "token", "owner", "repo", 1)
	if err == nil {
		t.Fatal("expected error for bad URL, got nil")
	}
}

// TestGetWebHooks_URLParseError tests GetWebHooks when URL parsing fails.
func TestGetWebHooks_URLParseError(t *testing.T) {
	client := &thirdapi.Client{
		BaseURL:    &url.URL{Scheme: "://", Opaque: "bad"},
		HttpClient: http.DefaultClient,
	}
	w := &wrapper{client}
	svc := &RepositoryService{client: w}

	_, err := svc.GetWebHooks(context.Background(), "token", "owner", "repo", 1, 10)
	if err == nil {
		t.Fatal("expected error for bad URL, got nil")
	}
}

// TestAPIErrorMethods tests APIError helper methods.
func TestAPIErrorMethods(t *testing.T) {
	err500 := thirdapi.NewAPIError("gitea", "Test", 500, "internal error")
	if !err500.IsRetryable() {
		t.Error("500 should be retryable")
	}
	if err500.IsNotFound() {
		t.Error("500 should not be not found")
	}
	if err500.IsAuthError() {
		t.Error("500 should not be auth error")
	}

	err429 := thirdapi.NewAPIError("gitea", "Test", 429, "rate limit")
	if !err429.IsRetryable() {
		t.Error("429 should be retryable")
	}

	err404 := thirdapi.NewAPIError("gitea", "Test", 404, "not found")
	if !err404.IsNotFound() {
		t.Error("404 should be not found")
	}
	if err404.IsRetryable() {
		t.Error("404 should not be retryable")
	}

	err401 := thirdapi.NewAPIError("gitea", "Test", 401, "unauthorized")
	if !err401.IsAuthError() {
		t.Error("401 should be auth error")
	}

	err403 := thirdapi.NewAPIError("gitea", "Test", 403, "forbidden")
	if !err403.IsAuthError() {
		t.Error("403 should be auth error")
	}

	err200 := thirdapi.NewAPIError("gitea", "Test", 200, "")
	if err200.IsRetryable() {
		t.Error("200 should not be retryable")
	}
	if err200.IsNotFound() {
		t.Error("200 should not be not found")
	}
	if err200.IsAuthError() {
		t.Error("200 should not be auth error")
	}
}

// TestAPIErrorString tests APIError.Error() string formatting.
func TestAPIErrorString(t *testing.T) {
	withBody := thirdapi.NewAPIError("gitea", "GetRepos", 500, "server error")
	s := withBody.Error()
	if s == "" {
		t.Error("Error() returned empty string")
	}

	withoutBody := thirdapi.NewAPIError("gitea", "GetRepos", 500, "")
	s2 := withoutBody.Error()
	if s2 == "" {
		t.Error("Error() returned empty string")
	}
}

// TestAPIErrorTruncation tests that long error bodies are truncated.
func TestAPIErrorTruncation(t *testing.T) {
	longBody := make([]byte, 1000)
	for i := range longBody {
		longBody[i] = 'x'
	}

	err := thirdapi.NewAPIError("gitea", "Test", 500, string(longBody))
	if len(err.Body) > 600 {
		t.Errorf("Body length = %d, expected <= 600 (512 + truncation message)", len(err.Body))
	}
}

// TestConvertFunctions_NilInputs tests converter functions with nil inputs.
func TestConvertFunctions_NilInputs(t *testing.T) {
	repos := convertRepositoryList(nil)
	if len(repos) != 0 {
		t.Errorf("convertRepositoryList(nil) length = %d, want 0", len(repos))
	}

	branches := convertBranchList(nil)
	if len(branches) != 0 {
		t.Errorf("convertBranchList(nil) length = %d, want 0", len(branches))
	}

	hooks := convertHookList(nil)
	if len(hooks) != 0 {
		t.Errorf("convertHookList(nil) length = %d, want 0", len(hooks))
	}
}

// TestHttpClientTimeout tests that the client has a reasonable timeout.
func TestHttpClientTimeout(t *testing.T) {
	client := NewDefault()
	if client.HttpClient.Timeout < 5*time.Second {
		t.Errorf("HttpClient.Timeout = %v, want >= 5s", client.HttpClient.Timeout)
	}
	if client.HttpClient.Timeout > 30*time.Second {
		t.Errorf("HttpClient.Timeout = %v, want <= 30s", client.HttpClient.Timeout)
	}
}
