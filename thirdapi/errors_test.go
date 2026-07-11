package thirdapi

import (
	"errors"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			name: "with body",
			err:  NewAPIError("gitea", "GetRepos", 404, "not found"),
			want: "gitea api GetRepos failed (status 404): not found",
		},
		{
			name: "without body",
			err:  NewAPIError("github", "DeleteHooks", 204, ""),
			want: "github api DeleteHooks failed (status 204)",
		},
		{
			name: "500 server error",
			err:  NewAPIError("gitlab", "CreateWebHooks", 500, "internal server error"),
			want: "gitlab api CreateWebHooks failed (status 500): internal server error",
		},
		{
			name: "401 unauthorized",
			err:  NewAPIError("gitee", "GetRepos", 401, "bad credentials"),
			want: "gitee api GetRepos failed (status 401): bad credentials",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("APIError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIError_IsServerError(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{201, false},
		{400, false},
		{404, false},
		{499, false},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{599, true},
		{600, false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := NewAPIError("test", "op", tt.code, "")
			if got := err.IsServerError(); got != tt.want {
				t.Errorf("IsServerError() for status %d = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestAPIError_IsClientError(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{201, false},
		{399, false},
		{400, true},
		{401, true},
		{403, true},
		{404, true},
		{422, true},
		{499, true},
		{500, false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := NewAPIError("test", "op", tt.code, "")
			if got := err.IsClientError(); got != tt.want {
				t.Errorf("IsClientError() for status %d = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestAPIError_IsNotFound(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{400, false},
		{403, false},
		{404, true},
		{405, false},
		{500, false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := NewAPIError("test", "op", tt.code, "")
			if got := err.IsNotFound(); got != tt.want {
				t.Errorf("IsNotFound() for status %d = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestAPIError_IsUnauthorized(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{400, false},
		{401, true},
		{403, false},
		{500, false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := NewAPIError("test", "op", tt.code, "")
			if got := err.IsUnauthorized(); got != tt.want {
				t.Errorf("IsUnauthorized() for status %d = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestAPIError_IsForbidden(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{401, false},
		{403, true},
		{404, false},
		{500, false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := NewAPIError("test", "op", tt.code, "")
			if got := err.IsForbidden(); got != tt.want {
				t.Errorf("IsForbidden() for status %d = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestAPIError_ErrorsAs(t *testing.T) {
	// Verify that errors.As works correctly with APIError
	var err error = NewAPIError("gitea", "GetRepos", 404, "not found")

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As should succeed for *APIError")
	}
	if apiErr.Provider != "gitea" {
		t.Errorf("Provider = %q, want %q", apiErr.Provider, "gitea")
	}
	if apiErr.Operation != "GetRepos" {
		t.Errorf("Operation = %q, want %q", apiErr.Operation, "GetRepos")
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, 404)
	}
	if apiErr.Body != "not found" {
		t.Errorf("Body = %q, want %q", apiErr.Body, "not found")
	}
}

func TestAPIError_ErrorsAs_WrappedError(t *testing.T) {
	// Simulate a wrapped error chain
	inner := NewAPIError("github", "DeleteHooks", 403, "forbidden")
	wrapped := wrapError("operation failed", inner)

	var apiErr *APIError
	if !errors.As(wrapped, &apiErr) {
		t.Fatal("errors.As should succeed for wrapped *APIError")
	}
	if !apiErr.IsForbidden() {
		t.Error("expected IsForbidden() to be true")
	}
	if apiErr.Provider != "github" {
		t.Errorf("Provider = %q, want %q", apiErr.Provider, "github")
	}
}

// wrapError is a test helper that wraps an error with a message.
func wrapError(msg string, err error) error {
	return &wrappedError{msg: msg, cause: err}
}

type wrappedError struct {
	msg   string
	cause error
}

func (e *wrappedError) Error() string {
	return e.msg + ": " + e.cause.Error()
}

func (e *wrappedError) Unwrap() error {
	return e.cause
}

func TestAPIError_NilBody(t *testing.T) {
	err := NewAPIError("gitlab", "GetRepos", 500, "")
	if err.Error() != "gitlab api GetRepos failed (status 500)" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestAPIError_LargeBody(t *testing.T) {
	// Test with a large response body
	body := string(make([]byte, 10000))
	err := NewAPIError("gitee", "GetRepos", 500, body)
	if err.Body != body {
		t.Error("Body should be preserved even for large responses")
	}
}

func TestNewAPIError_Fields(t *testing.T) {
	err := NewAPIError("myprovider", "MyOp", 418, "I'm a teapot")
	if err.Provider != "myprovider" {
		t.Errorf("Provider = %q, want %q", err.Provider, "myprovider")
	}
	if err.Operation != "MyOp" {
		t.Errorf("Operation = %q, want %q", err.Operation, "MyOp")
	}
	if err.StatusCode != 418 {
		t.Errorf("StatusCode = %d, want %d", err.StatusCode, 418)
	}
	if err.Body != "I'm a teapot" {
		t.Errorf("Body = %q, want %q", err.Body, "I'm a teapot")
	}
}
