package bean

import (
	"errors"
	"fmt"
)

// Sentinel errors for trigger parameter validation.
// These can be used with errors.Is() for programmatic error checking.
var (
	// ErrPipelineIdRequired is returned when the pipeline ID field is empty.
	ErrPipelineIdRequired = errors.New("pipeline ID is required")

	// ErrTriggerTypeRequired is returned when the trigger type field is empty.
	ErrTriggerTypeRequired = errors.New("trigger type is required")

	// ErrTriggerNameRequired is returned when the trigger name field is empty.
	ErrTriggerNameRequired = errors.New("trigger name is required")

	// ErrTriggerParamsRequired is returned when the trigger params field is empty.
	ErrTriggerParamsRequired = errors.New("trigger params is required")
)

type TriggerParam struct {
	Id         string `json:"id"`
	PipelineId string `json:"pipelineId"`
	Types      string `json:"types"`
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	Params     string `json:"params"`
	Enabled    bool   ` json:"enabled"`
}

// Check validates the trigger parameters and returns a descriptive error
// wrapping the appropriate sentinel error if validation fails.
func (c *TriggerParam) Check() error {
	if c == nil {
		return fmt.Errorf("trigger param is nil")
	}
	if c.PipelineId == "" {
		return fmt.Errorf("validate trigger: %w", ErrPipelineIdRequired)
	}
	if c.Types == "" {
		return fmt.Errorf("validate trigger: %w", ErrTriggerTypeRequired)
	}
	if c.Name == "" {
		return fmt.Errorf("validate trigger: %w", ErrTriggerNameRequired)
	}
	if c.Params == "" {
		return fmt.Errorf("validate trigger: %w", ErrTriggerParamsRequired)
	}
	return nil
}
