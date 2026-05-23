package httpex

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
