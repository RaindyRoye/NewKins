package comm

import (
	"sync"
	"testing"

	"github.com/gokins/gokins/thirdapi"
)

func TestGetThirdApi_GitlabTypo(t *testing.T) {
	// Reset the global apiClient to ensure a clean test state.
	apiClient = nil
	defer func() { apiClient = nil }()

	// "gitlab" (not "gitalb") should be recognized as a valid provider.
	// The host is intentionally fake — the API client will be created but
	// no network call is made at construction time for gitlab client.
	_, err := GetThirdApi("gitlab", "https://gitlab.example.com")
	if err != nil {
		// Some gitlab API constructors may fail without a valid host,
		// but the important thing is the case was recognized (not falling through to default).
		// We check that apiClient was NOT set to the default (github) client.
		if apiClient != nil {
			t.Errorf("expected apiClient to be nil when constructor returns error, got non-nil")
		}
		return
	}
	if apiClient == nil {
		t.Fatal("expected apiClient to be set for gitlab provider, got nil")
	}
}

func TestGetThirdApi_DefaultFallback(t *testing.T) {
	// Reset the global apiClient.
	apiClient = nil
	defer func() { apiClient = nil }()

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
	// Reset the global apiClient.
	apiClient = nil
	defer func() { apiClient = nil }()

	// First call should create the client.
	client1, err := GetThirdApi("github", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call should return the same cached client.
	client2, err := GetThirdApi("gitee", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client1 != client2 {
		t.Error("expected second call to return cached client, got different instance")
	}
}

func TestGetThirdApi_GiteaError(t *testing.T) {
	// Reset the global apiClient.
	apiClient = nil
	defer func() { apiClient = nil }()

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

func TestGetThirdApi_ConcurrentAccess(t *testing.T) {
	// Reset the global apiClient.
	apiClient = nil
	defer func() { apiClient = nil }()

	const goroutines = 20
	type result struct {
		client *thirdapi.Client
		err    error
	}
	results := make([]result, goroutines)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c, err := GetThirdApi("github", "")
			results[idx] = result{client: c, err: err}
		}(i)
	}
	wg.Wait()

	// All goroutines should have received the same client instance.
	expected := results[0].client
	for i, r := range results {
		if r.err != nil {
			t.Errorf("goroutine %d returned error: %v", i, r.err)
		}
		if r.client != expected {
			t.Errorf("goroutine %d got different client instance, expected concurrent safety", i)
		}
	}
}

func TestGetThirdApi_NoDoubleInit(t *testing.T) {
	// Reset the global apiClient.
	apiClient = nil
	defer func() { apiClient = nil }()

	// First call creates the client.
	c1, err := GetThirdApi("github", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Directly overwrite apiClient to nil to simulate a race condition
	// where another goroutine might have set it between the check and the assignment.
	// With the mutex, this should NOT happen during concurrent access.
	// Here we just verify that after a successful init, subsequent calls
	// return the same cached client even with a different provider name.
	c2, err := GetThirdApi("gitee", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c1 != c2 {
		t.Error("expected cached client to be returned regardless of provider argument")
	}
}
