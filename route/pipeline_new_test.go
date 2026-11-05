package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/utils"
	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupPipelineNewTestDB(t *testing.T) *xorm.Engine {
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
		create_time DATETIME,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline table: %v", err)
	}

	// Create t_pipeline_conf table
	_, err = db.Exec(`CREATE TABLE t_pipeline_conf (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		pipeline_id VARCHAR(64),
		yml_content TEXT,
		url VARCHAR(500),
		username VARCHAR(255),
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
		name VARCHAR(255),
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

	// Create t_org table
	_, err = db.Exec(`CREATE TABLE t_org (
		id VARCHAR(64) PRIMARY KEY,
		aid BIGINT,
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
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		created DATETIME,
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_org_pipe table: %v", err)
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

func TestPipelineController_new_InvalidYaml(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	np := &bean.NewPipeline{
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Content:     "invalid: yaml: content:",
		Url:         "https://github.com/test/repo",
		Username:    "testuser",
		AccessToken: "token123",
	}

	c, w := makePipelineGinContext(t, np, adminUser)
	ctrl := PipelineController{}
	ctrl.new(c, np)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestPipelineController_new_ValidPipeline(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	validYml := `stages:
  - stage: build
    name: Build Stage
    steps:
      - step: shell
        name: echo
        commands:
          - echo "hello"`

	np := &bean.NewPipeline{
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
		Content:     validYml,
		Url:         "https://github.com/test/repo",
		Username:    "testuser",
		AccessToken: "token123",
	}

	c, w := makePipelineGinContext(t, np, adminUser)
	ctrl := PipelineController{}
	ctrl.new(c, np)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify pipeline was created
	var pipeline model.TPipeline
	ok, err := comm.Db.Where("name=?", "test-pipeline").Get(&pipeline)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline not found after creation")
	}
	if pipeline.DisplayName != "Test Pipeline" {
		t.Errorf("pipeline display_name = %q, want %q", pipeline.DisplayName, "Test Pipeline")
	}

	// Verify pipeline conf was created
	var conf model.TPipelineConf
	ok, err = comm.Db.Where("pipeline_id=?", pipeline.Id).Get(&conf)
	if err != nil {
		t.Fatalf("query pipeline conf: %v", err)
	}
	if !ok {
		t.Fatal("pipeline conf not found after creation")
	}
	if conf.Url != "https://github.com/test/repo" {
		t.Errorf("conf url = %q, want %q", conf.Url, "https://github.com/test/repo")
	}
}

func TestPipelineController_new_WithVars(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	validYml := `stages:
  - stage: build
    name: Build
    steps:
      - step: shell
        name: echo
        commands:
          - echo "test"`

	np := &bean.NewPipeline{
		Name:        "pipeline-with-vars",
		DisplayName: "Pipeline With Vars",
		Content:     validYml,
		Url:         "https://github.com/test/repo",
		Vars: []*bean.NewPipelineVar{
			{Name: "VAR1", Value: "value1", Public: true},
			{Name: "VAR2", Value: "value2", Public: false},
		},
	}

	c, w := makePipelineGinContext(t, np, adminUser)
	ctrl := PipelineController{}
	ctrl.new(c, np)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify variables were created
	var pipeline model.TPipeline
	ok, err := comm.Db.Where("name=?", "pipeline-with-vars").Get(&pipeline)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline not found")
	}

	var vars []model.TPipelineVar
	err = comm.Db.Where("pipeline_id=?", pipeline.Id).Find(&vars)
	if err != nil {
		t.Fatalf("query pipeline vars: %v", err)
	}
	if len(vars) != 2 {
		t.Errorf("expected 2 variables, got %d", len(vars))
	}
}

func TestPipelineController_new_InvalidParams(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Missing name
	np := &bean.NewPipeline{
		Name:    "",
		Content: "stages: []",
	}

	c, w := makePipelineGinContext(t, np, adminUser)
	ctrl := PipelineController{}
	ctrl.new(c, np)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_save_EmptyPipelineId(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("name", "test")
	m.Set("content", "stages: []")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_save_ValidUpdate(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline first
	pipeline := &model.TPipeline{
		Id:          utils.NewXid(),
		Uid:         "admin",
		Name:        "original-name",
		DisplayName: "Original Name",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	conf := &model.TPipelineConf{
		PipelineId: pipeline.Id,
		YmlContent: "original content",
	}
	_, err = db.InsertOne(conf)
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	validYml := `stages:
  - stage: test
    name: Test
    steps:
      - step: shell
        name: test
        commands:
          - echo "test"`

	m := &hbtp.Map{}
	m.Set("pipelineId", pipeline.Id)
	m.Set("name", "updated-name")
	m.Set("displayName", "Updated Name")
	m.Set("content", validYml)
	m.Set("url", "https://github.com/updated/repo")
	m.Set("username", "newuser")
	m.Set("accessToken", "newtoken")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify pipeline was updated
	var updated model.TPipeline
	ok, err := comm.Db.Where("id=?", pipeline.Id).Get(&updated)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline not found")
	}
	if updated.Name != "updated-name" {
		t.Errorf("pipeline name = %q, want %q", updated.Name, "updated-name")
	}
}

func TestPipelineController_varSave_EmptyParams(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	pv := &bean.PipelineVar{
		Name:       "",
		Value:      "value",
		PipelineId: "pipe-1",
	}

	c, w := makePipelineGinContext(t, pv, adminUser)
	ctrl := PipelineController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varSave_NewVariable(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline first
	pipeline := &model.TPipeline{
		Id:   utils.NewXid(),
		Uid:  "admin",
		Name: "test-pipeline",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	pv := &bean.PipelineVar{
		Name:       "NEW_VAR",
		Value:      "new_value",
		PipelineId: pipeline.Id,
		Public:     true,
	}

	c, w := makePipelineGinContext(t, pv, adminUser)
	ctrl := PipelineController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify variable was created
	var v model.TPipelineVar
	ok, err := comm.Db.Where("pipeline_id=? AND name=?", pipeline.Id, "NEW_VAR").Get(&v)
	if err != nil {
		t.Fatalf("query variable: %v", err)
	}
	if !ok {
		t.Fatal("variable not found")
	}
	if v.Value != "new_value" {
		t.Errorf("variable value = %q, want %q", v.Value, "new_value")
	}
	if v.Public != 1 {
		t.Errorf("variable public = %d, want 1", v.Public)
	}
}

func TestPipelineController_varSave_DuplicateName(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline first
	pipeline := &model.TPipeline{
		Id:   utils.NewXid(),
		Uid:  "admin",
		Name: "test-pipeline",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create an existing variable
	existingVar := &model.TPipelineVar{
		Uid:        "admin",
		PipelineId: pipeline.Id,
		Name:       "EXISTING_VAR",
		Value:      "old_value",
		Public:     0,
	}
	_, err = db.InsertOne(existingVar)
	if err != nil {
		t.Fatalf("insert existing variable: %v", err)
	}

	// Try to create a new variable with the same name
	pv := &bean.PipelineVar{
		Name:       "EXISTING_VAR",
		Value:      "new_value",
		PipelineId: pipeline.Id,
	}

	c, w := makePipelineGinContext(t, pv, adminUser)
	ctrl := PipelineController{}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusConflict {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestPipelineController_pipelineVersions_EmptyPipelineId(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", int64(1))

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_pipelineVersions_WithVersions(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline
	pipeline := &model.TPipeline{
		Id:          utils.NewXid(),
		Uid:         "admin",
		Name:        "test-pipeline",
		DisplayName: "Test Pipeline",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create pipeline versions
	for i := 0; i < 3; i++ {
		pv := &model.TPipelineVersion{
			Id:                  utils.NewXid(),
			Uid:                 "admin",
			Number:              int64(i + 1),
			Events:              "run",
			Sha:                 "abc123",
			PipelineName:        "test-pipeline",
			PipelineDisplayName: "Test Pipeline",
			PipelineId:          pipeline.Id,
			Created:             time.Now(),
		}
		_, err := db.InsertOne(pv)
		if err != nil {
			t.Fatalf("insert pipeline version: %v", err)
		}
	}

	m := &hbtp.Map{}
	m.Set("pipelineId", pipeline.Id)
	m.Set("page", int64(1))

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["data"] == nil {
		t.Error("expected 'data' in response")
	}
}

func TestPipelineController_info_EmptyId(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("id", "")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_info_NotFound(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_vars_EmptyPipelineId(t *testing.T) {
	setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	m := &hbtp.Map{}
	m.Set("pipelineId", "")

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_vars_WithVariables(t *testing.T) {
	db := setupPipelineNewTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Create a pipeline
	pipeline := &model.TPipeline{
		Id:   utils.NewXid(),
		Uid:  "admin",
		Name: "test-pipeline",
	}
	_, err := db.InsertOne(pipeline)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	// Create variables
	for i := 0; i < 3; i++ {
		v := &model.TPipelineVar{
			Uid:        "admin",
			PipelineId: pipeline.Id,
			Name:       "VAR" + string(rune('0'+i)),
			Value:      "value" + string(rune('0'+i)),
			Public:     i % 2,
		}
		_, err := db.InsertOne(v)
		if err != nil {
			t.Fatalf("insert variable: %v", err)
		}
	}

	m := &hbtp.Map{}
	m.Set("pipelineId", pipeline.Id)
	m.Set("page", int64(1))

	c, w := makePipelineGinContext(t, m, adminUser)
	ctrl := PipelineController{}
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
