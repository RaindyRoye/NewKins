package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupHookTestRouter(t *testing.T, db *xorm.Engine) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	hc := &HookController{}
	hookGroup := r.Group("/trigger")
	hc.Routes(hookGroup)
	return r
}

func createHookTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create the trigger table matching the model schema
	_, err = db.Exec(`CREATE TABLE t_trigger (
		id VARCHAR(64) NOT NULL,
		aid BIGINT NOT NULL,
		uid VARCHAR(64),
		pipeline_id VARCHAR(64),
		types VARCHAR(50),
		name VARCHAR(100),
		"desc" VARCHAR(255),
		params TEXT,
		enabled INT DEFAULT 0,
		created DATETIME,
		updated DATETIME,
		PRIMARY KEY (id, aid)
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestHookController_Routes(t *testing.T) {
	c := &HookController{}
	if got := c.GetPath(); got != "/trigger" {
		t.Errorf("GetPath() = %q, want %q", got, "/trigger")
	}
}

func TestHooks_EmptyTriggerId(t *testing.T) {
	db := createHookTestDb(t)
	r := setupHookTestRouter(t, db)

	// Gin will route /trigger/hook/ to the handler with an empty param
	// But since the route is /trigger/hook/:triggerId, an empty triggerId
	// won't match the route at all, resulting in 404
	req := httptest.NewRequest(http.MethodPost, "/trigger/hook/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Gin returns 404 for routes that don't match (empty param)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for empty triggerId, got %d", w.Code)
	}
}

func TestHooks_NonexistentTrigger(t *testing.T) {
	db := createHookTestDb(t)
	r := setupHookTestRouter(t, db)

	req := httptest.NewRequest(http.MethodPost, "/trigger/hook/nonexistent-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent trigger, got %d", w.Code)
	}
}

func TestHooks_DisabledTrigger(t *testing.T) {
	db := createHookTestDb(t)
	r := setupHookTestRouter(t, db)

	// Insert a disabled trigger (enabled = 0)
	_, err := db.Exec(`INSERT INTO t_trigger (id, aid, pipeline_id, types, enabled) VALUES ('trig-disabled', 1, 'pipe-1', 'hook', 0)`)
	if err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/trigger/hook/trig-disabled", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// The query is "WHERE id = ? and enabled != 0", so disabled triggers return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for disabled trigger, got %d", w.Code)
	}
}

func TestWeb_EmptySecret(t *testing.T) {
	db := createHookTestDb(t)
	r := setupHookTestRouter(t, db)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/trigger/web/trig-1", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty secret, got %d", w.Code)
	}
}

func TestWeb_EmptyTriggerIdAndSecret(t *testing.T) {
	db := createHookTestDb(t)
	r := setupHookTestRouter(t, db)

	body := bytes.NewBufferString(`{}`)
	// Empty triggerId won't match the route pattern
	req := httptest.NewRequest(http.MethodPost, "/trigger/web/", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for empty triggerId, got %d", w.Code)
	}
}

func TestWeb_NonexistentTrigger(t *testing.T) {
	db := createHookTestDb(t)
	r := setupHookTestRouter(t, db)

	body, _ := json.Marshal(map[string]any{"secret": "test-secret"})
	req := httptest.NewRequest(http.MethodPost, "/trigger/web/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent trigger, got %d", w.Code)
	}
}

func TestWeb_DisabledTrigger(t *testing.T) {
	db := createHookTestDb(t)
	r := setupHookTestRouter(t, db)

	_, err := db.Exec(`INSERT INTO t_trigger (id, aid, pipeline_id, types, enabled) VALUES ('trig-dis', 2, 'pipe-1', 'web', 0)`)
	if err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"secret": "test-secret"})
	req := httptest.NewRequest(http.MethodPost, "/trigger/web/trig-dis", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for disabled trigger, got %d", w.Code)
	}
}

func TestWeb_InvalidJSON(t *testing.T) {
	db := createHookTestDb(t)
	r := setupHookTestRouter(t, db)

	req := httptest.NewRequest(http.MethodPost, "/trigger/web/trig-1", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// GinReqParseJson should return 400 for invalid JSON
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}
