package server

import "errors"

// Sentinel errors for the server package.
// These can be used with errors.Is() for programmatic error checking.
var (
	// ErrPathEmpty is returned when a file path parameter is empty.
	ErrPathEmpty = errors.New("path parameter is empty")

	// ErrPathInvalid is returned when a file path contains traversal sequences.
	ErrPathInvalid = errors.New("invalid path")

	// ErrFileNotFound is returned when a requested file does not exist in the static bundle.
	ErrFileNotFound = errors.New("file not found")
)
