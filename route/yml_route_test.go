package route

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestYmlController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &YmlController{}
	ctrl.Routes(r.Group("/api/yml"))
	// If we get here without panic, routes were registered successfully
}
