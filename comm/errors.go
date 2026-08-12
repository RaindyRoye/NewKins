package comm

import "errors"

// Sentinel errors for the comm package.
// Callers can use errors.Is() to check for these specific conditions
// without relying on string matching.

// ErrAssetNotFound is returned when a requested embedded asset cannot be found.
var ErrAssetNotFound = errors.New("asset not found")

// ErrDecompressionLimit is returned when a compressed asset exceeds the
// maximum allowed decompressed size (10 MiB). This protects against
// decompression bombs (CWE-409 / G110).
var ErrDecompressionLimit = errors.New("decompressed size exceeds limit")

// ErrInvalidDataType is returned when findCount receives a non-pointer
// or non-slice data argument.
var ErrInvalidDataType = errors.New("invalid data type")

// ErrPageGenMissingOrderBy is returned when FindPages is called with a
// PageGen whose SQL does not contain a required "\nORDER BY" clause.
var ErrPageGenMissingOrderBy = errors.New("SQL missing ORDER BY clause")
