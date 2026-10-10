package bean

import "errors"

// Sentinel errors for TriggerParam validation. Use errors.Is to check
// for specific validation failures programmatically.
var (
	ErrPipelineIdRequired    = errors.New("pipeline ID is required")
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
