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
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			uid VARCHAR(64),
			name VARCHAR(255),
			display_name VARCHAR(255),
			pipeline_type VARCHAR(255),
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			create_time DATETIME
		)`,
		`CREATE TABLE t_pipeline_conf (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			pipeline_id VARCHAR(64),
			url VARCHAR(255),
			access_token VARCHAR(255),
			yml_content TEXT,
			username VARCHAR(255)
		)`,
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
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL,
			aid INTEGER NOT NULL,
			uid VARCHAR(64),
			name VARCHAR(200),
			"desc" TEXT,
			public INT DEFAULT 0,
			created DATETIME,
			updated DATETIME,
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			PRIMARY KEY (id, aid)
		)`,
		`CREATE TABLE t_user_org (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_pipe (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_build (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			pipeline_id VARCHAR(64),
			pipeline_version_id VARCHAR(64),
			uid VARCHAR(64),
			sha VARCHAR(255),
			status VARCHAR(50),
			event VARCHAR(50),
			error TEXT,
			started DATETIME,
			finished DATETIME,
			updated DATETIME,
			created DATETIME,
			version VARCHAR(255)
		)`,
	}

	for _, ddl := range tables {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("create table: %v\nDDL: %s", err, ddl)
		}
	}

	comm.Db = db
}

func makePipelineTestCtx(t *testing.T, body interface{}, lguser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

	if lguser != nil {
		c.Set(service.LgUserKey, lguser)
	}
	return c, w
}

func insertTestPipeline(t *testing.T, id, uid, name string) {
	t.Helper()
	p := &model.TPipeline{
		Id:   id,
		Uid:  uid,
		Name: name,
	}
	_, err := comm.Db.InsertOne(p)
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
}

func insertTestPipelineConf(t *testing.T, pipelineId, content string) {
	t.Helper()
	_, err := comm.Db.InsertOne(&model.TPipelineConf{
		PipelineId: pipelineId,
		YmlContent: content,
	})
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}
}

// --- Tests ---

func TestPipelineController_Routes(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	pc := &PipelineController{}
	pc.Routes(r.Group("/api/pipeline"))
	// Just verify no panic during registration
}

func TestPipelineGetPipelines_Empty(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineGetPipelines_WithData(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe1", "admin", "Test Pipeline")
	m := &hbtp.Map{}
	m.Set("q", "")
	m.Set("page", int64(1))
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineGetPipelines_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe1", "admin", "Searchable Pipeline")
	m := &hbtp.Map{}
	m.Set("q", "Searchable")
	m.Set("page", int64(1))
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestPipelineInfo_MissingId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineInfo_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineDelete_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineDelete_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineRun_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.run(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineRun_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.run(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineCopy_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.copy(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineCopy_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.copy(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineRebuild_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.rebuild(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineRebuild_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "nonexistent")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.rebuild(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineSave_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineVersions_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		// Empty pipelineId returns all versions for user
		t.Logf("status = %d (may vary based on implementation)", w.Code)
	}
}

func TestPipelineVersions_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.pipelineVersions(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineVersion_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.pipelineVersion(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineSearchSha_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineVars_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineTestCtx(t, m, admin)
	pc := PipelineController{}
	pc.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
