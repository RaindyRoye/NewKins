package route

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupYmlTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_yml_template (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		yml_content TEXT,
		deleted INTEGER DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_yml_template table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_yml_plugin (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		yml_content TEXT,
		deleted INTEGER DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_yml_plugin table: %v", err)
	}

	comm.Db = db
}

func TestYmlController_GetPath_New(t *testing.T) {
	ctrl := YmlController{}
	if got := ctrl.GetPath(); got != "/api/yml" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/yml")
	}
}

func TestYmlTemplates_Empty(t *testing.T) {
	setupYmlTestDB(t)

	ctrl := YmlController{}
	req := httptest.NewRequest("POST", "/api/yml/templates", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ctrl.templates(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result []*model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 templates, got %d", len(result))
	}
}

func TestYmlTemplates_WithData(t *testing.T) {
	setupYmlTestDB(t)

	// Insert test template
	_, err := comm.Db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES (?, ?, 0)`,
		"Test Template", "yml content here")
	if err != nil {
		t.Fatalf("insert template: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/yml/templates", nil)

	ctrl := &YmlController{}
	ctrl.templates(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result []model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("unmarshal: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 template, got %d", len(result))
	}
}

func TestYmlPlugins_Success(t *testing.T) {
	setupYmlTestDB(t)

	ctrl := YmlController{}
	req := httptest.NewRequest("POST", "/api/yml/plugins", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	ctrl.plugins(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result []*model.TYmlPlugin
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Verify we got a valid response (may be empty if no plugins configured)
	if result == nil {
		t.Errorf("expected non-nil plugin list")
	}
}
