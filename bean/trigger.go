package bean

import "errors"

// Sentinel errors for trigger validation.
//
// These errors can be used with errors.Is() to check for specific validation
// failures without relying on exact error message strings.
var (
	// ErrPipelineIdRequired is returned when a trigger has no pipeline ID.
	ErrPipelineIdRequired = errors.New("pipeline ID is required")
	// ErrTriggerTypeRequired is returned when a trigger has no type.
	ErrTriggerTypeRequired = errors.New("trigger type is required")
	// ErrTriggerNameRequired is returned when a trigger has no name.
	ErrTriggerNameRequired = errors.New("trigger name is required")
	// ErrTriggerParamsRequired is returned when a trigger has no params.
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
		return ErrPipelineIdRequired
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
