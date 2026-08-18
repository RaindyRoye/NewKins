package util

import "errors"

// Sentinel errors for the util package.
// These errors can be used with errors.Is() to check for specific error
// conditions without relying on exact error message strings.
var (
	// ErrRepositoryNil is returned when a repository reference is unexpectedly nil.
	ErrRepositoryNil = errors.New("repository is nil")

	// ErrInvalidGitHash is returned when a string is not a valid git hash.
	ErrInvalidGitHash = errors.New("invalid git hash")

	// ErrInvalidSigningMethod is returned when a JWT token uses an unexpected signing algorithm.
	ErrInvalidSigningMethod = errors.New("unexpected signing method")
)
