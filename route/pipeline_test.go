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

func setupPipelineTestDB(t *testing.T) *xorm.Engine {
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

	// Create t_pipeline table
	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(255),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		create_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline table: %v", err)
	}

	// Create t_pipeline_info view or table (some code uses this)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline_info (
		id VARCHAR(64) PRIMARY KEY,
		aid BIGINT,
		uid VARCHAR(64),
		name VARCHAR(100),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(50),
		yml_content TEXT,
		url VARCHAR(500),
		username VARCHAR(100),
		access_token VARCHAR(500),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_info table: %v", err)
	}

	// Create t_pipeline_conf table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline_conf (
		id VARCHAR(64) PRIMARY KEY,
		aid BIGINT,
		pipeline_id VARCHAR(64),
		yml_content TEXT,
		url VARCHAR(500),
		username VARCHAR(100),
		access_token VARCHAR(500)
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_conf table: %v", err)
	}

	// Create t_pipeline_var table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline_var (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		pipeline_id VARCHAR(64),
		name VARCHAR(255),
		value TEXT,
		remarks VARCHAR(255),
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_var table: %v", err)
	}

	// Create t_org_pipe table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_org_pipe (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		created DATETIME,
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_org_pipe table: %v", err)
	}

	// Create t_org table
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

	// Create t_user_org table
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

	// Create t_user table
	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		name VARCHAR(100),
		pass VARCHAR(255),
		nick VARCHAR(100),
		avatar VARCHAR(500),
		created DATETIME,
		login_time DATETIME,
		active INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_user table: %v", err)
	}

	// Create t_user_info table
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
		t.Fatalf("create t_user_info table: %v", err)
	}

	// Create t_pipeline_version table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline_version (
		id VARCHAR(64) PRIMARY KEY,
		aid BIGINT,
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

	// Create t_build table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_build (
		id VARCHAR(64) PRIMARY KEY,
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
	)`)
	if err != nil {
		t.Fatalf("create t_build table: %v", err)
	}

	comm.Db = db
	return db
}

func makePipeGinContext(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestPipelineController_orgPipelines_EmptyOrgId(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"orgId": ""}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_orgPipelines_OrgNotFound(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"orgId": "nonexistent"}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_orgPipelines_Success(t *testing.T) {
	db := setupPipelineTestDB(t)

	// Create an organization
	_, err := db.Insert(&model.TOrgInfo{
		Id:      "org-1",
		Name:    "test-org",
		Deleted: 0,
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Create a pipeline
	_, err = db.Insert(&model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline",
		Deleted: 0,
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Link pipeline to org
	_, err = db.Exec(`INSERT INTO t_org_pipe (org_id, pipe_id, created) VALUES (?, ?, ?)`,
		"org-1", "pipe-1", time.Now())
	if err != nil {
		t.Fatalf("insert org_pipe: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"orgId": "org-1", "page": int64(1)}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("page", int64(1))
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_getPipelines_Success(t *testing.T) {
	db := setupPipelineTestDB(t)

	// Create pipelines
	_, err := db.Insert(&model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline-1",
		Deleted: 0,
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"page": int64(1)}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("page", int64(1))
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_delete_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"id": ""}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_delete_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_delete_Success(t *testing.T) {
	db := setupPipelineTestDB(t)

	// Create a pipeline
	_, err := db.Insert(&model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline",
		Deleted: 0,
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"id": "pipe-1"}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify pipeline is marked as deleted
	pipe := &model.TPipeline{}
	ok, err := db.Where("id=?", "pipe-1").Get(pipe)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline not found after delete")
	}
	if pipe.Deleted != 1 {
		t.Errorf("pipeline deleted = %d, want 1", pipe.Deleted)
	}
}

func TestPipelineController_info_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"id": ""}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_info_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_save_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"pipelineId": ""}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_searchSha_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"id": ""}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	ctrl.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_searchSha_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_searchSha_Success(t *testing.T) {
	db := setupPipelineTestDB(t)

	// Create a pipeline
	_, err := db.Insert(&model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline",
		Deleted: 0,
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create pipeline versions with SHA
	_, err = db.Insert(&model.TPipelineVersion{
		Id:         "pv-1",
		PipelineId: "pipe-1",
		Sha:        "abc123",
		Created:    time.Now(),
	})
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"id": "pipe-1"}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_vars_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"pipelineId": ""}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_vars_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"pipelineId": "nonexistent"}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_vars_Success(t *testing.T) {
	db := setupPipelineTestDB(t)

	// Create a pipeline
	_, err := db.Insert(&model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline",
		Deleted: 0,
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create pipeline vars
	_, err = db.Exec(`INSERT INTO t_pipeline_var (pipeline_id, name, value, remarks, public) VALUES (?, ?, ?, ?, ?)`,
		"pipe-1", "VAR1", "value1", "test var", 1)
	if err != nil {
		t.Fatalf("insert pipeline var: %v", err)
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"pipelineId": "pipe-1", "page": int64(1)}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("page", int64(1))
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_varDel_InvalidAid(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"aid": int64(0)}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varDel_NotFound(t *testing.T) {
	setupPipelineTestDB(t)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"aid": int64(999)}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(999))
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_varDel_Success(t *testing.T) {
	db := setupPipelineTestDB(t)

	// Create a pipeline
	_, err := db.Insert(&model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipeline",
		Deleted: 0,
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create a pipeline var
	_, err = db.Exec(`INSERT INTO t_pipeline_var (pipeline_id, name, value, remarks, public) VALUES (?, ?, ?, ?, ?)`,
		"pipe-1", "VAR1", "value1", "test var", 1)
	if err != nil {
		t.Fatalf("insert pipeline var: %v", err)
	}

	// Get the aid of the inserted var
	type aidRow struct {
		Aid int64 `xorm:"aid"`
	}
	var row aidRow
	ok, err := db.SQL("SELECT aid FROM t_pipeline_var WHERE name=?", "VAR1").Get(&row)
	if err != nil {
		t.Fatalf("query pipeline var aid: %v", err)
	}
	if !ok {
		t.Fatal("pipeline var not found")
	}

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipeGinContext(t, hbtp.Map{"aid": row.Aid}, adminUser)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", row.Aid)
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify the var is deleted
	count, err := db.Where("aid=?", row.Aid).Count(&model.TPipelineVar{})
	if err != nil {
		t.Fatalf("count pipeline vars: %v", err)
	}
	if count != 0 {
		t.Errorf("pipeline var count = %d, want 0", count)
	}
}
