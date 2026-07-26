package route

import (
	"bytes"
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

// setupTriggerTestDB creates an in-memory SQLite database with tables needed
// by the TriggerController and PipelineVersionController tests.
func setupTriggerTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		comm.Db = origDb
	})

	tables := []string{
		`CREATE TABLE t_user (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			name VARCHAR(100),
			pass VARCHAR(255),
			nick VARCHAR(100),
			avatar VARCHAR(500),
			created DATETIME,
			login_time DATETIME,
			active INT DEFAULT 0
		)`,
		`CREATE TABLE t_pipeline (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			uid VARCHAR(64),
			name VARCHAR(255),
			display_name VARCHAR(255),
			pipeline_type VARCHAR(255),
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			create_time DATETIME
		)`,
		`CREATE TABLE t_trigger (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			pipeline_id VARCHAR(64),
			types VARCHAR(50),
			name VARCHAR(100),
			"desc" VARCHAR(255),
			params TEXT,
			enabled INT DEFAULT 0,
			created DATETIME,
			updated DATETIME
		)`,
		`CREATE TABLE t_trigger_run (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			tid VARCHAR(64),
			pipe_version_id VARCHAR(64),
			infos TEXT,
			error VARCHAR(255),
			created DATETIME
		)`,
		`CREATE TABLE t_pipeline_version (
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
		)`,
		`CREATE TABLE t_build (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			pipeline_id VARCHAR(64),
			pipeline_version_id VARCHAR(64),
			status VARCHAR(100),
			error VARCHAR(500),
			event VARCHAR(100),
			started DATETIME,
			finished DATETIME,
			created DATETIME,
			updated DATETIME,
			version VARCHAR(255)
		)`,
	}

	for _, ddl := range tables {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("exec DDL: %v", err)
		}
	}

	comm.Db = db
	return db
}

func makeTriggerGinCtx(t *testing.T, body interface{}, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest("POST", "/test", bytes.NewReader(b))
	} else {
		req = httptest.NewRequest("POST", "/test", nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	if user != nil {
		c.Set(service.LgUserKey, user)
	}
	return c, w
}

// --- TriggerController.triggers ---

func TestTriggerController_triggers_MissingPipelineId(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	m := &hbtp.Map{}
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := TriggerController{}
	ctrl.triggers(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTriggerController_triggers_PipelineNotFound(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := TriggerController{}
	ctrl.triggers(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestTriggerController_triggers_Success(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TPipeline{Id: "pipe1", Name: "test-pipe"})
	_, _ = comm.Db.InsertOne(&model.TTrigger{
		Id:         "trig1",
		PipelineId: "pipe1",
		Types:      "webhook",
		Name:       "test-trigger",
		Enabled:    1,
		Created:    time.Now(),
	})

	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe1")
	m.Set("page", int64(1))
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := TriggerController{}
	ctrl.triggers(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- TriggerController.save ---

func TestTriggerController_save_MissingPipelineId(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	tp := &bean.TriggerParam{
		PipelineId: "",
		Types:      "webhook",
		Name:       "test",
		Params:     "{}",
	}
	c, w := makeTriggerGinCtx(t, tp, admin)
	ctrl := TriggerController{}
	ctrl.save(c, tp)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestTriggerController_save_MissingTypes(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	tp := &bean.TriggerParam{
		PipelineId: "pipe1",
		Types:      "",
		Name:       "test",
		Params:     "{}",
	}
	c, w := makeTriggerGinCtx(t, tp, admin)
	ctrl := TriggerController{}
	ctrl.save(c, tp)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestTriggerController_save_PipelineNotFound(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	tp := &bean.TriggerParam{
		PipelineId: "nonexistent",
		Types:      "webhook",
		Name:       "test",
		Params:     "{}",
	}
	c, w := makeTriggerGinCtx(t, tp, admin)
	ctrl := TriggerController{}
	ctrl.save(c, tp)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestTriggerController_save_SuccessCreate(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TPipeline{Id: "pipe1", Uid: "admin", Name: "test-pipe"})

	tp := &bean.TriggerParam{
		PipelineId: "pipe1",
		Types:      "webhook",
		Name:       "test-trigger",
		Params:     `{"secret":"test"}`,
		Enabled:    true,
	}
	c, w := makeTriggerGinCtx(t, tp, admin)
	ctrl := TriggerController{}
	ctrl.save(c, tp)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- TriggerController.delete ---

func TestTriggerController_delete_MissingId(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	m := &hbtp.Map{}
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := TriggerController{}
	ctrl.delete(c, m)

	// Missing id: DB query with empty id returns not found
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestTriggerController_delete_NotFound(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := TriggerController{}
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestTriggerController_delete_Success(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TPipeline{Id: "pipe1", Uid: "admin", Name: "test-pipe"})
	_, _ = comm.Db.InsertOne(&model.TTrigger{
		Id:         "trig1",
		PipelineId: "pipe1",
		Types:      "webhook",
		Name:       "test-trigger",
		Enabled:    1,
		Created:    time.Now(),
	})

	m := &hbtp.Map{}
	m.Set("id", "trig1")
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := TriggerController{}
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- TriggerController.runs ---

func TestTriggerController_runs_MissingId(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	m := &hbtp.Map{}
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := TriggerController{}
	ctrl.runs(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestTriggerController_runs_NotFound(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := TriggerController{}
	ctrl.runs(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestTriggerController_runs_Success(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TPipeline{Id: "pipe1", Uid: "admin", Name: "test-pipe"})
	_, _ = comm.Db.InsertOne(&model.TTrigger{
		Id:         "trig1",
		PipelineId: "pipe1",
		Types:      "webhook",
		Name:       "test-trigger",
		Enabled:    1,
		Created:    time.Now(),
	})
	_, _ = comm.Db.InsertOne(&model.TTriggerRun{
		Id:      "run1",
		Tid:     "trig1",
		Created: time.Now(),
	})

	m := &hbtp.Map{}
	m.Set("id", "trig1")
	m.Set("page", int64(1))
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := TriggerController{}
	ctrl.runs(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- PipelineVersionController.delete ---

func TestPipelineVersionController_delete_MissingId(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	m := &hbtp.Map{}
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := PipelineVersionController{}
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineVersionController_delete_NotFound(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := PipelineVersionController{}
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineVersionController_delete_Success(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TPipeline{Id: "pipe1", Uid: "admin", Name: "test-pipe"})
	_, _ = comm.Db.InsertOne(&model.TPipelineVersion{
		Id:         "ver1",
		PipelineId: "pipe1",
		Number:     1,
		Created:    time.Now(),
	})

	m := &hbtp.Map{}
	m.Set("id", "ver1")
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := PipelineVersionController{}
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineVersionController_delete_PipelineNotFound(t *testing.T) {
	setupTriggerTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "superadmin", Active: 1}
	_, _ = comm.Db.InsertOne(&model.TPipelineVersion{
		Id:         "ver1",
		PipelineId: "nonexistent",
		Number:     1,
		Created:    time.Now(),
	})

	m := &hbtp.Map{}
	m.Set("id", "ver1")
	c, w := makeTriggerGinCtx(t, m, admin)
	ctrl := PipelineVersionController{}
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
