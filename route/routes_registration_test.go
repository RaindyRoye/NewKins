package route

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestLoginController_Routes verifies that the login controller routes are
// registered correctly and respond to requests.
func TestLoginController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &LoginController{}
	ctrl.Routes(r.Group("/api/lg"))

	// /api/lg/info should be reachable
	req := httptest.NewRequest(http.MethodPost, "/api/lg/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/lg/info, got %d, body: %s", w.Code, w.Body.String())
	}
}

// TestYmlController_Routes verifies that the yml controller routes are
// registered correctly.
func TestYmlController_Routes(t *testing.T) {
	setupYmlTestDB(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &YmlController{}
	ctrl.Routes(r.Group("/api/yml"))

	// /api/yml/templates should be reachable
	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/yml/templates via routes, got %d", w.Code)
	}
}

// TestTriggerController_Routes verifies that the trigger controller routes are
// registered correctly.
func TestTriggerController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &TriggerController{}
	// Routes uses MidUserCheck which requires a logged-in user.
	// We verify the group path is registered by hitting a known route
	// without auth — it should return 403 from MidUserCheck.
	ctrl.Routes(r.Group("/api/trigger"))

	req := httptest.NewRequest(http.MethodPost, "/api/trigger/triggers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// MidUserCheck returns 403 when no user is present
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 without auth, got %d", w.Code)
	}
}

// TestPipelineVersionController_Routes verifies routes registration.
func TestPipelineVersionController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &PipelineVersionController{}
	ctrl.Routes(r.Group("/api/pipelineVersion"))

	req := httptest.NewRequest(http.MethodPost, "/api/pipelineVersion/delete", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// MidUserCheck returns 403 when no user is present
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 without auth, got %d", w.Code)
	}
}
