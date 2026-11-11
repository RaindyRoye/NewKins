package route

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/util"
)

// TestLoginController_Routes tests that Routes registers endpoints without panic
func TestLoginController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := LoginController{}

	// Should not panic when registering routes
	ctrl.Routes(router)
}

// TestLoginController_info tests the info endpoint
func TestLoginController_info(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := LoginController{}

	router.POST("/info", ctrl.info)

	req, _ := http.NewRequest("POST", "/info", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 200 with login info
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// TestOrgController_Routes tests that Routes registers endpoints without panic
func TestOrgController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := OrgController{}

	// Should not panic when registering routes
	ctrl.Routes(router)
}

// TestRuntimeController_Routes tests that Routes registers endpoints without panic
func TestRuntimeController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := RuntimeController{}

	// Should not panic when registering routes
	ctrl.Routes(router)
}

// TestRuntimeController_stages tests the stages endpoint with missing param
func TestRuntimeController_stages(t *testing.T) {
	setupUserTestDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := RuntimeController{}

	router.POST("/stages", util.GinReqParseJson(ctrl.stages))

	// Test with missing pvId
	body := `{"pvId":""}`
	req, _ := http.NewRequest("POST", "/stages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestRuntimeController_cmds tests the cmds endpoint with missing param
func TestRuntimeController_cmds(t *testing.T) {
	setupUserTestDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := RuntimeController{}

	router.POST("/cmds", util.GinReqParseJson(ctrl.cmds))

	// Test with missing stepId
	body := `{"stepId":""}`
	req, _ := http.NewRequest("POST", "/cmds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestRuntimeController_build tests the build endpoint with missing param
func TestRuntimeController_build(t *testing.T) {
	setupUserTestDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := RuntimeController{}

	router.POST("/build", util.GinReqParseJson(ctrl.build))

	// Test with missing buildId
	body := `{"buildId":""}`
	req, _ := http.NewRequest("POST", "/build", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestRuntimeController_cancel tests the cancel endpoint with missing param
func TestRuntimeController_cancel(t *testing.T) {
	setupUserTestDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := RuntimeController{}

	router.POST("/cancel", util.GinReqParseJson(ctrl.cancel))

	// Test with missing buildId
	body := `{"buildId":""}`
	req, _ := http.NewRequest("POST", "/cancel", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestRuntimeController_logs tests the logs endpoint with missing param
func TestRuntimeController_logs(t *testing.T) {
	setupUserTestDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := RuntimeController{}

	router.POST("/logs", util.GinReqParseJson(ctrl.logs))

	// Test with missing stepId
	body := `{"stepId":"","buildId":"test"}`
	req, _ := http.NewRequest("POST", "/logs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestTriggerController_Routes tests that Routes registers endpoints without panic
func TestTriggerController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := TriggerController{}

	// Should not panic when registering routes
	ctrl.Routes(router)
}

// TestTriggerController_triggers tests the triggers endpoint with missing param
func TestTriggerController_triggers(t *testing.T) {
	setupUserTestDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := TriggerController{}

	router.POST("/triggers", util.GinReqParseJson(ctrl.triggers))

	// Test with missing pipelineId
	body := `{"pipelineId":""}`
	req, _ := http.NewRequest("POST", "/triggers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestUserController_Routes tests that Routes registers endpoints without panic
func TestUserController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := UserController{}

	// Should not panic when registering routes
	ctrl.Routes(router)
}

// TestPipelineVersionController_Routes tests that Routes registers endpoints without panic
func TestPipelineVersionController_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := PipelineVersionController{}

	// Should not panic when registering routes
	ctrl.Routes(router)
}

// TestPipelineVersionController_delete tests the delete endpoint with missing param
func TestPipelineVersionController_delete(t *testing.T) {
	setupUserTestDB(t)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	ctrl := PipelineVersionController{}

	router.POST("/delete", util.GinReqParseJson(ctrl.delete))

	// Test with missing id
	body := `{"id":""}`
	req, _ := http.NewRequest("POST", "/delete", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
