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

	// ErrDatabaseOpen is returned when the database engine cannot be opened.
	ErrDatabaseOpen = errors.New("open database")
	// ErrCacheOpen is returned when the cache database cannot be opened.
	ErrCacheOpen = errors.New("open cache database")
	// ErrMigrationFailed is returned when a database migration fails.
	ErrMigrationFailed = errors.New("database migration failed")
	// ErrDatabasePing is returned when a database connectivity check fails.
	ErrDatabasePing = errors.New("database ping failed")
)
