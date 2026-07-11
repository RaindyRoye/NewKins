package thirdapi

import "fmt"

// APIError represents an HTTP error response from a third-party API.
// Callers can use errors.As to extract the status code and response body
// for conditional error handling (e.g., retry on 5xx, skip on 404).
type APIError struct {
	// Provider is the API provider name (e.g., "gitea", "github").
	Provider string
	// Operation is the method/operation that failed (e.g., "GetRepos").
	Operation string
	// StatusCode is the HTTP response status code.
	StatusCode int
	// Body is the raw response body text (may be empty).
	Body string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("%s api %s failed (status %d): %s", e.Provider, e.Operation, e.StatusCode, e.Body)
	}
	return fmt.Sprintf("%s api %s failed (status %d)", e.Provider, e.Operation, e.StatusCode)
}

// IsServerError returns true if the error is a 5xx server error,
// which may indicate a transient failure worth retrying.
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// IsClientError returns true if the error is a 4xx client error,
// which usually indicates a permanent failure (bad request, not found, etc.).
func (e *APIError) IsClientError() bool {
	return e.StatusCode >= 400 && e.StatusCode < 500
}

// IsNotFound returns true if the error is a 404 Not Found response.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsUnauthorized returns true if the error is a 401 Unauthorized response.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsForbidden returns true if the error is a 403 Forbidden response.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == 403
}

// NewAPIError creates a new APIError with the given parameters.
// This is a helper for use in thirdapi sub-packages.
func NewAPIError(provider, operation string, statusCode int, body string) *APIError {
	return &APIError{
		Provider:   provider,
		Operation:  operation,
		StatusCode: statusCode,
		Body:       body,
	}
}
