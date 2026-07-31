package httpex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientWithTimeout_ContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	client := clientWithTimeout(ctx, 10*time.Second)
	if client.Timeout != 100*time.Millisecond {
		t.Errorf("expected timeout to be 100ms (context deadline), got %v", client.Timeout)
	}
}

func TestClientWithTimeout_ExplicitTimeout(t *testing.T) {
	ctx := context.Background()
	client := clientWithTimeout(ctx, 5*time.Second)
	if client.Timeout != 5*time.Second {
		t.Errorf("expected timeout to be 5s (explicit), got %v", client.Timeout)
	}
}

func TestClientWithTimeout_ContextShorter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client := clientWithTimeout(ctx, 10*time.Second)
	// Allow some tolerance for timing
	if client.Timeout > 2*time.Second {
		t.Errorf("expected timeout <= 2s (context), got %v", client.Timeout)
	}
}

func TestClientWithTimeout_ExplicitShorter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := clientWithTimeout(ctx, 2*time.Second)
	if client.Timeout != 2*time.Second {
		t.Errorf("expected timeout to be 2s (explicit shorter), got %v", client.Timeout)
	}
}

func TestClientWithTimeout_ZeroTimeout(t *testing.T) {
	ctx := context.Background()
	client := clientWithTimeout(ctx, 0)
	if client.Timeout != 0 {
		t.Errorf("expected timeout to be 0 (no timeout), got %v", client.Timeout)
	}
}

func TestClientWithTimeout_ContextAlreadyExpired(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(10 * time.Millisecond)

	client := clientWithTimeout(ctx, 5*time.Second)
	// When context is expired, remaining <= 0, so should fall back to explicit timeout
	if client.Timeout != 5*time.Second {
		t.Errorf("expected timeout to be 5s (fallback), got %v", client.Timeout)
	}
}

// TestPostCtx_ContextCancellation verifies that PostCtx respects context cancellation.
func TestPostCtx_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := PostCtx(ctx, server.URL, nil, 10*time.Second)
	if err == nil {
		t.Fatal("expected error due to cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestPostsCtx_ContextCancellation verifies that PostsCtx respects context cancellation.
func TestPostsCtx_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := PostsCtx(ctx, server.URL, nil, 10*time.Second)
	if err == nil {
		t.Fatal("expected error due to cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestPostJSONCtx_ContextCancellation verifies that PostJSONCtx respects context cancellation.
func TestPostJSONCtx_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := PostJSONCtx(ctx, server.URL, map[string]string{"test": "value"}, 10*time.Second)
	if err == nil {
		t.Fatal("expected error due to cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestPostResultCtx_ContextCancellation verifies that PostResultCtx respects context cancellation.
func TestPostResultCtx_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var result map[string]interface{}
	_, _, err := PostResultCtx(ctx, server.URL, nil, &result, 10*time.Second)
	if err == nil {
		t.Fatal("expected error due to cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestPostJSONResultCtx_ContextCancellation verifies that PostJSONResultCtx respects context cancellation.
func TestPostJSONResultCtx_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var result map[string]interface{}
	_, _, err := PostJSONResultCtx(ctx, server.URL, map[string]string{"test": "value"}, &result, 10*time.Second)
	if err == nil {
		t.Fatal("expected error due to cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

// TestPostResultCtx_Success tests successful PostResultCtx call.
func TestPostResultCtx_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","data":"test"}`))
	}))
	defer server.Close()

	var result map[string]interface{}
	code, body, err := PostResultCtx(context.Background(), server.URL, nil, &result, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != http.StatusOK {
		t.Errorf("expected status 200, got %d", code)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", result["status"])
	}
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

// TestPostResultCtx_NilResult tests that PostResultCtx rejects nil result parameter.
func TestPostResultCtx_NilResult(t *testing.T) {
	_, _, err := PostResultCtx(context.Background(), "http://example.com", nil, nil, 5*time.Second)
	if err == nil {
		t.Fatal("expected error for nil result, got nil")
	}
}

// TestPostJSONResultCtx_NilResult tests that PostJSONResultCtx rejects nil result parameter.
func TestPostJSONResultCtx_NilResult(t *testing.T) {
	_, _, err := PostJSONResultCtx(context.Background(), "http://example.com", map[string]string{}, nil, 5*time.Second)
	if err == nil {
		t.Fatal("expected error for nil result, got nil")
	}
}
