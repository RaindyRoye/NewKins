package middleware

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// captureLog redirects logrus output to a buffer for the duration of the test.
func captureLog(t *testing.T) (*bytes.Buffer, func()) {
	t.Helper()
	buf := &bytes.Buffer{}
	origOut := logrus.StandardLogger().Out
	origFmt := logrus.StandardLogger().Formatter
	origLvl := logrus.GetLevel()
	logrus.SetOutput(buf)
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	logrus.SetLevel(logrus.DebugLevel)
	return buf, func() {
		logrus.SetOutput(origOut)
		logrus.SetFormatter(origFmt)
		logrus.SetLevel(origLvl)
	}
}

func setupRouter(middleware gin.HandlerFunc, handler gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(MidRequestID())
	r.Use(middleware)
	r.GET("/test", handler)
	r.GET("/healthz", func(c *gin.Context) { c.String(200, "ok") })
	r.GET("/gokinsui/app.js", func(c *gin.Context) { c.String(200, "js") })
	return r
}

func TestMidAccessLog_BasicRequest(t *testing.T) {
	buf, cleanup := captureLog(t)
	defer cleanup()

	r := setupRouter(MidAccessLog(), func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?q=1", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("User-Agent", "TestBot/1.0")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	log := buf.String()
	if !strings.Contains(log, "method=GET") {
		t.Errorf("log should contain method=GET, got: %s", log)
	}
	if !strings.Contains(log, "path=/test") {
		t.Errorf("log should contain path=/test, got: %s", log)
	}
	if !strings.Contains(log, "status=200") {
		t.Errorf("log should contain status=200, got: %s", log)
	}
	if !strings.Contains(log, "request_id=") {
		t.Errorf("log should contain request_id, got: %s", log)
	}
	if !strings.Contains(log, "ip=192.168.1.100") {
		t.Errorf("log should contain client IP, got: %s", log)
	}
}

func TestMidAccessLog_SkipsHealthCheck(t *testing.T) {
	buf, cleanup := captureLog(t)
	defer cleanup()

	r := setupRouter(MidAccessLog(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	log := buf.String()
	if strings.Contains(log, "path=/healthz") {
		t.Errorf("health check should not be logged, got: %s", log)
	}
}

func TestMidAccessLog_ServerError(t *testing.T) {
	buf, cleanup := captureLog(t)
	defer cleanup()

	r := setupRouter(MidAccessLog(), func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "boom")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	log := buf.String()
	if !strings.Contains(log, "level=error") {
		t.Errorf("5xx should log at error level, got: %s", log)
	}
	if !strings.Contains(log, "server error") {
		t.Errorf("5xx should contain 'server error', got: %s", log)
	}
}

func TestMidAccessLog_SlowRequest(t *testing.T) {
	buf, cleanup := captureLog(t)
	defer cleanup()

	r := setupRouter(MidAccessLog(), func(c *gin.Context) {
		time.Sleep(accessLogSlowThreshold + 50*time.Millisecond)
		c.String(http.StatusOK, "slow")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	log := buf.String()
	if !strings.Contains(log, "level=warning") {
		t.Errorf("slow request should log at warn level, got: %s", log)
	}
	if !strings.Contains(log, "slow request") {
		t.Errorf("slow request should contain 'slow request', got: %s", log)
	}
}

func TestMidAccessLog_NormalInfo(t *testing.T) {
	buf, cleanup := captureLog(t)
	defer cleanup()

	r := setupRouter(MidAccessLog(), func(c *gin.Context) {
		c.String(http.StatusOK, "fast")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	log := buf.String()
	if !strings.Contains(log, "level=info") {
		t.Errorf("normal fast request should log at info level, got: %s", log)
	}
}

func TestMidAccessLog_ClientError(t *testing.T) {
	buf, cleanup := captureLog(t)
	defer cleanup()

	r := setupRouter(MidAccessLog(), func(c *gin.Context) {
		c.String(http.StatusNotFound, "nope")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	log := buf.String()
	// 4xx should be Info level (not error, not slow)
	if !strings.Contains(log, "level=info") {
		t.Errorf("4xx should log at info level, got: %s", log)
	}
	if !strings.Contains(log, "status=404") {
		t.Errorf("should contain status=404, got: %s", log)
	}
}

func TestMidAccessLog_LongUserAgentTruncated(t *testing.T) {
	buf, cleanup := captureLog(t)
	defer cleanup()

	r := setupRouter(MidAccessLog(), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	// User-Agent longer than 200 chars
	req.Header.Set("User-Agent", strings.Repeat("x", 300))
	r.ServeHTTP(w, req)

	log := buf.String()
	if strings.Contains(log, "ua=") {
		t.Errorf("long user-agent should be excluded from log, got: %s", log)
	}
}

func TestMidAccessLog_ResponseSize(t *testing.T) {
	buf, cleanup := captureLog(t)
	defer cleanup()

	body := "hello world response body"
	r := setupRouter(MidAccessLog(), func(c *gin.Context) {
		c.String(http.StatusOK, body)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	log := buf.String()
	if !strings.Contains(log, "size=") {
		t.Errorf("log should contain response size, got: %s", log)
	}
}

// mockGinResponseWriter implements gin.ResponseWriter for unit testing statusWriter.
type mockGinResponseWriter struct {
	header     http.Header
	body       *bytes.Buffer
	statusCode int
}

func newMockGinRW() *mockGinResponseWriter {
	return &mockGinResponseWriter{
		header:     http.Header{},
		body:       &bytes.Buffer{},
		statusCode: http.StatusOK,
	}
}

func (m *mockGinResponseWriter) Header() http.Header        { return m.header }
func (m *mockGinResponseWriter) Write(b []byte) (int, error) { return m.body.Write(b) }
func (m *mockGinResponseWriter) WriteHeader(code int)        { m.statusCode = code }
func (m *mockGinResponseWriter) Status() int                 { return m.statusCode }
func (m *mockGinResponseWriter) Size() int                   { return m.body.Len() }
func (m *mockGinResponseWriter) WriteString(s string) (int, error) { return m.body.WriteString(s) }
func (m *mockGinResponseWriter) Written() bool               { return m.body.Len() > 0 }
func (m *mockGinResponseWriter) WriteHeaderNow()             {}
func (m *mockGinResponseWriter) Pusher() http.Pusher         { return nil }
func (m *mockGinResponseWriter) CloseNotify() <-chan bool {
	ch := make(chan bool)
	close(ch)
	return ch
}
func (m *mockGinResponseWriter) Flush() {}
func (m *mockGinResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, http.ErrNotSupported
}

func TestStatusWriter_CapturesStatus(t *testing.T) {
	mw := newMockGinRW()
	sw := &statusWriter{
		ResponseWriter: mw,
		status:         http.StatusOK,
	}
	sw.WriteHeader(http.StatusCreated)
	if sw.status != http.StatusCreated {
		t.Errorf("expected status 201, got %d", sw.status)
	}
	if mw.statusCode != http.StatusCreated {
		t.Errorf("expected underlying statusCode 201, got %d", mw.statusCode)
	}
}

func TestStatusWriter_CapturesWrittenBytes(t *testing.T) {
	mw := newMockGinRW()
	sw := &statusWriter{
		ResponseWriter: mw,
		status:         http.StatusOK,
	}
	n, err := sw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}
	if sw.written != 5 {
		t.Errorf("expected sw.written=5, got %d", sw.written)
	}
	// Write more
	_, _ = sw.Write([]byte(" world"))
	if sw.written != 11 {
		t.Errorf("expected sw.written=11 after second write, got %d", sw.written)
	}
}
