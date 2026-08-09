package httpex

import "errors"

// Sentinel errors for the httpex package.
// These can be used with errors.Is() for programmatic error checking.
var (
	// ErrResultNil is returned when a nil result pointer is passed to PostResult or PostJSONResult.
	ErrResultNil = errors.New("result is nil")
)
