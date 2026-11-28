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
	"github.com/sirupsen/logrus"
)

var (
	apiClient    *thirdapi.Client
	apiClientMux sync.Mutex
)

// GetThirdApi returns the cached third-party API client, creating one if
// necessary. It is safe for concurrent use. The first successful call
// caches the client; subsequent calls return the same instance regardless
// of the provider argument.
func GetThirdApi(s string, host string) (*thirdapi.Client, error) {
	apiClientMux.Lock()
	defer apiClientMux.Unlock()

	if apiClient != nil {
		return apiClient, nil
	}
	switch s {
	case "gitee":
		apiClient = giteeapi.NewDefault()
	case "github":
		apiClient = githubapi.NewDefault()
	case "gitlab":
		client, err := gitlabapi.New(host + "/api/v4")
		if err != nil {
			return nil, fmt.Errorf("create gitlab client (host=%s): %w", host, err)
		}
		apiClient = client
	case "giteepremium":
		client, err := giteepremiumapi.New(host + "/api/v5")
		if err != nil {
			return nil, fmt.Errorf("create giteepremium client (host=%s): %w", host, err)
		}
		apiClient = client
	case "gitea":
		client, err := giteaapi.New(host + "/api/v1")
		if err != nil {
			return nil, fmt.Errorf("create gitea client (host=%s): %w", host, err)
		}
		apiClient = client
	default:
		logrus.Debug("GetThirdApi default : 'github' ")
		apiClient = githubapi.NewDefault()
	}
	return apiClient, nil
}
