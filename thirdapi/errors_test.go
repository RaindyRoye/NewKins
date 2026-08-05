package thirdapi

import (
	"errors"
	"strings"
	"testing"
)

func TestAPIError_Error_WithBody(t *testing.T) {
	e := &APIError{
		Provider:   "github",
		Operation:  "GetRepos",
		StatusCode: 500,
		Body:       `{"message":"internal error"}`,
	}
	got := e.Error()
	if !strings.Contains(got, "github") {
		t.Errorf("expected provider in error, got: %s", got)
	}
	if !strings.Contains(got, "GetRepos") {
		t.Errorf("expected operation in error, got: %s", got)
	}
	if !strings.Contains(got, "500") {
		t.Errorf("expected status code in error, got: %s", got)
	}
	if !strings.Contains(got, "internal error") {
		t.Errorf("expected body in error, got: %s", got)
	}
}

func TestAPIError_Error_WithoutBody(t *testing.T) {
	e := &APIError{
		Provider:   "gitlab",
		Operation:  "DeleteHooks",
		StatusCode: 403,
		Body:       "",
	}
	got := e.Error()
	if !strings.Contains(got, "gitlab") {
		t.Errorf("expected provider in error, got: %s", got)
	}
	if !strings.Contains(got, "403") {
		t.Errorf("expected status code in error, got: %s", got)
	}
	// Should not contain trailing colon+space when body is empty
	if strings.HasSuffix(got, ": ") {
		t.Errorf("error should not end with ': ' when body is empty, got: %s", got)
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{"429 rate limit", 429, true},
		{"500 internal", 500, true},
		{"502 bad gateway", 502, true},
		{"503 service unavailable", 503, true},
		{"504 gateway timeout", 504, true},
		{"200 ok", 200, false},
		{"400 bad request", 400, false},
		{"401 unauthorized", 401, false},
		{"403 forbidden", 403, false},
		{"404 not found", 404, false},
		{"409 conflict", 409, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &APIError{StatusCode: tt.statusCode}
			if got := e.IsRetryable(); got != tt.want {
				t.Errorf("IsRetryable() for %d = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestAPIError_IsNotFound(t *testing.T) {
	tests := []struct {
		statusCode int
		want       bool
	}{
		{404, true},
		{200, false},
		{500, false},
		{410, false}, // gone, not 404
	}
	for _, tt := range tests {
		e := &APIError{StatusCode: tt.statusCode}
		if got := e.IsNotFound(); got != tt.want {
			t.Errorf("IsNotFound() for %d = %v, want %v", tt.statusCode, got, tt.want)
		}
	}
}

func TestAPIError_IsAuthError(t *testing.T) {
	tests := []struct {
		statusCode int
		want       bool
	}{
		{401, true},
		{403, true},
		{200, false},
		{404, false},
		{500, false},
		{429, false},
	}
	for _, tt := range tests {
		e := &APIError{StatusCode: tt.statusCode}
		if got := e.IsAuthError(); got != tt.want {
			t.Errorf("IsAuthError() for %d = %v, want %v", tt.statusCode, got, tt.want)
		}
	}
}

func TestNewAPIError_ShortBody(t *testing.T) {
	body := "short error"
	e := NewAPIError("github", "GetRepos", 500, body)
	if e.Provider != "github" {
		t.Errorf("Provider = %q, want %q", e.Provider, "github")
	}
	if e.Operation != "GetRepos" {
		t.Errorf("Operation = %q, want %q", e.Operation, "GetRepos")
	}
	if e.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want %d", e.StatusCode, 500)
	}
	if e.Body != body {
		t.Errorf("Body = %q, want %q", e.Body, body)
	}
}

func TestNewAPIError_LongBody_Truncates(t *testing.T) {
	longBody := strings.Repeat("a", maxBodyLen+200)
	e := NewAPIError("gitlab", "CreateWebHooks", 422, longBody)

	if len(e.Body) <= maxBodyLen {
		t.Errorf("expected truncated body to include suffix, got len=%d", len(e.Body))
	}
	if !strings.HasSuffix(e.Body, "...(truncated)") {
		t.Errorf("expected body to end with '...(truncated)', got: ...%s", e.Body[len(e.Body)-20:])
	}
}

func TestNewAPIError_ExactMaxBodyLen(t *testing.T) {
	exactBody := strings.Repeat("b", maxBodyLen)
	e := NewAPIError("gitea", "GetRepos", 200, exactBody)
	if e.Body != exactBody {
		t.Errorf("body at exact maxBodyLen should not be truncated")
	}
}

func TestNewAPIError_EmptyBody(t *testing.T) {
	e := NewAPIError("gitee", "DeleteHooks", 204, "")
	if e.Body != "" {
		t.Errorf("Body = %q, want empty", e.Body)
	}
}

func TestAPIError_ErrorsAs(t *testing.T) {
	// Verify that errors.As works with *APIError
	var apiErr *APIError
	err := NewAPIError("github", "GetRepos", 403, "forbidden")
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As should match *APIError")
	}
	if apiErr.Provider != "github" {
		t.Errorf("Provider = %q, want %q", apiErr.Provider, "github")
	}
	if !apiErr.IsAuthError() {
		t.Error("expected IsAuthError() to be true for 403")
	}
}

func TestAPIError_Wrapped(t *testing.T) {
	// Verify that APIError works when wrapped with fmt.Errorf
	inner := NewAPIError("github", "GetRepos", 500, "server error")
	wrapped := errors.Join(errors.New("operation failed"), inner)

	var apiErr *APIError
	if !errors.As(wrapped, &apiErr) {
		t.Fatal("errors.As should find *APIError through wrapping")
	}
	if !apiErr.IsRetryable() {
		t.Error("expected IsRetryable() for status 500")
	}
}

func TestAPIError_AllMethods_Combined(t *testing.T) {
	// Test that the methods are independent — a status can't be both retryable and auth error
	// (except hypothetically if we add more categories). Just sanity check 500.
	e := &APIError{StatusCode: 500}
	if !e.IsRetryable() {
		t.Error("500 should be retryable")
	}
	if e.IsNotFound() {
		t.Error("500 should not be not-found")
	}
	if e.IsAuthError() {
		t.Error("500 should not be auth error")
	}
}
