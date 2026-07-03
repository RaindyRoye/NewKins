package engine

import "errors"

// Sentinel errors for the engine package. Callers can use errors.Is() to
// check for specific failure conditions without relying on error message text.
//
// These cover the most common error categories in the runner, timer, and
// job engines. Additional sentinel errors can be added here as new error
// categories emerge.
var (
	// ErrBuildNotFound is returned when a build ID does not match any
	// active build task in the build engine.
	ErrBuildNotFound = errors.New("build not found")

	// ErrJobNotFound is returned when a job ID does not match any job
	// within the specified build.
	ErrJobNotFound = errors.New("job not found")

	// ErrCmdNotFound is returned when a command ID does not match any
	// command within the specified job.
	ErrCmdNotFound = errors.New("cmd not found")

	// ErrInvalidInput is returned when a required parameter is empty
	// or nil.
	ErrInvalidInput = errors.New("invalid input")

	// ErrArtifactoryNotFound is returned when an artifactory identifier
	// does not match any configured repository.
	ErrArtifactoryNotFound = errors.New("artifactory not found")

	// ErrArtifactNotFound is returned when an artifact package or version
	// is not found in the database.
	ErrArtifactNotFound = errors.New("artifact not found")

	// ErrPluginNotFound is returned when a step's plugin identifier does
	// not match any registered executor.
	ErrPluginNotFound = errors.New("plugin not found")

	// ErrTriggerNotFound is returned when a trigger ID does not match any
	// enabled trigger.
	ErrTriggerNotFound = errors.New("trigger not found or disabled")

	// ErrNoJobAvailable is returned when PullJob times out without finding
	// a matching job for the requested runner and plugins.
	ErrNoJobAvailable = errors.New("no job available")
)
