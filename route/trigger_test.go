package route

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupTriggerTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	comm.Db = db

	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(255),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		create_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create pipeline table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_trigger (
		id VARCHAR(64) NOT NULL,
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		pipeline_id VARCHAR(64),
		types VARCHAR(50),
		name VARCHAR(100),
		"desc" VARCHAR(255),
		params TEXT,
		enabled INT DEFAULT 0,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create trigger table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_trigger_run (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		tid VARCHAR(64),
		pipe_version_id VARCHAR(64),
		infos TEXT,
		error VARCHAR(255),
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create trigger_run table: %v", err)
	}

	return db
}

func makeTriggerGinCtx(t *testing.T, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if user != nil {
		c.Set(service.LgUserKey, user)
	}
	return c, w
}

func TestTriggers_MissingPipelineId(t *testing.T) {
	setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeTriggerGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", int64(1))
	ctrl.triggers(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing pipelineId, got %d", w.Code)
	}
}

func TestTriggers_PipelineNotFound(t *testing.T) {
	setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeTriggerGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent-pipe")
	m.Set("page", int64(1))
	ctrl.triggers(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestTriggers_Success(t *testing.T) {
	db := setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipeline := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	trigger := &model.TTrigger{
		Id: "trig-1", Aid: 1, PipelineId: "pipe-1",
		Types: "timer", Name: "trigger-1", Enabled: 1,
		Created: time.Now(), Updated: time.Now(),
	}
	if _, err := db.Insert(trigger); err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	c, w := makeTriggerGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("page", int64(1))
	ctrl.triggers(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["host"] == nil {
		t.Error("expected host in response")
	}
}

func TestTriggers_WithQueryFilter(t *testing.T) {
	db := setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipeline := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	triggers := []*model.TTrigger{
		{Id: "trig-1", Aid: 1, PipelineId: "pipe-1", Types: "timer", Name: "timer-trigger", Enabled: 1, Created: time.Now()},
		{Id: "trig-2", Aid: 2, PipelineId: "pipe-1", Types: "webhook", Name: "webhook-trigger", Enabled: 1, Created: time.Now()},
	}
	for _, tr := range triggers {
		if _, err := db.Insert(tr); err != nil {
			t.Fatalf("insert trigger: %v", err)
		}
	}

	c, w := makeTriggerGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("types", "timer")
	m.Set("page", int64(1))
	ctrl.triggers(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestSave_MissingFields(t *testing.T) {
	setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeTriggerGinCtx(t, user)

	tp := &bean.TriggerParam{}
	ctrl.save(c, tp)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", w.Code)
	}
}

func TestSave_PipelineNotFound(t *testing.T) {
	setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeTriggerGinCtx(t, user)

	tp := &bean.TriggerParam{
		PipelineId: "nonexistent",
		Types:      "timer",
		Name:       "test-trigger",
		Params:     "{}",
	}
	ctrl.save(c, tp)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestSave_CreateNewTrigger(t *testing.T) {
	db := setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipeline := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	c, w := makeTriggerGinCtx(t, user)
	tp := &bean.TriggerParam{
		PipelineId: "pipe-1",
		Types:      "webhook", // Use webhook instead of timer to avoid timer engine issues
		Name:       "new-trigger",
		Desc:       "A test trigger",
		Params:     `{"url":"http://example.com"}`,
		Enabled:    true,
	}
	ctrl.save(c, tp)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	count, err := db.Where("name = ? AND pipeline_id = ?", "new-trigger", "pipe-1").Count(&model.TTrigger{})
	if err != nil {
		t.Fatalf("count triggers: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 trigger created, got %d", count)
	}
}

func TestSave_UpdateExistingTrigger(t *testing.T) {
	db := setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipeline := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	trigger := &model.TTrigger{
		Id: "trig-1", Aid: 1, PipelineId: "pipe-1", Types: "webhook",
		Name: "old-name", Enabled: 0, Params: `{"url":"http://old.com"}`,
		Created: time.Now(), Updated: time.Now(), Uid: "user-1",
	}
	if _, err := db.Insert(trigger); err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	c, w := makeTriggerGinCtx(t, user)
	tp := &bean.TriggerParam{
		Id: "trig-1", PipelineId: "pipe-1", Types: "webhook",
		Name: "new-name", Enabled: true, Params: `{"url":"http://new.com"}`,
	}
	ctrl.save(c, tp)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var updated model.TTrigger
	if _, err := db.Where("id = ?", "trig-1").Get(&updated); err != nil {
		t.Fatalf("get trigger: %v", err)
	}
	if updated.Name != "new-name" {
		t.Errorf("expected name 'new-name', got %q", updated.Name)
	}
	if updated.Enabled != 1 {
		t.Errorf("expected enabled = 1, got %d", updated.Enabled)
	}
}

func TestDelete_MissingId(t *testing.T) {
	setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeTriggerGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.delete(c, m)

	// Empty ID returns 404 (not found) since the DB query returns no results
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing id, got %d", w.Code)
	}
}

func TestDelete_TriggerNotFound(t *testing.T) {
	setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeTriggerGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent trigger, got %d", w.Code)
	}
}

func TestDelete_Success(t *testing.T) {
	db := setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipeline := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Use webhook type to avoid timer engine nil pointer panic
	trigger := &model.TTrigger{
		Id: "trig-1", Aid: 1, PipelineId: "pipe-1", Types: "webhook",
		Name: "to-delete", Enabled: 1, Params: `{"url":"http://example.com"}`,
		Created: time.Now(), Updated: time.Now(), Uid: "user-1",
	}
	if _, err := db.Insert(trigger); err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	c, w := makeTriggerGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "trig-1")
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify trigger was deleted
	var count int64
	count, err := db.Where("id = ?", "trig-1").Count(&model.TTrigger{})
	if err != nil {
		t.Fatalf("count triggers: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 triggers after delete, got %d", count)
	}
}

func TestRuns_MissingId(t *testing.T) {
	setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeTriggerGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.runs(c, m)

	// API returns 404 when trigger not found (empty id matches no records)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing id, got %d", w.Code)
	}
}

func TestRuns_TriggerNotFound(t *testing.T) {
	setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makeTriggerGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.runs(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent trigger, got %d", w.Code)
	}
}

func TestRuns_Success(t *testing.T) {
	db := setupTriggerTestDb(t)
	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipeline := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	trigger := &model.TTrigger{
		Id: "trig-1", Aid: 1, PipelineId: "pipe-1", Types: "timer",
		Name: "test-trigger", Enabled: 1, Created: time.Now(), Updated: time.Now(), Uid: "user-1",
	}
	if _, err := db.Insert(trigger); err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	runs := []*model.TTriggerRun{
		{Id: "run-1", Aid: 1, Tid: "trig-1", Created: time.Now()},
		{Id: "run-2", Aid: 2, Tid: "trig-1", Created: time.Now()},
	}
	for _, r := range runs {
		if _, err := db.Insert(r); err != nil {
			t.Fatalf("insert trigger run: %v", err)
		}
	}

	c, w := makeTriggerGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "trig-1")
	m.Set("page", int64(1))
	ctrl.runs(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestBatchRunPipelineVersions_EmptyCtx(t *testing.T) {
	result, err := batchRunPipelineVersions(context.TODO(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestBatchRunPipelineVersions_NoIDs(t *testing.T) {
	result, err := batchRunPipelineVersions(context.TODO(), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result for empty IDs, got %d entries", len(result))
	}
}
