package thirdapi

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestAPIError_Error_WithBody(t *testing.T) {
	err := NewAPIError("gitea", "GetRepos", 500, "internal server error")
	want := "gitea api GetRepos failed (status 500): internal server error"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestAPIError_Error_WithoutBody(t *testing.T) {
	err := NewAPIError("github", "DeleteHooks", 204, "")
	want := "github api DeleteHooks failed (status 204)"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestAPIError_Error_TruncatesLongBody(t *testing.T) {
	longBody := strings.Repeat("x", 300)
	err := NewAPIError("gitee", "CreateWebHooks", 422, longBody)
	msg := err.Error()
	if len(msg) > 400 {
		t.Errorf("error message too long: %d chars", len(msg))
	}
	if !strings.HasSuffix(msg, "...") {
		t.Errorf("expected truncation marker '...', got %q", msg)
	}
}

func TestAPIError_ErrorsIs(t *testing.T) {
	err := NewAPIError("gitlab", "GetRepoBranches", 404, "not found")
	if !errors.Is(err, ErrAPIRequestFailed) {
		t.Error("errors.Is(err, ErrAPIRequestFailed) should be true")
	}
}

func TestAPIError_ErrorsAs(t *testing.T) {
	err := NewAPIError("gitea", "GetWebHooks", 503, "service unavailable")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As should succeed")
	}
	if apiErr.Provider != "gitea" {
		t.Errorf("Provider = %q, want %q", apiErr.Provider, "gitea")
	}
	if apiErr.Operation != "GetWebHooks" {
		t.Errorf("Operation = %q, want %q", apiErr.Operation, "GetWebHooks")
	}
	if apiErr.StatusCode != 503 {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, 503)
	}
	if apiErr.Body != "service unavailable" {
		t.Errorf("Body = %q, want %q", apiErr.Body, "service unavailable")
	}
}

func TestAPIError_Wrapped_ErrorsIs(t *testing.T) {
	inner := NewAPIError("github", "GetRepos", 403, "forbidden")
	wrapped := fmt.Errorf("operation failed: %w", inner)
	if !errors.Is(wrapped, ErrAPIRequestFailed) {
		t.Error("wrapped error should still match ErrAPIRequestFailed via errors.Is")
	}
	var apiErr *APIError
	if !errors.As(wrapped, &apiErr) {
		t.Fatal("errors.As should succeed through fmt.Errorf wrapping")
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, 403)
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	err := NewAPIError("gitee", "DeleteHooks", 500, "err")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As should succeed")
	}
	if apiErr.Unwrap() != ErrAPIRequestFailed {
		t.Error("Unwrap should return ErrAPIRequestFailed")
	}
}

func TestAPIError_NotMatchOtherSentinels(t *testing.T) {
	err := NewAPIError("gitea", "GetRepos", 500, "err")
	otherSentinel := errors.New("some other error")
	if errors.Is(err, otherSentinel) {
		t.Error("should not match unrelated sentinel errors")
	}
}

func TestErrAPIRequestFailed_IsDistinct(t *testing.T) {
	if ErrAPIRequestFailed == nil {
		t.Fatal("ErrAPIRequestFailed must not be nil")
	}
	if ErrAPIRequestFailed.Error() != "api request failed" {
		t.Errorf("message = %q, want %q", ErrAPIRequestFailed.Error(), "api request failed")
	}
}

func TestAPIError_ProviderNames(t *testing.T) {
	providers := []string{"gitea", "github", "gitee", "gitlab", "gitee-premium"}
	for _, p := range providers {
		err := NewAPIError(p, "TestOp", 400, "")
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("errors.As failed for provider %q", p)
		}
		if apiErr.Provider != p {
			t.Errorf("Provider = %q, want %q", apiErr.Provider, p)
		}
	}
}
