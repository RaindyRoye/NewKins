package thirdapi

import "fmt"

// APIError represents a structured error from a third-party API call.
// It captures the provider name, the operation that failed, the HTTP
// status code, and (optionally) the response body returned by the server.
//
// Callers can use errors.As to inspect APIError fields and make decisions
// based on the status code (e.g., retry on 5xx, abort on 403).
type APIError struct {
	// Provider is the name of the API provider (e.g., "gitea", "github").
	Provider string
	// Operation is the method that produced the error (e.g., "GetRepos").
	Operation string
	// StatusCode is the HTTP status code returned by the remote server.
	StatusCode int
	// Body is the raw response body, truncated to 512 bytes for safety.
	Body string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("%s api %s failed (status %d): %s",
			e.Provider, e.Operation, e.StatusCode, e.Body)
	}
	return fmt.Sprintf("%s api %s failed (status %d)",
		e.Provider, e.Operation, e.StatusCode)
}

// IsRetryable returns true if the HTTP status code indicates a transient
// failure that may succeed on retry (429 Too Many Requests or 5xx server errors).
func (e *APIError) IsRetryable() bool {
	return e.StatusCode == 429 || e.StatusCode >= 500
}

// IsNotFound returns true if the HTTP status code is 404.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsAuthError returns true if the HTTP status code indicates an
// authentication or authorization failure (401 or 403).
func (e *APIError) IsAuthError() bool {
	return e.StatusCode == 401 || e.StatusCode == 403
}

// maxBodyLen is the maximum number of bytes to retain from a response
// body when constructing an APIError. This prevents unbounded memory
// use when a server returns a very large error page.
const maxBodyLen = 512

// NewAPIError creates an APIError with the response body truncated to
// maxBodyLen bytes. This is the recommended constructor for all thirdapi
// packages.
func NewAPIError(provider, operation string, statusCode int, body string) *APIError {
	if len(body) > maxBodyLen {
		body = body[:maxBodyLen] + "...(truncated)"
	}
	return &APIError{
		Provider:   provider,
		Operation:  operation,
		StatusCode: statusCode,
		Body:       body,
	}
}
