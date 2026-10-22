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

	// ErrInvalidTriggerType is returned when a trigger's type field does not match the expected value.
	ErrInvalidTriggerType = errors.New("invalid trigger type")

	// ErrArtifactoryDisabled is returned when an operation targets a disabled artifactory.
	ErrArtifactoryDisabled = errors.New("artifactory is disabled")

	// ErrUnknownWebhookType is returned when a webhook type string does not match any supported provider.
	ErrUnknownWebhookType = errors.New("unknown webhook type")

	// ErrAssetNotFound is returned when a requested embedded asset is not found.
	ErrAssetNotFound = errors.New("asset not found")

	// ErrInvalidConfig is returned when configuration validation fails.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrDuplicateEntry is returned when a duplicate entry is detected during validation.
	ErrDuplicateEntry = errors.New("duplicate entry")

	// ErrRepositoryNil is returned when a repository reference is unexpectedly nil.
	ErrRepositoryNil = errors.New("repository is nil")

	// ErrRepoNil is returned when a build's repo parameter is nil.
	ErrRepoNil = errors.New("repo parameter is nil")

	// ErrStagesEmpty is returned when a build has no stages defined.
	ErrStagesEmpty = errors.New("build stages is empty")

	// ErrStageBuildIdMismatch is returned when a stage's BuildId does not match the build's Id.
	ErrStageBuildIdMismatch = errors.New("stage build id mismatch")

	// ErrStageNameEmpty is returned when a stage name is empty.
	ErrStageNameEmpty = errors.New("stage name is empty")

	// ErrStageDuplicate is returned when two stages share the same name.
	ErrStageDuplicate = errors.New("duplicate stage name")

	// ErrJobBuildIdMismatch is returned when a job's BuildId does not match the build's Id.
	ErrJobBuildIdMismatch = errors.New("job build id mismatch")

	// ErrJobStageIdMismatch is returned when a job's StageId does not match the stage's Id.
	ErrJobStageIdMismatch = errors.New("job stage id mismatch")

	// ErrStepPluginEmpty is returned when a step's plugin field is empty.
	ErrStepPluginEmpty = errors.New("step plugin is empty")

	// ErrStepNameEmpty is returned when a step name is empty.
	ErrStepNameEmpty = errors.New("step name is empty")

	// ErrJobDuplicate is returned when two jobs in the same stage share the same name.
	ErrJobDuplicate = errors.New("duplicate job name")

	// ErrJobCommands is returned when a job's commands cannot be parsed.
	ErrJobCommands = errors.New("job commands generation failed")

	// ErrStageNotFound is returned when a runtime stage is not found in the build's stage map.
	ErrStageNotFound = errors.New("stage not found in build")

	// ErrWaitNotFound is returned when a wait dependency (step name) is not found in the stage.
	ErrWaitNotFound = errors.New("wait dependency not found")

	// ErrWaitSelf is returned when a step declares a wait dependency on itself.
	ErrWaitSelf = errors.New("wait dependency references self")

	// ErrWaitFailed is returned when a waited-on step ends in error.
	ErrWaitFailed = errors.New("waited step ended in error")

	// ErrJobPut is returned when putting a job into the job engine fails.
	ErrJobPut = errors.New("failed to put job into engine")

	// ErrCreateBuildPath is returned when the build working directory cannot be created.
	ErrCreateBuildPath = errors.New("failed to create build path")

	// ErrCloneRepo is returned when a git clone operation fails.
	ErrCloneRepo = errors.New("failed to clone repository")
)
