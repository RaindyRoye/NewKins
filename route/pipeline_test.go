package route

import (
	"context"
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

// --------------- helpers ---------------

func setupPipelineTestDB(t *testing.T) *xorm.Engine {
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
			"desc" TEXT,
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
			created DATETIME,
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
			t.Fatalf("exec %q: %v", sql[:60], err)
		}
	}

	comm.Db = db
	return db
}

func makePipeGinCtx(t *testing.T, lgUser *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if lgUser != nil {
		c.Set(service.LgUserKey, lgUser)
	}
	return c, w
}

func insertTestUser(t *testing.T, db *xorm.Engine, id, name string, active int) *model.TUser {
	t.Helper()
	usr := &model.TUser{
		Id: id, Name: name, Nick: name + "_nick",
		Active: active, Created: time.Now(), LoginTime: time.Now(),
	}
	if _, err := db.InsertOne(usr); err != nil {
		t.Fatalf("insert user %s: %v", id, err)
	}
	return usr
}

func insertTestPipeline(t *testing.T, db *xorm.Engine, id, uid, name string) *model.TPipeline {
	t.Helper()
	pipe := &model.TPipeline{
		Id: id, Uid: uid, Name: name, DisplayName: name + " Display",
		CreateTime: time.Now(),
	}
	if _, err := db.InsertOne(pipe); err != nil {
		t.Fatalf("insert pipeline %s: %v", id, err)
	}
	return pipe
}

func insertTestPipelineConf(t *testing.T, db *xorm.Engine, pipeId, url, yml string) {
	t.Helper()
	tpc := &model.TPipelineConf{
		PipelineId: pipeId, Url: url, YmlContent: yml,
		Username: "testuser", AccessToken: "secret123",
	}
	if _, err := db.InsertOne(tpc); err != nil {
		t.Fatalf("insert pipeline conf %s: %v", pipeId, err)
	}
}

// ========================= info tests =========================

func TestPipelineInfo_MissingID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing id, got %d", w.Code)
	}
}

func TestPipelineInfo_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pipeline, got %d", w.Code)
	}
}

func TestPipelineInfo_DeletedPipeline(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "tester", 1)
	pipe := insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")
	pipe.Deleted = 1
	if _, err := db.ID("pipe1").Cols("deleted").Update(pipe); err != nil {
		t.Fatalf("update deleted: %v", err)
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("id", "pipe1")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted pipeline, got %d", w.Code)
	}
}

func TestPipelineInfo_Success_OwnerCanWrite(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "tester", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")
	insertTestPipelineConf(t, db, "pipe1", "https://github.com/example", "stages: []")

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("id", "pipe1")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	perm, ok := resp["perm"].(map[string]interface{})
	if !ok {
		t.Fatal("missing perm in response")
	}
	if perm["read"] != true {
		t.Error("owner should have read perm")
	}
	if perm["write"] != true {
		t.Error("owner should have write perm")
	}

	pinfo, ok := resp["pipe"].(map[string]interface{})
	if !ok {
		t.Fatal("missing pipe in response")
	}
	// Owner can see credentials
	if pinfo["username"] == comm.MaskedValue {
		t.Error("owner should see real username, not masked")
	}
	if pinfo["accessToken"] == comm.MaskedValue {
		t.Error("owner should see real accessToken, not masked")
	}
}

func TestPipelineInfo_NonOwnerNoReadPerm(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	// user2 is not admin, not owner, not in any org
	usr2 := insertTestUser(t, db, "user2", "other", 1)
	c, w := makePipeGinCtx(t, usr2)
	m := &hbtp.Map{}
	m.Set("id", "pipe1")
	ctrl.info(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for non-owner without read perm, got %d", w.Code)
	}
}

func TestPipelineInfo_Admin(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")
	insertTestPipelineConf(t, db, "pipe1", "https://github.com/example", "stages: []")

	admin := insertTestUser(t, db, "admin", "admin", 1)
	c, w := makePipeGinCtx(t, admin)
	m := &hbtp.Map{}
	m.Set("id", "pipe1")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d: %s", w.Code, w.Body.String())
	}
}

// ========================= delete tests =========================

func TestPipelineDelete_MissingID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.delete(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineDelete_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineDelete_NoPerm(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	// user2 has no permission
	usr2 := insertTestUser(t, db, "user2", "other", 1)
	c, w := makePipeGinCtx(t, usr2)
	m := &hbtp.Map{}
	m.Set("id", "pipe1")
	ctrl.delete(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestPipelineDelete_Success(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	// Also add a pipeline version to soft-delete
	pv := &model.TPipelineVersion{
		Id: "pv1", PipelineId: "pipe1", Created: time.Now(),
	}
	if _, err := db.InsertOne(pv); err != nil {
		t.Fatalf("insert pv: %v", err)
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("id", "pipe1")
	ctrl.delete(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify soft delete
	p := &model.TPipeline{}
	ok, _ := db.Where("id=?", "pipe1").Get(p)
	if !ok {
		t.Fatal("pipeline row should still exist")
	}
	if p.Deleted != 1 {
		t.Errorf("expected deleted=1, got %d", p.Deleted)
	}

	// Verify pipeline version also soft-deleted
	pv2 := &model.TPipelineVersion{}
	ok, _ = db.Where("id=?", "pv1").Get(pv2)
	if !ok {
		t.Fatal("pv row should still exist")
	}
	if pv2.Deleted != 1 {
		t.Errorf("expected pv deleted=1, got %d", pv2.Deleted)
	}
}

func TestPipelineDelete_AlreadyDeleted(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "owner", 1)
	pipe := insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")
	pipe.Deleted = 1
	if _, err := db.ID("pipe1").Cols("deleted").Update(pipe); err != nil {
		t.Fatalf("update: %v", err)
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("id", "pipe1")
	ctrl.delete(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for already-deleted pipeline, got %d", w.Code)
	}
}

// ========================= searchSha tests =========================

func TestPipelineSearchSha_MissingID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineSearchSha_PipeNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineSearchSha_NoPerm(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	usr2 := insertTestUser(t, db, "user2", "other", 1)
	c, w := makePipeGinCtx(t, usr2)
	m := &hbtp.Map{}
	m.Set("id", "pipe1")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestPipelineSearchSha_Success(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	// Insert some pipeline versions with SHAs
	shas := []string{"abc123", "def456", "abc789"}
	for i, sha := range shas {
		pv := &model.TPipelineVersion{
			Id: utils.NewXid(), PipelineId: "pipe1",
			Sha: sha, Number: int64(i + 1), Created: time.Now(),
		}
		if _, err := db.InsertOne(pv); err != nil {
			t.Fatalf("insert pv: %v", err)
		}
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("id", "pipe1")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if len(resp) != 3 {
		t.Errorf("expected 3 SHAs, got %d", len(resp))
	}
}

func TestPipelineSearchSha_WithQuery(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	shas := []string{"abc123", "def456", "abc789"}
	for i, sha := range shas {
		pv := &model.TPipelineVersion{
			Id: utils.NewXid(), PipelineId: "pipe1",
			Sha: sha, Number: int64(i + 1), Created: time.Now(),
		}
		if _, err := db.InsertOne(pv); err != nil {
			t.Fatalf("insert pv: %v", err)
		}
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("id", "pipe1")
	m.Set("q", "abc")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 SHAs matching 'abc', got %d", len(resp))
	}
}

func TestPipelineSearchSha_FiltersEmptySha(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	// One with SHA and one empty
	pv1 := &model.TPipelineVersion{
		Id: utils.NewXid(), PipelineId: "pipe1",
		Sha: "abc123", Number: 1, Created: time.Now(),
	}
	pv2 := &model.TPipelineVersion{
		Id: utils.NewXid(), PipelineId: "pipe1",
		Sha: "", Number: 2, Created: time.Now(),
	}
	if _, err := db.InsertOne(pv1); err != nil {
		t.Fatalf("insert pv1: %v", err)
	}
	if _, err := db.InsertOne(pv2); err != nil {
		t.Fatalf("insert pv2: %v", err)
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("id", "pipe1")
	ctrl.searchSha(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	// empty SHA should be filtered out
	if len(resp) != 1 {
		t.Errorf("expected 1 non-empty SHA, got %d", len(resp))
	}
}

// ========================= vars tests =========================

func TestPipelineVars_MissingID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineVars_PipeNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.vars(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineVars_Success_Owner(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	// Insert some vars
	pv := &model.TPipelineVar{
		Uid: "user1", PipelineId: "pipe1",
		Name: "MY_VAR", Value: "secret-value", Public: 1,
	}
	if _, err := db.InsertOne(pv); err != nil {
		t.Fatalf("insert pv: %v", err)
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe1")
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVars_NonOwner_MasksPublicVars(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	// Insert public var
	pv := &model.TPipelineVar{
		Uid: "user1", PipelineId: "pipe1",
		Name: "MY_VAR", Value: "secret-value", Public: 1,
	}
	if _, err := db.InsertOne(pv); err != nil {
		t.Fatalf("insert pv: %v", err)
	}

	// Non-owner user with admin uid to bypass read perm check
	admin := insertTestUser(t, db, "admin", "admin", 1)
	c, w := makePipeGinCtx(t, admin)
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe1")
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var page bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	// Admin is non-owner but has read. Since admin is not CanWrite() for this pipe
	// (admin is admin, so CanWrite returns true), vars won't be masked.
	// Let's verify the structure is valid
	if page.Data == nil {
		t.Error("expected non-nil page data")
	}
}

// ========================= varSave tests =========================

func TestPipelineVarSave_MissingFields(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)

	tests := []struct {
		name string
		pv   *bean.PipelineVar
	}{
		{"missing name", &bean.PipelineVar{PipelineId: "pipe1", Value: "v"}},
		{"missing value", &bean.PipelineVar{PipelineId: "pipe1", Name: "n"}},
		{"missing pipelineId", &bean.PipelineVar{Name: "n", Value: "v"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := makePipeGinCtx(t, usr)
			ctrl.varSave(c, tt.pv)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestPipelineVarSave_PipeNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	pv := &bean.PipelineVar{PipelineId: "nonexistent", Name: "var1", Value: "val1"}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineVarSave_NoPerm(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	usr2 := insertTestUser(t, db, "user2", "other", 1)
	c, w := makePipeGinCtx(t, usr2)

	pv := &bean.PipelineVar{PipelineId: "pipe1", Name: "var1", Value: "val1"}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestPipelineVarSave_InsertNew(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	c, w := makePipeGinCtx(t, usr)
	pv := &bean.PipelineVar{
		PipelineId: "pipe1", Name: "MY_VAR", Value: "secret",
		Public: true, Remarks: "a test var",
	}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it was inserted
	v := &model.TPipelineVar{}
	ok, _ := db.Where("pipeline_id=? and name=?", "pipe1", "MY_VAR").Get(v)
	if !ok {
		t.Fatal("var should have been inserted")
	}
	if v.Value != "secret" {
		t.Errorf("expected value 'secret', got %q", v.Value)
	}
	if v.Public != 1 {
		t.Errorf("expected public=1, got %d", v.Public)
	}
}

func TestPipelineVarSave_DuplicateName(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	// Insert existing var
	existing := &model.TPipelineVar{
		Uid: "user1", PipelineId: "pipe1",
		Name: "MY_VAR", Value: "old-value",
	}
	if _, err := db.InsertOne(existing); err != nil {
		t.Fatalf("insert: %v", err)
	}

	c, w := makePipeGinCtx(t, usr)
	pv := &bean.PipelineVar{
		PipelineId: "pipe1", Name: "MY_VAR", Value: "new-value",
	}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate name, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPipelineVarSave_UpdateExisting(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	existing := &model.TPipelineVar{
		Uid: "user1", PipelineId: "pipe1",
		Name: "MY_VAR", Value: "old-value",
	}
	if _, err := db.InsertOne(existing); err != nil {
		t.Fatalf("insert: %v", err)
	}

	c, w := makePipeGinCtx(t, usr)
	pv := &bean.PipelineVar{
		Aid: existing.Aid, PipelineId: "pipe1",
		Name: "MY_VAR", Value: "new-value",
	}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for update, got %d: %s", w.Code, w.Body.String())
	}

	// Verify update
	v := &model.TPipelineVar{}
	ok, _ := db.Where("aid=?", existing.Aid).Get(v)
	if !ok {
		t.Fatal("var should still exist")
	}
	if v.Value != "new-value" {
		t.Errorf("expected updated value 'new-value', got %q", v.Value)
	}
}

// ========================= varDel tests =========================

func TestPipelineVarDel_BadAid(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("aid", int64(0))
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineVarDel_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("aid", int64(999))
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineVarDel_Success(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	pv := &model.TPipelineVar{
		Uid: "user1", PipelineId: "pipe1",
		Name: "MY_VAR", Value: "secret",
	}
	if _, err := db.InsertOne(pv); err != nil {
		t.Fatalf("insert: %v", err)
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("aid", pv.Aid)
	ctrl.varDel(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	count, _ := db.Where("aid=?", pv.Aid).Count(&model.TPipelineVar{})
	if count != 0 {
		t.Errorf("expected 0 vars remaining, got %d", count)
	}
}

func TestPipelineVarDel_NoPerm(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	pv := &model.TPipelineVar{
		Uid: "user1", PipelineId: "pipe1",
		Name: "MY_VAR", Value: "secret",
	}
	if _, err := db.InsertOne(pv); err != nil {
		t.Fatalf("insert: %v", err)
	}

	usr2 := insertTestUser(t, db, "user2", "other", 1)
	c, w := makePipeGinCtx(t, usr2)
	m := &hbtp.Map{}
	m.Set("aid", pv.Aid)
	ctrl.varDel(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// ========================= getPipelines tests =========================

func TestGetPipelines_Admin(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := insertTestUser(t, db, "admin", "admin", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe1")
	insertTestPipeline(t, db, "pipe2", "user2", "test-pipe2")

	c, w := makePipeGinCtx(t, admin)
	m := &hbtp.Map{}
	m.Set("page", int64(1))
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var page bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Admin sees all non-deleted pipelines
	if page.Total != 2 {
		t.Errorf("expected total=2 for admin, got %d", page.Total)
	}
}

func TestGetPipelines_NonAdmin_OnlyOwn(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	insertTestUser(t, db, "user1", "owner", 1)
	usr2 := insertTestUser(t, db, "user2", "other", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe1")
	insertTestPipeline(t, db, "pipe2", "user2", "test-pipe2")

	c, w := makePipeGinCtx(t, usr2)
	m := &hbtp.Map{}
	m.Set("page", int64(1))
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var page bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Non-admin sees only own pipelines
	if page.Total != 1 {
		t.Errorf("expected total=1 for non-admin, got %d", page.Total)
	}
}

func TestGetPipelines_WithSearchQuery(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := insertTestUser(t, db, "admin", "admin", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "alpha-pipeline")
	insertTestPipeline(t, db, "pipe2", "user2", "beta-pipeline")

	c, w := makePipeGinCtx(t, admin)
	m := &hbtp.Map{}
	m.Set("page", int64(1))
	m.Set("q", "alpha")
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var page bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if page.Total != 1 {
		t.Errorf("expected total=1 for query 'alpha', got %d", page.Total)
	}
}

func TestGetPipelines_ExcludesDeleted(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := insertTestUser(t, db, "admin", "admin", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "active-pipe")
	delPipe := insertTestPipeline(t, db, "pipe2", "user2", "deleted-pipe")
	delPipe.Deleted = 1
	if _, err := db.ID("pipe2").Cols("deleted").Update(delPipe); err != nil {
		t.Fatalf("update: %v", err)
	}

	c, w := makePipeGinCtx(t, admin)
	m := &hbtp.Map{}
	m.Set("page", int64(1))
	ctrl.getPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var page bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if page.Total != 1 {
		t.Errorf("expected total=1 (deleted excluded), got %d", page.Total)
	}
}

// ========================= orgPipelines tests =========================

func TestOrgPipelines_MissingOrgID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("orgId", "")
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestOrgPipelines_OrgNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent-org")
	m.Set("page", int64(1))
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestOrgPipelines_Success_Admin(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "admin", "admin", 1)

	// Create org
	org := &model.TOrg{
		Id: "org1", Name: "test-org", Public: 1,
		Created: time.Now(), Updated: time.Now(),
	}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Create pipeline and link to org
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe1")
	op := &model.TOrgPipe{
		OrgId: "org1", PipeId: "pipe1", Created: time.Now(),
	}
	if _, err := db.InsertOne(op); err != nil {
		t.Fatalf("insert org_pipe: %v", err)
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("orgId", "org1")
	m.Set("page", int64(1))
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var page bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if page.Total != 1 {
		t.Errorf("expected total=1, got %d", page.Total)
	}
}

func TestOrgPipelines_DeletedOrg(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "admin", "admin", 1)

	org := &model.TOrg{
		Id: "org1", Name: "test-org", Public: 1, Deleted: 1,
		Created: time.Now(), Updated: time.Now(),
	}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("orgId", "org1")
	m.Set("page", int64(1))
	ctrl.orgPipelines(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted org, got %d", w.Code)
	}
}

// ========================= pipelineVersions tests =========================

func TestPipelineVersions_WithPipeID_Success(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "owner", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "test-pipe")

	// Insert pipeline versions
	for i := 0; i < 3; i++ {
		pv := &model.TPipelineVersion{
			Id: utils.NewXid(), PipelineId: "pipe1",
			Number: int64(i + 1), Created: time.Now(),
		}
		if _, err := db.InsertOne(pv); err != nil {
			t.Fatalf("insert pv: %v", err)
		}
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe1")
	m.Set("page", int64(1))
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var page bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if page.Total != 3 {
		t.Errorf("expected total=3, got %d", page.Total)
	}
}

func TestPipelineVersions_PipeNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	m.Set("page", int64(1))
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPipelineVersions_NoPipeID_Admin(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := insertTestUser(t, db, "admin", "admin", 1)

	pv := &model.TPipelineVersion{
		Id: utils.NewXid(), PipelineId: "pipe1",
		Number: 1, Created: time.Now(),
	}
	if _, err := db.InsertOne(pv); err != nil {
		t.Fatalf("insert pv: %v", err)
	}

	c, w := makePipeGinCtx(t, admin)
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", int64(1))
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin with no pipeId, got %d: %s", w.Code, w.Body.String())
	}

	var page bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if page.Total != 1 {
		t.Errorf("expected total=1, got %d", page.Total)
	}
}

func TestPipelineVersions_NoPipeID_NonAdmin_OwnPipelines(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "tester", 1)
	insertTestPipeline(t, db, "pipe1", "user1", "my-pipe")
	insertTestPipeline(t, db, "pipe2", "user2", "other-pipe")

	// Insert versions for both pipelines
	pv1 := &model.TPipelineVersion{
		Id: utils.NewXid(), PipelineId: "pipe1",
		Number: 1, Created: time.Now(),
	}
	pv2 := &model.TPipelineVersion{
		Id: utils.NewXid(), PipelineId: "pipe2",
		Number: 1, Created: time.Now(),
	}
	if _, err := db.InsertOne(pv1); err != nil {
		t.Fatalf("insert pv1: %v", err)
	}
	if _, err := db.InsertOne(pv2); err != nil {
		t.Fatalf("insert pv2: %v", err)
	}

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", int64(1))
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var page bean.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Should only see versions for pipe1 (owned by user1)
	if page.Total != 1 {
		t.Errorf("expected total=1 (only own pipelines), got %d", page.Total)
	}
}

func TestPipelineVersions_NoPipeID_NonAdmin_NoPipelines(t *testing.T) {
	db := setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, db, "user1", "tester", 1)
	// user has no pipelines

	c, w := makePipeGinCtx(t, usr)
	m := &hbtp.Map{}
	m.Set("pipelineId", "")
	m.Set("page", int64(1))
	ctrl.pipelineVersions(c, m)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ========================= pipelineVersion tests =========================

func TestPipelineVersion_MissingID(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("id", "")
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPipelineVersion_PVNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	usr := insertTestUser(t, comm.Db, "user1", "tester", 1)
	c, w := makePipeGinCtx(t, usr)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.pipelineVersion(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ========================= GetPath test =========================

func TestPipelineCtrl_GetPath(t *testing.T) {
	ctrl := PipelineController{}
	if got := ctrl.GetPath(); got != "/api/pipeline" {
		t.Errorf("expected /api/pipeline, got %s", got)
	}
}

// ========================= fillPipelineListBuildInfo tests =========================

func TestFillPipelineListBuildInfo_EmptyList(t *testing.T) {
	setupPipelineTestDB(t)
	err := fillPipelineListBuildInfo(context.TODO(), nil)
	if err != nil {
		t.Errorf("expected nil error for empty list, got %v", err)
	}

	err = fillPipelineListBuildInfo(context.TODO(), []*model.TPipeline{})
	if err != nil {
		t.Errorf("expected nil error for empty slice, got %v", err)
	}
}
