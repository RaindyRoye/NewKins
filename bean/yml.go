package bean

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for YAML pipeline validation.
// Use errors.Is() to check for specific conditions.
var (
	// ErrStagesEmpty is returned when the pipeline has no stages defined.
	ErrStagesEmpty = errors.New("stages is empty")
	// ErrStageNameEmpty is returned when a stage has no name.
	ErrStageNameEmpty = errors.New("stage name is empty")
	// ErrStepsEmpty is returned when a stage has no steps defined.
	ErrStepsEmpty = errors.New("steps is empty")
	// ErrStepPluginEmpty is returned when a step has no plugin specified.
	ErrStepPluginEmpty = errors.New("step plugin is empty")
	// ErrStepNameEmpty is returned when a step has no name.
	ErrStepNameEmpty = errors.New("step name is empty")
)

type Pipeline struct {
	Version  string              `yaml:"version,omitempty" json:"version"`
	Triggers map[string]*Trigger `yaml:"triggers,omitempty" json:"triggers"`
	Vars     map[string]string   `yaml:"vars,omitempty" json:"vars"`
	Stages   []*Stage            `yaml:"stages,omitempty" json:"stages"`
}

type Trigger struct {
	AutoCancel     bool       `yaml:"autoCancel,omitempty" json:"autoCancel,omitempty"`
	Timeout        string     `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Branches       *Condition `yaml:"branches,omitempty" json:"branches,omitempty"`
	Tags           *Condition `yaml:"tags,omitempty" json:"tags,omitempty"`
	Paths          *Condition `yaml:"paths,omitempty" json:"paths,omitempty"`
	Notes          *Condition `yaml:"notes,omitempty" json:"notes,omitempty"`
	CommitMessages *Condition `yaml:"commitMessages,omitempty" json:"commitMessages,omitempty"`
}

type Condition struct {
	Include []string `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`
}

type Stage struct {
	Stage       string  `yaml:"stage" json:"stage"`
	Name        string  `yaml:"name,omitempty" json:"name"`
	DisplayName string  `yaml:"displayName,omitempty" json:"displayName"`
	Repo        string  `yaml:"repo,omitempty" json:"repo"`
	Steps       []*Step `yaml:"steps,omitempty" json:"steps"`
}

type Step struct {
	Step         string            `yaml:"step" json:"step"`
	DisplayName  string            `yaml:"displayName,omitempty" json:"displayName"`
	Name         string            `yaml:"name,omitempty" json:"name"`
	Disable      bool              `yaml:"disable,omitempty" json:"disable"`
	MustCopy     bool              `yaml:"mustCopy,omitempty" json:"mustCopy"`
	Repo         string            `yaml:"repo,omitempty" json:"repo"`
	Input        map[string]string `yaml:"input,omitempty" json:"input"`
	Env          map[string]string `yaml:"env,omitempty" json:"env"`
	Commands     any               `yaml:"commands,omitempty" json:"commands"`
	Waits        []string          `yaml:"wait,omitempty" json:"wait"`
	Image        string            `yaml:"image,omitempty" json:"image"`
	Artifacts    []*Artifact       `yaml:"artifacts,omitempty" json:"artifacts"`
	UseArtifacts []*UseArtifacts   `yaml:"useArtifacts,omitempty" json:"useArtifacts"`
}

type Artifact struct {
	Scope      string `yaml:"scope,omitempty" json:"scope"`
	Repository string `yaml:"repository,omitempty" json:"repository"`
	Name       string `yaml:"name,omitempty" json:"name"`
	Path       string `yaml:"path,omitempty" json:"path"`
}

type UseArtifacts struct {
	Scope      string `yaml:"scope" json:"scope"`
	Repository string `yaml:"repository" json:"repository"`
	Name       string `yaml:"name" json:"name"`
	IsUrl      bool   `yaml:"isUrl" json:"isUrl"`
	Alias      string `yaml:"alias" json:"alias"`
	Path       string `yaml:"path" json:"path"`

	FromStage string `yaml:"fromStage" json:"sourceStage"`
	FromStep  string `yaml:"fromStep" json:"sourceStep"`
}

func (c *Pipeline) ToJson() ([]byte, error) {
	c.ConvertCmd()
	return json.Marshal(c)
}

func (c *Pipeline) ConvertCmd() {
	for _, stage := range c.Stages {
		for _, step := range stage.Steps {
			switch v := step.Commands.(type) {
			case string:
				step.Commands = v
			case []any:
				ls := make([]string, 0, len(v))
				for _, v1 := range v {
					ls = append(ls, fmt.Sprintf("%v", v1))
				}
				step.Commands = ls
			default:
				if v != nil {
					step.Commands = fmt.Sprintf("%v", v)
				}
			}
		}
	}
}

func (c *Pipeline) Check() error {
	stages := make(map[string]map[string]*Step)
	if len(c.Stages) == 0 {
		return fmt.Errorf("pipeline check: %w", ErrStagesEmpty)
	}
	for _, v := range c.Stages {
		if v.Name == "" {
			return fmt.Errorf("pipeline check: %w", ErrStageNameEmpty)
		}
		if len(v.Steps) == 0 {
			return fmt.Errorf("pipeline check: %w", ErrStepsEmpty)
		}
		if _, ok := stages[v.Name]; ok {
			return fmt.Errorf("duplicate stage name: %s", v.Name)
		}
		m := map[string]*Step{}
		stages[v.Name] = m
		for _, e := range v.Steps {
			if strings.TrimSpace(e.Step) == "" {
				return fmt.Errorf("pipeline check: %w", ErrStepPluginEmpty)
			}
			if e.Name == "" {
				return fmt.Errorf("pipeline check: %w", ErrStepNameEmpty)
			}
			if _, ok := m[e.Name]; ok {
				return fmt.Errorf("duplicate step name: %s", e.Name)
			}
			m[e.Name] = e
		}
	}
	return nil
}
