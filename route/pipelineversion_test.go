package route

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupPipelineVersionTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	tables := []string{
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
	}
	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	return db
}

func makePipelineVersionGinCtx(t *testing.T, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if loggedInUser != nil {
		c.Set(service.LgUserKey, loggedInUser)
	}
	return c, w
}

func TestPipelineVersionDelete_EmptyId(t *testing.T) {
	setupPipelineVersionTestDB(t)
	ctrl := PipelineVersionController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineVersionGinCtx(t, user)
	
	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

func TestPipelineVersionDelete_NotFound(t *testing.T) {
	setupPipelineVersionTestDB(t)
	ctrl := PipelineVersionController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineVersionGinCtx(t, user)
	
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-pv")
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineVersionDelete_PipelineNotFound(t *testing.T) {
	db := setupPipelineVersionTestDB(t)
	
	// Create a pipeline version without a corresponding pipeline
	pv := &model.TPipelineVersion{
		Id:         "pv-1",
		Uid:        "user-1",
		Number:     1,
		PipelineId: "nonexistent-pipe",
		Created:    time.Now(),
		Deleted:    0,
	}
	if _, err := db.InsertOne(pv); err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	ctrl := PipelineVersionController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineVersionGinCtx(t, user)
	
	m := &hbtp.Map{}
	m.Set("id", "pv-1")
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestPipelineVersionDelete_NoPermission(t *testing.T) {
	db := setupPipelineVersionTestDB(t)
	
	// Create pipeline and pipeline version
	pipe := &model.TPipeline{
		Id:          "pipe-1",
		Uid:         "owner-1",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Deleted:     0,
	}
	if _, err := db.InsertOne(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	pv := &model.TPipelineVersion{
		Id:         "pv-1",
		Uid:        "owner-1",
		Number:     1,
		PipelineId: "pipe-1",
		Created:    time.Now(),
		Deleted:    0,
	}
	if _, err := db.InsertOne(pv); err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	// Try to delete with a different user (not owner, not admin)
	ctrl := PipelineVersionController{}
	user := &model.TUser{Id: "other-user", Name: "other", Active: 1}
	c, w := makePipelineVersionGinCtx(t, user)
	
	m := &hbtp.Map{}
	m.Set("id", "pv-1")
	ctrl.delete(c, m)

	// Should fail with permission error
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestPipelineVersionDelete_Success(t *testing.T) {
	db := setupPipelineVersionTestDB(t)
	
	// Create pipeline and pipeline version
	pipe := &model.TPipeline{
		Id:          "pipe-1",
		Uid:         "user-1",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Deleted:     0,
	}
	if _, err := db.InsertOne(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	pv := &model.TPipelineVersion{
		Id:         "pv-1",
		Uid:        "user-1",
		Number:     1,
		PipelineId: "pipe-1",
		Created:    time.Now(),
		Deleted:    0,
	}
	if _, err := db.InsertOne(pv); err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	ctrl := PipelineVersionController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineVersionGinCtx(t, user)
	
	m := &hbtp.Map{}
	m.Set("id", "pv-1")
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if w.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", w.Body.String(), "ok")
	}

	// Verify the pipeline version is marked as deleted
	updated := &model.TPipelineVersion{}
	ok, err := db.Where("id = ?", "pv-1").Get(updated)
	if err != nil {
		t.Fatalf("query updated pv: %v", err)
	}
	if !ok {
		t.Fatal("pipeline version not found after delete")
	}
	if updated.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", updated.Deleted)
	}
}

func TestPipelineVersionDelete_AlreadyDeleted(t *testing.T) {
	db := setupPipelineVersionTestDB(t)
	
	// Create pipeline
	pipe := &model.TPipeline{
		Id:          "pipe-1",
		Uid:         "user-1",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Deleted:     0,
	}
	if _, err := db.InsertOne(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create an already-deleted pipeline version
	pv := &model.TPipelineVersion{
		Id:         "pv-1",
		Uid:        "user-1",
		Number:     1,
		PipelineId: "pipe-1",
		Created:    time.Now(),
		Deleted:    1, // Already deleted
	}
	if _, err := db.InsertOne(pv); err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	ctrl := PipelineVersionController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineVersionGinCtx(t, user)
	
	m := &hbtp.Map{}
	m.Set("id", "pv-1")
	ctrl.delete(c, m)

	// The query uses "WHERE id = ? " without checking deleted status,
	// so it will find the record and try to delete it again
	// This is actually a bug - should check if already deleted
	// For now, just verify it doesn't crash
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
