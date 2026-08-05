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

func setupYmlTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create t_yml_template table
	_, err = db.Exec(`CREATE TABLE t_yml_template (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(64),
		yml_content TEXT,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_yml_template table: %v", err)
	}

	// Create t_yml_plugin table
	_, err = db.Exec(`CREATE TABLE t_yml_plugin (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(64),
		yml_content TEXT,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_yml_plugin table: %v", err)
	}

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	return db
}

func TestYmlController_templates_Empty(t *testing.T) {
	setupYmlTestDB(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &YmlController{}
	r.POST("/api/yml/templates", ctrl.templates)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var templates []model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &templates); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("expected 0 templates, got %d", len(templates))
	}
}

func TestYmlController_templates_WithData(t *testing.T) {
	db := setupYmlTestDB(t)

	// Insert test templates
	_, err := db.Insert(&model.TYmlTemplate{
		Name:       "template1",
		YmlContent: "version: 1.0",
		Deleted:    0,
	})
	if err != nil {
		t.Fatalf("insert template: %v", err)
	}

	_, err = db.Insert(&model.TYmlTemplate{
		Name:       "template2",
		YmlContent: "version: 2.0",
		Deleted:    0,
	})
	if err != nil {
		t.Fatalf("insert template: %v", err)
	}

	// Insert a deleted template (should not appear)
	_, err = db.Insert(&model.TYmlTemplate{
		Name:       "deleted-template",
		YmlContent: "deleted",
		Deleted:    1,
	})
	if err != nil {
		t.Fatalf("insert deleted template: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &YmlController{}
	r.POST("/api/yml/templates", ctrl.templates)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var templates []model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &templates); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(templates))
	}

	// Verify deleted template is not included
	for _, tmpl := range templates {
		if tmpl.Name == "deleted-template" {
			t.Error("deleted template should not be returned")
		}
	}
}

func TestYmlController_plugins_Empty(t *testing.T) {
	setupYmlTestDB(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &YmlController{}
	r.POST("/api/yml/plugins", ctrl.plugins)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/plugins", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var plugins []model.TYmlPlugin
	if err := json.Unmarshal(w.Body.Bytes(), &plugins); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	// plugins() also adds engine.Mgr.Plugins(), so we just verify it's a valid array
	if plugins == nil {
		t.Error("expected non-nil plugins array")
	}
}

func TestYmlController_plugins_WithData(t *testing.T) {
	db := setupYmlTestDB(t)

	// Insert test plugins
	_, err := db.Insert(&model.TYmlPlugin{
		Name:       "plugin1",
		YmlContent: "plugin content 1",
		Deleted:    0,
	})
	if err != nil {
		t.Fatalf("insert plugin: %v", err)
	}

	_, err = db.Insert(&model.TYmlPlugin{
		Name:       "plugin2",
		YmlContent: "plugin content 2",
		Deleted:    0,
	})
	if err != nil {
		t.Fatalf("insert plugin: %v", err)
	}

	// Insert a deleted plugin (should not appear)
	_, err = db.Insert(&model.TYmlPlugin{
		Name:       "deleted-plugin",
		YmlContent: "deleted",
		Deleted:    1,
	})
	if err != nil {
		t.Fatalf("insert deleted plugin: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := &YmlController{}
	r.POST("/api/yml/plugins", ctrl.plugins)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/plugins", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var plugins []model.TYmlPlugin
	if err := json.Unmarshal(w.Body.Bytes(), &plugins); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	// Should have at least 2 plugins from DB (plus any from engine.Mgr.Plugins())
	if len(plugins) < 2 {
		t.Errorf("expected at least 2 plugins, got %d", len(plugins))
	}

	// Verify deleted plugin is not included
	for _, plug := range plugins {
		if plug.Name == "deleted-plugin" {
			t.Error("deleted plugin should not be returned")
		}
	}
}
