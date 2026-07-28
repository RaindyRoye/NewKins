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
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_yml_template (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(64),
		yml_content TEXT,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_yml_template: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_yml_plugin (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(64),
		yml_content TEXT,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_yml_plugin: %v", err)
	}

	comm.Db = db
	return db
}

func TestYmlController_templates_Empty(t *testing.T) {
	setupYmlTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()

	yc := &YmlController{}
	r.POST("/api/yml/templates", yc.templates)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result []*model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d items", len(result))
	}
}

func TestYmlController_templates_WithData(t *testing.T) {
	db := setupYmlTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Insert test templates
	_, err := db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('tpl1', 'stages: [build]', 0)`)
	if err != nil {
		t.Fatalf("insert template: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('tpl2', 'stages: [test]', 0)`)
	if err != nil {
		t.Fatalf("insert template: %v", err)
	}
	// Insert a deleted template (should not appear)
	_, err = db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('tpl3', 'old', 1)`)
	if err != nil {
		t.Fatalf("insert deleted template: %v", err)
	}

	yc := &YmlController{}
	r.POST("/api/yml/templates", yc.templates)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result []*model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 templates, got %d", len(result))
	}
}

func TestYmlController_plugins_EmptyDB(t *testing.T) {
	setupYmlTestDB(t)
	gin.SetMode(gin.TestMode)

	// Initialize engine.Mgr.jobEgn so Plugins() doesn't nil-panic
	initTestJobEngine(t)

	r := gin.New()
	yc := &YmlController{}
	r.POST("/api/yml/plugins", yc.plugins)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/plugins", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestYmlController_plugins_WithDBPlugins(t *testing.T) {
	db := setupYmlTestDB(t)
	gin.SetMode(gin.TestMode)

	initTestJobEngine(t)

	_, err := db.Exec(`INSERT INTO t_yml_plugin (name, yml_content, deleted) VALUES ('my-plugin', 'content', 0)`)
	if err != nil {
		t.Fatalf("insert plugin: %v", err)
	}

	r := gin.New()
	yc := &YmlController{}
	r.POST("/api/yml/plugins", yc.plugins)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/plugins", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result []*model.TYmlPlugin
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	// Should have at least the DB plugin
	found := false
	for _, p := range result {
		if p.Name == "my-plugin" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected my-plugin in results")
	}
}
