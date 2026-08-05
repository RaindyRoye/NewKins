package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

func setupPipelineVersionTestDB(t *testing.T) {
	t.Helper()
	db := setupRuntimeTestDB(t)

	// Create t_pipeline_version table explicitly
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline_version (
		id VARCHAR(64) PRIMARY KEY,
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
		t.Fatalf("create t_pipeline_version table: %v", err)
	}

	// Also create t_user_org and t_org for perm checks
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_user_org (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		perm_adm INT DEFAULT 0,
		perm_rw INT DEFAULT 0,
		perm_exec INT DEFAULT 0,
		perm_down INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_user_org table: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_org (
		id VARCHAR(64) PRIMARY KEY,
		aid BIGINT,
		uid VARCHAR(64),
		name VARCHAR(100),
		"desc" VARCHAR(500),
		avatar VARCHAR(500),
		public INT DEFAULT 0,
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_org table: %v", err)
	}
}

func makePvGinContext(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	} else {
		req = httptest.NewRequest("POST", "/test", nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	if loggedInUser != nil {
		c.Set(service.LgUserKey, loggedInUser)
	}
	return c, w
}

func TestPipelineVersionController_delete_EmptyId(t *testing.T) {
	setupPipelineVersionTestDB(t)

	c, w := makePvGinContext(t, hbtp.Map{}, nil)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	ctrl := PipelineVersionController{}
	m := &hbtp.Map{}
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineVersionController_delete_NotFound(t *testing.T) {
	setupPipelineVersionTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePvGinContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)

	ctrl := PipelineVersionController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineVersionController_delete_PipelineNotFound(t *testing.T) {
	db := setupRuntimeTestDB(t)

	// Create t_pipeline_version table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline_version (
		id VARCHAR(64) PRIMARY KEY,
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
		t.Fatalf("create t_pipeline_version table: %v", err)
	}

	// Insert a pipeline version whose pipeline doesn't exist
	_, err = db.Insert(&model.TPipelineVersion{
		Id:         "pv-1",
		PipelineId: "nonexistent-pipe",
		Created:    time.Now(),
	})
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePvGinContext(t, hbtp.Map{"id": "pv-1"}, adminUser)

	ctrl := PipelineVersionController{}
	m := &hbtp.Map{}
	m.Set("id", "pv-1")
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineVersionController_delete_Success(t *testing.T) {
	db := setupRuntimeTestDB(t)

	// Create t_pipeline_version table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline_version (
		id VARCHAR(64) PRIMARY KEY,
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
		t.Fatalf("create t_pipeline_version table: %v", err)
	}

	// Insert a pipeline
	_, err = db.Insert(&model.TPipeline{
		Id:   "pipe-1",
		Uid:  "admin",
		Name: "test-pipeline",
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Insert a pipeline version
	_, err = db.Insert(&model.TPipelineVersion{
		Id:         "pv-1",
		PipelineId: "pipe-1",
		Created:    time.Now(),
	})
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePvGinContext(t, hbtp.Map{"id": "pv-1"}, adminUser)

	ctrl := PipelineVersionController{}
	m := &hbtp.Map{}
	m.Set("id", "pv-1")
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify the pipeline version is marked as deleted
	pv := &model.TPipelineVersion{}
	ok, err := db.Where("id=?", "pv-1").Get(pv)
	if err != nil {
		t.Fatalf("query pipeline version: %v", err)
	}
	if !ok {
		t.Fatal("pipeline version not found after delete")
	}
	if pv.Deleted != 1 {
		t.Errorf("pipeline version deleted = %d, want 1", pv.Deleted)
	}
}
