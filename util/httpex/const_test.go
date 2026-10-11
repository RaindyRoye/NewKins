package httpex

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestResMsgUrl_WithUrl(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/redirect", nil)

	ResMsgUrl(c, "Please wait", "https://example.com/target")

	if w.Code != 302 {
		t.Errorf("expected status 302, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Please wait") {
		t.Errorf("expected body to contain message, got %q", body)
	}
	if !strings.Contains(body, "https://example.com/target") {
		t.Errorf("expected body to contain redirect URL, got %q", body)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content type, got %q", ct)
	}
}

func TestResMsgUrl_WithoutUrl(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/redirect", nil)

	ResMsgUrl(c, "Redirecting...")

	if w.Code != 302 {
		t.Errorf("expected status 302, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Redirecting...") {
		t.Errorf("expected body to contain message, got %q", body)
	}
	// When no URL is provided, the template should replace {{url}} with ""
	if strings.Contains(body, "{{url}}") {
		t.Error("template placeholder {{url}} was not replaced")
	}
}

func TestResMsgUrl_EmptyMessage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/redirect", nil)

	ResMsgUrl(c, "", "https://example.com")

	if w.Code != 302 {
		t.Errorf("expected status 302, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "https://example.com") {
		t.Errorf("expected redirect URL in body, got %q", body)
	}
}

func TestResMsgUrl_HTMLTemplateStructure(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/redirect", nil)

	ResMsgUrl(c, "Test message", "/dashboard")

	body := w.Body.String()
	// Verify the HTML structure contains expected elements
	if !strings.Contains(body, "<html") {
		t.Error("expected HTML document structure")
	}
	if !strings.Contains(body, "<script>") {
		t.Error("expected script tag for redirect logic")
	}
	if !strings.Contains(body, "window.location") {
		t.Error("expected JavaScript redirect")
	}
}

func TestResMsgUrl_NilContext(t *testing.T) {
	// Verify ResMsgUrl handles a proper gin context
	g := gin.New()
	g.GET("/test", func(c *gin.Context) {
		ResMsgUrl(c, "hello", "/target")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	g.ServeHTTP(w, req)

	if w.Code != 302 {
		t.Errorf("expected status 302 through gin router, got %d", w.Code)
	}
}
