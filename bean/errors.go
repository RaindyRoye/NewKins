package bean

import "errors"

// Sentinel errors for pipeline YAML validation.
// These errors can be checked with errors.Is() for programmatic error handling.
var (
	// ErrStagesEmpty indicates that the pipeline has no stages defined.
	ErrStagesEmpty = errors.New("stages is empty")

	// ErrStageNameEmpty indicates that a stage has no name.
	ErrStageNameEmpty = errors.New("stage name is empty")

	// ErrStepsEmpty indicates that a stage has no steps defined.
	ErrStepsEmpty = errors.New("steps is empty")

	// ErrStepPluginEmpty indicates that a step has no plugin specified.
	ErrStepPluginEmpty = errors.New("step plugin is empty")

	// ErrStepNameEmpty indicates that a step has no name.
	ErrStepNameEmpty = errors.New("step name is empty")

	// ErrDuplicateStage indicates that a stage name is duplicated.
	ErrDuplicateStage = errors.New("duplicate stage name")

	// ErrDuplicateStep indicates that a step name is duplicated within a stage.
	ErrDuplicateStep = errors.New("duplicate step name")
)

// Sentinel errors for trigger validation.
var (
	// ErrPipelineIdRequired indicates that the pipeline ID is missing.
	ErrPipelineIdRequired = errors.New("pipeline ID is required")

	// ErrTriggerTypeRequired indicates that the trigger type is missing.
	ErrTriggerTypeRequired = errors.New("trigger type is required")

	// ErrTriggerNameRequired indicates that the trigger name is missing.
	ErrTriggerNameRequired = errors.New("trigger name is required")

	// ErrTriggerParamsRequired indicates that the trigger params are missing.
	ErrTriggerParamsRequired = errors.New("trigger params is required")
)
