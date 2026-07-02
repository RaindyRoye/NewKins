package engine

import "errors"

// Sentinel errors for the engine package.
// These can be used with errors.Is() for programmatic error checking.
var (
	// ErrBuildNotFound is returned when a build ID does not exist in the build engine.
	ErrBuildNotFound = errors.New("build not found")

	// ErrJobNotFound is returned when a job ID does not exist in the build.
	ErrJobNotFound = errors.New("job not found")

	// ErrCmdNotFound is returned when a command ID does not exist in the job.
	ErrCmdNotFound = errors.New("command not found")

	// ErrPluginNotFound is returned when a step plugin is not registered.
	ErrPluginNotFound = errors.New("plugin not found")

	// ErrNoJobAvailable is returned when no job is available for the runner after timeout.
	ErrNoJobAvailable = errors.New("no job available")

	// ErrInvalidFilesystem is returned when an invalid filesystem type is specified.
	ErrInvalidFilesystem = errors.New("invalid filesystem type")

	// ErrEmptyParameter is returned when a required parameter is empty.
	ErrEmptyParameter = errors.New("required parameter is empty")

	// ErrArtifactoryNotFound is returned when an artifactory identifier is not found.
	ErrArtifactoryNotFound = errors.New("artifactory not found")

	// ErrArtifactNotFound is returned when an artifact package or version is not found.
	ErrArtifactNotFound = errors.New("artifact not found")

	// ErrPermissionDenied is returned when the user lacks required permissions.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrTimerNotFound is returned when a timer ID does not exist.
	ErrTimerNotFound = errors.New("timer not found")

	// ErrInvalidTriggerType is returned when a trigger has an unexpected type.
	ErrInvalidTriggerType = errors.New("invalid trigger type")

	// ErrStepPluginEmpty is returned when a step's plugin field is empty.
	ErrStepPluginEmpty = errors.New("step plugin empty")
)
