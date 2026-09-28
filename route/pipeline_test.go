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
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			name VARCHAR(200),
			desc TEXT,
			public INT DEFAULT 0,
			created DATETIME,
			updated DATETIME,
			deleted INT DEFAULT 0,
			deleted_time DATETIME
		)`,
		`CREATE TABLE t_user_org (
			aid BIGINT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			created DATETIME,
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0
		)`,
		`CREATE TABLE t_pipeline (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			uid VARCHAR(64),
			name VARCHAR(255),
			display_name VARCHAR(255),
			pipeline_type VARCHAR(255),
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			created DATETIME,
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
		`CREATE TABLE t_pipeline_var (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			pipeline_id VARCHAR(64),
			name VARCHAR(255),
			value TEXT,
			remarks VARCHAR(255),
			public INT DEFAULT 0
		)`,
		`CREATE TABLE t_org_pipe (
			aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64),
			created DATETIME,
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

func createPipelineTestUser(t *testing.T, name, nick string, isAdmin bool) *model.TUser {
	t.Helper()
	usr := &model.TUser{
		Id:        utils.NewXid(),
		Name:      name,
		Nick:      nick,
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	if isAdmin {
		usr.Id = "admin"
		usr.Name = "admin"
	}
	if _, err := comm.Db.InsertOne(usr); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return usr
}

func createPipelineTestOrg(t *testing.T, owner *model.TUser, name string) *model.TOrg {
	t.Helper()
	org := &model.TOrg{
		Id:      utils.NewXid(),
		Uid:     owner.Id,
		Name:    name,
		Public:  1,
		Created: time.Now(),
		Updated: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("create org: %v", err)
	}
	// Add owner as admin
	uo := &model.TUserOrg{
		Uid:     owner.Id,
		OrgId:   org.Id,
		PermAdm: 1,
		Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(uo); err != nil {
		t.Fatalf("create user org: %v", err)
	}
	return org
}

func createPipelineTestPipeline(t *testing.T, owner *model.TUser, name string, org *model.TOrg) *model.TPipeline {
	t.Helper()
	pipe := &model.TPipeline{
		Id:          utils.NewXid(),
		Uid:         owner.Id,
		Name:        name,
		DisplayName: name,
		CreateTime:  time.Now(),
	}
	if _, err := comm.Db.InsertOne(pipe); err != nil {
		t.Fatalf("create pipeline: %v", err)
	}
	conf := &model.TPipelineConf{
		PipelineId:  pipe.Id,
		YmlContent:  "stages: []",
		Url:         "https://github.com/test/repo",
		Username:    "testuser",
		AccessToken: "testtoken",
	}
	if _, err := comm.Db.InsertOne(conf); err != nil {
		t.Fatalf("create pipeline conf: %v", err)
	}
	if org != nil {
		op := &model.TOrgPipe{
			OrgId:   org.Id,
			PipeId:  pipe.Id,
			Created: time.Now(),
		}
		if _, err := comm.Db.InsertOne(op); err != nil {
			t.Fatalf("create org pipe: %v", err)
		}
	}
	return pipe
}

// ===== GetPath and Routes =====

func TestPipelineController_GetPathFromPipeline(t *testing.T) {
	c := &PipelineController{}
	if got := c.GetPath(); got != "/api/pipeline" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/pipeline")
	}
}

func TestPipelineController_Routes(t *testing.T) {
	setupPipelineTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	pc := &PipelineController{}
	pc.Routes(r.Group("/api/pipeline"))
	// Routes registered successfully
}

// ===== orgPipelines =====

func TestPipelineController_orgPipelines_EmptyOrgId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_orgPipelines_NotFoundOrg(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_orgPipelines_Success(t *testing.T) {
	setupPipelineTestDB(t)
	admin := createPipelineTestUser(t, "admin", "Admin", true)
	org := createPipelineTestOrg(t, admin, "test-org")
	createPipelineTestPipeline(t, admin, "pipe1", org)
	createPipelineTestPipeline(t, admin, "pipe2", org)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("orgId", org.Id)
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
}

// ===== getPipelines =====

func TestPipelineController_getPipelines_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	admin := createPipelineTestUser(t, "admin", "Admin", true)
	createPipelineTestPipeline(t, admin, "pipe1", nil)
	createPipelineTestPipeline(t, admin, "pipe2", nil)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
}

func TestPipelineController_getPipelines_WithSearch(t *testing.T) {
	setupPipelineTestDB(t)
	admin := createPipelineTestUser(t, "admin", "Admin", true)
	createPipelineTestPipeline(t, admin, "alpha-pipe", nil)
	createPipelineTestPipeline(t, admin, "beta-pipe", nil)
	createPipelineTestPipeline(t, admin, "gamma", nil)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("q", "pipe")
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("total = %d, want 2 (alpha-pipe + beta-pipe)", resp.Total)
	}
}

// ===== delete =====

func TestPipelineController_delete_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_delete_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_delete_Success(t *testing.T) {
	setupPipelineTestDB(t)
	admin := createPipelineTestUser(t, "admin", "Admin", true)
	pipe := createPipelineTestPipeline(t, admin, "to-delete", nil)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", pipe.Id)
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify soft delete
	p := &model.TPipeline{}
	ok, err := comm.Db.Where("id=?", pipe.Id).Get(p)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok || p.Deleted != 1 {
		t.Error("pipeline not marked as deleted")
	}
}

// ===== info =====

func TestPipelineController_info_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_info_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_info_Success(t *testing.T) {
	setupPipelineTestDB(t)
	admin := createPipelineTestUser(t, "admin", "Admin", true)
	pipe := createPipelineTestPipeline(t, admin, "test-pipe", nil)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", pipe.Id)
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["pipe"] == nil {
		t.Error("expected pipe in response")
	}
	if resp["perm"] == nil {
		t.Error("expected perm in response")
	}
}

// ===== save =====

func TestPipelineController_save_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_save_InvalidYaml(t *testing.T) {
	setupPipelineTestDB(t)
	admin := createPipelineTestUser(t, "admin", "Admin", true)
	pipe := createPipelineTestPipeline(t, admin, "test-pipe", nil)

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", pipe.Id)
	m.Set("name", "updated-name")
	m.Set("content", "invalid: yaml: content:")
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.save(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

// ===== vars =====

func TestPipelineController_vars_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_vars_NotFoundPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_vars_Success(t *testing.T) {
	setupPipelineTestDB(t)
	admin := createPipelineTestUser(t, "admin", "Admin", true)
	pipe := createPipelineTestPipeline(t, admin, "test-pipe", nil)

	// Add a variable
	pv := &model.TPipelineVar{
		Uid:        admin.Id,
		PipelineId: pipe.Id,
		Name:       "TEST_VAR",
		Value:      "test_value",
		Public:     1,
	}
	if _, err := comm.Db.InsertOne(pv); err != nil {
		t.Fatalf("insert var: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", pipe.Id)
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

// ===== varSave =====

func TestPipelineController_varSave_EmptyParams(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		Name:       "",
		Value:      "val",
		PipelineId: "pipe-id",
	}
	c, w := makePipelineGinCtx(t, pv, createPipelineTestUser(t, "user1", "User", false))
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varSave_NotFoundPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		Name:       "VAR_NAME",
		Value:      "var_value",
		PipelineId: "nonexistent",
	}
	c, w := makePipelineGinCtx(t, pv, createPipelineTestUser(t, "user1", "User", false))
	ctrl.varSave(c, pv)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_varSave_Success(t *testing.T) {
	setupPipelineTestDB(t)
	admin := createPipelineTestUser(t, "admin", "Admin", true)
	pipe := createPipelineTestPipeline(t, admin, "test-pipe", nil)

	ctrl := PipelineController{}
	pv := &bean.PipelineVar{
		Name:       "NEW_VAR",
		Value:      "new_value",
		PipelineId: pipe.Id,
		Public:     true,
	}
	c, w := makePipelineGinCtx(t, pv, admin)
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify variable was created
	tpv := &model.TPipelineVar{}
	ok, err := comm.Db.Where("pipeline_id=? and name=?", pipe.Id, "NEW_VAR").Get(tpv)
	if err != nil {
		t.Fatalf("query var: %v", err)
	}
	if !ok {
		t.Error("variable not created")
	}
}

// ===== varDel =====

func TestPipelineController_varDel_InvalidAid(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varDel_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(99999))
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_varDel_Success(t *testing.T) {
	setupPipelineTestDB(t)
	admin := createPipelineTestUser(t, "admin", "Admin", true)
	pipe := createPipelineTestPipeline(t, admin, "test-pipe", nil)

	// Add a variable to delete
	pv := &model.TPipelineVar{
		Uid:        admin.Id,
		PipelineId: pipe.Id,
		Name:       "TO_DELETE",
		Value:      "delete_me",
		Public:     0,
	}
	if _, err := comm.Db.InsertOne(pv); err != nil {
		t.Fatalf("insert var: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("aid", int64(pv.Aid))
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}

	// Verify deletion
	pv2 := &model.TPipelineVar{}
	ok, err := comm.Db.Where("aid=?", pv.Aid).Get(pv2)
	if err != nil {
		t.Fatalf("query var: %v", err)
	}
	if ok {
		t.Error("variable not deleted")
	}
}

// ===== searchSha =====

func TestPipelineController_searchSha_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_searchSha_NotFoundPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.searchSha(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_searchSha_Success(t *testing.T) {
	setupPipelineTestDB(t)
	admin := createPipelineTestUser(t, "admin", "Admin", true)
	pipe := createPipelineTestPipeline(t, admin, "test-pipe", nil)

	// Add a pipeline version with SHA
	pv := &model.TPipelineVersion{
		Id:         utils.NewXid(),
		PipelineId: pipe.Id,
		Sha:        "abc123def456",
		Created:    time.Now(),
	}
	if _, err := comm.Db.InsertOne(pv); err != nil {
		t.Fatalf("insert version: %v", err)
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", pipe.Id)
	m.Set("q", "abc")
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("len(resp) = %d, want 1", len(resp))
	}
}

// ===== run =====

func TestPipelineController_run_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.run(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ===== copy =====

func TestPipelineController_copy_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.copy(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_copy_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.copy(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ===== rebuild =====

func TestPipelineController_rebuild_EmptyPipelineVersionId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.rebuild(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_rebuild_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "nonexistent")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.rebuild(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ===== pipelineVersion =====

func TestPipelineController_pipelineVersion_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_pipelineVersion_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ===== pipelineVersions =====

func TestPipelineController_pipelineVersions_Success(t *testing.T) {
	setupPipelineTestDB(t)
	admin := createPipelineTestUser(t, "admin", "Admin", true)
	pipe := createPipelineTestPipeline(t, admin, "test-pipe", nil)

	// Add versions
	for i := 0; i < 3; i++ {
		pv := &model.TPipelineVersion{
			Id:         utils.NewXid(),
			PipelineId: pipe.Id,
			Number:     int64(i + 1),
			Created:    time.Now(),
		}
		if _, err := comm.Db.InsertOne(pv); err != nil {
			t.Fatalf("insert version: %v", err)
		}
	}

	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", pipe.Id)
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, admin)
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
}

func TestPipelineController_pipelineVersions_NotFoundPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	m.Set("page", int64(1))
	c, w := makePipelineGinCtx(t, m, createPipelineTestUser(t, "user1", "User", false))
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
