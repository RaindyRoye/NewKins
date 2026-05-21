package util

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGinRegController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g := gin.New()

	called := false
	ctrl := &testController{
		path: "/test",
		routes: func(g gin.IRoutes) {
			called = true
		},
	}
	GinRegController(g, ctrl)
	if !called {
		t.Error("GinRegController should call Routes method")
	}
}

func TestGinRegController_NilGuard(t *testing.T) {
	// Should not panic with nil
	GinRegController(nil, nil)
}

func TestMidAccessAllowFun_OPTIONS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g := gin.New()
	g.Use(MidAccessAllowFun)
	g.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	g.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS should return 204, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Error("should set Access-Control-Allow-Origin header")
	}
}

type testController struct {
	path   string
	routes func(gin.IRoutes)
}

func (c *testController) GetPath() string {
	return c.path
}
func (c *testController) Routes(g gin.IRoutes) {
	c.routes(g)
}
