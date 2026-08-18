package bean

import "errors"

// Sentinel errors for the bean package.
// These errors can be used with errors.Is() to check for specific error
// conditions without relying on exact error message strings.
var (
	// ErrStagesEmpty is returned when a pipeline has no stages defined.
	ErrStagesEmpty = errors.New("stages is empty")

	// ErrStageNameEmpty is returned when a stage name is missing.
	ErrStageNameEmpty = errors.New("stage name is empty")

	// ErrStepsEmpty is returned when a stage has no steps defined.
	ErrStepsEmpty = errors.New("steps is empty")

	// ErrStepPluginEmpty is returned when a step plugin identifier is missing.
	ErrStepPluginEmpty = errors.New("step plugin is empty")

	// ErrStepNameEmpty is returned when a step name is missing.
	ErrStepNameEmpty = errors.New("step name is empty")

	// ErrDuplicateStageName is returned when two stages share the same name.
	ErrDuplicateStageName = errors.New("duplicate stage name")

	// ErrDuplicateStepName is returned when two steps share the same name within a stage.
	ErrDuplicateStepName = errors.New("duplicate step name")
)
