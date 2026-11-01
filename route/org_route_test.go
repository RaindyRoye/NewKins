package route

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOrgController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &OrgController{}
	ctrl.Routes(r.Group("/api/org"))
	// If we get here without panic, routes were registered successfully
}
