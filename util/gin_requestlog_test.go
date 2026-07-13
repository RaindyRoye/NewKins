package util

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestMidRequestLog_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidRequestID())
	router.Use(MidRequestLog())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Capture log output
	var logOutput strings.Builder
	origOutput := logrus.StandardLogger().Out
	origFormatter := logrus.StandardLogger().Formatter
	logrus.SetOutput(&logOutput)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer func() {
		logrus.SetOutput(origOutput)
		logrus.SetFormatter(origFormatter)
	}()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify log was written
	if logOutput.Len() == 0 {
		t.Error("expected log output, got none")
	}

	// Verify log contains expected fields
	output := logOutput.String()
	if !strings.Contains(output, "request completed") {
		t.Errorf("expected 'request completed' in log, got: %s", output)
	}
	if !strings.Contains(output, "/test") {
		t.Errorf("expected '/test' in log, got: %s", output)
	}
	if !strings.Contains(output, "200") {
		t.Errorf("expected '200' in log, got: %s", output)
	}
}

func TestMidRequestLog_ClientError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidRequestID())
	router.Use(MidRequestLog())

	router.GET("/bad", func(c *gin.Context) {
		c.String(http.StatusBadRequest, "bad request")
	})

	var logOutput strings.Builder
	origOutput := logrus.StandardLogger().Out
	origFormatter := logrus.StandardLogger().Formatter
	logrus.SetOutput(&logOutput)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer func() {
		logrus.SetOutput(origOutput)
		logrus.SetFormatter(origFormatter)
	}()

	req := httptest.NewRequest("GET", "/bad", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	output := logOutput.String()
	if !strings.Contains(output, "request completed with client error") {
		t.Errorf("expected 'request completed with client error' in log, got: %s", output)
	}
}

func TestMidRequestLog_ServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidRequestID())
	router.Use(MidRequestLog())

	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "server error")
	})

	var logOutput strings.Builder
	origOutput := logrus.StandardLogger().Out
	origFormatter := logrus.StandardLogger().Formatter
	logrus.SetOutput(&logOutput)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer func() {
		logrus.SetOutput(origOutput)
		logrus.SetFormatter(origFormatter)
	}()

	req := httptest.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	output := logOutput.String()
	if !strings.Contains(output, "request completed with server error") {
		t.Errorf("expected 'request completed with server error' in log, got: %s", output)
	}
}

func TestMidRequestLog_HealthCheckSkipped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidRequestID())
	router.Use(MidRequestLog())

	router.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	var logOutput strings.Builder
	origOutput := logrus.StandardLogger().Out
	origFormatter := logrus.StandardLogger().Formatter
	logrus.SetOutput(&logOutput)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer func() {
		logrus.SetOutput(origOutput)
		logrus.SetFormatter(origFormatter)
	}()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Health check should not produce log output
	if logOutput.Len() > 0 {
		t.Errorf("expected no log output for health check, got: %s", logOutput.String())
	}
}

func TestMidRequestLog_WithQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidRequestID())
	router.Use(MidRequestLog())

	router.GET("/search", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	var logOutput strings.Builder
	origOutput := logrus.StandardLogger().Out
	origFormatter := logrus.StandardLogger().Formatter
	logrus.SetOutput(&logOutput)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer func() {
		logrus.SetOutput(origOutput)
		logrus.SetFormatter(origFormatter)
	}()

	req := httptest.NewRequest("GET", "/search?q=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	output := logOutput.String()
	if !strings.Contains(output, "q=test") {
		t.Errorf("expected 'q=test' in log, got: %s", output)
	}
}

func TestMidRequestLog_WithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MidRequestID())
	router.Use(MidRequestLog())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	var logOutput strings.Builder
	origOutput := logrus.StandardLogger().Out
	origFormatter := logrus.StandardLogger().Formatter
	logrus.SetOutput(&logOutput)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	defer func() {
		logrus.SetOutput(origOutput)
		logrus.SetFormatter(origFormatter)
	}()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(RequestIDHeader, "custom-request-id-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	output := logOutput.String()
	if !strings.Contains(output, "custom-request-id-123") {
		t.Errorf("expected 'custom-request-id-123' in log, got: %s", output)
	}
}
