package server

import "errors"

// Sentinel errors for the server package.
var (
	// ErrPathEmpty is returned when a file path parameter is empty.
	ErrPathEmpty = errors.New("path parameter is empty")
	// ErrInvalidPath is returned when a file path is invalid or contains traversal sequences.
	ErrInvalidPath = errors.New("invalid path")
	// ErrFileNotFound is returned when a requested file cannot be found.
	ErrFileNotFound = errors.New("file not found")
)
