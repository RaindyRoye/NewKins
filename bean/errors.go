package bean

import "errors"

// Sentinel errors for the bean package.
//
// These errors can be used with errors.Is() to check for specific error
// conditions without relying on exact error message strings. All sentinel
// errors are wrapped using fmt.Errorf with %w in the call sites, so
// errors.Is() works through the error chain.
//
// Usage in call sites:
//
//	return fmt.Errorf("duplicate stage name: %s: %w", v.Name, ErrDuplicateStage)
//
// Usage in callers/tests:
//
//	if errors.Is(err, bean.ErrDuplicateStage) { ... }
var (
	// ErrStagesEmpty is returned when a pipeline has no stages.
	ErrStagesEmpty = errors.New("stages is empty")

	// ErrStageNameEmpty is returned when a stage has an empty name.
	ErrStageNameEmpty = errors.New("stage name is empty")

	// ErrStepsEmpty is returned when a stage has no steps.
	ErrStepsEmpty = errors.New("steps is empty")

	// ErrStepPluginEmpty is returned when a step has an empty plugin name.
	ErrStepPluginEmpty = errors.New("step plugin is empty")

	// ErrStepNameEmpty is returned when a step has an empty name.
	ErrStepNameEmpty = errors.New("step name is empty")

	// ErrDuplicateStage is returned when two stages share the same name.
	ErrDuplicateStage = errors.New("duplicate stage name")

	// ErrDuplicateStep is returned when two steps within the same stage share the same name.
	ErrDuplicateStep = errors.New("duplicate step name")

	// ErrTriggerFieldEmpty is returned when a required TriggerParam field is empty.
	// The wrapped error message identifies which specific field is missing.
	ErrTriggerFieldEmpty = errors.New("trigger field is required")
)
