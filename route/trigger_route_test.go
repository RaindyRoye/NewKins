package route

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTriggerController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &TriggerController{}
	ctrl.Routes(r.Group("/api/trigger"))
	// If we get here without panic, routes were registered successfully
}
