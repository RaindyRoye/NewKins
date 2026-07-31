package giteaapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/thirdapi"
)

func TestGiteaGetRepos_APIError(t *testing.T) {
	// Create a test server that returns 403 Forbidden
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"rate limit exceeded"}`))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Repositories.GetRepos(context.Background(), "token", "user", "all", "full_name", "desc", 1, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T: %v", err, err)
	}

	if apiErr.Provider != "gitea" {
		t.Errorf("Provider = %q, want %q", apiErr.Provider, "gitea")
	}
	if apiErr.Operation != "GetRepos" {
		t.Errorf("Operation = %q, want %q", apiErr.Operation, "GetRepos")
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want 403", apiErr.StatusCode)
	}
	if !apiErr.IsAuthError() {
		t.Error("expected IsAuthError() true for 403")
	}
	if apiErr.IsRetryable() {
		t.Error("expected IsRetryable() false for 403")
	}
}

func TestGiteaDeleteHooks_APIError(t *testing.T) {
	// Create a test server that returns 404 Not Found
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"hook not found"}`))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.Repositories.DeleteHooks(context.Background(), "token", "owner", "repo", "123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T: %v", err, err)
	}

	if apiErr.Provider != "gitea" {
		t.Errorf("Provider = %q, want %q", apiErr.Provider, "gitea")
	}
	if apiErr.Operation != "DeleteHooks" {
		t.Errorf("Operation = %q, want %q", apiErr.Operation, "DeleteHooks")
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() true for 404")
	}
}

func TestGiteaCreateWebHooks_APIError(t *testing.T) {
	// Create a test server that returns 500 Internal Server Error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Repositories.CreateWebHooks(context.Background(), "token", "owner", "repo", "https://example.com/hook", "secret")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T: %v", err, err)
	}

	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
	if !apiErr.IsRetryable() {
		t.Error("expected IsRetryable() true for 500")
	}
}

func TestGiteaGetRepoBranches_APIError(t *testing.T) {
	// Create a test server that returns 429 Too Many Requests
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"rate limit"}`))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Repositories.GetRepoBranches(context.Background(), "token", "owner", "repo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T: %v", err, err)
	}

	if apiErr.StatusCode != 429 {
		t.Errorf("StatusCode = %d, want 429", apiErr.StatusCode)
	}
	if !apiErr.IsRetryable() {
		t.Error("expected IsRetryable() true for 429")
	}
}

func TestGiteaGetWebHooks_APIError(t *testing.T) {
	// Create a test server that returns 401 Unauthorized
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid token"}`))
	}))
	defer ts.Close()

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Repositories.GetWebHooks(context.Background(), "token", "owner", "repo", 1, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *thirdapi.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *thirdapi.APIError, got %T: %v", err, err)
	}

	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if !apiErr.IsAuthError() {
		t.Error("expected IsAuthError() true for 401")
	}
}
