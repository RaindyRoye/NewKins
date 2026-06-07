package service

import "errors"

// Sentinel errors for the service package.
// These can be used with errors.Is() for programmatic error checking.
var (
	// ErrPipelineNotFound is returned when a pipeline configuration is not found in the database.
	ErrPipelineNotFound = errors.New("pipeline not found")

	// ErrPipelineYmlEmpty is returned when a pipeline's YAML content is empty.
	ErrPipelineYmlEmpty = errors.New("pipeline YAML content is empty")

	// ErrTriggerNoParams is returned when a trigger has no configuration parameters.
	ErrTriggerNoParams = errors.New("trigger has no configuration parameters")

	// ErrHookTypeEmpty is returned when the hookType parameter is missing from trigger config.
	ErrHookTypeEmpty = errors.New("hook type is empty")

	// ErrWebhookParseFailed is returned when the webhook payload cannot be parsed.
	ErrWebhookParseFailed = errors.New("failed to parse webhook payload")

	// ErrWebhookEventMismatch is returned when the webhook event type does not match the filter.
	ErrWebhookEventMismatch = errors.New("webhook event type does not match")

	// ErrBranchMismatch is returned when the webhook branch does not match the filter.
	ErrBranchMismatch = errors.New("branch does not match filter")

	// ErrTriggerNoSecret is returned when a trigger requires a secret but none is configured.
	ErrTriggerNoSecret = errors.New("trigger secret is not configured")

	// ErrTriggerSecretMismatch is returned when the provided secret does not match the configured one.
	ErrTriggerSecretMismatch = errors.New("trigger secret does not match")

	// ErrPermissionDenied is returned when the trigger creator lacks required permissions.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrParamDataNil is returned when parameter data is nil.
	ErrParamDataNil = errors.New("parameter data is nil")

	// ErrParamNotFound is returned when a parameter is not found.
	ErrParamNotFound = errors.New("parameter not found")
)
