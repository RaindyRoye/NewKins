package bean

import "errors"

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
		return errors.New("pipeline ID is required")
	}
	if c.Types == "" {
		return errors.New("trigger type is required")
	}
	if c.Name == "" {
		return errors.New("trigger name is required")
	}
	if c.Params == "" {
		return errors.New("trigger params is required")
	}
	return nil
}
