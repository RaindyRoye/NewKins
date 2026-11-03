package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gokins/gokins/hook"
)

// TestParseCommentHook tests parsing of issue comment webhooks
func TestParseCommentHook(t *testing.T) {
	// Create a mock HTTP server to simulate GitHub API for PR details
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a mock PR response
		response := map[string]any{
			"id":    1001,
			"title": "Test PR",
			"head": map[string]any{
				"ref": "feature-branch",
				"sha": "abc123",
				"repo": map[string]any{
					"id":         2,
					"name":       "fork-repo",
					"full_name":  "forker/test-repo",
					"clone_url":  "https://github.com/forker/test-repo.git",
					"html_url":   "https://github.com/forker/test-repo",
					"git_url":    "git://github.com/forker/test-repo.git",
					"ssh_url":    "git@github.com:forker/test-repo.git",
					"svn_url":    "https://github.com/forker/test-repo",
					"created_at": "2021-01-01T00:00:00Z",
					"owner": map[string]any{
						"login": "forker",
					},
				},
			},
			"base": map[string]any{
				"ref": "main",
				"sha": "def456",
				"repo": map[string]any{
					"id":         1,
					"name":       "test-repo",
					"full_name":  "owner/test-repo",
					"clone_url":  "https://github.com/owner/test-repo.git",
					"html_url":   "https://github.com/owner/test-repo",
					"git_url":    "git://github.com/owner/test-repo.git",
					"ssh_url":    "git@github.com:owner/test-repo.git",
					"svn_url":    "https://github.com/owner/test-repo",
					"created_at": "2021-01-01T00:00:00Z",
					"owner": map[string]any{
						"login": "owner",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	payload := map[string]any{
		"action": "created",
		"issue": map[string]any{
			"id":     123,
			"number": 42,
			"title":  "Test Issue",
			"pull_request": map[string]any{
				"url": mockServer.URL + "/repos/owner/repo/pulls/42",
			},
		},
		"comment": map[string]any{
			"id":   456,
			"body": "/rebuild",
			"user": map[string]any{
				"login": "testuser",
			},
		},
		"repository": map[string]any{
			"id":        789,
			"name":      "test-repo",
			"full_name": "owner/test-repo",
			"owner": map[string]any{
				"login": "owner",
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	result, err := parseCommentHook(data)
	if err != nil {
		t.Fatalf("parseCommentHook failed: %v", err)
	}

	if result == nil {
		t.Fatal("parseCommentHook returned nil")
	}

	// Verify basic fields were parsed
	if result.Comment.Body != "/rebuild" {
		t.Errorf("expected comment body '/rebuild', got '%s'", result.Comment.Body)
	}

	if result.Action != hook.EventsTypeComment {
		t.Errorf("expected action %q, got %q", hook.EventsTypeComment, result.Action)
	}
}

// TestParseCommentHookInvalidJSON tests error handling for malformed JSON
func TestParseCommentHookInvalidJSON(t *testing.T) {
	invalidData := []byte(`{invalid json`)

	_, err := parseCommentHook(invalidData)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestConvertPullRequestURL tests fetching PR details from GitHub API
func TestConvertPullRequestURL(t *testing.T) {
	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":    1001,
			"title": "Test PR",
			"head": map[string]any{
				"ref": "feature-branch",
				"sha": "abc123",
				"repo": map[string]any{
					"id":        2,
					"name":      "fork-repo",
					"full_name": "forker/test-repo",
				},
			},
			"base": map[string]any{
				"ref": "main",
				"sha": "def456",
				"repo": map[string]any{
					"id":        1,
					"name":      "test-repo",
					"full_name": "owner/test-repo",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	prURL, err := convertPullRequestURL(mockServer.URL + "/repos/owner/repo/pulls/42")
	if err != nil {
		t.Fatalf("convertPullRequestURL failed: %v", err)
	}

	if prURL == nil {
		t.Fatal("convertPullRequestURL returned nil")
	}

	if prURL.Head.Ref != "feature-branch" {
		t.Errorf("expected head ref 'feature-branch', got '%s'", prURL.Head.Ref)
	}

	if prURL.Base.Ref != "main" {
		t.Errorf("expected base ref 'main', got '%s'", prURL.Base.Ref)
	}
}

// TestConvertPullRequestURLNetworkError tests error handling when API is unreachable
func TestConvertPullRequestURLNetworkError(t *testing.T) {
	// Use an invalid URL to simulate network error
	_, err := convertPullRequestURL("http://invalid-url-that-does-not-exist.local/api/pulls/1")
	if err == nil {
		t.Fatal("expected error for network failure, got nil")
	}
}

// TestConvertPullRequestURLInvalidJSON tests error handling when API returns invalid JSON
func TestConvertPullRequestURLInvalidJSON(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer mockServer.Close()

	_, err := convertPullRequestURL(mockServer.URL + "/api/pulls/1")
	if err == nil {
		t.Fatal("expected error for invalid JSON response, got nil")
	}
}
