package route

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoginController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &LoginController{}
	ctrl.Routes(r.Group("/api/lg"))
	// If we get here without panic, routes were registered successfully
}
