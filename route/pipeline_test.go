package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupPipelineTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) NOT NULL,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(255),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		create_time DATETIME,
		PRIMARY KEY (id)
	)`)
	if err != nil {
		t.Fatalf("create pipeline table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline_conf (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		pipeline_id VARCHAR(64) NOT NULL,
		url VARCHAR(255),
		access_token VARCHAR(255),
		yml_content TEXT,
		username VARCHAR(255)
	)`)
	if err != nil {
		t.Fatalf("create pipeline_conf table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline_var (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		pipeline_id VARCHAR(64),
		name VARCHAR(255),
		value TEXT,
		remarks VARCHAR(255),
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create pipeline_var table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline_version (
		id VARCHAR(64) NOT NULL,
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
		repo_clone_url VARCHAR(255),
		PRIMARY KEY (id)
	)`)
	if err != nil {
		t.Fatalf("create pipeline_version table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org_pipe (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		created DATETIME,
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create org_pipe table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_build (
		id VARCHAR(64) NOT NULL,
		pipeline_id VARCHAR(64),
		pipeline_version_id VARCHAR(64),
		status VARCHAR(100),
		error VARCHAR(500),
		event VARCHAR(100),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		version VARCHAR(255),
		PRIMARY KEY (id)
	)`)
	if err != nil {
		t.Fatalf("create build table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL,
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(100),
		pass VARCHAR(255),
		nick VARCHAR(100),
		avatar VARCHAR(500),
		created DATETIME,
		login_time DATETIME,
		active INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create user table: %v", err)
	}

	comm.Db = db
}

func makePipelineTestContext(t *testing.T, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

	c.Set(service.LgUserKey, &model.TUser{
		Id:     "test-user",
		Name:   "tester",
		Nick:   "Test User",
		Active: 1,
	})
	return c, w
}

func TestPipelineController_GetPipelinePath(t *testing.T) {
	c := &PipelineController{}
	if got := c.GetPath(); got != "/api/pipeline" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/pipeline")
	}
}

func TestPipelineController_Routes(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	pc := &PipelineController{}
	pc.Routes(r.Group("/api/pipeline"))
}

func TestPipelineInfo_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineTestContext(t, m)
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty id, got %d", w.Code)
	}
}

func TestPipelineInfo_NonexistentId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-id")
	c, w := makePipelineTestContext(t, m)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineDelete_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineTestContext(t, m)
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty id, got %d", w.Code)
	}
}

func TestPipelineDelete_NonexistentId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-id")
	c, w := makePipelineTestContext(t, m)
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineRun_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestContext(t, m)
	ctrl.run(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pipelineId, got %d", w.Code)
	}
}

func TestPipelineRun_NonexistentPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent-id")
	c, w := makePipelineTestContext(t, m)
	ctrl.run(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineCopy_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestContext(t, m)
	ctrl.copy(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pipelineId, got %d", w.Code)
	}
}

func TestPipelineCopy_NonexistentPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent-id")
	c, w := makePipelineTestContext(t, m)
	ctrl.copy(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineRebuild_EmptyPipelineVersionId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "")
	c, w := makePipelineTestContext(t, m)
	ctrl.rebuild(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pipelineVersionId, got %d", w.Code)
	}
}

func TestPipelineRebuild_NonexistentVersion(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "nonexistent-id")
	c, w := makePipelineTestContext(t, m)
	ctrl.rebuild(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent version, got %d", w.Code)
	}
}

func TestPipelineSearchSha_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineTestContext(t, m)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty id, got %d", w.Code)
	}
}

func TestPipelineSearchSha_NonexistentPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-id")
	c, w := makePipelineTestContext(t, m)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineVars_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestContext(t, m)
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pipelineId, got %d", w.Code)
	}
}

func TestPipelineVars_NonexistentPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent-id")
	c, w := makePipelineTestContext(t, m)
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineVarDel_InvalidAid(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	c, w := makePipelineTestContext(t, m)
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid aid, got %d", w.Code)
	}
}

func TestPipelineVarDel_NonexistentAid(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(999))
	c, w := makePipelineTestContext(t, m)
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent var, got %d", w.Code)
	}
}

func TestPipelineOrgPipelines_EmptyOrgId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	c, w := makePipelineTestContext(t, m)
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty orgId, got %d", w.Code)
	}
}

func TestPipelineGetPipelines_Success(t *testing.T) {
	setupPipelineTestDB(t)

	// Insert test pipelines
	_, err := comm.Db.Exec(`INSERT INTO t_pipeline (id, uid, name, display_name, deleted) 
		VALUES ('pipe-1', 'test-user', 'test-pipe', 'Test Pipeline', 0)`)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("q", "test")
	m.Set("page", int64(1))
	c, w := makePipelineTestContext(t, m)
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid request, got %d", w.Code)
	}
}

func TestPipelineVersion_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineTestContext(t, m)
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty id, got %d", w.Code)
	}
}

func TestPipelineVersion_NonexistentId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-id")
	c, w := makePipelineTestContext(t, m)
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent version, got %d", w.Code)
	}
}

func TestFillPipelineListBuildInfo_EmptyList(t *testing.T) {
	setupPipelineTestDB(t)
	err := fillPipelineListBuildInfo(nil, []*model.TPipeline{})
	if err != nil {
		t.Errorf("unexpected error for empty list: %v", err)
	}
}
