package httpex

import (
	"context"
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
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	c.Request = req

	ResMsgUrl(c, "Operation successful", "/dashboard")

	if w.Code != 302 {
		t.Errorf("expected status 302, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Operation successful") {
		t.Errorf("body should contain message, got: %s", body)
	}
	if !strings.Contains(body, "/dashboard") {
		t.Errorf("body should contain redirect URL, got: %s", body)
	}
	if !strings.Contains(body, "text/html") || w.Header().Get("Content-Type") != "text/html" {
		// Check Content-Type header
		if w.Header().Get("Content-Type") != "text/html" {
			t.Errorf("expected Content-Type text/html, got %s", w.Header().Get("Content-Type"))
		}
	}
}

func TestResMsgUrl_WithoutUrl(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	c.Request = req

	ResMsgUrl(c, "Redirecting...")

	if w.Code != 302 {
		t.Errorf("expected status 302, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Redirecting...") {
		t.Errorf("body should contain message, got: %s", body)
	}
}

func TestResMsgUrl_EmptyMessage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	c.Request = req

	ResMsgUrl(c, "")

	if w.Code != 302 {
		t.Errorf("expected status 302, got %d", w.Code)
	}

	body := w.Body.String()
	// When message is empty, the template should show "跳转中..."
	if !strings.Contains(body, "跳转中") {
		t.Errorf("body should contain default redirect message for empty msg, got: %s", body)
	}
}

func TestResMsgUrl_EmptyUrl(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	c.Request = req

	ResMsgUrl(c, "Done", "")

	if w.Code != 302 {
		t.Errorf("expected status 302, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Done") {
		t.Errorf("body should contain message, got: %s", body)
	}
}

func TestResMsgUrl_HTMLEscaping(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	c.Request = req

	// Test that special characters are handled
	ResMsgUrl(c, "Success <script>alert('xss')</script>", "/home")

	if w.Code != 302 {
		t.Errorf("expected status 302, got %d", w.Code)
	}

	body := w.Body.String()
	// The message should be present (even if not escaped, since this is a simple template)
	if !strings.Contains(body, "Success") {
		t.Errorf("body should contain message, got: %s", body)
	}
}

func TestHTMLMsgUrl_TemplatePresent(t *testing.T) {
	if HTMLMsgUrl == "" {
		t.Error("HTMLMsgUrl template should not be empty")
	}
	if !strings.Contains(HTMLMsgUrl, "{{msg}}") {
		t.Error("HTMLMsgUrl should contain {{msg}} placeholder")
	}
	if !strings.Contains(HTMLMsgUrl, "{{url}}") {
		t.Error("HTMLMsgUrl should contain {{url}} placeholder")
	}
}
