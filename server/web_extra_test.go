package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// TestMidUiHandle_InstalledWithExistingRoute verifies that midUiHandle
// does not interfere when a route handler successfully responds.
func TestMidUiHandle_InstalledWithExistingRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test response")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = true

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "test response" {
		t.Errorf("expected 'test response', got %q", w.Body.String())
	}
}

// TestMidUiHandle_NotInstalledWithInstallRoute verifies that /install
// is accessible even when the system is not installed.
func TestMidUiHandle_NotInstalledWithInstallRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/install", func(c *gin.Context) {
		c.String(http.StatusOK, "install page")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/install", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for /install, got %d", w.Code)
	}
}

// TestMidUiHandle_NotInstalledWithUIAsset verifies that /gokinsui/* paths
// are accessible even when the system is not installed.
func TestMidUiHandle_NotInstalledWithUIAsset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(midUiHandle)
	router.GET("/gokinsui/test.js", func(c *gin.Context) {
		c.String(http.StatusOK, "console.log('test');")
	})

	origInstalled := comm.Installed
	defer func() { comm.Installed = origInstalled }()
	comm.Installed = false

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/gokinsui/test.js", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for /gokinsui/test.js, got %d", w.Code)
	}
}

// TestGetRdr_EmptyStaticPkg verifies that getRdr returns an error when StaticPkg is empty.
func TestGetRdr_EmptyStaticPkg(t *testing.T) {
	// Reset the singleton state
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()
	comm.StaticPkg = ""

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error when StaticPkg is empty, got nil")
	}
}

// TestGetRdr_InvalidBase64 verifies that getRdr returns an error for invalid base64.
func TestGetRdr_InvalidBase64(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()
	comm.StaticPkg = "invalid-base64-!!!"

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

// TestGetRdr_InvalidZip verifies that getRdr returns an error for valid base64 but invalid zip.
func TestGetRdr_InvalidZip(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()
	// Valid base64 but not a valid zip file
	comm.StaticPkg = base64.StdEncoding.EncodeToString([]byte("not a zip file"))

	_, err := getRdr()
	if err == nil {
		t.Fatal("expected error for invalid zip, got nil")
	}
}

// TestGetFile_WithValidZip verifies that getFile can extract files from a valid zip archive.
func TestGetFile_WithValidZip(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Create a minimal zip file with one entry
	// PK\x03\x04 = local file header signature
	// This is a minimal valid zip with a single empty file named "test.txt"
	zipData := []byte{
		0x50, 0x4b, 0x03, 0x04, // Local file header signature
		0x14, 0x00, // Version needed to extract
		0x00, 0x00, // General purpose bit flag
		0x00, 0x00, // Compression method (stored)
		0x00, 0x00, // Last mod file time
		0x00, 0x00, // Last mod file date
		0x00, 0x00, 0x00, 0x00, // CRC-32
		0x00, 0x00, 0x00, 0x00, // Compressed size
		0x00, 0x00, 0x00, 0x00, // Uncompressed size
		0x08, 0x00, // File name length (8 bytes)
		0x00, 0x00, // Extra field length
	}
	zipData = append(zipData, []byte("test.txt")...) // File name
	// Central directory
	zipData = append(zipData, []byte{
		0x50, 0x4b, 0x01, 0x02, // Central directory signature
		0x14, 0x00, // Version made by
		0x14, 0x00, // Version needed to extract
		0x00, 0x00, // General purpose bit flag
		0x00, 0x00, // Compression method
		0x00, 0x00, // Last mod file time
		0x00, 0x00, // Last mod file date
		0x00, 0x00, 0x00, 0x00, // CRC-32
		0x00, 0x00, 0x00, 0x00, // Compressed size
		0x00, 0x00, 0x00, 0x00, // Uncompressed size
		0x08, 0x00, // File name length
		0x00, 0x00, // Extra field length
		0x00, 0x00, // File comment length
		0x00, 0x00, // Disk number start
		0x00, 0x00, // Internal file attributes
		0x00, 0x00, 0x00, 0x00, // External file attributes
		0x00, 0x00, 0x00, 0x00, // Relative offset of local header
	}...)
	zipData = append(zipData, []byte("test.txt")...) // File name
	// End of central directory
	zipData = append(zipData, []byte{
		0x50, 0x4b, 0x05, 0x06, // End of central directory signature
		0x00, 0x00, // Number of this disk
		0x00, 0x00, // Disk where central directory starts
		0x01, 0x00, // Number of central directory records on this disk
		0x01, 0x00, // Total number of central directory records
		0x36, 0x00, 0x00, 0x00, // Size of central directory
		0x36, 0x00, 0x00, 0x00, // Offset of start of central directory
		0x00, 0x00, // Comment length
	}...)

	comm.StaticPkg = base64.StdEncoding.EncodeToString(zipData)

	file, err := getFile("test.txt")
	if err != nil {
		t.Fatalf("getFile failed: %v", err)
	}
	if file == nil {
		t.Fatal("expected file, got nil")
	}
	if file.Name != "test.txt" {
		t.Errorf("expected file name 'test.txt', got %q", file.Name)
	}
}

// TestGetFile_FileNotFoundInZip verifies that getFile returns an error when
// the requested file does not exist in the zip archive.
func TestGetFile_FileNotFoundInZip(t *testing.T) {
	rder = nil
	rderOnce = sync.Once{}
	rderErr = nil

	origStaticPkg := comm.StaticPkg
	defer func() { comm.StaticPkg = origStaticPkg }()

	// Use the same minimal zip as above
	zipData := []byte{
		0x50, 0x4b, 0x03, 0x04, 0x14, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x08, 0x00, 0x00, 0x00,
	}
	zipData = append(zipData, []byte("test.txt")...)
	zipData = append(zipData, []byte{
		0x50, 0x4b, 0x01, 0x02, 0x14, 0x00, 0x14, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}...)
	zipData = append(zipData, []byte("test.txt")...)
	zipData = append(zipData, []byte{
		0x50, 0x4b, 0x05, 0x06, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00,
		0x36, 0x00, 0x00, 0x00, 0x36, 0x00, 0x00, 0x00, 0x00, 0x00,
	}...)

	comm.StaticPkg = base64.StdEncoding.EncodeToString(zipData)

	_, err := getFile("nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestReadyzEndpoint_WithDbButNoCache verifies that readyz returns 503 when
// Db is connected but BCache is nil.
func TestReadyzEndpoint_WithDbButNoCache(t *testing.T) {
	origDb := comm.Db
	origCache := comm.BCache
	defer func() {
		if origDb != nil {
			_ = origDb.Close()
		}
		comm.Db = origDb
		comm.BCache = origCache
	}()

	// Create a mock database
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	comm.Db = db
	comm.BCache = nil

	router := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "/readyz", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 when cache is nil, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["cache"] != "disconnected" {
		t.Errorf("expected cache='disconnected', got %v", resp["cache"])
	}
}
