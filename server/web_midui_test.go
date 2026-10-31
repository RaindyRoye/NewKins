package server

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
)

func TestMidUiHandle_Non404Status(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest("GET", "/ok", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_RedirectToInstall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	comm.Installed = false
	defer func() { comm.Installed = true }()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/somepath", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	req := httptest.NewRequest("GET", "/somepath", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
}

func TestMidUiHandle_NotInstalled_AllowGokinsUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	comm.Installed = false
	defer func() { comm.Installed = true }()

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/gokinsui/app.js", func(c *gin.Context) {
		c.String(http.StatusOK, "js content")
	})

	req := httptest.NewRequest("GET", "/gokinsui/app.js", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMidUiHandle_FileNotFound_NoIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	comm.Installed = true

	// Create an empty zip (valid but no files)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	zw.Close()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	// Reset zip reader to reload
	rder = nil
	rderErr = nil
	rderOnce = sync.Once{}

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	req := httptest.NewRequest("GET", "/notexist.html", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect to /, got %d", w.Code)
	}
}

func TestMidUiHandle_ValidHTMLFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	comm.Installed = true

	// Create a minimal zip with an HTML file
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("index.html")
	fw.Write([]byte("<html><body>Hello</body></html>"))
	zw.Close()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())

	// Reset zip reader to reload
	rder = nil
	rderErr = nil
	rderOnce = sync.Once{}

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	req := httptest.NewRequest("GET", "/index.html", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/html" {
		t.Errorf("expected Content-Type text/html, got %s", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", w.Header().Get("Cache-Control"))
	}
}

func TestMidUiHandle_ValidCSSFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	comm.Installed = true

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("style.css")
	fw.Write([]byte("body { color: red; }"))
	zw.Close()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())
	rder = nil
	rderErr = nil
	rderOnce = sync.Once{}

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	req := httptest.NewRequest("GET", "/style.css", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/css" {
		t.Errorf("expected Content-Type text/css, got %s", w.Header().Get("Content-Type"))
	}
}

func TestMidUiHandle_ValidJSFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	comm.Installed = true

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("app.js")
	fw.Write([]byte("console.log('hello');"))
	zw.Close()

	comm.StaticPkg = base64.StdEncoding.EncodeToString(buf.Bytes())
	rder = nil
	rderErr = nil
	rderOnce = sync.Once{}

	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	req := httptest.NewRequest("GET", "/app.js", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/javascript" {
		t.Errorf("expected Content-Type application/javascript, got %s", w.Header().Get("Content-Type"))
	}
}


