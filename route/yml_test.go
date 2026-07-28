package route

import (
	"bytes"
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

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })
	return db
}

func setupYmlRouter(t *testing.T, r *gin.Engine) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	if r == nil {
		r = gin.New()
	}
	yc := &YmlController{}
	group := r.Group("/api/yml")
	yc.Routes(group)
	return r
}

func TestYmlController_Routes(t *testing.T) {
	c := &YmlController{}
	if got := c.GetPath(); got != "/api/yml" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/yml")
	}
}

func TestYmlTemplates_EmptyDB(t *testing.T) {
	setupYmlTestDB(t)
	r := setupYmlRouter(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
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
		t.Errorf("expected 0 templates, got %d", len(result))
	}
}

func TestYmlTemplates_WithTemplates(t *testing.T) {
	db := setupYmlTestDB(t)

	// Insert templates - one active, one deleted
	_, err := db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('tmpl-active', 'version: 1', 0)`)
	if err != nil {
		t.Fatalf("insert template: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('tmpl-deleted', 'version: 2', 1)`)
	if err != nil {
		t.Fatalf("insert deleted template: %v", err)
	}

	r := setupYmlRouter(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result []*model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 active template, got %d", len(result))
	}
	if result[0].Name != "tmpl-active" {
		t.Errorf("template name = %q, want %q", result[0].Name, "tmpl-active")
	}
}

func TestYmlPlugins_EmptyDB(t *testing.T) {
	setupYmlTestDB(t)
	r := setupYmlRouter(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/plugins", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestYmlPlugins_WithPlugins(t *testing.T) {
	db := setupYmlTestDB(t)

	_, err := db.Exec(`INSERT INTO t_yml_plugin (name, yml_content, deleted) VALUES ('plugin-a', 'content-a', 0)`)
	if err != nil {
		t.Fatalf("insert plugin: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_yml_plugin (name, yml_content, deleted) VALUES ('plugin-deleted', 'content-del', 1)`)
	if err != nil {
		t.Fatalf("insert deleted plugin: %v", err)
	}

	r := setupYmlRouter(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/plugins", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result []*model.TYmlPlugin
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	// Should have 1 active plugin from DB (deleted ones are filtered out)
	if len(result) < 1 {
		t.Fatalf("expected at least 1 plugin, got %d", len(result))
	}
	found := false
	for _, p := range result {
		if p.Name == "plugin-a" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected plugin-a in response")
	}
}

func TestYmlTemplates_NoBindParams(t *testing.T) {
	// The templates handler takes no bind parameters, so GinReqParseJson
	// will pass a zero-value arg. Invalid JSON is ignored.
	setupYmlTestDB(t)
	r := setupYmlRouter(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should succeed since the handler has no bind params
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
