package comm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetThirdApi_Gitee(t *testing.T) {
	// Reset global state
	origClient := apiClient
	apiClient = nil
	defer func() { apiClient = origClient }()

	client, err := GetThirdApi("gitee", "")
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestGetThirdApi_GitHub(t *testing.T) {
	origClient := apiClient
	apiClient = nil
	defer func() { apiClient = origClient }()

	client, err := GetThirdApi("github", "")
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestGetThirdApi_GitLab(t *testing.T) {
	origClient := apiClient
	apiClient = nil
	defer func() { apiClient = origClient }()

	client, err := GetThirdApi("gitlab", "https://gitlab.example.com")
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestGetThirdApi_GiteePremium(t *testing.T) {
	origClient := apiClient
	apiClient = nil
	defer func() { apiClient = origClient }()

	client, err := GetThirdApi("giteepremium", "https://gitee.example.com")
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestGetThirdApi_Gitea(t *testing.T) {
	origClient := apiClient
	apiClient = nil
	defer func() { apiClient = origClient }()

	client, err := GetThirdApi("gitea", "https://gitea.example.com")
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestGetThirdApi_Default(t *testing.T) {
	origClient := apiClient
	apiClient = nil
	defer func() { apiClient = origClient }()

	client, err := GetThirdApi("unknown", "")
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestGetThirdApi_Cached(t *testing.T) {
	origClient := apiClient
	apiClient = nil
	defer func() { apiClient = origClient }()

	// First call creates the client
	client1, err := GetThirdApi("github", "")
	assert.NoError(t, err)
	assert.NotNil(t, client1)

	// Second call should return the same cached client
	client2, err := GetThirdApi("gitee", "")
	assert.NoError(t, err)
	assert.NotNil(t, client2)

	// Should be the same instance (cached)
	assert.Equal(t, client1, client2)
}
