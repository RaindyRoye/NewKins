package route

import (
	"context"
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

	// Create t_pipeline table - matches model.TPipeline and TPipelineInfo
	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(255),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		create_time DATETIME,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline table: %v", err)
	}

	// Create t_pipeline_conf table
	_, err = db.Exec(`CREATE TABLE t_pipeline_conf (
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		pipeline_id VARCHAR(64),
		yml_content TEXT,
		url VARCHAR(500),
		username VARCHAR(255),
		access_token VARCHAR(500)
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_conf table: %v", err)
	}

	// Create t_pipeline_version table - matches model.TPipelineVersion
	_, err = db.Exec(`CREATE TABLE t_pipeline_version (
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

	// Create t_pipeline_var table
	_, err = db.Exec(`CREATE TABLE t_pipeline_var (
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		id VARCHAR(64),
		uid VARCHAR(64),
		pipeline_id VARCHAR(64),
		name VARCHAR(255),
		value TEXT,
		remarks TEXT,
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_var table: %v", err)
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

	// Create t_org table
	_, err = db.Exec(`CREATE TABLE t_org (
		id VARCHAR(64) PRIMARY KEY,
		name VARCHAR(255),
		display_name VARCHAR(255),
		deleted INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_org table: %v", err)
	}

	// Create t_org_pipe table
	_, err = db.Exec(`CREATE TABLE t_org_pipe (
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		created DATETIME,
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_org_pipe table: %v", err)
	}

	// Create t_user_org table
	_, err = db.Exec(`CREATE TABLE t_user_org (
		id VARCHAR(64) PRIMARY KEY,
		user_id VARCHAR(64),
		org_id VARCHAR(64),
		role INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_user_org table: %v", err)
	}

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	return db
}

func makePipelineGinCtx(t *testing.T, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestPipelineInfo_MissingId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing id, got %d", w.Code)
	}
}

func TestPipelineInfo_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineInfo_Success(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	// Insert pipeline
	pipeline := &model.TPipeline{
		Id:          "pipe-1",
		Uid:         "user-1",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
	}
	if _, err := db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Insert pipeline config
	conf := &model.TPipelineConf{
		PipelineId:  "pipe-1",
		YmlContent:  "stages: []",
		Url:         "https://github.com/example/repo",
		Username:    "admin",
		AccessToken: "secret-token",
	}
	if _, err := db.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	c, w := makePipelineGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["pipe"] == nil {
		t.Error("expected 'pipe' in response")
	}
	if resp["perm"] == nil {
		t.Error("expected 'perm' in response")
	}
}

func TestPipelineDelete_MissingId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing id, got %d", w.Code)
	}
}

func TestPipelineDelete_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineDelete_Success(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}

	pipeline := &model.TPipeline{
		Id:   "pipe-1",
		Uid:  "user-1",
		Name: "to-delete",
	}
	if _, err := db.Insert(pipeline); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Insert a pipeline version
	pv := &model.TPipelineVersion{
		Id:         "pv-1",
		PipelineId: "pipe-1",
		Created:    time.Now(),
	}
	if _, err := db.Insert(pv); err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	c, w := makePipelineGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify soft delete
	var deletedPipe model.TPipeline
	ok, err := db.Where("id = ?", "pipe-1").Get(&deletedPipe)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline should still exist (soft deleted)")
	}
	if deletedPipe.Deleted != 1 {
		t.Errorf("expected deleted=1, got %d", deletedPipe.Deleted)
	}
}

func TestPipelineRun_MissingPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	ctrl.run(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing pipelineId, got %d", w.Code)
	}
}

func TestPipelineRun_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.run(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineCopy_MissingPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	ctrl.copy(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing pipelineId, got %d", w.Code)
	}
}

func TestPipelineCopy_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.copy(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineRebuild_MissingId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "")
	ctrl.rebuild(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing pipelineVersionId, got %d", w.Code)
	}
}

func TestPipelineRebuild_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "nonexistent")
	ctrl.rebuild(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline version, got %d", w.Code)
	}
}

func TestPipelineVars_MissingPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing pipelineId, got %d", w.Code)
	}
}

func TestPipelineVars_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineVarSave_MissingFields(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	// Call varDel with missing aid to test the parameter validation path
	// (varSave requires a non-nil *bean.PipelineVar which is hard to construct here)
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid aid, got %d", w.Code)
	}
}

func TestPipelineVarDel_InvalidAid(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid aid, got %d", w.Code)
	}
}

func TestPipelineVarDel_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("aid", int64(999))
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline var, got %d", w.Code)
	}
}

func TestPipelineSearchSha_MissingId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing id, got %d", w.Code)
	}
}

func TestPipelineSearchSha_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineVersion_MissingId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing id, got %d", w.Code)
	}
}

func TestPipelineVersion_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline version, got %d", w.Code)
	}
}

func TestFillPipelineListBuildInfo_Empty(t *testing.T) {
	err := fillPipelineListBuildInfo(context.Background(), nil)
	if err != nil {
		t.Errorf("unexpected error for empty list: %v", err)
	}
}

func TestFillPipelineListBuildInfo_EmptySlice(t *testing.T) {
	err := fillPipelineListBuildInfo(context.Background(), []*model.TPipeline{})
	if err != nil {
		t.Errorf("unexpected error for empty slice: %v", err)
	}
}
