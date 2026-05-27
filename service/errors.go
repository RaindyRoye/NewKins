package service

import "errors"

// Sentinel errors for the service package.
// These can be used with errors.Is() for programmatic error checking.
var (
	// ErrPipelineNotFound is returned when a pipeline configuration is not found in the database.
	ErrPipelineNotFound = errors.New("流水线不存在")

	// ErrPipelineYmlEmpty is returned when a pipeline's YAML content is empty.
	ErrPipelineYmlEmpty = errors.New("流水线Yaml为空")

	// ErrTriggerNoParams is returned when a trigger has no configuration parameters.
	ErrTriggerNoParams = errors.New("触发器没有配置参数")

	// ErrHookTypeEmpty is returned when the hookType parameter is missing from trigger config.
	ErrHookTypeEmpty = errors.New("hookType为空")

	// ErrWebhookParseFailed is returned when the webhook payload cannot be parsed.
	ErrWebhookParseFailed = errors.New("webhook解析失败")

	// ErrWebhookEventMismatch is returned when the webhook event type does not match the filter.
	ErrWebhookEventMismatch = errors.New("webhook事件不匹配")

	// ErrBranchMismatch is returned when the webhook branch does not match the filter.
	ErrBranchMismatch = errors.New("分支不匹配")

	// ErrTriggerNoSecret is returned when a trigger requires a secret but none is configured.
	ErrTriggerNoSecret = errors.New("触发器没有配置密钥")

	// ErrTriggerSecretMismatch is returned when the provided secret does not match the configured one.
	ErrTriggerSecretMismatch = errors.New("密钥不正确")

	// ErrPermissionDenied is returned when the trigger creator lacks required permissions.
	ErrPermissionDenied = errors.New("触发器创建者没有权限")

	// ErrParamDataNil is returned when parameter data is nil.
	ErrParamDataNil = errors.New("data is nil")

	// ErrParamNotFound is returned when a parameter is not found.
	ErrParamNotFound = errors.New("not found param")
)
