package route

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPipelineVersionController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &PipelineVersionController{}
	ctrl.Routes(r.Group("/api/pipeline/version"))
	// If we get here without panic, routes were registered successfully
}
