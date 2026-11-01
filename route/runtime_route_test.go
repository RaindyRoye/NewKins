package route

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRuntimeController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &RuntimeController{}
	ctrl.Routes(r.Group("/api/runtime"))
	// If we get here without panic, routes were registered successfully
}
