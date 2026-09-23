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

func setupPipelineTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
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
			perm_org INT DEFAULT 0,
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
			create_time DATETIME,
			created DATETIME
		)`,
		`CREATE TABLE t_pipeline_conf (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			pipeline_id VARCHAR(64) NOT NULL,
			url VARCHAR(255),
			access_token VARCHAR(255),
			yml_content TEXT,
			username VARCHAR(255)
		)`,
		`CREATE TABLE t_pipeline_var (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			pipeline_id VARCHAR(64),
			name VARCHAR(255),
			value TEXT,
			remarks VARCHAR(255),
			public INT DEFAULT 0
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
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("exec %q: %v", sql[:40], err)
		}
	}

	comm.Db = db
}

func makePipelineGinCtx(t *testing.T, body interface{}, lgUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		bts, _ := json.Marshal(body)
		req = httptest.NewRequest("POST", "/test", bytes.NewReader(bts))
	} else {
		req = httptest.NewRequest("POST", "/test", nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	if lgUser != nil {
		c.Set(service.LgUserKey, lgUser)
	}
	return c, w
}

func createPipelineTestUser(t *testing.T) *model.TUser {
	t.Helper()
	usr := &model.TUser{
		Id:        utils.NewXid(),
		Name:      "testuser",
		Nick:      "Test User",
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	if _, err := comm.Db.InsertOne(usr); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return usr
}

func createTestPipeline(t *testing.T, uid, name, displayName string) *model.TPipeline {
	t.Helper()
	pipe := &model.TPipeline{
		Id:           utils.NewXid(),
		Uid:          uid,
		Name:         name,
		DisplayName:  displayName,
		PipelineType: "",
	}
	if _, err := comm.Db.InsertOne(pipe); err != nil {
		t.Fatalf("create pipeline: %v", err)
	}
	return pipe
}

func createTestPipelineConf(t *testing.T, pipelineId, ymlContent, url, username, token string) *model.TPipelineConf {
	t.Helper()
	conf := &model.TPipelineConf{
		PipelineId:  pipelineId,
		YmlContent:  ymlContent,
		Url:         url,
		Username:    username,
		AccessToken: token,
	}
	if _, err := comm.Db.InsertOne(conf); err != nil {
		t.Fatalf("create pipeline conf: %v", err)
	}
	return conf
}

// TestPipelineController_info_Success tests retrieving pipeline info
func TestPipelineController_info_Success(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	pipe := createTestPipeline(t, user.Id, "test-pipeline", "Test Pipeline")
	createTestPipelineConf(t, pipe.Id, "version: 1\nstages: []", "https://github.com/test/repo", "user", "token123")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", pipe.Id)
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["pipe"] == nil {
		t.Error("response missing pipe field")
	}
	if resp["perm"] == nil {
		t.Error("response missing perm field")
	}
}

// TestPipelineController_info_NotFound tests info for non-existent pipeline
func TestPipelineController_info_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestPipelineController_info_EmptyID tests info with empty pipeline ID
func TestPipelineController_info_EmptyID(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestPipelineController_delete_Success tests deleting a pipeline
func TestPipelineController_delete_Success(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	pipe := createTestPipeline(t, user.Id, "test-pipeline", "Test Pipeline")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", pipe.Id)
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify pipeline is marked as deleted
	updated := &model.TPipeline{}
	ok, err := comm.Db.Where("id = ?", pipe.Id).Get(updated)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline not found")
	}
	if updated.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", updated.Deleted)
	}
}

// TestPipelineController_delete_NotFound tests deleting non-existent pipeline
func TestPipelineController_delete_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestPipelineController_delete_EmptyID tests deleting with empty ID
func TestPipelineController_delete_EmptyID(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestPipelineController_new_Success tests creating a new pipeline
func TestPipelineController_new_Success(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)

	// Create user_info with perm_pipe = 1
	userInfo := &model.TUserInfo{
		Id:       user.Id,
		PermPipe: 1,
	}
	if _, err := comm.Db.InsertOne(userInfo); err != nil {
		t.Fatalf("create user info: %v", err)
	}

	ymlContent := `version: 1
stages:
  - name: build
    steps:
      - step: build-plugin
        name: compile
        commands:
          - go build
`
	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:        "new-pipeline",
		DisplayName: "New Pipeline",
		Content:     ymlContent,
		OrgId:       "",
		Url:         "https://github.com/test/repo",
		Username:    "user",
		AccessToken: "token123",
	}
	c, w := makePipelineGinCtx(t, npipe, user)
	ctrl.new(c, npipe)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp model.TPipeline
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Id == "" {
		t.Error("response missing pipeline ID")
	}
	if resp.Name != "new-pipeline" {
		t.Errorf("name = %q, want %q", resp.Name, "new-pipeline")
	}

	// Verify pipeline was created
	created := &model.TPipeline{}
	ok, err := comm.Db.Where("id = ?", resp.Id).Get(created)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline not found in DB")
	}
}

// TestPipelineController_new_InvalidYAML tests creating with invalid YAML
func TestPipelineController_new_InvalidYAML(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)

	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:    "test-pipeline",
		Content: "invalid: yaml: content:",
	}
	c, w := makePipelineGinCtx(t, npipe, user)
	ctrl.new(c, npipe)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestPipelineController_new_EmptyName tests creating with empty name
func TestPipelineController_new_EmptyName(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)

	ctrl := PipelineController{}
	npipe := &bean.NewPipeline{
		Name:    "",
		Content: "version: 1",
	}
	c, w := makePipelineGinCtx(t, npipe, user)
	ctrl.new(c, npipe)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestPipelineController_vars_Success tests listing pipeline variables
func TestPipelineController_vars_Success(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	pipe := createTestPipeline(t, user.Id, "test-pipeline", "Test Pipeline")

	// Create test variables
	v1 := &model.TPipelineVar{
		PipelineId: pipe.Id,
		Name:       "DB_HOST",
		Value:      "localhost",
		Public:     0,
	}
	v2 := &model.TPipelineVar{
		PipelineId: pipe.Id,
		Name:       "API_KEY",
		Value:      "secret123",
		Public:     0,
	}
	if _, err := comm.Db.InsertOne(v1); err != nil {
		t.Fatalf("insert v1: %v", err)
	}
	if _, err := comm.Db.InsertOne(v2); err != nil {
		t.Fatalf("insert v2: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", pipe.Id)
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// TestPipelineController_vars_WithSearch tests listing variables with search query
func TestPipelineController_vars_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	pipe := createTestPipeline(t, user.Id, "test-pipeline", "Test Pipeline")

	v1 := &model.TPipelineVar{
		PipelineId: pipe.Id,
		Name:       "DB_HOST",
		Value:      "localhost",
		Public:     0,
	}
	v2 := &model.TPipelineVar{
		PipelineId: pipe.Id,
		Name:       "API_KEY",
		Value:      "secret123",
		Public:     0,
	}
	if _, err := comm.Db.InsertOne(v1); err != nil {
		t.Fatalf("insert v1: %v", err)
	}
	if _, err := comm.Db.InsertOne(v2); err != nil {
		t.Fatalf("insert v2: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", pipe.Id)
	m.Set("q", "DB")
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// TestPipelineController_vars_EmptyPipelineID tests vars with empty pipeline ID
func TestPipelineController_vars_EmptyPipelineID(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestPipelineController_varSave_Success tests saving a pipeline variable
func TestPipelineController_varSave_Success(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	pipe := createTestPipeline(t, user.Id, "test-pipeline", "Test Pipeline")

	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		PipelineId: pipe.Id,
		Name:       "NEW_VAR",
		Value:      "new_value",
		Remarks:    "test variable",
		Public:     false,
	}
	c, w := makePipelineGinCtx(t, pv, user)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify variable was created
	created := &model.TPipelineVar{}
	ok, err := comm.Db.Where("pipeline_id = ? AND name = ?", pipe.Id, "NEW_VAR").Get(created)
	if err != nil {
		t.Fatalf("query variable: %v", err)
	}
	if !ok {
		t.Fatal("variable not found in DB")
	}
	if created.Value != "new_value" {
		t.Errorf("value = %q, want %q", created.Value, "new_value")
	}
}

// TestPipelineController_varSave_Update tests updating an existing variable
func TestPipelineController_varSave_Update(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	pipe := createTestPipeline(t, user.Id, "test-pipeline", "Test Pipeline")

	existing := &model.TPipelineVar{
		PipelineId: pipe.Id,
		Name:       "EXISTING_VAR",
		Value:      "old_value",
		Public:     0,
	}
	if _, err := comm.Db.InsertOne(existing); err != nil {
		t.Fatalf("insert existing: %v", err)
	}

	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		Aid:        existing.Aid,
		PipelineId: pipe.Id,
		Name:       "EXISTING_VAR",
		Value:      "new_value",
		Public:     false,
	}
	c, w := makePipelineGinCtx(t, pv, user)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify variable was updated
	updated := &model.TPipelineVar{}
	ok, err := comm.Db.Where("aid = ?", existing.Aid).Get(updated)
	if err != nil {
		t.Fatalf("query variable: %v", err)
	}
	if !ok {
		t.Fatal("variable not found")
	}
	if updated.Value != "new_value" {
		t.Errorf("value = %q, want %q", updated.Value, "new_value")
	}
}

// TestPipelineController_varSave_EmptyValue tests saving with empty value
func TestPipelineController_varSave_EmptyValue(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	pipe := createTestPipeline(t, user.Id, "test-pipeline", "Test Pipeline")

	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		PipelineId: pipe.Id,
		Name:       "TEST_VAR",
		Value:      "",
	}
	c, w := makePipelineGinCtx(t, pv, user)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestPipelineController_varSave_DuplicateName tests saving with duplicate name
func TestPipelineController_varSave_DuplicateName(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	pipe := createTestPipeline(t, user.Id, "test-pipeline", "Test Pipeline")

	existing := &model.TPipelineVar{
		PipelineId: pipe.Id,
		Name:       "DUPLICATE_VAR",
		Value:      "value1",
		Public:     0,
	}
	if _, err := comm.Db.InsertOne(existing); err != nil {
		t.Fatalf("insert existing: %v", err)
	}

	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		PipelineId: pipe.Id,
		Name:       "DUPLICATE_VAR",
		Value:      "value2",
	}
	c, w := makePipelineGinCtx(t, pv, user)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// TestPipelineController_varDel_Success tests deleting a pipeline variable
func TestPipelineController_varDel_Success(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	pipe := createTestPipeline(t, user.Id, "test-pipeline", "Test Pipeline")

	pv := &model.TPipelineVar{
		PipelineId: pipe.Id,
		Name:       "TO_DELETE",
		Value:      "value",
		Public:     0,
	}
	if _, err := comm.Db.InsertOne(pv); err != nil {
		t.Fatalf("insert variable: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", pv.Aid)
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify variable was deleted
	deleted := &model.TPipelineVar{}
	ok, err := comm.Db.Where("aid = ?", pv.Aid).Get(deleted)
	if err != nil {
		t.Fatalf("query variable: %v", err)
	}
	if ok {
		t.Error("variable still exists after deletion")
	}
}

// TestPipelineController_varDel_NotFound tests deleting non-existent variable
func TestPipelineController_varDel_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(99999))
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestPipelineController_varDel_InvalidAID tests deleting with invalid aid
func TestPipelineController_varDel_InvalidAID(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestPipelineController_searchSha_Success tests searching for SHA commits
func TestPipelineController_searchSha_Success(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	pipe := createTestPipeline(t, user.Id, "test-pipeline", "Test Pipeline")

	// Create pipeline versions with SHAs
	pv1 := &model.TPipelineVersion{
		Id:         utils.NewXid(),
		PipelineId: pipe.Id,
		Sha:        "abc123",
		Created:    time.Now(),
	}
	pv2 := &model.TPipelineVersion{
		Id:         utils.NewXid(),
		PipelineId: pipe.Id,
		Sha:        "def456",
		Created:    time.Now(),
	}
	if _, err := comm.Db.InsertOne(pv1); err != nil {
		t.Fatalf("insert pv1: %v", err)
	}
	if _, err := comm.Db.InsertOne(pv2); err != nil {
		t.Fatalf("insert pv2: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", pipe.Id)
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("len(resp) = %d, want 2", len(resp))
	}
}

// TestPipelineController_searchSha_WithQuery tests searching with query filter
func TestPipelineController_searchSha_WithQuery(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	pipe := createTestPipeline(t, user.Id, "test-pipeline", "Test Pipeline")

	pv1 := &model.TPipelineVersion{
		Id:         utils.NewXid(),
		PipelineId: pipe.Id,
		Sha:        "abc123",
		Created:    time.Now(),
	}
	pv2 := &model.TPipelineVersion{
		Id:         utils.NewXid(),
		PipelineId: pipe.Id,
		Sha:        "def456",
		Created:    time.Now(),
	}
	if _, err := comm.Db.InsertOne(pv1); err != nil {
		t.Fatalf("insert pv1: %v", err)
	}
	if _, err := comm.Db.InsertOne(pv2); err != nil {
		t.Fatalf("insert pv2: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", pipe.Id)
	m.Set("q", "abc")
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("len(resp) = %d, want 1", len(resp))
	}
	if resp[0]["name"] != "abc123" {
		t.Errorf("sha = %q, want %q", resp[0]["name"], "abc123")
	}
}

// TestPipelineController_searchSha_EmptyID tests searching with empty ID
func TestPipelineController_searchSha_EmptyID(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestPipelineController_getPipelines_Success tests listing user's pipelines
func TestPipelineController_getPipelines_Success(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	createTestPipeline(t, user.Id, "pipeline1", "Pipeline 1")
	createTestPipeline(t, user.Id, "pipeline2", "Pipeline 2")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// TestPipelineController_getPipelines_WithSearch tests listing with search query
func TestPipelineController_getPipelines_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	user := createPipelineTestUser(t)
	createTestPipeline(t, user.Id, "frontend-app", "Frontend App")
	createTestPipeline(t, user.Id, "backend-api", "Backend API")

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("q", "frontend")
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, user)
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
