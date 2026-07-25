package thirdapi

import (
	"errors"
	"fmt"
)

// ErrAPIRequestFailed is a sentinel error returned when an HTTP API call
// to a third-party provider (Gitea, GitHub, Gitee, GitLab, etc.) receives
// a non-success status code. Callers can use errors.Is to detect this class
// of error and errors.As to extract the status code and response body.
//
// Example:
//
//	var apiErr *thirdapi.APIError
//	if errors.As(err, &apiErr) {
//	    log.Printf("API returned %d: %s", apiErr.StatusCode, apiErr.Body)
//	}
var ErrAPIRequestFailed = errors.New("api request failed")

// APIError represents a non-success HTTP response from a third-party API.
// It wraps ErrAPIRequestFailed so that errors.Is(err, ErrAPIRequestFailed)
// returns true. The Provider, Operation, StatusCode, and Body fields give
// callers enough context to produce actionable log messages or retry logic.
type APIError struct {
	// Provider is the API provider name (e.g. "gitea", "github", "gitee", "gitlab").
	Provider string
	// Operation is the method or endpoint name (e.g. "GetRepos", "DeleteHooks").
	Operation string
	// StatusCode is the HTTP status code returned by the server.
	StatusCode int
	// Body is the (possibly truncated) response body for debugging.
	Body string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Body != "" {
		body := e.Body
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		return fmt.Sprintf("%s api %s failed (status %d): %s", e.Provider, e.Operation, e.StatusCode, body)
	}
	return fmt.Sprintf("%s api %s failed (status %d)", e.Provider, e.Operation, e.StatusCode)
}

// Is implements the errors.Is interface so that errors.Is(err, ErrAPIRequestFailed) works.
func (e *APIError) Is(target error) bool {
	return target == ErrAPIRequestFailed
}

// Unwrap is not needed since Is() handles the sentinel check, but we
// implement it for completeness in case the sentinel changes.
func (e *APIError) Unwrap() error {
	return ErrAPIRequestFailed
}

// NewAPIError creates a new APIError wrapped with ErrAPIRequestFailed.
// Callers should return the result directly: return nil, NewAPIError(...)
func NewAPIError(provider, operation string, statusCode int, body string) error {
	return &APIError{
		Provider:   provider,
		Operation:  operation,
		StatusCode: statusCode,
		Body:       body,
	}
}
