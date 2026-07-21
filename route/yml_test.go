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
		t.Fatalf("failed to create t_yml_template: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_yml_plugin (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(64),
		yml_content TEXT,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("failed to create t_yml_plugin: %v", err)
	}

	comm.Db = db
	return db
}

func setupYmlTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	setupYmlTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	yc := &YmlController{}
	yc.Routes(r.Group("/api/yml"))
	return r
}

func TestYmlController_Templates_Empty(t *testing.T) {
	r := setupYmlTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result []model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 templates, got %d", len(result))
	}
}

func TestYmlController_Templates_WithData(t *testing.T) {
	r := setupYmlTestRouter(t)

	// Insert test data
	_, _ = comm.Db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('tmpl1', 'content1', 0)`)
	_, _ = comm.Db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('tmpl2', 'content2', 0)`)
	_, _ = comm.Db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('tmpl_deleted', 'deleted', 1)`)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result []model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 templates, got %d", len(result))
	}
}

func TestYmlController_Templates_OnlyDeleted(t *testing.T) {
	r := setupYmlTestRouter(t)

	_, _ = comm.Db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('d1', 'content', 1)`)
	_, _ = comm.Db.Exec(`INSERT INTO t_yml_template (name, yml_content, deleted) VALUES ('d2', 'content', 1)`)

	req := httptest.NewRequest(http.MethodPost, "/api/yml/templates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result []model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 templates (all deleted), got %d", len(result))
	}
}

func TestYmlController_Plugins_WithDatabasePlugins(t *testing.T) {
	// Skip this test because it requires engine.Mgr to be initialized,
	// which is a complex global dependency that needs full engine startup.
	// The endpoint calls engine.Mgr.Plugins() which dereferences Mgr.jobEgn.
	t.Skip("Requires engine.Mgr initialization - integration test only")
}

func TestYmlController_Plugins_Empty(t *testing.T) {
	// Skip this test because it requires engine.Mgr to be initialized,
	// which is a complex global dependency that needs full engine startup.
	// The endpoint calls engine.Mgr.Plugins() which dereferences Mgr.jobEgn.
	t.Skip("Requires engine.Mgr initialization - integration test only")
}

func TestYmlController_Plugins_DeletedFilter(t *testing.T) {
	// Skip this test because it requires engine.Mgr to be initialized,
	// which is a complex global dependency that needs full engine startup.
	// The endpoint calls engine.Mgr.Plugins() which dereferences Mgr.jobEgn.
	t.Skip("Requires engine.Mgr initialization - integration test only")
}
