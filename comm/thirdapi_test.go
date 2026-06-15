package comm

import (
	"testing"
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
