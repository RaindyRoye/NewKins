package route

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

	// Create tables needed by runtime handlers
	tables := []string{
		`CREATE TABLE t_stage (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
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
		)`,
		`CREATE TABLE t_step (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
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
		)`,
		`CREATE TABLE t_cmd_line (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
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
		)`,
		`CREATE TABLE t_build (
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
	}
	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	origDb := comm.Db
	comm.Db = db
	t.Cleanup(func() { comm.Db = origDb })

	return db
}

func makeRuntimeGinCtx(t *testing.T, loggedInUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if loggedInUser != nil {
		c.Set(service.LgUserKey, loggedInUser)
	}
	return c, w
}

// --- stages handler tests ---

func TestRuntimeStages_EmptyPvId(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	c, w := makeRuntimeGinCtx(t, nil)
	m := &hbtp.Map{}
	m.Set("pvId", "")
	ctrl.stages(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w.Body.String() != "param err" {
		t.Errorf("body = %q, want %q", w.Body.String(), "param err")
	}
}

func TestRuntimeStages_NoStages(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	c, w := makeRuntimeGinCtx(t, nil)
	m := &hbtp.Map{}
	m.Set("pvId", "nonexistent-pv")
	ctrl.stages(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if ids, ok := resp["ids"].([]interface{}); ok && len(ids) != 0 {
		t.Errorf("expected empty ids, got %d", len(ids))
	}
}

func TestRuntimeStages_WithData(t *testing.T) {
	db := setupRuntimeTestDB(t)

	// Insert a stage
	stage := &model.RunStage{
		Id:                "stage-1",
		PipelineVersionId: "pv-1",
		BuildId:           "build-1",
		Name:              "build",
		Status:            "running",
		Created:           time.Now(),
	}
	if _, err := db.InsertOne(stage); err != nil {
		t.Fatalf("insert stage: %v", err)
	}

	// Insert steps for this stage
	step1 := &model.RunStep{
		Id:      "step-1",
		StageId: "stage-1",
		Name:    "checkout",
		Status:  "success",
		Waits:   `["step-0"]`,
		Sort:    1,
	}
	step2 := &model.RunStep{
		Id:      "step-2",
		StageId: "stage-1",
		Name:    "test",
		Status:  "pending",
		Waits:   `["step-1"]`,
		Sort:    2,
	}
	if _, err := db.InsertOne(step1); err != nil {
		t.Fatalf("insert step1: %v", err)
	}
	if _, err := db.InsertOne(step2); err != nil {
		t.Fatalf("insert step2: %v", err)
	}

	ctrl := RuntimeController{}
	c, w := makeRuntimeGinCtx(t, nil)
	m := &hbtp.Map{}
	m.Set("pvId", "pv-1")
	ctrl.stages(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	ids, ok := resp["ids"].([]interface{})
	if !ok {
		t.Fatal("response missing 'ids' field")
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 stage id, got %d", len(ids))
	}

	stagesMap, ok := resp["stages"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing 'stages' map")
	}
	if _, exists := stagesMap["stage-1"]; !exists {
		t.Error("stages map should contain 'stage-1'")
	}

	stepsMap, ok := resp["steps"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing 'steps' map")
	}
	if len(stepsMap) != 2 {
		t.Errorf("expected 2 steps, got %d", len(stepsMap))
	}
}

// --- cmds handler tests ---

func TestRuntimeCmds_EmptyStepId(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	c, w := makeRuntimeGinCtx(t, nil)
	m := &hbtp.Map{}
	m.Set("stepId", "")
	ctrl.cmds(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntimeCmds_NoCommands(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	c, w := makeRuntimeGinCtx(t, nil)
	m := &hbtp.Map{}
	m.Set("stepId", "nonexistent-step")
	ctrl.cmds(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["stepId"] != "nonexistent-step" {
		t.Errorf("stepId = %v, want %q", resp["stepId"], "nonexistent-step")
	}
}

func TestRuntimeCmds_WithCommands(t *testing.T) {
	db := setupRuntimeTestDB(t)

	cmds := []*model.TCmdLine{
		{Id: "cmd-1", StepId: "step-1", Num: 1, Status: "success", Code: 0, Content: "echo hello", Created: time.Now()},
		{Id: "cmd-2", StepId: "step-1", Num: 2, Status: "running", Code: -1, Content: "make test", Created: time.Now()},
	}
	for _, cmd := range cmds {
		if _, err := db.InsertOne(cmd); err != nil {
			t.Fatalf("insert cmd: %v", err)
		}
	}

	ctrl := RuntimeController{}
	c, w := makeRuntimeGinCtx(t, nil)
	m := &hbtp.Map{}
	m.Set("stepId", "step-1")
	ctrl.cmds(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	cmdList, ok := resp["cmds"].([]interface{})
	if !ok {
		t.Fatal("response missing 'cmds' field")
	}
	if len(cmdList) != 2 {
		t.Errorf("expected 2 cmds, got %d", len(cmdList))
	}
}

// --- cancel handler tests ---

func TestRuntimeCancel_EmptyBuildId(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	c, w := makeRuntimeGinCtx(t, nil)
	m := &hbtp.Map{}
	m.Set("buildId", "")
	ctrl.cancel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntimeCancel_BuildNotFound(t *testing.T) {
	setupRuntimeTestDB(t)
	user := &model.TUser{Id: "user-1", Name: "tester", Active: 1}
	ctrl := RuntimeController{}
	c, w := makeRuntimeGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("buildId", "nonexistent-build")
	ctrl.cancel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- logs handler tests ---

func TestRuntimeLogs_EmptyStepId(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	c, w := makeRuntimeGinCtx(t, nil)
	m := &hbtp.Map{}
	m.Set("stepId", "")
	ctrl.logs(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntimeLogs_FileNotFound(t *testing.T) {
	setupRuntimeTestDB(t)
	oldWorkPath := comm.WorkPath
	comm.WorkPath = "/nonexistent"
	t.Cleanup(func() { comm.WorkPath = oldWorkPath })

	ctrl := RuntimeController{}
	c, w := makeRuntimeGinCtx(t, nil)
	m := &hbtp.Map{}
	m.Set("stepId", "step-1")
	m.Set("buildId", "build-1")
	ctrl.logs(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRuntimeLogs_ReadsLogFile(t *testing.T) {
	db := setupRuntimeTestDB(t)
	_ = db

	// Create a temp directory structure for build logs
	tmpDir := t.TempDir()
	oldWorkPath := comm.WorkPath
	comm.WorkPath = tmpDir
	t.Cleanup(func() { comm.WorkPath = oldWorkPath })

	// Create build/job log directory and file
	// Path: WorkPath/build/{buildId}/jobs/{stepId}/build.log
	logDir := filepath.Join(tmpDir, "build", "build-1", "jobs", "step-1")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("create log dir: %v", err)
	}

	logContent := `{"id":"1","content":"hello world","times":"2026-01-01T00:00:00Z"}
{"id":"2","content":"second line","times":"2026-01-01T00:00:01Z"}
`
	if err := os.WriteFile(filepath.Join(logDir, "build.log"), []byte(logContent), 0600); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	ctrl := RuntimeController{}
	c, w := makeRuntimeGinCtx(t, nil)
	m := &hbtp.Map{}
	m.Set("stepId", "step-1")
	m.Set("buildId", "build-1")
	ctrl.logs(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp["stepId"] != "step-1" {
		t.Errorf("stepId = %v, want %q", resp["stepId"], "step-1")
	}
	logs, ok := resp["logs"].([]interface{})
	if !ok {
		t.Fatal("response missing 'logs' field")
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 log entries, got %d", len(logs))
	}
}

// --- batchRunPipelineVersions tests ---

func TestBatchRunPipelineVersions_Empty(t *testing.T) {
	setupRuntimeTestDB(t)
	result, err := batchRunPipelineVersions(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}
