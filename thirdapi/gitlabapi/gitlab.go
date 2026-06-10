package gitlabapi

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gokins/gokins/thirdapi"
)

func New(uri string) (*thirdapi.Client, error) {
	base, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("gitlab: parse API URL: %w", err)
	}
	client := &wrapper{new(thirdapi.Client)}
	client.BaseURL = base
	c := &http.Client{
		Timeout: time.Second * 8,
	}
	client.HttpClient = c
	client.Repositories = &RepositoryService{client}
	return client.Client, nil
}

func NewDefault() *thirdapi.Client {
	client, _ := New(BaseApiGitlab)
	return client
}

type wrapper struct {
	*thirdapi.Client
}
