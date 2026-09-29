package comm

import "errors"

// Sentinel errors for comm package
var (
	// ErrInvalidDriver is returned when an unsupported database driver is specified.
	ErrInvalidDriver = errors.New("unsupported datasource driver")
	// ErrMissingDriver is returned when the datasource driver is not specified.
	ErrMissingDriver = errors.New("datasource.driver is required")
	// ErrMissingURL is returned when the datasource URL is not specified.
	ErrMissingURL = errors.New("datasource.url is required")
	// ErrInvalidRunLimit is returned when server.runLimit is negative.
	ErrInvalidRunLimit = errors.New("server.runLimit must be non-negative")
	// ErrFindCountInvalidData is returned when findCount receives invalid data type.
	ErrFindCountInvalidData = errors.New("findCount: data must be a non-nil pointer to a slice")
	// ErrFindCountNotSlice is returned when findCount receives non-slice type.
	ErrFindCountNotSlice = errors.New("findCount: expected pointer to slice")
	// ErrFindPagesMissingOrderBy is returned when FindPages SQL lacks ORDER BY.
	ErrFindPagesMissingOrderBy = errors.New("FindPages: SQL must contain ORDER BY clause")
)
