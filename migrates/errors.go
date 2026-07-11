package migrates

import "errors"

// Sentinel errors for migration operations.
// Use errors.Is() to check for specific error types.
var (
	// ErrDatabaseConfigMissing indicates that required database configuration is missing.
	ErrDatabaseConfigMissing = errors.New("database config not found")

	// ErrMigrationFailed indicates that database migration failed.
	ErrMigrationFailed = errors.New("migration failed")

	// ErrOpenDatabase indicates that opening the database connection failed.
	ErrOpenDatabase = errors.New("failed to open database")

	// ErrPingDatabase indicates that pinging the database failed.
	ErrPingDatabase = errors.New("failed to ping database")

	// ErrCreateDatabase indicates that creating the database failed.
	ErrCreateDatabase = errors.New("failed to create database")

	// ErrInitDriver indicates that initializing the migration driver failed.
	ErrInitDriver = errors.New("failed to initialize migration driver")

	// ErrInitSource indicates that initializing the bindata source failed.
	ErrInitSource = errors.New("failed to initialize bindata source")

	// ErrCreateMigrateInstance indicates that creating the migrate instance failed.
	ErrCreateMigrateInstance = errors.New("failed to create migrate instance")

	// ErrRunMigration indicates that running migrations failed.
	ErrRunMigration = errors.New("failed to run migration")
)
