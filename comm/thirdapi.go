package comm

import (
	"fmt"
	"sync"

	"github.com/gokins/gokins/thirdapi"
	"github.com/gokins/gokins/thirdapi/giteaapi"
	"github.com/gokins/gokins/thirdapi/giteeapi"
	"github.com/gokins/gokins/thirdapi/giteepremiumapi"
	"github.com/gokins/gokins/thirdapi/githubapi"
	"github.com/gokins/gokins/thirdapi/gitlabapi"
)

var (
	apiClient     *thirdapi.Client
	apiClientOnce sync.Once
	apiClientErr  error
)

// ResetThirdApiClient clears the cached API client so the next call to
// GetThirdApi creates a fresh one. This is primarily useful in tests.
func ResetThirdApiClient() {
	apiClientOnce = sync.Once{}
	apiClient = nil
	apiClientErr = nil
}

// GetThirdApi returns a cached third-party API client for the given provider.
// The client is created lazily on first call and reused thereafter.
// Supported providers: "gitee", "github", "gitlab", "giteepremium", "gitea".
// Falls back to GitHub if the provider is unrecognized.
func GetThirdApi(s string, host string) (*thirdapi.Client, error) {
	apiClientOnce.Do(func() {
		var client *thirdapi.Client
		switch s {
		case "gitee":
			client = giteeapi.NewDefault()
		case "github":
			client = githubapi.NewDefault()
		case "gitlab":
			c, err := gitlabapi.New(host + "/api/v4")
			if err != nil {
				apiClientErr = fmt.Errorf("create gitlab client (host=%s): %w", host, err)
				return
			}
			client = c
		case "giteepremium":
			c, err := giteepremiumapi.New(host + "/api/v5")
			if err != nil {
				apiClientErr = fmt.Errorf("create giteepremium client (host=%s): %w", host, err)
				return
			}
			client = c
		case "gitea":
			c, err := giteaapi.New(host + "/api/v1")
			if err != nil {
				apiClientErr = fmt.Errorf("create gitea client (host=%s): %w", host, err)
				return
			}
			client = c
		default:
			client = githubapi.NewDefault()
		}
		apiClient = client
	})
	return apiClient, apiClientErr
}
