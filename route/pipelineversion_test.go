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
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupPipelineVersionTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

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
		t.Fatalf("create t_pipeline_version: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(255),
		access_token VARCHAR(255),
		url VARCHAR(500),
		username VARCHAR(255),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		create_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline_info (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(255),
		access_token VARCHAR(255),
		url VARCHAR(500),
		username VARCHAR(255),
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_info: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org_pipe (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		org_id VARCHAR(64),
		pipeline_id VARCHAR(64)
	)`)
	if err != nil {
		t.Fatalf("create t_org_pipe: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_info (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		phone VARCHAR(100),
		email VARCHAR(200),
		birthday DATETIME,
		remark TEXT,
		perm_user INT,
		perm_org INT,
		perm_pipe INT
	)`)
	if err != nil {
		t.Fatalf("create t_user_info: %v", err)
	}

	comm.Db = db
	return db
}

func makePVGinCtx(t *testing.T, body interface{}, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

	if user != nil {
		c.Set(service.LgUserKey, user)
	}
	return c, w
}

func TestPipelineVersionController_Path(t *testing.T) {
	c := &PipelineVersionController{}
	if got := c.GetPath(); got != "/api/pipelineVersion" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/pipelineVersion")
	}
}

func TestPV_Delete_EmptyId(t *testing.T) {
	setupPipelineVersionTestDB(t)
	ctrl := PipelineVersionController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePVGinCtx(t, m, nil)
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPV_Delete_VersionNotFound(t *testing.T) {
	setupPipelineVersionTestDB(t)
	ctrl := PipelineVersionController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-pv")
	c, w := makePVGinCtx(t, m, nil)
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestPV_Delete_PipelineNotFound(t *testing.T) {
	db := setupPipelineVersionTestDB(t)
	ctrl := PipelineVersionController{}

	// Insert a pipeline version without a corresponding pipeline
	pv := &model.TPipelineVersion{
		Id:         "pv-orphan",
		PipelineId: "nonexistent-pipeline",
		Number:     1,
		Created:    time.Now(),
	}
	_, _ = db.InsertOne(pv)

	m := &hbtp.Map{}
	m.Set("id", "pv-orphan")
	c, w := makePVGinCtx(t, m, nil)
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

func TestPV_Delete_NoPermission(t *testing.T) {
	db := setupPipelineVersionTestDB(t)
	ctrl := PipelineVersionController{}

	// Insert pipeline and pipeline version
	pipe := &model.TPipeline{
		Id:          "pipe-1",
		Name:        "test-pipe",
		DisplayName: "Test Pipeline",
		CreateTime:  time.Now(),
	}
	_, _ = db.InsertOne(pipe)

	pv := &model.TPipelineVersion{
		Id:         "pv-1",
		PipelineId: "pipe-1",
		Number:     1,
		Created:    time.Now(),
	}
	_, _ = db.InsertOne(pv)

	// User without write permission (non-admin, no perm_pipe)
	user := &model.TUser{Id: "usr-noperm", Name: "noperm", Active: 1}

	m := &hbtp.Map{}
	m.Set("id", "pv-1")
	c, w := makePVGinCtx(t, m, user)
	ctrl.delete(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func TestPV_Delete_Success(t *testing.T) {
	db := setupPipelineVersionTestDB(t)
	ctrl := PipelineVersionController{}

	// Insert pipeline and pipeline version
	pipe := &model.TPipeline{
		Id:          "pipe-del",
		Name:        "del-pipe",
		DisplayName: "Delete Pipeline",
		CreateTime:  time.Now(),
	}
	_, _ = db.InsertOne(pipe)

	pv := &model.TPipelineVersion{
		Id:         "pv-del",
		PipelineId: "pipe-del",
		Number:     1,
		Created:    time.Now(),
	}
	_, _ = db.InsertOne(pv)

	// Admin user with full permissions
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	_, _ = db.Exec(`INSERT INTO t_user_info (id, perm_user, perm_org, perm_pipe) VALUES ('admin', 1, 1, 1)`)

	m := &hbtp.Map{}
	m.Set("id", "pv-del")
	c, w := makePVGinCtx(t, m, user)
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify the pipeline version is soft-deleted
	updated := &model.TPipelineVersion{}
	ok, err := db.Where("id = ?", "pv-del").Get(updated)
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
