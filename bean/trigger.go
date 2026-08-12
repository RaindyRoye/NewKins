package bean

import "fmt"

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
		return fmt.Errorf("%w: pipelineId", ErrTriggerFieldEmpty)
	}
	if c.Types == "" {
		return fmt.Errorf("%w: types", ErrTriggerFieldEmpty)
	}
	if c.Name == "" {
		return fmt.Errorf("%w: name", ErrTriggerFieldEmpty)
	}
	if c.Params == "" {
		return fmt.Errorf("%w: params", ErrTriggerFieldEmpty)
	}
	return nil
}
