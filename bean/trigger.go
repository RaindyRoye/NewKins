package bean

import "errors"

// Sentinel errors for trigger parameter validation.
// These can be used with errors.Is to check specific validation failures.
var (
	ErrPipelineIDRequired    = errors.New("pipeline ID is required")
	ErrTriggerTypeRequired   = errors.New("trigger type is required")
	ErrTriggerNameRequired   = errors.New("trigger name is required")
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

// Check validates that all required fields are present.
// Returns sentinel errors that can be checked with errors.Is.
func (c *TriggerParam) Check() error {
	if c.PipelineId == "" {
		return ErrPipelineIDRequired
	}
	if c.Types == "" {
		return ErrTriggerTypeRequired
	}
	if c.Name == "" {
		return ErrTriggerNameRequired
	}
	if c.Params == "" {
		return ErrTriggerParamsRequired
	}
	return nil
}
