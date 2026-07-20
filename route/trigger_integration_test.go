package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupTriggerTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create tables
	_, err = db.Exec(`CREATE TABLE t_trigger (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		id VARCHAR(64) NOT NULL UNIQUE,
		pipeline_id VARCHAR(64) NOT NULL,
		uid VARCHAR(64),
		name VARCHAR(255),
		types VARCHAR(50),
		desc VARCHAR(255),
		params TEXT,
		enabled INT DEFAULT 0,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_trigger table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_trigger_run (
		id VARCHAR(64) NOT NULL,
		aid BIGINT NOT NULL,
		tid VARCHAR(64),
		pipe_version_id VARCHAR(64),
		infos TEXT,
		error VARCHAR(255),
		created DATETIME,
		PRIMARY KEY (id, aid)
	)`)
	if err != nil {
		t.Fatalf("create t_trigger_run table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(50),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		create_time DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		name VARCHAR(100),
		pass VARCHAR(255),
		nick VARCHAR(100),
		avatar VARCHAR(500),
		created DATETIME,
		login_time DATETIME,
		active INT DEFAULT 1
	)`)
	if err != nil {
		t.Fatalf("create t_user table: %v", err)
	}

	comm.Db = db
	return db
}

func setupTriggerRouter(t *testing.T, db *xorm.Engine) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// Create test user in database
	testUser := &model.TUser{
		Id:      "test-user",
		Name:    "tester",
		Nick:    "Test User",
		Active:  1,
		Created: time.Now(),
	}
	_, err := db.InsertOne(testUser)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Generate JWT token for test user using HS512 (matches util.CreateToken)
	loginKey := "test-secret-key-for-trigger-tests-2024"
	comm.Cfg.Server.LoginKey = loginKey
	tokenClaims := jwt.MapClaims{
		"uid": testUser.Id,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, tokenClaims)
	tokenString, err := token.SignedString([]byte(loginKey))
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}

	// Create router with authentication via cookie
	r := gin.New()
	r.Use(func(c *gin.Context) {
		// Set cookie with test token (cookie name must be "gokinstk")
		c.Request.AddCookie(&http.Cookie{
			Name:  "gokinstk",
			Value: tokenString,
		})
		c.Next()
	})

	tc := &TriggerController{}
	triggerGroup := r.Group("/api/trigger")
	tc.Routes(triggerGroup)
	return r
}

func TestTriggerController_triggers_EmptyPipelineId(t *testing.T) {
	db := setupTriggerTestDB(t)
	r := setupTriggerRouter(t, db)

	body, _ := json.Marshal(map[string]any{"pipelineId": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/trigger/triggers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pipelineId, got %d", w.Code)
	}
}

func TestTriggerController_triggers_PipelineNotFound(t *testing.T) {
	db := setupTriggerTestDB(t)
	r := setupTriggerRouter(t, db)

	body, _ := json.Marshal(map[string]any{"pipelineId": "nonexistent"})
	req := httptest.NewRequest(http.MethodPost, "/api/trigger/triggers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestTriggerController_triggers_Success(t *testing.T) {
	db := setupTriggerTestDB(t)
	r := setupTriggerRouter(t, db)

	// Create a pipeline
	_, err := db.Exec(`INSERT INTO t_pipeline (id, uid, name, deleted) VALUES ('pipe-1', 'test-user', 'test-pipe', 0)`)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create triggers
	_, err = db.Exec(`INSERT INTO t_trigger (id, aid, uid, pipeline_id, types, name, enabled) VALUES ('trig-1', 1, 'test-user', 'pipe-1', 'webhook', 'test-trigger', 1)`)
	if err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"pipelineId": "pipe-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/trigger/triggers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestTriggerController_save_ValidationError(t *testing.T) {
	db := setupTriggerTestDB(t)
	r := setupTriggerRouter(t, db)

	// Missing required fields
	body, _ := json.Marshal(map[string]any{
		"pipelineId": "",
		"name":       "",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/trigger/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for validation error, got %d", w.Code)
	}
}

func TestTriggerController_save_PipelineNotFound(t *testing.T) {
	db := setupTriggerTestDB(t)
	r := setupTriggerRouter(t, db)

	body, _ := json.Marshal(map[string]any{
		"pipelineId": "nonexistent",
		"name":       "test",
		"types":      "webhook",
		"params":     `{"hookType": "github"}`,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/trigger/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestTriggerController_save_Success(t *testing.T) {
	db := setupTriggerTestDB(t)
	r := setupTriggerRouter(t, db)

	// Create a pipeline
	_, err := db.Exec(`INSERT INTO t_pipeline (id, uid, name, deleted) VALUES ('pipe-1', 'test-user', 'test-pipe', 0)`)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	body, _ := json.Marshal(map[string]any{
		"pipelineId": "pipe-1",
		"name":       "new-trigger",
		"types":      "webhook",
		"params":     `{"hookType": "github"}`,
		"enabled":    true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/trigger/save", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Expect 200 OK - trigger should be created successfully
	// Note: aid is auto-incremented by SQLite when not specified
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestTriggerController_delete_NotFound(t *testing.T) {
	db := setupTriggerTestDB(t)
	r := setupTriggerRouter(t, db)

	body, _ := json.Marshal(map[string]any{"id": "nonexistent"})
	req := httptest.NewRequest(http.MethodPost, "/api/trigger/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent trigger, got %d", w.Code)
	}
}

func TestTriggerController_delete_Success(t *testing.T) {
	db := setupTriggerTestDB(t)
	r := setupTriggerRouter(t, db)

	// Create pipeline and trigger
	_, err := db.Exec(`INSERT INTO t_pipeline (id, uid, name, deleted) VALUES ('pipe-1', 'test-user', 'test-pipe', 0)`)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_trigger (id, aid, uid, pipeline_id, types, name, enabled) VALUES ('trig-1', 1, 'test-user', 'pipe-1', 'webhook', 'test', 1)`)
	if err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"id": "trig-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/trigger/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify trigger is deleted
	var count int64
	count, err = db.Where("id=?", "trig-1").Count(&model.TTrigger{})
	if err != nil {
		t.Fatalf("count triggers: %v", err)
	}
	if count != 0 {
		t.Errorf("expected trigger to be deleted, count=%d", count)
	}
}

func TestTriggerController_runs_NotFound(t *testing.T) {
	db := setupTriggerTestDB(t)
	r := setupTriggerRouter(t, db)

	body, _ := json.Marshal(map[string]any{"id": "nonexistent"})
	req := httptest.NewRequest(http.MethodPost, "/api/trigger/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent trigger, got %d", w.Code)
	}
}

func TestTriggerController_runs_Success(t *testing.T) {
	db := setupTriggerTestDB(t)
	r := setupTriggerRouter(t, db)

	// Create pipeline and trigger
	_, err := db.Exec(`INSERT INTO t_pipeline (id, uid, name, deleted) VALUES ('pipe-1', 'test-user', 'test-pipe', 0)`)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	_, err = db.Exec(`INSERT INTO t_trigger (id, aid, uid, pipeline_id, types, name, enabled) VALUES ('trig-1', 1, 'test-user', 'pipe-1', 'webhook', 'test', 1)`)
	if err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	// Create trigger runs
	_, err = db.Exec(`INSERT INTO t_trigger_run (id, aid, tid, created) VALUES ('run-1', 1, 'trig-1', ?)`, time.Now())
	if err != nil {
		t.Fatalf("insert trigger run: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"id": "trig-1", "page": 1})
	req := httptest.NewRequest(http.MethodPost, "/api/trigger/runs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}
