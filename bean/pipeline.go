package bean

import "errors"

// Sentinel errors for NewPipeline validation.
var (
	ErrPipelineNameRequired    = errors.New("pipeline name is required")
	ErrPipelineContentRequired = errors.New("pipeline content is required")
)

type NewPipeline struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	Content     string            `json:"content"`
	OrgId       string            `json:"orgId"`
	AccessToken string            `json:"accessToken"`
	Url         string            `json:"url"`
	Username    string            `json:"username"`
	Vars        []*NewPipelineVar `json:"vars"`
}

type NewPipelineVar struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Remarks string `json:"remarks"`
	Public  bool   `json:"public"`
}

// Check validates that the required fields Name and Content are non-empty.
// Returns nil on success, or a sentinel error describing the missing field.
func (p *NewPipeline) Check() error {
	if p.Name == "" {
		return ErrPipelineNameRequired
	}
	if p.Content == "" {
		return ErrPipelineContentRequired
	}
	return nil
}
