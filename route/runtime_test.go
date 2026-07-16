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

func setupRuntimeTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
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
		`CREATE TABLE t_build (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			pipeline_id VARCHAR(64),
			pipeline_version_id VARCHAR(64),
			status VARCHAR(50),
			error TEXT,
			created DATETIME,
			updated DATETIME
		)`,
		`CREATE TABLE t_stage (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			pipeline_version_id VARCHAR(64),
			build_id VARCHAR(64),
			name VARCHAR(200),
			display_name VARCHAR(200),
			stage VARCHAR(200),
			status VARCHAR(50),
			error TEXT,
			sort INT,
			started DATETIME,
			finished DATETIME,
			created DATETIME,
			updated DATETIME
		)`,
		`CREATE TABLE t_step (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			stage_id VARCHAR(64),
			build_id VARCHAR(64),
			pipeline_version_id VARCHAR(64),
			name VARCHAR(200),
			display_name VARCHAR(200),
			step VARCHAR(200),
			status VARCHAR(50),
			exit_code BIGINT,
			error TEXT,
			sort INT,
			waits TEXT,
			errignore INT,
			started DATETIME,
			finished DATETIME,
			created DATETIME,
			updated DATETIME
		)`,
		`CREATE TABLE t_cmd_line (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			group_id VARCHAR(64),
			build_id VARCHAR(64),
			step_id VARCHAR(64),
			num INT,
			code INT,
			content TEXT,
			status VARCHAR(50),
			started DATETIME,
			finished DATETIME,
			created DATETIME
		)`,
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	comm.Db = db
}

func TestRuntimeController_stages_MissingPvId(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	body := hbtp.Map{"pvId": ""}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := RuntimeController{}
	ctrl.stages(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntimeController_stages_EmptyResult(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	body := hbtp.Map{"pvId": "nonexistent"}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := RuntimeController{}
	ctrl.stages(c, &body)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRuntimeController_cmds_MissingStepId(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	body := hbtp.Map{"stepId": ""}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := RuntimeController{}
	ctrl.cmds(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntimeController_cmds_EmptyResult(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	body := hbtp.Map{"stepId": "nonexistent"}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := RuntimeController{}
	ctrl.cmds(c, &body)

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRuntimeController_build_MissingBuildId(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	body := hbtp.Map{"buildId": ""}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := RuntimeController{}
	ctrl.build(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntimeController_build_NotFound(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	body := hbtp.Map{"buildId": "nonexistent"}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// engine.Mgr is nil in test environment, so skip this test
	t.Skip("engine.Mgr not initialized in unit test environment")
}

func TestRuntimeController_cancel_MissingBuildId(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	body := hbtp.Map{"buildId": ""}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := RuntimeController{}
	ctrl.cancel(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntimeController_cancel_NotFound(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	// Create a build record in DB but don't start the engine
	build := &model.TBuild{Id: "test-build-123"}
	_, _ = comm.Db.InsertOne(build)

	body := hbtp.Map{"buildId": "test-build-123"}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// This will fail because engine.Mgr is nil, so skip this test
	t.Skip("engine.Mgr not initialized in unit test environment")
	_ = w
}

func TestRuntimeController_logs_MissingStepId(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	body := hbtp.Map{"stepId": ""}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := RuntimeController{}
	ctrl.logs(c, &body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntimeController_logs_FileNotFound(t *testing.T) {
	setupRuntimeTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c.Set(service.LgUserKey, adminUser)

	// Set a temporary work path
	oldWorkPath := comm.WorkPath
	comm.WorkPath = "/tmp/gokins-test-nonexistent"
	t.Cleanup(func() { comm.WorkPath = oldWorkPath })

	body := hbtp.Map{"stepId": "test-step", "buildId": "test-build"}
	bodyBytes, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/test", bytes.NewReader(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	ctrl := RuntimeController{}
	ctrl.logs(c, &body)

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}
