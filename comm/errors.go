package comm

import "errors"

// Sentinel errors for the comm package.
//
// These errors can be used with errors.Is() to check for specific error
// conditions without relying on exact error message strings. Call sites
// wrap them using fmt.Errorf with %w so errors.Is() works through the chain.
//
// Usage in call sites:
//
//	return fmt.Errorf("findCount: %w: got %T", ErrInvalidArgType, data)
//
// Usage in callers/tests:
//
//	if errors.Is(err, comm.ErrInvalidArgType) { ... }
var (
	// ErrInvalidArgType is returned when a function receives an argument of an unexpected type.
	ErrInvalidArgType = errors.New("invalid argument type")

	// ErrMissingOrderBy is returned when a paged query is missing the required ORDER BY clause.
	ErrMissingOrderBy = errors.New("SQL must contain '\\nORDER BY' clause")

	// ErrAssetNotFound is returned when a requested embedded asset is not found.
	ErrAssetNotFound = errors.New("asset not found")

	// ErrDecompressionLimit is returned when a compressed asset exceeds the maximum allowed size.
	ErrDecompressionLimit = errors.New("decompressed size exceeds limit")

	// ErrDriverRequired is returned when the datasource driver field is empty.
	ErrDriverRequired = errors.New("datasource.driver is required")

	// ErrUnsupportedDriver is returned when the datasource driver is not one of the supported values.
	ErrUnsupportedDriver = errors.New("unsupported datasource driver")

	// ErrURLRequired is returned when the datasource URL field is empty.
	ErrURLRequired = errors.New("datasource.url is required")

	// ErrInvalidRunLimit is returned when server.runLimit is negative.
	ErrInvalidRunLimit = errors.New("server.runLimit must be non-negative")

	// ErrConfigNotFound is returned when no configuration file is found in the working directory.
	ErrConfigNotFound = errors.New("no configuration file found")

	// ErrConfigExists is returned when a configuration file already exists and --force is not set.
	ErrConfigExists = errors.New("configuration file already exists")
)
