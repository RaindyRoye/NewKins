package route

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	"xorm.io/xorm"
)

func setupPipelineTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	db := setupRuntimeTestDB(t)

	// Create t_pipeline_conf table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline_conf (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		pipeline_id VARCHAR(64),
		url VARCHAR(255),
		access_token VARCHAR(255),
		yml_content TEXT,
		username VARCHAR(255)
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_conf table: %v", err)
	}

	// Create t_pipeline_var table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline_var (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		pipeline_id VARCHAR(64),
		name VARCHAR(255),
		value TEXT,
		remarks VARCHAR(255),
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_var table: %v", err)
	}

	// Create t_org_pipe table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_org_pipe (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		created DATETIME,
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_org_pipe table: %v", err)
	}

	// Create t_pipeline_version table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline_version (
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
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_version table: %v", err)
	}

	// Create t_org table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_org (
		id VARCHAR(64) PRIMARY KEY,
		aid BIGINT,
		uid VARCHAR(64),
		name VARCHAR(100),
		"desc" VARCHAR(500),
		avatar VARCHAR(500),
		public INT DEFAULT 0,
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_org table: %v", err)
	}

	// Add created column to t_pipeline (used by TPipelineInfo)
	_, err = db.Exec(`ALTER TABLE t_pipeline ADD COLUMN created DATETIME`)
	if err != nil {
		// column may already exist - ignore
		_ = err
	}

	return db
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

func insertTestPipelineConf(t *testing.T, pipelineId string) {
	t.Helper()
	tpc := &model.TPipelineConf{
		PipelineId:  pipelineId,
		Url:         "https://github.com/test/repo",
		AccessToken: "test-token",
		YmlContent:  "stages: []",
		Username:    "testuser",
	}
	_, err := comm.Db.InsertOne(tpc)
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}
}

// --- copy tests ---

func TestPipelineController_copy_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.copy(c, &hbtp.Map{"pipelineId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_copy_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.copy(c, &hbtp.Map{"pipelineId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_copy_Success(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-1", "admin", "original-pipe")
	insertTestPipelineConf(t, "pipe-1")

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.copy(c, &hbtp.Map{"pipelineId": "pipe-1"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["id"] == nil {
		t.Error("expected new pipeline id in response")
	}
	if resp["name"] == nil || resp["name"] != "original-pipe_copy" {
		t.Errorf("expected name 'original-pipe_copy', got %v", resp["name"])
	}
}

// --- searchSha tests ---

func TestPipelineController_searchSha_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.searchSha(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_searchSha_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.searchSha(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_searchSha_Success(t *testing.T) {
	db := setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-1", "admin", "test-pipe")

	// Insert pipeline versions with SHAs
	for _, sha := range []string{"abc123", "def456", "ghi789"} {
		_, err := db.Insert(&model.TPipelineVersion{
			Id:         "pv-" + sha,
			PipelineId: "pipe-1",
			Sha:        sha,
			Created:    time.Now(),
		})
		if err != nil {
			t.Fatalf("insert pipeline version: %v", err)
		}
	}

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.searchSha(c, &hbtp.Map{"id": "pipe-1"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) != 3 {
		t.Errorf("expected 3 SHAs, got %d", len(resp))
	}
}

func TestPipelineController_searchSha_WithQuery(t *testing.T) {
	db := setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-1", "admin", "test-pipe")

	for _, sha := range []string{"abc123", "def456", "abc789"} {
		_, err := db.Insert(&model.TPipelineVersion{
			Id:         "pv-" + sha,
			PipelineId: "pipe-1",
			Sha:        sha,
			Created:    time.Now(),
		})
		if err != nil {
			t.Fatalf("insert pipeline version: %v", err)
		}
	}

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.searchSha(c, &hbtp.Map{"id": "pipe-1", "q": "abc"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 SHAs matching 'abc', got %d", len(resp))
	}
}

// --- vars tests ---

func TestPipelineController_vars_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.vars(c, &hbtp.Map{"pipelineId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_vars_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.vars(c, &hbtp.Map{"pipelineId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_vars_Success(t *testing.T) {
	db := setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-1", "admin", "test-pipe")

	// Insert vars
	for _, v := range []struct{ name, value string }{
		{"VAR1", "val1"}, {"VAR2", "val2"},
	} {
		_, err := db.Insert(&model.TPipelineVar{
			PipelineId: "pipe-1",
			Name:       v.name,
			Value:      v.value,
			Uid:        "admin",
		})
		if err != nil {
			t.Fatalf("insert var: %v", err)
		}
	}

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.vars(c, &hbtp.Map{"pipelineId": "pipe-1"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_vars_WithQuery(t *testing.T) {
	db := setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-1", "admin", "test-pipe")

	for _, v := range []struct{ name, value string }{
		{"DB_HOST", "localhost"}, {"DB_PORT", "5432"}, {"APP_NAME", "myapp"},
	} {
		_, err := db.Insert(&model.TPipelineVar{
			PipelineId: "pipe-1",
			Name:       v.name,
			Value:      v.value,
			Uid:        "admin",
		})
		if err != nil {
			t.Fatalf("insert var: %v", err)
		}
	}

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.vars(c, &hbtp.Map{"pipelineId": "pipe-1", "q": "DB"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- varSave tests ---

func TestPipelineController_varSave_EmptyParams(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.varSave(c, &bean.PipelineVar{PipelineId: "", Name: "", Value: ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varSave_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "nonexistent",
		Name:       "VAR1",
		Value:      "val1",
	})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_varSave_CreateNew(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-1", "admin", "test-pipe")

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "pipe-1",
		Name:       "NEW_VAR",
		Value:      "new_value",
		Public:     true,
	})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify var was created
	pv := &model.TPipelineVar{}
	ok, err := comm.Db.Where("pipeline_id = ? AND name = ?", "pipe-1", "NEW_VAR").Get(pv)
	if err != nil {
		t.Fatalf("query var: %v", err)
	}
	if !ok {
		t.Error("expected var to be created")
	}
	if pv.Public != 1 {
		t.Errorf("public = %d, want 1", pv.Public)
	}
}

func TestPipelineController_varSave_DuplicateName(t *testing.T) {
	db := setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-1", "admin", "test-pipe")

	// Insert existing var
	_, err := db.Insert(&model.TPipelineVar{
		PipelineId: "pipe-1",
		Name:       "EXISTING",
		Value:      "old_value",
		Uid:        "admin",
	})
	if err != nil {
		t.Fatalf("insert var: %v", err)
	}

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.varSave(c, &bean.PipelineVar{
		PipelineId: "pipe-1",
		Name:       "EXISTING",
		Value:      "new_value",
	})

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

// --- varDel tests ---

func TestPipelineController_varDel_InvalidAid(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.varDel(c, &hbtp.Map{"aid": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varDel_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.varDel(c, &hbtp.Map{"aid": int64(9999)})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_varDel_Success(t *testing.T) {
	db := setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-1", "admin", "test-pipe")

	_, err := db.Insert(&model.TPipelineVar{
		PipelineId: "pipe-1",
		Name:       "DEL_VAR",
		Value:      "del_value",
		Uid:        "admin",
	})
	if err != nil {
		t.Fatalf("insert var: %v", err)
	}

	// Get the aid
	pv := &model.TPipelineVar{}
	ok, err := db.Where("pipeline_id = ? AND name = ?", "pipe-1", "DEL_VAR").Get(pv)
	if err != nil || !ok {
		t.Fatalf("get var: ok=%v err=%v", ok, err)
	}

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.varDel(c, &hbtp.Map{"aid": pv.Aid})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify deleted
	count, err := db.Where("aid = ?", pv.Aid).Count(&model.TPipelineVar{})
	if err != nil {
		t.Fatalf("count vars: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 vars after delete, got %d", count)
	}
}

// --- pipelineVersion tests ---

func TestPipelineController_pipelineVersion_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.pipelineVersion(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_pipelineVersion_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.pipelineVersion(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- pipelineVersions tests ---

func TestPipelineController_pipelineVersions_WithPipelineId_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.pipelineVersions(c, &hbtp.Map{"pipelineId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_pipelineVersions_AdminNoPipelineId(t *testing.T) {
	db := setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Insert pipeline versions for admin
	for i := 0; i < 3; i++ {
		_, err := db.Insert(&model.TPipelineVersion{
			Id:         "pv-" + time.Now().Format("150405") + "-" + string(rune('a'+i)),
			PipelineId: "pipe-any",
			Created:    time.Now(),
		})
		if err != nil {
			t.Fatalf("insert pv: %v", err)
		}
	}

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.pipelineVersions(c, &hbtp.Map{"pipelineId": "", "page": int64(1)})

	// Admin with no pipelineId sees all non-deleted versions
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- fillPipelineListBuildInfo tests ---

func TestFillPipelineListBuildInfo_EmptyList(t *testing.T) {
	err := fillPipelineListBuildInfo(context.TODO(), nil)
	if err != nil {
		t.Errorf("expected nil error for empty list, got %v", err)
	}
}

func TestFillPipelineListBuildInfo_NoPipelines(t *testing.T) {
	err := fillPipelineListBuildInfo(context.TODO(), []*model.TPipeline{})
	if err != nil {
		t.Errorf("expected nil error for empty pipelines, got %v", err)
	}
}

// --- new pipeline tests ---

func TestPipelineController_new_InvalidCheck(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.new(c, &bean.NewPipeline{Name: "", Content: ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_new_InvalidYaml(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.new(c, &bean.NewPipeline{Name: "test", Content: ":::invalid yaml"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- delete pipeline tests ---

func TestPipelineController_delete_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.delete(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_delete_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.delete(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_delete_Success(t *testing.T) {
	db := setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-del", "admin", "to-delete")

	// Insert a pipeline version to be deleted along with the pipeline
	_, err := db.Insert(&model.TPipelineVersion{
		Id:         "pv-del",
		PipelineId: "pipe-del",
		Created:    time.Now(),
	})
	if err != nil {
		t.Fatalf("insert pv: %v", err)
	}

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.delete(c, &hbtp.Map{"id": "pipe-del"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify pipeline is marked deleted
	p := &model.TPipeline{}
	ok, err := comm.Db.Where("id = ?", "pipe-del").Get(p)
	if err != nil {
		t.Fatalf("query pipeline: %v", err)
	}
	if !ok {
		t.Fatal("pipeline not found")
	}
	if p.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", p.Deleted)
	}

	// Verify pipeline version is also marked deleted
	pv := &model.TPipelineVersion{}
	ok, err = comm.Db.Where("id = ?", "pv-del").Get(pv)
	if err != nil {
		t.Fatalf("query pv: %v", err)
	}
	if !ok {
		t.Fatal("pv not found")
	}
	if pv.Deleted != 1 {
		t.Errorf("pv deleted = %d, want 1", pv.Deleted)
	}
}

// --- info pipeline tests ---

func TestPipelineController_info_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.info(c, &hbtp.Map{"id": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_info_NotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.info(c, &hbtp.Map{"id": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_info_Success(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-1", "admin", "test-pipe")
	insertTestPipelineConf(t, "pipe-1")

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.info(c, &hbtp.Map{"id": "pipe-1"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["pipe"] == nil {
		t.Error("expected 'pipe' in response")
	}
	if resp["perm"] == nil {
		t.Error("expected 'perm' in response")
	}
}

// --- save pipeline tests ---

func TestPipelineController_save_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.save(c, &hbtp.Map{"pipelineId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_save_InvalidYaml(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-1", "admin", "test-pipe")

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.save(c, &hbtp.Map{
		"pipelineId": "pipe-1",
		"content":    ":::invalid yaml",
		"name":       "updated",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- GetPath test ---

// --- run pipeline tests ---

func TestPipelineController_run_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.run(c, &hbtp.Map{"pipelineId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_run_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.run(c, &hbtp.Map{"pipelineId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- rebuild pipeline tests ---

func TestPipelineController_rebuild_EmptyPvId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.rebuild(c, &hbtp.Map{"pipelineVersionId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_rebuild_PvNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.rebuild(c, &hbtp.Map{"pipelineVersionId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- orgPipelines tests ---

func TestPipelineController_orgPipelines_EmptyOrgId(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.orgPipelines(c, &hbtp.Map{"orgId": ""})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_orgPipelines_OrgNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.orgPipelines(c, &hbtp.Map{"orgId": "nonexistent"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- getPipelines tests ---

func TestPipelineController_getPipelines_AdminEmpty(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.getPipelines(c, &hbtp.Map{"page": int64(1)})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_getPipelines_WithQuery(t *testing.T) {
	setupPipelineTestDB(t)
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	insertTestPipeline(t, "pipe-1", "admin", "alpha-pipe")
	insertTestPipeline(t, "pipe-2", "other", "beta-pipe")

	c, w := makeRuntimeGinContext(t, nil, admin)
	ctrl := PipelineController{}
	ctrl.getPipelines(c, &hbtp.Map{"page": int64(1), "q": "alpha"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- non-admin user tests ---

func TestPipelineController_copy_NonAdminDenied(t *testing.T) {
	db := setupPipelineTestDB(t)

	user := &model.TUser{Id: "user1", Name: "user1", Active: 1}
	_, err := db.Insert(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	insertTestPipeline(t, "pipe-1", "user1", "test-pipe")
	insertTestPipelineConf(t, "pipe-1")

	c, w := makeRuntimeGinContext(t, nil, user)
	ctrl := PipelineController{}
	ctrl.copy(c, &hbtp.Map{"pipelineId": "pipe-1"})

	// Non-admin without PermPipe should get 403
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

// --- LgUserKey constant check ---

func TestLgUserKeyIsSet(t *testing.T) {
	if service.LgUserKey == "" {
		t.Error("LgUserKey should not be empty")
	}
}
