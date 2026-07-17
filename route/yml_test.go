package route

import (
	"context"
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
	origCtx := comm.Ctx
	t.Cleanup(func() {
		comm.Db = origDb
		comm.Ctx = origCtx
	})

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_yml_template (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(64),
		yml_content LONGTEXT,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_yml_template: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_yml_plugin (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(64),
		yml_content LONGTEXT,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_yml_plugin: %v", err)
	}

	comm.Db = db
	comm.Ctx = context.Background()
}

func makeYmlGinContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c, w
}

func TestYmlController_templates_Empty(t *testing.T) {
	setupYmlTestDB(t)
	c, w := makeYmlGinContext(t)
	ctrl := YmlController{}
	ctrl.templates(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var templates []*model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &templates); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("expected 0 templates, got %d", len(templates))
	}
}

func TestYmlController_templates_WithData(t *testing.T) {
	setupYmlTestDB(t)
	_, err := comm.Db.Insert(&model.TYmlTemplate{
		Name:       "template1",
		YmlContent: "version: 1.0",
		Deleted:    0,
	})
	if err != nil {
		t.Fatalf("insert template: %v", err)
	}

	_, err = comm.Db.Insert(&model.TYmlTemplate{
		Name:       "template2",
		YmlContent: "version: 2.0",
		Deleted:    0,
	})
	if err != nil {
		t.Fatalf("insert template: %v", err)
	}

	// Insert a deleted template that should not appear
	_, err = comm.Db.Insert(&model.TYmlTemplate{
		Name:    "deleted-template",
		Deleted: 1,
	})
	if err != nil {
		t.Fatalf("insert deleted template: %v", err)
	}

	c, w := makeYmlGinContext(t)
	ctrl := YmlController{}
	ctrl.templates(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var templates []*model.TYmlTemplate
	if err := json.Unmarshal(w.Body.Bytes(), &templates); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(templates))
	}
}

// Note: YmlController.plugins() tests are skipped because they require
// engine.Mgr to be fully initialized via engine.Start(), which depends on
// runtime configuration and cannot be easily mocked in unit tests.
