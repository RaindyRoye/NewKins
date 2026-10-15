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

func setupPipelineTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create required tables
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
		`CREATE TABLE t_user_info (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			phone VARCHAR(100),
			email VARCHAR(200),
			birthday DATETIME,
			remark TEXT,
			perm_user INT,
			perm_org INT,
			perm_pipe INT
		)`,
		`CREATE TABLE t_pipeline (
			id VARCHAR(64) PRIMARY KEY,
			uid VARCHAR(64),
			name VARCHAR(255),
			display_name VARCHAR(255),
			pipeline_type VARCHAR(255),
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			created DATETIME
		)`,
		`CREATE TABLE t_pipeline_conf (
			aid BIGINT,
			pipeline_id VARCHAR(64),
			yml_content TEXT,
			url VARCHAR(500),
			username VARCHAR(200),
			access_token VARCHAR(500)
		)`,
		`CREATE TABLE t_pipeline_var (
			aid BIGINT,
			uid VARCHAR(64),
			pipeline_id VARCHAR(64),
			name VARCHAR(200),
			value TEXT,
			remark TEXT,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_pipeline_version (
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
		)`,
		`CREATE TABLE t_build (
			id TEXT PRIMARY KEY,
			pipeline_id TEXT,
			pipeline_version_id TEXT,
			status TEXT,
			error TEXT,
			event TEXT,
			time_stamp DATETIME,
			title TEXT,
			message TEXT,
			started DATETIME,
			finished DATETIME,
			created DATETIME,
			updated DATETIME,
			version TEXT
		)`,
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			name VARCHAR(200),
			desc TEXT,
			avatar VARCHAR(500),
			public INT DEFAULT 0,
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			created DATETIME,
			updated DATETIME
		)`,
		`CREATE TABLE t_org_pipe (
			aid BIGINT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_user_org (
			aid BIGINT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0,
			created DATETIME
		)`,
	}
	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("failed to create table: %v\nSQL: %s", err, sql)
		}
	}
	comm.Db = db
}
func createTestPipeline(t *testing.T, uid, name, displayName string) *model.TPipeline {
	t.Helper()
	id := "pipe_" + time.Now().Format("20060102150405.000000")
	now := time.Now()
	pipe := &model.TPipeline{
		Id:          id,
		Uid:         uid,
		Name:        name,
		DisplayName: displayName,
		Created:     now,
	}
	_, err := comm.Db.InsertOne(pipe)
	if err != nil {
		t.Fatalf("failed to create test pipeline: %v", err)
	}
	// Create pipeline config
	conf := &model.TPipelineConf{
		PipelineId: pipe.Id,
		YmlContent: "repo:\n  url: https://github.com/test/repo\nstages:\n  - name: build\n    steps:\n      - name: test\n        command: echo test",
		Url:        "https://github.com/test/repo",
		Username:   "testuser",
		AccessToken: "test-token",
	}
	_, err = comm.Db.InsertOne(conf)
	if err != nil {
		t.Fatalf("failed to create test pipeline config: %v", err)
	}
	return pipe
}
func createTestOrg(t *testing.T, uid, name string, public int) *model.TOrg {
	t.Helper()
	id := "org_" + time.Now().Format("20060102150405.000000")
	now := time.Now()
	org := &model.TOrg{
		Id:      id,
		Uid:     uid,
		Name:    name,
		Created: now,
		Updated: now,
		Public:  public,
	}
	_, err := comm.Db.InsertOne(org)
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}
	return org
}

func makePipelineTestContext(t *testing.T, body interface{}, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestPipelineController_info_MissingID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	c, w := makePipelineTestContext(t, nil, nil)
	ctrl.info(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_info_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.info(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_info_Success(t *testing.T) {
	setupPipelineTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	pipe := createTestPipeline(t, user.Id, "test-pipe", "Test Pipeline")

	ctrl := PipelineController{}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.info(c, &hbtp.Map{"id": pipe.Id})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify response contains pipeline info
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["pipe"] == nil {
		t.Error("response should contain 'pipe' field")
	}
	if resp["perm"] == nil {
		t.Error("response should contain 'perm' field")
	}
}

func TestPipelineController_delete_MissingID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	c, w := makePipelineTestContext(t, nil, nil)
	ctrl.delete(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_delete_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.delete(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_delete_Success(t *testing.T) {
	setupPipelineTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	pipe := createTestPipeline(t, user.Id, "test-pipe", "Test Pipeline")

	ctrl := PipelineController{}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.delete(c, &hbtp.Map{"id": pipe.Id})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify pipeline is marked as deleted
	updated := &model.TPipeline{}
	ok, err := comm.Db.Where("id=?", pipe.Id).Get(updated)
	if err != nil {
		t.Fatalf("query updated pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline not found after update")
	}
	if updated.Deleted != 1 {
		t.Errorf("pipeline deleted = %d, want 1", updated.Deleted)
	}
}

func TestPipelineController_run_MissingPipelineID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	c, w := makePipelineTestContext(t, nil, nil)
	ctrl.run(c, &hbtp.Map{"pipelineId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_run_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.run(c, &hbtp.Map{"pipelineId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_copy_MissingPipelineID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	c, w := makePipelineTestContext(t, nil, nil)
	ctrl.copy(c, &hbtp.Map{"pipelineId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_copy_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.copy(c, &hbtp.Map{"pipelineId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_getPipelines_EmptyDB(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.getPipelines(c, &hbtp.Map{"q": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_getPipelines_WithPipelines(t *testing.T) {
	setupPipelineTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	createTestPipeline(t, user.Id, "pipe1", "Pipeline 1")
	createTestPipeline(t, user.Id, "pipe2", "Pipeline 2")

	ctrl := PipelineController{}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.getPipelines(c, &hbtp.Map{"q": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_getPipelines_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	createTestPipeline(t, user.Id, "alpha-pipe", "Alpha Pipeline")
	createTestPipeline(t, user.Id, "beta-pipe", "Beta Pipeline")

	ctrl := PipelineController{}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.getPipelines(c, &hbtp.Map{"q": "alpha", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_orgPipelines_MissingOrgID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	c, w := makePipelineTestContext(t, nil, nil)
	ctrl.orgPipelines(c, &hbtp.Map{"orgId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_orgPipelines_OrgNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.orgPipelines(c, &hbtp.Map{"orgId": "nonexistent", "q": "", "page": int64(1)})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_orgPipelines_Success(t *testing.T) {
	setupPipelineTestDB(t)
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	org := createTestOrg(t, user.Id, "test-org", 1)

	ctrl := PipelineController{}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.orgPipelines(c, &hbtp.Map{"orgId": org.Id, "q": "", "page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_rebuild_MissingPipelineVersionID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	c, w := makePipelineTestContext(t, nil, nil)
	ctrl.rebuild(c, &hbtp.Map{"pipelineVersionId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_pipelineVersions_MissingPipelineID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	c, w := makePipelineTestContext(t, nil, user)
	ctrl.pipelineVersions(c, &hbtp.Map{"pipelineId": "", "page": int64(1)})

	// Should return 200 with empty result when pipelineId is empty
	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
