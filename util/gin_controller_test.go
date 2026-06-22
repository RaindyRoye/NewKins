package util

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// mockController implements GinController for testing
type mockController struct {
	path string
}

func (m *mockController) GetPath() string { return m.path }
func (m *mockController) Routes(g gin.IRoutes) {
	g.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})
}

func TestGinRegController_NilEngine(t *testing.T) {
	// Should not panic with nil engine
	GinRegController(nil, &mockController{path: "/test"})
}

func TestGinRegController_NilController(t *testing.T) {
	// Should not panic with nil controller
	e := gin.New()
	GinRegController(e, nil)
}

func TestGinRegController_RootPath(t *testing.T) {
	e := gin.New()
	ctrl := &mockController{path: "/"}
	GinRegController(e, ctrl)

	// Test that the route was registered
	w := performRequest(e, "GET", "/test")
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGinRegController_GroupPath(t *testing.T) {
	e := gin.New()
	ctrl := &mockController{path: "/api/v1"}
	GinRegController(e, ctrl)

	// Test that the route was registered under the group
	w := performRequest(e, "GET", "/api/v1/test")
	if w.Code != 200 {
		t.Errorf("expected 200 for /api/v1/test, got %d", w.Code)
	}

	// Should NOT be registered at root
	w = performRequest(e, "GET", "/test")
	if w.Code != 404 {
		t.Errorf("expected 404 for /test (should be under group), got %d", w.Code)
	}
}

func TestGinRegController_MultipleGroups(t *testing.T) {
	e := gin.New()
	GinRegController(e, &mockController{path: "/api/a"})
	GinRegController(e, &mockController{path: "/api/b"})

	w := performRequest(e, "GET", "/api/a/test")
	if w.Code != 200 {
		t.Errorf("expected 200 for /api/a/test, got %d", w.Code)
	}

	w = performRequest(e, "GET", "/api/b/test")
	if w.Code != 200 {
		t.Errorf("expected 200 for /api/b/test, got %d", w.Code)
	}
}

// performRequest creates a test request and returns the response
func performRequest(e *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	e.ServeHTTP(w, req)
	return w
}
