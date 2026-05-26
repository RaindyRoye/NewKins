package util

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRespInternalErr(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	testErr := errors.New("sensitive db connection string: postgres://user:pass@host/db")
	RespInternalErr(c, "test operation", testErr)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	body := w.Body.String()
	// Must NOT contain the original error message
	if body == "sensitive db connection string: postgres://user:pass@host/db" {
		t.Error("RespInternalErr leaked internal error details to client")
	}
	// Must contain generic message
	if body != "internal server error" {
		t.Errorf("expected 'internal server error', got %q", body)
	}
}

func TestRespErr(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	RespErr(c, http.StatusBadRequest, "validation failed", errors.New("field X is required"))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	body := w.Body.String()
	// Should contain the safe message
	if body != "validation failed" {
		t.Errorf("expected 'validation failed', got %q", body)
	}
}

func TestRespInternalErr_Status500(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	RespInternalErr(c, "any op", errors.New("some error"))
	if w.Code != 500 {
		t.Errorf("RespInternalErr must always return 500, got %d", w.Code)
	}
}

func TestRespErr_CustomStatus(t *testing.T) {
	tests := []struct {
		status int
		msg    string
	}{
		{400, "bad request"},
		{404, "not found"},
		{409, "conflict"},
		{422, "unprocessable entity"},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			RespErr(c, tt.status, tt.msg, errors.New("internal details"))
			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}
			if w.Body.String() != tt.msg {
				t.Errorf("expected body %q, got %q", tt.msg, w.Body.String())
			}
		})
	}
}
