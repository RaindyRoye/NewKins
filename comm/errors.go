package comm

import "errors"

// Sentinel errors for the comm package.
// These errors can be used with errors.Is() to check for specific error
// conditions without relying on exact error message strings.
var (
	// ErrConfigInvalid is returned when configuration validation fails.
	ErrConfigInvalid = errors.New("invalid configuration")

	// ErrFindCountInvalidArg is returned when findCount receives an invalid argument.
	ErrFindCountInvalidArg = errors.New("findCount: invalid argument")

	// ErrAssetNotFound is returned when a requested asset is not found.
	ErrAssetNotFound = errors.New("asset not found")

	// ErrSQLMissingOrderBy is returned when SQL queries lack required ORDER BY clauses.
	ErrSQLMissingOrderBy = errors.New("SQL missing ORDER BY clause")
)
