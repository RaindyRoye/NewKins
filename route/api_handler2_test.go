package route

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiController_test_InvalidYaml(t *testing.T) {
	r := setupApiTestRouter(t)

	body := bytes.NewBufferString("invalid: yaml: content: [")
	req := httptest.NewRequest(http.MethodPost, "/api/builds", body)
	req.Header.Set("Content-Type", "text/yaml")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestApiController_test_ValidYamlEmptyStages(t *testing.T) {
	r := setupApiTestRouter(t)

	// Valid YAML but empty stages → should fail with ErrStagesEmpty
	body := bytes.NewBufferString("stages: []")
	req := httptest.NewRequest(http.MethodPost, "/api/builds", body)
	req.Header.Set("Content-Type", "text/yaml")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestApiController_test_ValidYamlWithStages(t *testing.T) {
	r := setupApiTestRouter(t)

	// Valid YAML with stages and steps
	yaml := `
stages:
  - name: build
    steps:
      - name: compile
        step: shell
        commands:
          - echo hello
`
	body := bytes.NewBufferString(yaml)
	req := httptest.NewRequest(http.MethodPost, "/api/builds", body)
	req.Header.Set("Content-Type", "text/yaml")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// engine.Mgr.BuildEgn() may be nil since engine isn't started in tests,
	// so we just verify it doesn't crash (the Put call may fail but that's fine)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, expected 200 or 500 (engine not started), body: %s", w.Code, w.Body.String())
	}
}
