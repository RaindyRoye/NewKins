package migrates

import "errors"

// ErrDatabaseConfigMissing is returned when required database connection
// parameters (host, database name, or user) are empty.
// Callers can use errors.Is to check for this sentinel error.
var ErrDatabaseConfigMissing = errors.New("database config not found")
