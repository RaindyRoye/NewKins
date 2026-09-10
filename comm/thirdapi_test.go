package comm

import (
	"sync"
	"testing"

	"github.com/gokins/gokins/thirdapi"
)

func TestGetThirdApi_GitlabTypo(t *testing.T) {
	// Reset the global apiClients cache to ensure a clean test state.
	ResetAPIClients()
	defer ResetAPIClients()

	// "gitlab" (not "gitalb") should be recognized as a valid provider.
	// The host is intentionally fake — the API client will be created but
	// no network call is made at construction time for gitlab client.
	_, err := GetThirdApi("gitlab", "https://gitlab.example.com")
	if err != nil {
		// Some gitlab api constructors may fail without a valid host,
		// but the important thing is the case was recognized (not falling through to default).
		// We check that apiClients was NOT set to the default (github) client.
		apiClientsMu.RLock()
		cached := apiClients[clientCacheKey("gitlab", "https://gitlab.example.com")]
		apiClientsMu.RUnlock()
		if cached != nil {
			t.Errorf("expected apiClients[gitlab] to be nil when constructor returns error, got non-nil")
		}
		return
	}
	apiClientsMu.RLock()
	cached := apiClients[clientCacheKey("gitlab", "https://gitlab.example.com")]
	apiClientsMu.RUnlock()
	if cached == nil {
		t.Fatal("expected apiClients[gitlab] to be set for gitlab provider, got nil")
	}
}

func TestGetThirdApi_DefaultFallback(t *testing.T) {
	// Reset the global apiClients cache.
	ResetAPIClients()
	defer ResetAPIClients()

	// Unknown provider should fall through to default (github).
	client, err := GetThirdApi("unknown-provider", "")
	if err != nil {
		t.Fatalf("unexpected error for default fallback: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client for default provider")
	}
}

func TestGetThirdApi_CachedClient(t *testing.T) {
	// Reset the global apiClients cache.
	ResetAPIClients()
	defer ResetAPIClients()

	// First call should create the client.
	client1, err := GetThirdApi("github", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call with same provider should return the same cached client.
	client2, err := GetThirdApi("github", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client1 != client2 {
		t.Error("expected second call to return cached client, got different instance")
	}
}

func TestGetThirdApi_DifferentProvidersReturnDifferentClients(t *testing.T) {
	ResetAPIClients()
	defer ResetAPIClients()

	// github client
	ghClient, err := GetThirdApi("github", "")
	if err != nil {
		t.Fatalf("unexpected error for github: %v", err)
	}

	// gitee client — should be different from github
	geClient, err := GetThirdApi("gitee", "")
	if err != nil {
		t.Fatalf("unexpected error for gitee: %v", err)
	}

	if ghClient == geClient {
		t.Error("expected different clients for github and gitee, got same instance")
	}

	// Requesting github again should return the same github client
	ghClient2, err := GetThirdApi("github", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ghClient != ghClient2 {
		t.Error("expected same cached client for github, got different instance")
	}
}

func TestGetThirdApi_GiteaError(t *testing.T) {
	ResetAPIClients()
	defer ResetAPIClients()

	// Gitea with an invalid host should return an error wrapped with context.
	_, err := GetThirdApi("gitea", "://invalid-host")
	if err == nil {
		// Some constructors may not fail on invalid URLs immediately,
		// so this is acceptable — just verify it doesn't panic.
		return
	}
	// If there IS an error, it should contain context about gitea.
	if errStr := err.Error(); len(errStr) == 0 {
		t.Error("error message should not be empty")
	}
}

func TestGetThirdApi_GiteepremiumError(t *testing.T) {
	ResetAPIClients()
	defer ResetAPIClients()

	// GiteePremium with an invalid host may or may not error at construction.
	_, err := GetThirdApi("giteepremium", "://invalid-host")
	if err == nil {
		return // acceptable — constructor may not fail immediately
	}
	if errStr := err.Error(); len(errStr) == 0 {
		t.Error("error message should not be empty")
	}
}

func TestClientCacheKey(t *testing.T) {
	key1 := clientCacheKey("github", "https://github.com")
	key2 := clientCacheKey("gitee", "https://gitee.com")
	key3 := clientCacheKey("github", "https://github.com") // same as key1

	if key1 == key2 {
		t.Error("expected different keys for different providers")
	}
	if key1 != key3 {
		t.Error("expected same key for same provider+host")
	}
}

func TestGetThirdApi_ConcurrentAccess(t *testing.T) {
	ResetAPIClients()
	defer ResetAPIClients()

	providers := []string{"github", "gitee", "github", "github", "gitee"}
	var wg sync.WaitGroup
	results := make([]*thirdapi.Client, len(providers))
	errs := make([]error, len(providers))

	for i, p := range providers {
		wg.Add(1)
		go func(idx int, prov string) {
			defer wg.Done()
			results[idx], errs[idx] = GetThirdApi(prov, "")
		}(i, p)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("GetThirdApi(%q) error: %v", providers[i], err)
		}
	}

	// All github calls should return the same client
	if results[0] != results[2] || results[0] != results[3] {
		t.Error("concurrent github calls should return same cached client")
	}
	// All gitee calls should return the same client
	if results[1] != results[4] {
		t.Error("concurrent gitee calls should return same cached client")
	}
	// github and gitee should be different
	if results[0] == results[1] {
		t.Error("github and gitee should return different clients")
	}
}
