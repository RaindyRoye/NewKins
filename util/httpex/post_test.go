package httpex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestPostResult_NilResult(t *testing.T) {
	_, _, err := PostResult("http://example.com", nil, nil, 5*time.Second)
	if err == nil {
		t.Fatal("PostResult with nil result expected error, got nil")
	}
	if err.Error() != "result is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostResult_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			t.Errorf("encode error: %v", err)
		}
	}))
	defer server.Close()

	var result map[string]string
	code, body, err := PostResult(server.URL, nil, &result, 5*time.Second)
	if err != nil {
		t.Fatalf("PostResult returned unexpected error: %v", err)
	}
	if code != 200 {
		t.Fatalf("expected status 200, got %d", code)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty body")
	}
	if result["status"] != "ok" {
		t.Fatalf("expected status=ok, got %v", result)
	}
}

func TestPostResult_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte("internal error")); err != nil {
			t.Errorf("write error: %v", err)
		}
	}))
	defer server.Close()

	var result map[string]string
	code, body, err := PostResult(server.URL, nil, &result, 5*time.Second)
	if err == nil {
		t.Fatal("PostResult with non-200 expected error, got nil")
	}
	if code != 500 {
		t.Fatalf("expected status 500, got %d", code)
	}
	if string(body) != "internal error" {
		t.Fatalf("expected body 'internal error', got %q", string(body))
	}
}

func TestPostJSONResult_NilResult(t *testing.T) {
	_, _, err := PostJSONResult("http://example.com", nil, nil, 5*time.Second)
	if err == nil {
		t.Fatal("PostJSONResult with nil result expected error, got nil")
	}
	if err.Error() != "result is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostJSONResult_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
			t.Errorf("expected JSON content type, got %s", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]int{"count": 42}); err != nil {
			t.Errorf("encode error: %v", err)
		}
	}))
	defer server.Close()

	var result map[string]int
	code, _, err := PostJSONResult(server.URL, map[string]string{"key": "val"}, &result, 5*time.Second)
	if err != nil {
		t.Fatalf("PostJSONResult returned unexpected error: %v", err)
	}
	if code != 200 {
		t.Fatalf("expected status 200, got %d", code)
	}
	if result["count"] != 42 {
		t.Fatalf("expected count=42, got %v", result)
	}
}

func TestPosts_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write([]byte("created")); err != nil {
			t.Errorf("write error: %v", err)
		}
	}))
	defer server.Close()

	code, body, err := Posts(server.URL, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("Posts returned unexpected error: %v", err)
	}
	if code != 201 {
		t.Fatalf("expected status 201, got %d", code)
	}
	if string(body) != "created" {
		t.Fatalf("expected body 'created', got %q", string(body))
	}
}

func TestPostResult_ErrorWrapping(t *testing.T) {
	// Test that read errors are properly wrapped with %w
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hijack the connection to simulate a read error
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Skip("server doesn't support hijacking")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Skip("hijack failed")
		}
		// Close connection abruptly to cause read error
		_ = conn.Close()
	}))
	defer server.Close()

	var result map[string]string
	_, _, err := PostResult(server.URL, nil, &result, 5*time.Second)
	if err == nil {
		t.Fatal("PostResult with broken connection expected error, got nil")
	}
	// Verify the error can be unwrapped (proving %w is used)
	if !errors.Is(err, err) {
		t.Fatal("error should be self-referential with errors.Is")
	}
}

// --- Context-aware function tests ---

func TestPostCtx_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := PostCtx(ctx, server.URL, nil, 10*time.Second)
	if err == nil {
		t.Fatal("PostCtx with canceled context expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestPostsCtx_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, _, err := PostsCtx(ctx, server.URL, nil, 10*time.Second)
	if err == nil {
		t.Fatal("PostsCtx with canceled context expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestPostJSONCtx_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := PostJSONCtx(ctx, server.URL, map[string]string{"k": "v"}, 10*time.Second)
	if err == nil {
		t.Fatal("PostJSONCtx with canceled context expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestPostResultCtx_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"hello": "world"}); err != nil {
			t.Errorf("encode error: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	var result map[string]string
	code, _, err := PostResultCtx(ctx, server.URL, nil, &result, 5*time.Second)
	if err != nil {
		t.Fatalf("PostResultCtx returned unexpected error: %v", err)
	}
	if code != 200 {
		t.Fatalf("expected status 200, got %d", code)
	}
	if result["hello"] != "world" {
		t.Fatalf("expected hello=world, got %v", result)
	}
}

func TestPostResultCtx_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var result map[string]string
	_, _, err := PostResultCtx(ctx, server.URL, nil, &result, 10*time.Second)
	if err == nil {
		t.Fatal("PostResultCtx with canceled context expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestPostJSONResultCtx_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]int{"val": 99}); err != nil {
			t.Errorf("encode error: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	var result map[string]int
	code, _, err := PostJSONResultCtx(ctx, server.URL, map[string]string{"k": "v"}, &result, 5*time.Second)
	if err != nil {
		t.Fatalf("PostJSONResultCtx returned unexpected error: %v", err)
	}
	if code != 200 {
		t.Fatalf("expected status 200, got %d", code)
	}
	if result["val"] != 99 {
		t.Fatalf("expected val=99, got %v", result)
	}
}

func TestPostJSONResultCtx_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var result map[string]int
	_, _, err := PostJSONResultCtx(ctx, server.URL, nil, &result, 10*time.Second)
	if err == nil {
		t.Fatal("PostJSONResultCtx with canceled context expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestPostJSONResultCtx_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("not json")); err != nil {
			t.Errorf("write error: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	var result map[string]string
	_, _, err := PostJSONResultCtx(ctx, server.URL, nil, &result, 5*time.Second)
	if err == nil {
		t.Fatal("PostJSONResultCtx with invalid JSON expected error, got nil")
	}
	// Verify error wrapping - should be unwrappable to json.SyntaxError
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected wrapped json.SyntaxError, got: %T: %v", err, err)
	}
}

func TestPostResultCtx_NilResult(t *testing.T) {
	_, _, err := PostResultCtx(context.Background(), "http://example.com", nil, nil, 5*time.Second)
	if err == nil {
		t.Fatal("PostResultCtx with nil result expected error, got nil")
	}
	if err.Error() != "result is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostJSONResultCtx_NilResult(t *testing.T) {
	_, _, err := PostJSONResultCtx(context.Background(), "http://example.com", nil, nil, 5*time.Second)
	if err == nil {
		t.Fatal("PostJSONResultCtx with nil result expected error, got nil")
	}
	if err.Error() != "result is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Convenience wrapper tests (Post / PostJSON without context) ---

func TestPost_ConvenienceWrapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded; charset=utf-8" {
			t.Errorf("expected form content-type, got %s", ct)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	params := &url.Values{}
	params.Set("key", "value")
	resp, err := Post(server.URL, params, 5*time.Second)
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPost_NilParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := Post(server.URL, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("Post with nil params returned error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPostJSON_ConvenienceWrapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json; charset=utf-8" {
			t.Errorf("expected JSON content-type, got %s", ct)
		}
		// Verify the JSON body was sent correctly
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body error: %v", err)
		}
		if body["key"] != "val" {
			t.Errorf("expected key=val, got %v", body)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	resp, err := PostJSON(server.URL, map[string]string{"key": "val"}, 5*time.Second)
	if err != nil {
		t.Fatalf("PostJSON returned error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPostJSON_NilParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := PostJSON(server.URL, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("PostJSON with nil params returned error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

func TestPost_InvalidURL(t *testing.T) {
	_, err := Post("http://invalid.host.that.does.not.exist:99999", nil, 2*time.Second)
	if err == nil {
		t.Fatal("Post with invalid URL expected error, got nil")
	}
}

func TestPostJSON_InvalidURL(t *testing.T) {
	_, err := PostJSON("http://invalid.host.that.does.not.exist:99999", map[string]string{"k": "v"}, 2*time.Second)
	if err == nil {
		t.Fatal("PostJSON with invalid URL expected error, got nil")
	}
}

func TestPosts_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "test-value" {
			t.Errorf("expected X-Custom=test-value, got %s", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("with-header"))
	}))
	defer server.Close()

	hdr := http.Header{}
	hdr.Set("X-Custom", "test-value")
	code, body, err := Posts(server.URL, nil, 5*time.Second, hdr)
	if err != nil {
		t.Fatalf("Posts returned error: %v", err)
	}
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if string(body) != "with-header" {
		t.Fatalf("expected 'with-header', got %q", string(body))
	}
}
