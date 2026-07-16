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
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			name VARCHAR(200),
			display_name VARCHAR(200),
			pipeline_type VARCHAR(50),
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			create_time DATETIME
		)`,
		`CREATE TABLE t_pipeline_conf (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			pipeline_id VARCHAR(64),
			yml_content TEXT,
			url VARCHAR(500),
			username VARCHAR(200),
			access_token VARCHAR(500),
			created DATETIME,
			updated DATETIME
		)`,
		`CREATE TABLE t_pipeline_version (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			pipeline_id VARCHAR(64),
			uid VARCHAR(64),
			sha VARCHAR(100),
			status VARCHAR(50),
			deleted INT DEFAULT 0,
			created DATETIME,
			updated DATETIME
		)`,
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	comm.Db = db
}

func TestPipelineController_info_MissingID(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{"id": ""}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := PipelineController{}
	ctrl.info(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_info_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	body := hbtp.Map{"id": "nonexistent"}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := PipelineController{}
	ctrl.info(c, &body)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_delete_MissingID(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{"id": ""}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := PipelineController{}
	ctrl.delete(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_run_MissingPipelineID(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{"pipelineId": ""}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := PipelineController{}
	ctrl.run(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_copy_MissingPipelineID(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{"pipelineId": ""}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := PipelineController{}
	ctrl.copy(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_rebuild_MissingVersionID(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{"pipelineVersionId": ""}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := PipelineController{}
	ctrl.rebuild(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_orgPipelines_MissingOrgID(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	body := hbtp.Map{"orgId": "", "q": "", "page": int64(1)}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := PipelineController{}
	ctrl.orgPipelines(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_save_MissingPipelineID(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := hbtp.Map{"pipelineId": ""}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := PipelineController{}
	ctrl.save(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
