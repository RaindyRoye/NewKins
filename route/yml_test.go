package route

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupYmlTestRouter(t *testing.T, db *xorm.Engine) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	r := gin.New()
	yc := &YmlController{}
	yc.Routes(r.Group("/api/yml"))
	return r
}

func createYmlTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_yml_template (
		aid BIGINT NOT NULL PRIMARY KEY,
		name VARCHAR(64),
		yml_content TEXT,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_yml_template: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_yml_plugin (
		aid BIGINT NOT NULL PRIMARY KEY,
		name VARCHAR(64),
		yml_content TEXT,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_yml_plugin: %v", err)
	}

	return db
}

func TestTemplates_EmptyDb(t *testing.T) {
	db := createYmlTestDb(t)
	r := setupYmlTestRouter(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected 0 templates, got %d", len(resp))
	}
}

func TestTemplates_WithEntries(t *testing.T) {
	db := createYmlTestDb(t)
	r := setupYmlTestRouter(t, db)

	// Insert some templates (one active, one deleted)
	_, err := db.Exec(`INSERT INTO t_yml_template (aid, name, yml_content, deleted) VALUES (1, 'template1', 'content1', 0)`)
	if err != nil {
		t.Fatalf("insert template1: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_yml_template (aid, name, yml_content, deleted) VALUES (2, 'template2', 'content2', 1)`)
	if err != nil {
		t.Fatalf("insert template2 (deleted): %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	// Only the non-deleted template should appear
	if len(resp) != 1 {
		t.Errorf("expected 1 template, got %d", len(resp))
	}
	if len(resp) > 0 && resp[0]["name"] != "template1" {
		t.Errorf("template name = %v, want 'template1'", resp[0]["name"])
	}
}

func TestPlugins_EmptyDb(t *testing.T) {
	t.Skip("Requires engine.Mgr.jobEgn initialization - tested in integration tests")
}

func TestPlugins_WithDbEntries(t *testing.T) {
	t.Skip("Requires engine.Mgr.jobEgn initialization - tested in integration tests")
}

// Prevent unused import errors
var _ = context.Background
