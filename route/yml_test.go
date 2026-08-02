package route

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/engine"
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
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_yml_template (
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(64),
		yml_content TEXT,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_yml_template: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_yml_plugin (
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
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

func TestYmlController_Path(t *testing.T) {
	c := &YmlController{}
	if got := c.GetPath(); got != "/api/yml" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/yml")
	}
}

func TestYml_Templates_Empty(t *testing.T) {
	setupYmlTestDB(t)
	ctrl := YmlController{}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)

	ctrl.templates(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []*model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected 0 templates, got %d", len(resp))
	}
}

func TestYml_Templates_WithData(t *testing.T) {
	db := setupYmlTestDB(t)
	ctrl := YmlController{}

	// Insert templates (active and deleted)
	_, _ = db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('Go CI', 'steps:\n  - build', 0)`)
	_, _ = db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('Java CI', 'steps:\n  - mvn test', 0)`)
	_, _ = db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('Old Template', 'deprecated', 1)`)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)

	ctrl.templates(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []*model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Should only return non-deleted templates
	if len(resp) != 2 {
		t.Errorf("expected 2 active templates, got %d", len(resp))
	}
	for _, tmpl := range resp {
		if tmpl.Deleted == 1 {
			t.Errorf("deleted template %q should not be returned", tmpl.Name)
		}
	}
}

func TestYml_Plugins_Empty(t *testing.T) {
	setupYmlTestDB(t)

	// Stub engine.Mgr to avoid nil pointer (Plugins() calls jobEgn.Plugins())
	// Since engine.Mgr is a package-level var and jobEgn may be nil,
	// we test that the handler doesn't panic by ensuring Plugins() returns nil safely
	origMgr := engine.Mgr
	t.Cleanup(func() { engine.Mgr = origMgr })
	// Create a minimal manager with nil jobEgn - Plugins() will be called
	// We need to handle this carefully, so let's just check DB-only plugins
	engine.Mgr = &engine.Manager{}

	ctrl := YmlController{}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)

	ctrl.plugins(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestYml_Plugins_WithDbPlugins(t *testing.T) {
	db := setupYmlTestDB(t)

	origMgr := engine.Mgr
	t.Cleanup(func() { engine.Mgr = origMgr })
	engine.Mgr = &engine.Manager{}

	// Insert plugins (active and deleted)
	_, _ = db.Exec(`INSERT INTO t_yml_plugin (name, yml_content, deleted) VALUES ('docker', 'image: docker', 0)`)
	_, _ = db.Exec(`INSERT INTO t_yml_plugin (name, yml_content, deleted) VALUES ('npm', 'image: node', 0)`)
	_, _ = db.Exec(`INSERT INTO t_yml_plugin (name, yml_content, deleted) VALUES ('old-plugin', 'deprecated', 1)`)

	ctrl := YmlController{}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)

	ctrl.plugins(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []*model.TYmlPlugin
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Should return 2 DB plugins (non-deleted) + 0 engine plugins (nil jobEgn)
	if len(resp) < 2 {
		t.Errorf("expected at least 2 plugins, got %d", len(resp))
	}
	// Verify deleted plugin is not included
	for _, p := range resp {
		if p.Name == "old-plugin" {
			t.Error("deleted plugin 'old-plugin' should not be returned")
		}
	}
}
