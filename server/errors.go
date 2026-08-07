package server

import "errors"

// Sentinel errors for the server package.
//
// These errors can be used with errors.Is() to check for specific error
// conditions without relying on exact error message strings. All sentinel
// errors are wrapped using fmt.Errorf with %w in the call sites, so
// errors.Is() works through the error chain.
//
// Usage in call sites:
//
//	return fmt.Errorf("%w: path parameter is empty", ErrFileNotFound)
//
// Usage in callers/tests:
//
//	if errors.Is(err, server.ErrFileNotFound) { ... }
var (
	// ErrFileNotFound is returned when a requested file is not found in the static assets.
	ErrFileNotFound = errors.New("file not found")
)
