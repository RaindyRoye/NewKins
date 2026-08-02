package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/core/common"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupRuntimeTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create tables matching the models
	_, err = db.Exec(`CREATE TABLE t_stage (
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
	)`)
	if err != nil {
		t.Fatalf("create t_stage: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_step (
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
	)`)
	if err != nil {
		t.Fatalf("create t_step: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_cmd_line (
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
	)`)
	if err != nil {
		t.Fatalf("create t_cmd_line: %v", err)
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
		t.Fatalf("create t_build: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(255),
		access_token VARCHAR(255),
		url VARCHAR(500),
		username VARCHAR(255),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		create_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_pipeline_info (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		pipeline_type VARCHAR(255),
		access_token VARCHAR(255),
		url VARCHAR(500),
		username VARCHAR(255),
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_info: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org_pipe (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		org_id VARCHAR(64),
		pipeline_id VARCHAR(64)
	)`)
	if err != nil {
		t.Fatalf("create t_org_pipe: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_info (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		phone VARCHAR(100),
		email VARCHAR(200),
		birthday DATETIME,
		remark TEXT,
		perm_user INT,
		perm_org INT,
		perm_pipe INT
	)`)
	if err != nil {
		t.Fatalf("create t_user_info: %v", err)
	}

	comm.Db = db
	return db
}

func makeRuntimeGinCtx(t *testing.T, body interface{}, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
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

// --- GetPath / Routes ---

func TestRuntimeController_Path(t *testing.T) {
	c := &RuntimeController{}
	if got := c.GetPath(); got != "/api/runtime" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/runtime")
	}
}

// --- stages handler tests ---

func TestRuntime_Stages_EmptyPvId(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("pvId", "")
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.stages(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntime_Stages_NoStages(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("pvId", "pv-nonexistent")
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.stages(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	ids, ok := resp["ids"].([]interface{})
	if !ok || len(ids) != 0 {
		t.Errorf("expected empty ids, got %v", resp["ids"])
	}
}

func TestRuntime_Stages_WithStagesAndSteps(t *testing.T) {
	db := setupRuntimeTestDB(t)
	ctrl := RuntimeController{}

	// Insert stages
	now := time.Now()
	stage1 := &model.RunStage{
		Id: "stage-1", PipelineVersionId: "pv-1", BuildId: "bd-1",
		Name: "build", Stage: "build", Created: now,
	}
	stage2 := &model.RunStage{
		Id: "stage-2", PipelineVersionId: "pv-1", BuildId: "bd-1",
		Name: "test", Stage: "test", Created: now,
	}
	_, err := db.InsertOne(stage1)
	if err != nil {
		t.Fatalf("insert stage1: %v", err)
	}
	_, err = db.InsertOne(stage2)
	if err != nil {
		t.Fatalf("insert stage2: %v", err)
	}

	// Insert steps
	step1 := &model.RunStep{
		Id: "step-1", StageId: "stage-1", BuildId: "bd-1",
		PipelineVersionId: "pv-1", Name: "compile", Step: "compile",
		Waits: `["step-0"]`, Created: now,
	}
	step2 := &model.RunStep{
		Id: "step-2", StageId: "stage-1", BuildId: "bd-1",
		PipelineVersionId: "pv-1", Name: "lint", Step: "lint",
		Waits: `[]`, Created: now,
	}
	step3 := &model.RunStep{
		Id: "step-3", StageId: "stage-2", BuildId: "bd-1",
		PipelineVersionId: "pv-1", Name: "unit-test", Step: "test",
		Waits: `["step-1"]`, Created: now,
	}
	for _, s := range []*model.RunStep{step1, step2, step3} {
		if _, err := db.InsertOne(s); err != nil {
			t.Fatalf("insert step %s: %v", s.Id, err)
		}
	}

	m := &hbtp.Map{}
	m.Set("pvId", "pv-1")
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.stages(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Ids    []string                      `json:"ids"`
		Stages map[string]*model.RunStage    `json:"stages"`
		Steps  map[string]*model.RunStep     `json:"steps"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Ids) != 2 {
		t.Errorf("expected 2 stage ids, got %d", len(resp.Ids))
	}
	if len(resp.Stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(resp.Stages))
	}
	if len(resp.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(resp.Steps))
	}

	// Verify step waits were parsed
	if s, ok := resp.Steps["step-1"]; ok {
		if len(s.Waitings) != 1 || s.Waitings[0] != "step-0" {
			t.Errorf("step-1 waitings = %v, want [step-0]", s.Waitings)
		}
	} else {
		t.Error("step-1 not found in response")
	}
}

func TestRuntime_Stages_StepWithInvalidWaitsJSON(t *testing.T) {
	db := setupRuntimeTestDB(t)
	ctrl := RuntimeController{}

	now := time.Now()
	stage := &model.RunStage{
		Id: "stage-bad", PipelineVersionId: "pv-bad", BuildId: "bd-bad",
		Name: "build", Stage: "build", Created: now,
	}
	_, _ = db.InsertOne(stage)

	// Step with invalid JSON in waits field
	step := &model.RunStep{
		Id: "step-bad", StageId: "stage-bad", BuildId: "bd-bad",
		PipelineVersionId: "pv-bad", Name: "s1", Step: "s1",
		Waits: `{invalid json`, Created: now,
	}
	_, _ = db.InsertOne(step)

	m := &hbtp.Map{}
	m.Set("pvId", "pv-bad")
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.stages(c, m)

	// Should still return 200 (warns about bad JSON but doesn't fail)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- cmds handler tests ---

func TestRuntime_Cmds_EmptyStepId(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("stepId", "")
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.cmds(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntime_Cmds_NoCmds(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("stepId", "step-no-cmds")
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.cmds(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["stepId"] != "step-no-cmds" {
		t.Errorf("stepId = %v, want step-no-cmds", resp["stepId"])
	}
}

func TestRuntime_Cmds_WithCmdLines(t *testing.T) {
	db := setupRuntimeTestDB(t)
	ctrl := RuntimeController{}

	now := time.Now()
	cmd1 := &model.TCmdLine{
		Id: "cmd-1", StepId: "step-with-cmds", BuildId: "bd-1",
		Num: 1, Content: "echo hello", Status: "ok", Created: now,
	}
	cmd2 := &model.TCmdLine{
		Id: "cmd-2", StepId: "step-with-cmds", BuildId: "bd-1",
		Num: 2, Content: "go test ./...", Status: "ok", Created: now,
	}
	_, _ = db.InsertOne(cmd1)
	_, _ = db.InsertOne(cmd2)

	m := &hbtp.Map{}
	m.Set("stepId", "step-with-cmds")
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.cmds(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		StepId string           `json:"stepId"`
		Cmds   []*model.TCmdLine `json:"cmds"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.StepId != "step-with-cmds" {
		t.Errorf("stepId = %q, want %q", resp.StepId, "step-with-cmds")
	}
	if len(resp.Cmds) != 2 {
		t.Errorf("expected 2 cmds, got %d", len(resp.Cmds))
	}
}

// --- build handler tests ---

func TestRuntime_Build_EmptyBuildId(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("buildId", "")
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.build(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntime_Build_NotInEngine(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("buildId", "nonexistent-build")
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.build(c, m)

	// BuildEgn is nil in test, so Get should return false => 404
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- cancel handler tests ---

func TestRuntime_Cancel_EmptyBuildId(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("buildId", "")
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.cancel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntime_Cancel_BuildNotFound(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	user := &model.TUser{Id: "usr-1", Name: "admin", Active: 1}
	m := &hbtp.Map{}
	m.Set("buildId", "nonexistent-build")
	c, w := makeRuntimeGinCtx(t, m, user)
	ctrl.cancel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
}

// --- logs handler tests ---

func TestRuntime_Logs_EmptyStepId(t *testing.T) {
	setupRuntimeTestDB(t)
	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("stepId", "")
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.logs(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRuntime_Logs_FileNotFound(t *testing.T) {
	setupRuntimeTestDB(t)

	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })
	comm.WorkPath = t.TempDir()

	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("buildId", "bd-1")
	m.Set("stepId", "step-1")
	m.Set("offset", int64(0))
	m.Set("limit", int64(0))
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.logs(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRuntime_Logs_ValidLogFile(t *testing.T) {
	setupRuntimeTestDB(t)

	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })
	comm.WorkPath = tmpDir

	// Create log file with JSON lines
	buildId := "bd-log-1"
	stepId := "step-log-1"
	logDir := filepath.Join(tmpDir, common.PathBuild, buildId, common.PathJobs, stepId)
	if err := os.MkdirAll(logDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logContent := `{"id":"l1","content":"line 1","times":"2026-01-01T00:00:00Z","errs":false}
{"id":"l2","content":"line 2","times":"2026-01-01T00:00:01Z","errs":false}
{"id":"l3","content":"line 3","times":"2026-01-01T00:00:02Z","errs":true}
`
	logPath := filepath.Join(logDir, "build.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0600); err != nil {
		t.Fatalf("write log: %v", err)
	}

	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("buildId", buildId)
	m.Set("stepId", stepId)
	m.Set("offset", int64(0))
	m.Set("limit", int64(0))
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.logs(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		StepId  string `json:"stepId"`
		Lastoff int64  `json:"lastoff"`
		Logs    []struct {
			Id      string `json:"id"`
			Content string `json:"content"`
			Errs    bool   `json:"errs"`
			Offset  int64  `json:"offset"`
		} `json:"logs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.StepId != stepId {
		t.Errorf("stepId = %q, want %q", resp.StepId, stepId)
	}
	if len(resp.Logs) != 3 {
		t.Errorf("expected 3 log lines, got %d", len(resp.Logs))
	}
	if resp.Logs[0].Content != "line 1" {
		t.Errorf("first log content = %q, want %q", resp.Logs[0].Content, "line 1")
	}
	if !resp.Logs[2].Errs {
		t.Error("third log should have errs=true")
	}
}

func TestRuntime_Logs_WithOffset(t *testing.T) {
	setupRuntimeTestDB(t)

	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })
	comm.WorkPath = tmpDir

	buildId := "bd-off"
	stepId := "step-off"
	logDir := filepath.Join(tmpDir, common.PathBuild, buildId, common.PathJobs, stepId)
	if err := os.MkdirAll(logDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logContent := `{"id":"l1","content":"first","times":"2026-01-01T00:00:00Z","errs":false}
{"id":"l2","content":"second","times":"2026-01-01T00:00:01Z","errs":false}
`
	logPath := filepath.Join(logDir, "build.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0600); err != nil {
		t.Fatalf("write log: %v", err)
	}

	// Use offset that skips the first line (seek to after first newline)
	firstLineEnd := int64(bytes.IndexByte([]byte(logContent), '\n') + 1)

	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("buildId", buildId)
	m.Set("stepId", stepId)
	m.Set("offset", firstLineEnd)
	m.Set("limit", int64(0))
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.logs(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Logs []struct {
			Id      string `json:"id"`
			Content string `json:"content"`
		} `json:"logs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Should only get the second line
	if len(resp.Logs) != 1 {
		t.Errorf("expected 1 log line with offset, got %d", len(resp.Logs))
	}
	if len(resp.Logs) > 0 && resp.Logs[0].Id != "l2" {
		t.Errorf("log id = %q, want %q", resp.Logs[0].Id, "l2")
	}
}

func TestRuntime_Logs_WithLimit(t *testing.T) {
	setupRuntimeTestDB(t)

	tmpDir := t.TempDir()
	origWorkPath := comm.WorkPath
	t.Cleanup(func() { comm.WorkPath = origWorkPath })
	comm.WorkPath = tmpDir

	buildId := "bd-lim"
	stepId := "step-lim"
	logDir := filepath.Join(tmpDir, common.PathBuild, buildId, common.PathJobs, stepId)
	if err := os.MkdirAll(logDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logContent := `{"id":"l1","content":"a","times":"2026-01-01T00:00:00Z","errs":false}
{"id":"l2","content":"b","times":"2026-01-01T00:00:01Z","errs":false}
{"id":"l3","content":"c","times":"2026-01-01T00:00:02Z","errs":false}
`
	logPath := filepath.Join(logDir, "build.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0600); err != nil {
		t.Fatalf("write log: %v", err)
	}

	ctrl := RuntimeController{}
	m := &hbtp.Map{}
	m.Set("buildId", buildId)
	m.Set("stepId", stepId)
	m.Set("offset", int64(0))
	m.Set("limit", int64(2))
	c, w := makeRuntimeGinCtx(t, m, nil)
	ctrl.logs(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Logs []struct {
			Id string `json:"id"`
		} `json:"logs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// limit=2, should stop after 2 entries
	if len(resp.Logs) > 2 {
		t.Errorf("expected at most 2 log lines with limit=2, got %d", len(resp.Logs))
	}
}
