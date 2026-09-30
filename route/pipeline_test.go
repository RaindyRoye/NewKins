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
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

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

	// Create t_pipeline_conf table
	_, err = db.Exec(`CREATE TABLE t_pipeline_conf (
		id VARCHAR(64) PRIMARY KEY,
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
	_, err = db.Exec(`CREATE TABLE t_pipeline_var (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		pipeline_id VARCHAR(64),
		name VARCHAR(100),
		value TEXT,
		remarks VARCHAR(500),
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_var table: %v", err)
	}

	// Create t_pipeline_version table
	_, err = db.Exec(`CREATE TABLE t_pipeline_version (
		id VARCHAR(64) PRIMARY KEY,
		pipeline_id VARCHAR(64),
		uid VARCHAR(64),
		number BIGINT,
		sha VARCHAR(100),
		events VARCHAR(100),
		pipeline_name VARCHAR(255),
		pipeline_display_name VARCHAR(255),
		version VARCHAR(255),
		content TEXT,
		err VARCHAR(500),
		deleted INT DEFAULT 0,
		pr_number BIGINT,
		repo_clone_url VARCHAR(255),
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_version table: %v", err)
	}

	// Create t_build table
	_, err = db.Exec(`CREATE TABLE t_build (
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

	// Create t_user table
	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) PRIMARY KEY,
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
		id VARCHAR(64) PRIMARY KEY,
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

	// Create t_org table
	_, err = db.Exec(`CREATE TABLE t_org (
		id VARCHAR(64) PRIMARY KEY,
		name VARCHAR(100),
		display_name VARCHAR(255),
		deleted INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_org table: %v", err)
	}

	// Create t_org_pipe table
	_, err = db.Exec(`CREATE TABLE t_org_pipe (
		id VARCHAR(64) PRIMARY KEY,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		public INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_org_pipe table: %v", err)
	}

	// Create t_org_user table
	_, err = db.Exec(`CREATE TABLE t_org_user (
		id VARCHAR(64) PRIMARY KEY,
		org_id VARCHAR(64),
		user_id VARCHAR(64),
		perm_read INT DEFAULT 0,
		perm_write INT DEFAULT 0,
		perm_exec INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_org_user table: %v", err)
	}

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	return db
}

func makePipelineGinContext(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

// Test PipelineController.delete with empty ID
func TestPipelineController_delete_EmptyID(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"id": ""}, adminUser)
	ctrl := PipelineController{}
	ctrl.delete(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

// Test PipelineController.delete with nonexistent pipeline
func TestPipelineController_delete_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)
	ctrl := PipelineController{}
	ctrl.delete(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test PipelineController.info with empty ID
func TestPipelineController_info_EmptyID(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"id": ""}, adminUser)
	ctrl := PipelineController{}
	ctrl.info(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

// Test PipelineController.info with nonexistent pipeline
func TestPipelineController_info_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)
	ctrl := PipelineController{}
	ctrl.info(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test PipelineController.run with empty pipelineId
func TestPipelineController_run_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineId": ""}, adminUser)
	ctrl := PipelineController{}
	ctrl.run(c, &hbtp.Map{"pipelineId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

// Test PipelineController.run with nonexistent pipeline
func TestPipelineController_run_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineId": "nonexistent"}, adminUser)
	ctrl := PipelineController{}
	ctrl.run(c, &hbtp.Map{"pipelineId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test PipelineController.copy with empty pipelineId
func TestPipelineController_copy_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineId": ""}, adminUser)
	ctrl := PipelineController{}
	ctrl.copy(c, &hbtp.Map{"pipelineId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

// Test PipelineController.copy with nonexistent pipeline
func TestPipelineController_copy_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineId": "nonexistent"}, adminUser)
	ctrl := PipelineController{}
	ctrl.copy(c, &hbtp.Map{"pipelineId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test PipelineController.rebuild with empty pipelineVersionId
func TestPipelineController_rebuild_EmptyPipelineVersionId(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineVersionId": ""}, adminUser)
	ctrl := PipelineController{}
	ctrl.rebuild(c, &hbtp.Map{"pipelineVersionId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

// Test PipelineController.rebuild with nonexistent pipeline version
func TestPipelineController_rebuild_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineVersionId": "nonexistent"}, adminUser)
	ctrl := PipelineController{}
	ctrl.rebuild(c, &hbtp.Map{"pipelineVersionId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test PipelineController.pipelineVersion with empty ID
func TestPipelineController_pipelineVersion_EmptyID(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"id": ""}, adminUser)
	ctrl := PipelineController{}
	ctrl.pipelineVersion(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

// Test PipelineController.pipelineVersion with nonexistent ID
func TestPipelineController_pipelineVersion_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)
	ctrl := PipelineController{}
	ctrl.pipelineVersion(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test PipelineController.searchSha with empty ID
func TestPipelineController_searchSha_EmptyID(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"id": ""}, adminUser)
	ctrl := PipelineController{}
	ctrl.searchSha(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

// Test PipelineController.searchSha with nonexistent pipeline
func TestPipelineController_searchSha_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"id": "nonexistent"}, adminUser)
	ctrl := PipelineController{}
	ctrl.searchSha(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test PipelineController.vars with empty pipelineId
func TestPipelineController_vars_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineId": ""}, adminUser)
	ctrl := PipelineController{}
	ctrl.vars(c, &hbtp.Map{"pipelineId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

// Test PipelineController.vars with nonexistent pipeline
func TestPipelineController_vars_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineId": "nonexistent"}, adminUser)
	ctrl := PipelineController{}
	ctrl.vars(c, &hbtp.Map{"pipelineId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test PipelineController.varDel with invalid aid
func TestPipelineController_varDel_InvalidAid(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"aid": int64(0)}, adminUser)
	ctrl := PipelineController{}
	ctrl.varDel(c, &hbtp.Map{"aid": int64(0)})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

// Test PipelineController.varDel with nonexistent aid
func TestPipelineController_varDel_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"aid": int64(999)}, adminUser)
	ctrl := PipelineController{}
	ctrl.varDel(c, &hbtp.Map{"aid": int64(999)})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test PipelineController.save with empty pipelineId
func TestPipelineController_save_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineId": ""}, adminUser)
	ctrl := PipelineController{}
	ctrl.save(c, &hbtp.Map{"pipelineId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

// Test PipelineController.pipelineVersions with empty pipelineId (admin path)
func TestPipelineController_pipelineVersions_EmptyPipelineId_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineId": "", "page": int64(1)}, adminUser)
	ctrl := PipelineController{}
	ctrl.pipelineVersions(c, &hbtp.Map{"pipelineId": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// Test PipelineController.orgPipelines with empty orgId
func TestPipelineController_orgPipelines_EmptyOrgId(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"orgId": ""}, adminUser)
	ctrl := PipelineController{}
	ctrl.orgPipelines(c, &hbtp.Map{"orgId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

// Test PipelineController.orgPipelines with nonexistent org
func TestPipelineController_orgPipelines_NotFoundOrg(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"orgId": "nonexistent"}, adminUser)
	ctrl := PipelineController{}
	ctrl.orgPipelines(c, &hbtp.Map{"orgId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test PipelineController.getPipelines with empty query (admin path)
func TestPipelineController_getPipelines_EmptyQuery_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makePipelineGinContext(t, hbtp.Map{"q": "", "page": int64(1)}, adminUser)
	ctrl := PipelineController{}
	ctrl.getPipelines(c, &hbtp.Map{"q": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// Test PipelineController.getPipelines with non-admin user (filtered by uid)
func TestPipelineController_getPipelines_NonAdmin(t *testing.T) {
	db := setupPipelineTestDB(t)
	regularUser := &model.TUser{Id: "user1", Name: "user1", Active: 1}

	// Insert user
	_, err := db.Insert(regularUser)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, w := makePipelineGinContext(t, hbtp.Map{"q": "", "page": int64(1)}, regularUser)
	ctrl := PipelineController{}
	ctrl.getPipelines(c, &hbtp.Map{"q": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// Test PipelineController.vars with existing pipeline and vars
func TestPipelineController_vars_WithVars(t *testing.T) {
	db := setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline
	pipeline := &model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipe",
		Deleted: 0,
	}
	_, err := db.Insert(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create a pipeline var
	pv := &model.TPipelineVar{
		Aid:        1,
		Uid:        "admin",
		PipelineId: "pipe-1",
		Name:       "MY_VAR",
		Value:      "secret-value",
		Public:     0,
	}
	_, err = db.Insert(pv)
	if err != nil {
		t.Fatalf("insert pipeline var: %v", err)
	}

	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineId": "pipe-1", "page": int64(1)}, adminUser)
	ctrl := PipelineController{}
	ctrl.vars(c, &hbtp.Map{"pipelineId": "pipe-1", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// Test PipelineController.pipelineVersions with existing pipeline and versions
func TestPipelineController_pipelineVersions_WithVersions(t *testing.T) {
	db := setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline
	pipeline := &model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipe",
		Deleted: 0,
	}
	_, err := db.Insert(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create pipeline versions
	pv1 := &model.TPipelineVersion{
		Id:         "pv-1",
		PipelineId: "pipe-1",
		Uid:        "admin",
		Sha:        "abc123",
		Events:     "push",
		Deleted:    0,
		Created:    time.Now(),
	}
	_, err = db.Insert(pv1)
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	c, w := makePipelineGinContext(t, hbtp.Map{"pipelineId": "pipe-1", "page": int64(1)}, adminUser)
	ctrl := PipelineController{}
	ctrl.pipelineVersions(c, &hbtp.Map{"pipelineId": "pipe-1", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// Test PipelineController.searchSha with existing pipeline and SHAs
func TestPipelineController_searchSha_WithSHAs(t *testing.T) {
	db := setupPipelineTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline
	pipeline := &model.TPipeline{
		Id:      "pipe-1",
		Uid:     "admin",
		Name:    "test-pipe",
		Deleted: 0,
	}
	_, err := db.Insert(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create pipeline versions with SHAs
	pv1 := &model.TPipelineVersion{
		Id:         "pv-1",
		PipelineId: "pipe-1",
		Uid:        "admin",
		Sha:        "abc123def456",
		Events:     "push",
		Deleted:    0,
		Created:    time.Now(),
	}
	_, err = db.Insert(pv1)
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	c, w := makePipelineGinContext(t, hbtp.Map{"id": "pipe-1", "q": ""}, adminUser)
	ctrl := PipelineController{}
	ctrl.searchSha(c, &hbtp.Map{"id": "pipe-1", "q": ""})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) == 0 {
		t.Error("expected at least one SHA in response")
	}
}
