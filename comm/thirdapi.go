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

// apiClients caches third-party API clients keyed by "provider:host".
// This replaces the old single-global-client design which incorrectly
// returned a cached github client when asked for a different provider.
var (
	apiClients   = make(map[string]*thirdapi.Client)
	apiClientsMu sync.RWMutex
)

// clientCacheKey builds the cache key for a provider+host pair.
func clientCacheKey(provider, host string) string {
	return provider + ":" + host
}

// ResetAPIClients clears the third-party API client cache.
// This is primarily intended for use in tests to ensure isolation
// between test cases.
func ResetAPIClients() {
	apiClientsMu.Lock()
	apiClients = make(map[string]*thirdapi.Client)
	apiClientsMu.Unlock()
}

// GetThirdApi returns a cached (or newly created) third-party API client
// for the specified provider and host.
//
// Supported providers: "gitee", "github", "gitlab", "giteepremium", "gitea".
// Unknown providers fall back to the default GitHub client.
//
// Each unique (provider, host) pair gets its own cached client instance.
// This is safe for concurrent use.
func GetThirdApi(s string, host string) (*thirdapi.Client, error) {
	key := clientCacheKey(s, host)

	// Fast path: check cache under read lock.
	apiClientsMu.RLock()
	cached, ok := apiClients[key]
	apiClientsMu.RUnlock()
	if ok {
		return cached, nil
	}

	// Slow path: create a new client under write lock.
	apiClientsMu.Lock()
	defer apiClientsMu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have created it).
	if cached, ok := apiClients[key]; ok {
		return cached, nil
	}

	var client *thirdapi.Client
	switch s {
	case "gitee":
		client = giteeapi.NewDefault()
	case "github":
		client = githubapi.NewDefault()
	case "gitlab":
		c, err := gitlabapi.New(host + "/api/v4")
		if err != nil {
			return nil, fmt.Errorf("create gitlab client (host=%s): %w", host, err)
		}
		client = c
	case "giteepremium":
		c, err := giteepremiumapi.New(host + "/api/v5")
		if err != nil {
			return nil, fmt.Errorf("create giteepremium client (host=%s): %w", host, err)
		}
		client = c
	case "gitea":
		c, err := giteaapi.New(host + "/api/v1")
		if err != nil {
			return nil, fmt.Errorf("create gitea client (host=%s): %w", host, err)
		}
		client = c
	default:
		logrus.Debug("GetThirdApi default : 'github' ")
		client = githubapi.NewDefault()
	}

	apiClients[key] = client
	return client, nil
}
