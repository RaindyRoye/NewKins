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

func TestRuns_SuccessWithPipelineVersions(t *testing.T) {
	db := setupTriggerTestDb(t)

	// Also create t_pipeline_version and t_build tables for the batch query
	_, err := db.Exec(`CREATE TABLE t_pipeline_version (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		number BIGINT,
		events VARCHAR(100),
		sha VARCHAR(255),
		pipeline_name VARCHAR(255),
		pipeline_display_name VARCHAR(255),
		pipeline_id VARCHAR(64),
		version VARCHAR(255),
		content TEXT,
		created DATETIME,
		deleted INT DEFAULT 0,
		pr_number BIGINT,
		repo_clone_url VARCHAR(255)
	)`)
	if err != nil {
		t.Fatalf("create pipeline_version table: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE t_build (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		pipeline_version_id VARCHAR(64),
		status VARCHAR(100)
	)`)
	if err != nil {
		t.Fatalf("create build table: %v", err)
	}

	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipeline := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	trigger := &model.TTrigger{
		Id: "trig-1", Aid: 1, PipelineId: "pipe-1", Types: "webhook",
		Name: "test-trigger", Enabled: 1, Created: time.Now(), Updated: time.Now(), Uid: "user-1",
	}
	if _, err := db.Insert(trigger); err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	// Create pipeline version and build
	pv := &model.TPipelineVersion{
		Id: "pv-1", Number: 42, PipelineName: "pipe-1",
		PipelineDisplayName: "Test Pipeline", PipelineId: "pipe-1",
		Created: time.Now(),
	}
	if _, err := db.Insert(pv); err != nil {
		t.Fatalf("insert pv: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO t_build (id, pipeline_version_id, status) VALUES (?, ?, ?)`,
		"build-1", "pv-1", "success"); err != nil {
		t.Fatalf("insert build: %v", err)
	}

	// Create trigger run linked to the pipeline version
	run1 := &model.TTriggerRun{
		Id: "run-1", Aid: 1, Tid: "trig-1",
		PipeVersionId: "pv-1", Created: time.Now(),
	}
	if _, err := db.Insert(run1); err != nil {
		t.Fatalf("insert trigger run: %v", err)
	}

	c, w := makeTriggerGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "trig-1")
	m.Set("page", int64(1))
	ctrl.runs(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestRuns_WithTriggerRunErrors(t *testing.T) {
	db := setupTriggerTestDb(t)

	// Create t_pipeline_version and t_build tables
	_, err := db.Exec(`CREATE TABLE t_pipeline_version (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		number BIGINT,
		events VARCHAR(100),
		sha VARCHAR(255),
		pipeline_name VARCHAR(255),
		pipeline_display_name VARCHAR(255),
		pipeline_id VARCHAR(64),
		version VARCHAR(255),
		content TEXT,
		created DATETIME,
		deleted INT DEFAULT 0,
		pr_number BIGINT,
		repo_clone_url VARCHAR(255)
	)`)
	if err != nil {
		t.Fatalf("create pipeline_version table: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE t_build (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		pipeline_version_id VARCHAR(64),
		status VARCHAR(100)
	)`)
	if err != nil {
		t.Fatalf("create build table: %v", err)
	}

	ctrl := TriggerController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipeline := &model.TPipeline{Id: "pipe-1", Name: "test-pipeline", Uid: "user-1"}
	if _, err := db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	trigger := &model.TTrigger{
		Id: "trig-1", Aid: 1, PipelineId: "pipe-1", Types: "webhook",
		Name: "test-trigger", Enabled: 1, Created: time.Now(), Updated: time.Now(), Uid: "user-1",
	}
	if _, err := db.Insert(trigger); err != nil {
		t.Fatalf("insert trigger: %v", err)
	}

	// Create runs - one with error (no PipeVersionId lookup), one without error but no PipeVersionId
	run1 := &model.TTriggerRun{
		Id: "run-1", Aid: 1, Tid: "trig-1",
		Error: "some error", Created: time.Now(),
	}
	run2 := &model.TTriggerRun{
		Id: "run-2", Aid: 2, Tid: "trig-1",
		PipeVersionId: "", Created: time.Now(),
	}
	if _, err := db.Insert(run1); err != nil {
		t.Fatalf("insert run1: %v", err)
	}
	if _, err := db.Insert(run2); err != nil {
		t.Fatalf("insert run2: %v", err)
	}

	c, w := makeTriggerGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "trig-1")
	m.Set("page", int64(1))
	ctrl.runs(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
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

func setupBatchPvTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	comm.Db = db

	_, err = db.Exec(`CREATE TABLE t_pipeline_version (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		number BIGINT,
		events VARCHAR(100),
		sha VARCHAR(255),
		pipeline_name VARCHAR(255),
		pipeline_display_name VARCHAR(255),
		pipeline_id VARCHAR(64),
		version VARCHAR(255),
		content TEXT,
		created DATETIME,
		deleted INT DEFAULT 0,
		pr_number BIGINT,
		repo_clone_url VARCHAR(255)
	)`)
	if err != nil {
		t.Fatalf("create pipeline_version table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_build (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		pipeline_version_id VARCHAR(64),
		status VARCHAR(100)
	)`)
	if err != nil {
		t.Fatalf("create build table: %v", err)
	}

	return db
}

func TestBatchRunPipelineVersions_WithData(t *testing.T) {
	db := setupBatchPvTestDb(t)

	// Insert pipeline versions
	pv1 := &model.TPipelineVersion{
		Id: "pv-1", Number: 1, PipelineName: "pipe-a",
		PipelineDisplayName: "Pipeline A", PipelineId: "pid-1",
		Created: time.Now(),
	}
	pv2 := &model.TPipelineVersion{
		Id: "pv-2", Number: 2, PipelineName: "pipe-b",
		PipelineDisplayName: "Pipeline B", PipelineId: "pid-2",
		Created: time.Now(),
	}
	if _, err := db.Insert(pv1); err != nil {
		t.Fatalf("insert pv1: %v", err)
	}
	if _, err := db.Insert(pv2); err != nil {
		t.Fatalf("insert pv2: %v", err)
	}

	result, err := batchRunPipelineVersions(context.TODO(), []string{"pv-1", "pv-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result["pv-1"].Number != 1 {
		t.Errorf("expected pv-1 number=1, got %d", result["pv-1"].Number)
	}
	if result["pv-1"].PipelineName != "pipe-a" {
		t.Errorf("expected pv-1 pipelineName='pipe-a', got %q", result["pv-1"].PipelineName)
	}
	if result["pv-2"].Number != 2 {
		t.Errorf("expected pv-2 number=2, got %d", result["pv-2"].Number)
	}
}

func TestBatchRunPipelineVersions_NonexistentIDs(t *testing.T) {
	setupBatchPvTestDb(t)

	result, err := batchRunPipelineVersions(context.TODO(), []string{"nonexistent-1", "nonexistent-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results for nonexistent IDs, got %d", len(result))
	}
}

func TestBatchRunPipelineVersions_WithBuildJoin(t *testing.T) {
	db := setupBatchPvTestDb(t)

	pv := &model.TPipelineVersion{
		Id: "pv-1", Number: 5, PipelineName: "pipe-x",
		PipelineDisplayName: "Pipeline X", PipelineId: "pid-x",
		Created: time.Now(),
	}
	if _, err := db.Insert(pv); err != nil {
		t.Fatalf("insert pv: %v", err)
	}

	// Insert a build record linked to the pipeline version
	_, err := db.Exec(`INSERT INTO t_build (id, pipeline_version_id, status) VALUES (?, ?, ?)`,
		"build-1", "pv-1", "success")
	if err != nil {
		t.Fatalf("insert build: %v", err)
	}

	result, err := batchRunPipelineVersions(context.TODO(), []string{"pv-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result["pv-1"].Status != "success" {
		t.Errorf("expected status='success', got %q", result["pv-1"].Status)
	}
	if result["pv-1"].Number != 5 {
		t.Errorf("expected number=5, got %d", result["pv-1"].Number)
	}
}
