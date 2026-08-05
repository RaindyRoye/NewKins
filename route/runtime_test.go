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

func setupRuntimeTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create t_stage table
	_, err = db.Exec(`CREATE TABLE t_stage (
		id VARCHAR(64) PRIMARY KEY,
		pipeline_version_id VARCHAR(64),
		build_id VARCHAR(64),
		status VARCHAR(100),
		error VARCHAR(500),
		name VARCHAR(255),
		display_name VARCHAR(255),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		stage VARCHAR(255),
		sort BIGINT
	)`)
	if err != nil {
		t.Fatalf("create t_stage table: %v", err)
	}

	// Create t_step table
	_, err = db.Exec(`CREATE TABLE t_step (
		id VARCHAR(64) PRIMARY KEY,
		build_id VARCHAR(64),
		stage_id VARCHAR(100),
		display_name VARCHAR(255),
		pipeline_version_id VARCHAR(64),
		step VARCHAR(255),
		status VARCHAR(100),
		exit_code BIGINT,
		error VARCHAR(500),
		name VARCHAR(100),
		started DATETIME,
		finished DATETIME,
		created DATETIME,
		updated DATETIME,
		errignore INT,
		waits TEXT,
		sort BIGINT
	)`)
	if err != nil {
		t.Fatalf("create t_step table: %v", err)
	}

	// Create t_cmd_line table
	_, err = db.Exec(`CREATE TABLE t_cmd_line (
		id VARCHAR(64) PRIMARY KEY,
		group_id VARCHAR(64),
		build_id VARCHAR(64),
		step_id VARCHAR(64),
		status VARCHAR(50),
		num INT,
		code INT,
		content TEXT,
		created DATETIME,
		started DATETIME,
		finished DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_cmd_line table: %v", err)
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

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	return db
}

func makeRuntimeGinContext(t *testing.T, body interface{}, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

func TestRuntimeController_stages_EmptyPvId(t *testing.T) {
	setupRuntimeTestDB(t)
	c, w := makeRuntimeGinContext(t, hbtp.Map{"pvId": ""}, nil)
	ctrl := RuntimeController{}
	ctrl.stages(c, &hbtp.Map{"pvId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

func TestRuntimeController_stages_NoStages(t *testing.T) {
	setupRuntimeTestDB(t)
	c, w := makeRuntimeGinContext(t, hbtp.Map{"pvId": "pv-123"}, nil)
	ctrl := RuntimeController{}
	ctrl.stages(c, &hbtp.Map{"pvId": "pv-123"})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestRuntimeController_stages_WithStages(t *testing.T) {
	db := setupRuntimeTestDB(t)

	// Insert test stages
	_, err := db.Insert(&model.RunStage{
		Id:                "stage-1",
		PipelineVersionId: "pv-123",
		BuildId:           "build-1",
		Status:            "pending",
		Name:              "build",
		DisplayName:       "Build Stage",
		Created:           time.Now(),
	})
	if err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	// Insert test steps
	_, err = db.Insert(&model.RunStep{
		Id:                "step-1",
		BuildId:           "build-1",
		StageId:           "stage-1",
		PipelineVersionId: "pv-123",
		Step:              "shell",
		Status:            "pending",
		Name:              "compile",
		Created:           time.Now(),
		Sort:              1,
	})
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}

	c, w := makeRuntimeGinContext(t, hbtp.Map{"pvId": "pv-123"}, nil)
	ctrl := RuntimeController{}
	ctrl.stages(c, &hbtp.Map{"pvId": "pv-123"})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify response structure
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["ids"] == nil {
		t.Error("expected 'ids' in response")
	}
	if resp["stages"] == nil {
		t.Error("expected 'stages' in response")
	}
	if resp["steps"] == nil {
		t.Error("expected 'steps' in response")
	}
}

func TestRuntimeController_cmds_EmptyStepId(t *testing.T) {
	setupRuntimeTestDB(t)
	c, w := makeRuntimeGinContext(t, hbtp.Map{"stepId": ""}, nil)
	ctrl := RuntimeController{}
	ctrl.cmds(c, &hbtp.Map{"stepId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

func TestRuntimeController_cmds_NoCmds(t *testing.T) {
	setupRuntimeTestDB(t)
	c, w := makeRuntimeGinContext(t, hbtp.Map{"stepId": "step-123"}, nil)
	ctrl := RuntimeController{}
	ctrl.cmds(c, &hbtp.Map{"stepId": "step-123"})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestRuntimeController_cmds_WithCmds(t *testing.T) {
	db := setupRuntimeTestDB(t)

	// Insert test commands
	_, err := db.Insert(&model.TCmdLine{
		Id:      "cmd-1",
		BuildId: "build-1",
		StepId:  "step-123",
		Status:  "pending",
		Num:     1,
		Content: "echo hello",
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	_, err = db.Insert(&model.TCmdLine{
		Id:      "cmd-2",
		BuildId: "build-1",
		StepId:  "step-123",
		Status:  "pending",
		Num:     2,
		Content: "make build",
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert cmd: %v", err)
	}

	c, w := makeRuntimeGinContext(t, hbtp.Map{"stepId": "step-123"}, nil)
	ctrl := RuntimeController{}
	ctrl.cmds(c, &hbtp.Map{"stepId": "step-123"})

	if w.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify response structure
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["stepId"] != "step-123" {
		t.Errorf("stepId = %v, want 'step-123'", resp["stepId"])
	}
	if resp["cmds"] == nil {
		t.Error("expected 'cmds' in response")
	}
}

func TestRuntimeController_build_EmptyBuildId(t *testing.T) {
	setupRuntimeTestDB(t)
	c, w := makeRuntimeGinContext(t, hbtp.Map{"buildId": ""}, nil)
	ctrl := RuntimeController{}
	ctrl.build(c, &hbtp.Map{"buildId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

func TestRuntimeController_build_NotFound(t *testing.T) {
	setupRuntimeTestDB(t)
	c, w := makeRuntimeGinContext(t, hbtp.Map{"buildId": "nonexistent"}, nil)
	ctrl := RuntimeController{}
	ctrl.build(c, &hbtp.Map{"buildId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRuntimeController_cancel_EmptyBuildId(t *testing.T) {
	setupRuntimeTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{"buildId": ""}, adminUser)
	ctrl := RuntimeController{}
	ctrl.cancel(c, &hbtp.Map{"buildId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

func TestRuntimeController_cancel_BuildNotFound(t *testing.T) {
	setupRuntimeTestDB(t)
	adminUser := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{"buildId": "nonexistent"}, adminUser)
	ctrl := RuntimeController{}
	ctrl.cancel(c, &hbtp.Map{"buildId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRuntimeController_logs_EmptyStepId(t *testing.T) {
	setupRuntimeTestDB(t)
	c, w := makeRuntimeGinContext(t, hbtp.Map{"stepId": "", "buildId": "build-1"}, nil)
	ctrl := RuntimeController{}
	ctrl.logs(c, &hbtp.Map{"stepId": "", "buildId": "build-1"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

func TestRuntimeController_logs_FileNotFound(t *testing.T) {
	setupRuntimeTestDB(t)
	c, w := makeRuntimeGinContext(t, hbtp.Map{"stepId": "step-1", "buildId": "build-1"}, nil)
	ctrl := RuntimeController{}
	ctrl.logs(c, &hbtp.Map{"stepId": "step-1", "buildId": "build-1"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}
