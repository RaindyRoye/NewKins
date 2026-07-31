package thirdapi

import (
	"errors"
	"strings"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	e := &APIError{
		Provider:   "github",
		Operation:  "GetRepos",
		StatusCode: 403,
		Body:       "rate limit exceeded",
	}
	got := e.Error()
	want := "github api GetRepos failed (status 403): rate limit exceeded"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAPIError_ErrorNoBody(t *testing.T) {
	e := &APIError{
		Provider:   "gitea",
		Operation:  "GetRepos",
		StatusCode: 500,
	}
	got := e.Error()
	want := "gitea api GetRepos failed (status 500)"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	cases := []struct {
		code int
		want bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
	}
	for _, tc := range cases {
		e := &APIError{StatusCode: tc.code}
		if got := e.IsRetryable(); got != tc.want {
			t.Errorf("IsRetryable() for %d = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestAPIError_IsNotFound(t *testing.T) {
	e404 := &APIError{StatusCode: 404}
	if !e404.IsNotFound() {
		t.Error("expected IsNotFound() true for 404")
	}
	e200 := &APIError{StatusCode: 200}
	if e200.IsNotFound() {
		t.Error("expected IsNotFound() false for 200")
	}
}

func TestAPIError_IsAuthError(t *testing.T) {
	cases := []struct {
		code int
		want bool
	}{
		{200, false},
		{400, false},
		{401, true},
		{403, true},
		{404, false},
		{500, false},
	}
	for _, tc := range cases {
		e := &APIError{StatusCode: tc.code}
		if got := e.IsAuthError(); got != tc.want {
			t.Errorf("IsAuthError() for %d = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestNewAPIError_ShortBody(t *testing.T) {
	e := NewAPIError("gitlab", "GetRepos", 500, "internal error")
	if e.Provider != "gitlab" {
		t.Errorf("Provider = %q, want %q", e.Provider, "gitlab")
	}
	if e.Operation != "GetRepos" {
		t.Errorf("Operation = %q, want %q", e.Operation, "GetRepos")
	}
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", e.StatusCode)
	}
	if e.Body != "internal error" {
		t.Errorf("Body = %q, want %q", e.Body, "internal error")
	}
}

func TestNewAPIError_TruncatesLongBody(t *testing.T) {
	longBody := strings.Repeat("x", 1024)
	e := NewAPIError("gitee", "DeleteHooks", 422, longBody)
	if len(e.Body) > maxBodyLen+50 { // account for "...(truncated)" suffix
		t.Errorf("Body length %d exceeds expected max", len(e.Body))
	}
	if !strings.HasSuffix(e.Body, "...(truncated)") {
		t.Error("expected body to end with '...(truncated)'")
	}
}

func TestNewAPIError_EmptyBody(t *testing.T) {
	e := NewAPIError("gitea", "GetRepos", 204, "")
	if e.Body != "" {
		t.Errorf("expected empty Body, got %q", e.Body)
	}
	got := e.Error()
	want := "gitea api GetRepos failed (status 204)"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAPIError_AsErrors(t *testing.T) {
	// Verify APIError works with errors.As for type assertions
	var err error = NewAPIError("github", "CreateWebHooks", 422, "validation failed")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As should match *APIError")
	}
	if apiErr.Provider != "github" {
		t.Errorf("Provider = %q, want %q", apiErr.Provider, "github")
	}
	if apiErr.StatusCode != 422 {
		t.Errorf("StatusCode = %d, want 422", apiErr.StatusCode)
	}
	if !apiErr.IsAuthError() && apiErr.StatusCode == 422 {
		// 422 is not an auth error, just verifying the method works correctly
		t.Log("422 is correctly not classified as an auth error")
	}
}
