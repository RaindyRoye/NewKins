package bean

import "errors"

// Sentinel errors for the bean package.
//
// These errors can be used with errors.Is() to check for specific error
// conditions without relying on exact error message strings. Call sites
// wrap them using fmt.Errorf with %w so errors.Is() works through the chain.
//
// Usage in call sites:
//
//	return fmt.Errorf("%w: %s", ErrDuplicateStage, v.Name)
//
// Usage in callers/tests:
//
//	if errors.Is(err, bean.ErrDuplicateStage) { ... }
var (
	// ErrDuplicateStage is returned when two stages share the same name.
	ErrDuplicateStage = errors.New("duplicate stage name")

	// ErrDuplicateStep is returned when two steps within the same stage share the same name.
	ErrDuplicateStep = errors.New("duplicate step name")
)
