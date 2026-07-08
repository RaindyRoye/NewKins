package engine

import "errors"

// Sentinel errors for the engine package.
//
// These errors can be used with errors.Is() to check for specific error
// conditions without relying on exact error message strings. All sentinel
// errors are wrapped using fmt.Errorf with %w in the call sites, so
// errors.Is() works through the error chain.
//
// Usage in call sites:
//
//	return fmt.Errorf("update: %w: %q", ErrBuildNotFound, buildID)
//
// Usage in callers/tests:
//
//	if errors.Is(err, engine.ErrBuildNotFound) { ... }
var (
	// ErrBuildNotFound is returned when a build ID does not match any active build.
	ErrBuildNotFound = errors.New("build not found")

	// ErrJobNotFound is returned when a job ID does not match any job within the specified build.
	ErrJobNotFound = errors.New("job not found")

	// ErrCmdNotFound is returned when a command ID does not match any command within the specified job.
	ErrCmdNotFound = errors.New("cmd not found")

	// ErrInvalidFSType is returned when a filesystem type code does not resolve to any known path.
	ErrInvalidFSType = errors.New("invalid filesystem type, no path resolved")

	// ErrEmptyParams is returned when required function parameters are empty or nil.
	ErrEmptyParams = errors.New("required parameters must not be empty")

	// ErrArtifactoryNotFound is returned when an artifactory identifier does not match any record.
	ErrArtifactoryNotFound = errors.New("artifactory not found")

	// ErrArtifactNotFound is returned when an artifact package or version is not found.
	ErrArtifactNotFound = errors.New("artifact not found")

	// ErrPermissionDenied is returned when a user lacks the required permissions.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrPluginNotFound is returned when a step plugin is not registered in the job engine.
	ErrPluginNotFound = errors.New("plugin not found")
)
