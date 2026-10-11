package httpex

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestPost_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded; charset=utf-8" {
			t.Errorf("unexpected Content-Type: %s", ct)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	resp, err := Post(server.URL, nil, 5*time.Second)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestPost_WithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.FormValue("key") != "value" {
			t.Errorf("expected key=value, got %s", r.FormValue("key"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	params := &url.Values{}
	params.Set("key", "value")
	resp, err := Post(server.URL, params, 5*time.Second)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

func TestPost_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "test" {
			t.Errorf("expected X-Custom: test, got %s", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hds := http.Header{}
	hds.Set("X-Custom", "test")
	resp, err := Post(server.URL, nil, 5*time.Second, hds)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

func TestPost_InvalidURL(t *testing.T) {
	_, err := Post("http://127.0.0.1:1/nonexistent", nil, 1*time.Second)
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
}

func TestPostJSON_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json; charset=utf-8" {
			t.Errorf("expected JSON content type, got %s", ct)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode error: %v", err)
		}
		if body["action"] != "test" {
			t.Errorf("expected action=test, got %s", body["action"])
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	resp, err := PostJSON(server.URL, map[string]string{"action": "test"}, 5*time.Second)
	if err != nil {
		t.Fatalf("PostJSON error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 201 {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
}

func TestPostJSON_WithHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token123" {
			t.Errorf("expected auth header, got %s", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hds := http.Header{}
	hds.Set("Authorization", "Bearer token123")
	resp, err := PostJSON(server.URL, map[string]string{}, 5*time.Second, hds)
	if err != nil {
		t.Fatalf("PostJSON error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

func TestPostJSON_InvalidURL(t *testing.T) {
	_, err := PostJSON("http://127.0.0.1:1/nonexistent", map[string]string{}, 1*time.Second)
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
}

func TestPost_Unreachable(t *testing.T) {
	_, err := Post("http://127.0.0.1:1/invalid", nil, 1*time.Second)
	if err == nil {
		t.Fatal("expected connection error for unreachable host")
	}
}
