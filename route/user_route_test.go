package route

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUserController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &UserController{}
	ctrl.Routes(r.Group("/api/user"))
	// If we get here without panic, routes were registered successfully
}
