package bean

import (
	"errors"
	"fmt"
)

// Sentinel errors for trigger validation.
// Use errors.Is() to check for specific conditions.
var (
	// ErrTriggerPipelineIDRequired is returned when PipelineId is empty.
	ErrTriggerPipelineIDRequired = errors.New("pipeline ID is required")
	// ErrTriggerTypeRequired is returned when Types is empty.
	ErrTriggerTypeRequired = errors.New("trigger type is required")
	// ErrTriggerNameRequired is returned when Name is empty.
	ErrTriggerNameRequired = errors.New("trigger name is required")
	// ErrTriggerParamsRequired is returned when Params is empty.
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

func (c *TriggerParam) Check() error {
	if c.PipelineId == "" {
		return fmt.Errorf("trigger check: %w", ErrTriggerPipelineIDRequired)
	}
	if c.Types == "" {
		return fmt.Errorf("trigger check: %w", ErrTriggerTypeRequired)
	}
	if c.Name == "" {
		return fmt.Errorf("trigger check: %w", ErrTriggerNameRequired)
	}
	if c.Params == "" {
		return fmt.Errorf("trigger check: %w", ErrTriggerParamsRequired)
	}
	return nil
}
