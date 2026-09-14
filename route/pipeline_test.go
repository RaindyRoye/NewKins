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

func setupPipelineTestDb(t *testing.T) *xorm.Engine {
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
		create_time DATETIME,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create pipeline table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline_conf (
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		pipeline_id VARCHAR(64),
		yml_content TEXT,
		url VARCHAR(255),
		username VARCHAR(100),
		access_token VARCHAR(255)
	)`)
	if err != nil {
		t.Fatalf("create pipeline_conf table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline_version (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		pipeline_id VARCHAR(64),
		number BIGINT,
		events VARCHAR(100),
		sha VARCHAR(255),
		pipeline_name VARCHAR(255),
		pipeline_display_name VARCHAR(255),
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

	_, err = db.Exec(`CREATE TABLE t_pipeline_var (
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		pipeline_id VARCHAR(64),
		uid VARCHAR(64),
		name VARCHAR(100),
		value TEXT,
		remarks VARCHAR(255),
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create pipeline_var table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		name VARCHAR(100),
		display_name VARCHAR(255),
		deleted INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create org table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org_pipe (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		public INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create org_pipe table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		name VARCHAR(100),
		nick VARCHAR(100),
		avatar VARCHAR(500),
		pass VARCHAR(255),
		active INT DEFAULT 1,
		created DATETIME,
		login_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create user table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_build (
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
	)`)
	if err != nil {
		t.Fatalf("create build table: %v", err)
	}

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

func createTestUser(t *testing.T, db *xorm.Engine) *model.TUser {
	t.Helper()
	user := &model.TUser{
		Id:      "user-1",
		Name:    "tester",
		Nick:    "Test User",
		Active:  1,
		Created: time.Now(),
	}
	if _, err := db.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return user
}

func createTestPipeline(t *testing.T, db *xorm.Engine, id, name, displayName string) *model.TPipeline {
	t.Helper()
	pipe := &model.TPipeline{
		Id:          id,
		Uid:         "user-1",
		Name:        name,
		DisplayName: displayName,
	}
	if _, err := db.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	return pipe
}

func createTestPipelineConf(t *testing.T, db *xorm.Engine, pipelineId, yml, url, username, token string) *model.TPipelineConf {
	t.Helper()
	conf := &model.TPipelineConf{
		PipelineId:  pipelineId,
		YmlContent:  yml,
		Url:         url,
		Username:    username,
		AccessToken: token,
	}
	if _, err := db.Insert(conf); err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}
	return conf
}

func TestPipelineInfo_MissingId(t *testing.T) {
	setupPipelineTestDb(t)
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
	setupPipelineTestDb(t)
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
	db := setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := createTestUser(t, db)
	createTestPipeline(t, db, "pipe-1", "test-pipe", "Test Pipeline")
	createTestPipelineConf(t, db, "pipe-1", "version: 1.0", "https://github.com/test/repo", "user", "token")

	c, w := makePipelineGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["pipe"] == nil {
		t.Error("expected pipe in response")
	}
	if resp["perm"] == nil {
		t.Error("expected perm in response")
	}
}

func TestPipelineDelete_MissingId(t *testing.T) {
	setupPipelineTestDb(t)
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
	setupPipelineTestDb(t)
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
	db := setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := createTestUser(t, db)
	createTestPipeline(t, db, "pipe-1", "test-pipe", "Test Pipeline")

	c, w := makePipelineGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify pipeline is soft-deleted
	pipe := &model.TPipeline{}
	ok, err := db.Where("id = ?", "pipe-1").Get(pipe)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline should still exist (soft delete)")
	}
	if pipe.Deleted != 1 {
		t.Errorf("expected deleted=1, got %d", pipe.Deleted)
	}
}

func TestPipelineVars_MissingPipelineId(t *testing.T) {
	setupPipelineTestDb(t)
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
	setupPipelineTestDb(t)
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

func TestPipelineVars_Success(t *testing.T) {
	db := setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := createTestUser(t, db)
	createTestPipeline(t, db, "pipe-1", "test-pipe", "Test Pipeline")

	// Add some pipeline vars
	vars := []*model.TPipelineVar{
		{Aid: 1, PipelineId: "pipe-1", Uid: "user-1", Name: "VAR1", Value: "value1", Public: 0},
		{Aid: 2, PipelineId: "pipe-1", Uid: "user-1", Name: "VAR2", Value: "value2", Public: 1},
	}
	for _, v := range vars {
		if _, err := db.Insert(v); err != nil {
			t.Fatalf("insert var: %v", err)
		}
	}

	c, w := makePipelineGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-1")
	m.Set("page", int64(1))
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["data"] == nil {
		t.Error("expected data in response")
	}
}

func TestPipelineVarSave_MissingFields(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	pv := &bean.PipelineVar{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", w.Code)
	}
}

func TestPipelineVarSave_PipelineNotFound(t *testing.T) {
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	pv := &bean.PipelineVar{
		PipelineId: "nonexistent",
		Name:       "VAR1",
		Value:      "value1",
	}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineVarSave_CreateNew(t *testing.T) {
	db := setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := createTestUser(t, db)
	createTestPipeline(t, db, "pipe-1", "test-pipe", "Test Pipeline")

	c, w := makePipelineGinCtx(t, user)
	pv := &bean.PipelineVar{
		PipelineId: "pipe-1",
		Name:       "NEW_VAR",
		Value:      "new_value",
		Public:     true,
	}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify var was created
	count, err := db.Where("pipeline_id = ? AND name = ?", "pipe-1", "NEW_VAR").Count(&model.TPipelineVar{})
	if err != nil {
		t.Fatalf("count vars: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 var created, got %d", count)
	}
}

func TestPipelineVarSave_DuplicateName(t *testing.T) {
	db := setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := createTestUser(t, db)
	createTestPipeline(t, db, "pipe-1", "test-pipe", "Test Pipeline")

	// Create existing var
	if _, err := db.Insert(&model.TPipelineVar{
		Aid: 1, PipelineId: "pipe-1", Uid: "user-1",
		Name: "EXISTING", Value: "old_value",
	}); err != nil {
		t.Fatalf("insert existing var: %v", err)
	}

	c, w := makePipelineGinCtx(t, user)
	pv := &bean.PipelineVar{
		PipelineId: "pipe-1",
		Name:       "EXISTING",
		Value:      "new_value",
	}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate name, got %d", w.Code)
	}
}

func TestPipelineVarDel_InvalidAid(t *testing.T) {
	setupPipelineTestDb(t)
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
	setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	c, w := makePipelineGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("aid", int64(999))
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent var, got %d", w.Code)
	}
}

func TestPipelineVarDel_Success(t *testing.T) {
	db := setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := createTestUser(t, db)
	createTestPipeline(t, db, "pipe-1", "test-pipe", "Test Pipeline")

	// Create var to delete
	if _, err := db.Insert(&model.TPipelineVar{
		Aid: 1, PipelineId: "pipe-1", Uid: "user-1",
		Name: "TO_DELETE", Value: "value",
	}); err != nil {
		t.Fatalf("insert var: %v", err)
	}

	c, w := makePipelineGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("aid", int64(1))
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify var was deleted
	count, err := db.Where("aid = ?", 1).Count(&model.TPipelineVar{})
	if err != nil {
		t.Fatalf("count vars: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 vars after delete, got %d", count)
	}
}

func TestPipelineSearchSha_MissingId(t *testing.T) {
	setupPipelineTestDb(t)
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
	setupPipelineTestDb(t)
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

func TestPipelineSearchSha_Success(t *testing.T) {
	db := setupPipelineTestDb(t)
	ctrl := PipelineController{}
	user := createTestUser(t, db)
	createTestPipeline(t, db, "pipe-1", "test-pipe", "Test Pipeline")

	// Add pipeline versions with SHAs
	versions := []*model.TPipelineVersion{
		{Id: "pv-1", PipelineId: "pipe-1", Sha: "abc123", Created: time.Now()},
		{Id: "pv-2", PipelineId: "pipe-1", Sha: "def456", Created: time.Now()},
		{Id: "pv-3", PipelineId: "pipe-1", Sha: "abc789", Created: time.Now()},
	}
	for _, v := range versions {
		if _, err := db.Insert(v); err != nil {
			t.Fatalf("insert version: %v", err)
		}
	}

	c, w := makePipelineGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	m.Set("q", "abc")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 SHA matches, got %d", len(resp))
	}
}

func TestFillPipelineListBuildInfo_Empty(t *testing.T) {
	err := fillPipelineListBuildInfo(context.TODO(), nil)
	if err != nil {
		t.Fatalf("unexpected error for empty list: %v", err)
	}
}

func TestFillPipelineListBuildInfo_Success(t *testing.T) {
	db := setupPipelineTestDb(t)
	createTestUser(t, db)
	pipes := []*model.TPipeline{
		createTestPipeline(t, db, "pipe-1", "pipe1", "Pipeline 1"),
		createTestPipeline(t, db, "pipe-2", "pipe2", "Pipeline 2"),
	}

	err := fillPipelineListBuildInfo(context.TODO(), pipes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify user info was populated
	if pipes[0].Nick != "Test User" {
		t.Errorf("expected nick='Test User', got %q", pipes[0].Nick)
	}
}
