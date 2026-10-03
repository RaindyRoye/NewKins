package route

import (
	"net/http"
	"testing"
	"time"

	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

// setupPipelineTestDB creates an in-memory SQLite database with all tables
// needed by PipelineController methods.
func setupPipelineTestDB(t *testing.T) {
	t.Helper()
	db := setupRuntimeTestDB(t)

	// t_pipeline_conf
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline_conf (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		pipeline_id VARCHAR(64),
		url VARCHAR(255),
		access_token VARCHAR(255),
		yml_content TEXT,
		username VARCHAR(255)
	)`)
	if err != nil {
		t.Fatalf("create t_pipeline_conf: %v", err)
	}

	// t_pipeline_var
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
		t.Fatalf("create t_pipeline_var: %v", err)
	}

	// t_pipeline_version
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
		t.Fatalf("create t_pipeline_version: %v", err)
	}

	// t_org
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
		t.Fatalf("create t_org: %v", err)
	}

	// t_org_pipe
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_org_pipe (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		org_id VARCHAR(64),
		pipe_id VARCHAR(64),
		created DATETIME,
		public INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_org_pipe: %v", err)
	}

	// t_pipeline
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_pipeline (
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
		t.Fatalf("create t_pipeline: %v", err)
	}

	// t_artifact
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS t_artifact (
		id VARCHAR(64) PRIMARY KEY,
		uid VARCHAR(64),
		name VARCHAR(255),
		deleted INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_artifact: %v", err)
	}
}

// --- PipelineController.info ---

func TestPipelineController_info_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{}, admin)

	m := &hbtp.Map{}
	ctrl.info(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("info(empty): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_info_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{"id": "nonexistent"}, admin)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.info(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("info(notfound): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_info_DeletedPipeline(t *testing.T) {
	setupPipelineTestDB(t)
	_, err := comm.Db.Insert(&model.TPipeline{
		Id:      "pipe-del",
		Uid:     "admin",
		Name:    "deleted-pipe",
		Deleted: 1,
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", "pipe-del")
	ctrl.info(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("info(deleted): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_info_Success(t *testing.T) {
	setupPipelineTestDB(t)
	_, err := comm.Db.Insert(&model.TPipeline{
		Id:   "pipe-1",
		Uid:  "admin",
		Name: "test-pipe",
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	_, err = comm.Db.Insert(&model.TPipelineConf{
		PipelineId:  "pipe-1",
		YmlContent:  "stages: []",
		Url:         "https://example.com",
		Username:    "user",
		AccessToken: "secret",
	})
	if err != nil {
		t.Fatalf("insert pipeline conf: %v", err)
	}

	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", "pipe-1")
	ctrl.info(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("info(ok): status=%d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- PipelineController.delete ---

func TestPipelineController_delete_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{}, admin)

	m := &hbtp.Map{}
	ctrl.delete(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("delete(empty): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_delete_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.delete(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("delete(notfound): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_delete_Success(t *testing.T) {
	setupPipelineTestDB(t)
	_, err := comm.Db.Insert(&model.TPipeline{
		Id:   "pipe-del-ok",
		Uid:  "admin",
		Name: "to-delete",
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	_, err = comm.Db.Insert(&model.TPipelineVersion{
		Id:         "pv-1",
		PipelineId: "pipe-del-ok",
		Created:    time.Now(),
	})
	if err != nil {
		t.Fatalf("insert pipeline version: %v", err)
	}

	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", "pipe-del-ok")
	ctrl.delete(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("delete(ok): status=%d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify pipeline is marked deleted
	tp := &model.TPipeline{}
	ok, _ := comm.Db.Where("id=?", "pipe-del-ok").Get(tp)
	if !ok {
		t.Fatal("pipeline not found after delete")
	}
	if tp.Deleted != 1 {
		t.Errorf("pipeline.deleted = %d, want 1", tp.Deleted)
	}

	// Verify pipeline version is also marked deleted
	pv := &model.TPipelineVersion{}
	ok, _ = comm.Db.Where("id=?", "pv-1").Get(pv)
	if !ok {
		t.Fatal("pipeline version not found after delete")
	}
	if pv.Deleted != 1 {
		t.Errorf("pipeline_version.deleted = %d, want 1", pv.Deleted)
	}
}

// --- PipelineController.orgPipelines ---

func TestPipelineController_orgPipelines_EmptyOrgId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{}, admin)

	m := &hbtp.Map{}
	ctrl.orgPipelines(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("orgPipelines(empty): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_orgPipelines_OrgNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent-org")
	ctrl.orgPipelines(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("orgPipelines(notfound): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_orgPipelines_NoReadPerm(t *testing.T) {
	setupPipelineTestDB(t)
	// Create org
	_, err := comm.Db.Insert(&model.TOrg{
		Id:   "org-1",
		Uid:  "other-user",
		Name: "test-org",
	})
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}

	ctrl := PipelineController{}
	// Regular user without org membership
	user := &model.TUser{Id: "user-1", Name: "user-1", Active: 1}
	_, err = comm.Db.Insert(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, w := makeRuntimeGinContext(t, nil, user)
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	ctrl.orgPipelines(c, m)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("orgPipelines(noperm): status=%d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- PipelineController.getPipelines ---

func TestPipelineController_getPipelines_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	// Create some pipelines
	for i := 0; i < 3; i++ {
		_, err := comm.Db.Insert(&model.TPipeline{
			Id:   "pipe-admin-" + string(rune('a'+i)),
			Uid:  "admin",
			Name: "admin-pipe-" + string(rune('a'+i)),
		})
		if err != nil {
			t.Fatalf("insert pipeline: %v", err)
		}
	}

	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	// Set user as admin via user_info
	_, err := comm.Db.Insert(&model.TUserInfo{
		Id:       "admin",
		PermUser: 1,
		PermOrg:  1,
		PermPipe: 1,
	})
	if err != nil {
		t.Fatalf("insert user_info: %v", err)
	}

	c, w := makeRuntimeGinContext(t, nil, admin)
	m := &hbtp.Map{}
	ctrl.getPipelines(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("getPipelines(admin): status=%d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_getPipelines_RegularUser(t *testing.T) {
	setupPipelineTestDB(t)
	_, err := comm.Db.Insert(&model.TPipeline{
		Id:   "pipe-user1",
		Uid:  "user-1",
		Name: "user-pipe",
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	ctrl := PipelineController{}
	user := &model.TUser{Id: "user-1", Name: "user-1", Active: 1}
	_, err = comm.Db.Insert(user)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	c, w := makeRuntimeGinContext(t, nil, user)
	m := &hbtp.Map{}
	ctrl.getPipelines(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("getPipelines(user): status=%d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestPipelineController_getPipelines_WithQuery(t *testing.T) {
	setupPipelineTestDB(t)
	_, err := comm.Db.Insert(&model.TPipeline{
		Id:   "pipe-q1",
		Uid:  "admin",
		Name: "searchable-pipe",
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	_, err = comm.Db.Insert(&model.TUserInfo{Id: "admin", PermUser: 1, PermOrg: 1, PermPipe: 1})
	if err != nil {
		t.Fatalf("insert user_info: %v", err)
	}

	c, w := makeRuntimeGinContext(t, nil, admin)
	m := &hbtp.Map{}
	m.Set("q", "searchable")
	ctrl.getPipelines(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("getPipelines(query): status=%d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- PipelineController.vars ---

func TestPipelineController_vars_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{}, admin)

	m := &hbtp.Map{}
	ctrl.vars(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("vars(empty): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_vars_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.vars(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("vars(notfound): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_vars_Success(t *testing.T) {
	setupPipelineTestDB(t)
	_, err := comm.Db.Insert(&model.TPipeline{
		Id:   "pipe-vars",
		Uid:  "admin",
		Name: "vars-pipe",
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	_, err = comm.Db.Insert(&model.TPipelineVar{
		Uid:        "admin",
		PipelineId: "pipe-vars",
		Name:       "VAR_1",
		Value:      "value-1",
		Public:     0,
	})
	if err != nil {
		t.Fatalf("insert pipeline var: %v", err)
	}

	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("pipelineId", "pipe-vars")
	ctrl.vars(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("vars(ok): status=%d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- PipelineController.searchSha ---

func TestPipelineController_searchSha_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{}, admin)

	m := &hbtp.Map{}
	ctrl.searchSha(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("searchSha(empty): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_searchSha_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.searchSha(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("searchSha(notfound): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_searchSha_Success(t *testing.T) {
	setupPipelineTestDB(t)
	_, err := comm.Db.Insert(&model.TPipeline{
		Id:   "pipe-sha",
		Uid:  "admin",
		Name: "sha-pipe",
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	// Insert pipeline versions with SHAs
	for _, sha := range []string{"abc123", "def456", "abc789"} {
		_, err = comm.Db.Insert(&model.TPipelineVersion{
			Id:         "pv-sha-" + sha,
			PipelineId: "pipe-sha",
			Sha:        sha,
			Created:    time.Now(),
		})
		if err != nil {
			t.Fatalf("insert pipeline version: %v", err)
		}
	}

	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", "pipe-sha")
	m.Set("q", "abc")
	ctrl.searchSha(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("searchSha(ok): status=%d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- PipelineController.varDel ---

func TestPipelineController_varDel_InvalidAid(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{}, admin)

	m := &hbtp.Map{}
	ctrl.varDel(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("varDel(invalid): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_varDel_VarNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("aid", int64(99999))
	ctrl.varDel(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("varDel(notfound): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- PipelineController.pipelineVersions ---

func TestPipelineController_pipelineVersions_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.pipelineVersions(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("pipelineVersions(notfound): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestPipelineController_pipelineVersions_EmptyPipelineId_Admin(t *testing.T) {
	setupPipelineTestDB(t)
	// Insert a pipeline and pipeline version for the "all versions" path
	_, err := comm.Db.Insert(&model.TPipeline{
		Id:   "pipe-all",
		Uid:  "admin",
		Name: "all-pipe",
	})
	if err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	_, err = comm.Db.Insert(&model.TPipelineVersion{
		Id:         "pv-all",
		PipelineId: "pipe-all",
		Created:    time.Now(),
	})
	if err != nil {
		t.Fatalf("insert pv: %v", err)
	}

	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	_, err = comm.Db.Insert(&model.TUserInfo{Id: "admin", PermUser: 1, PermOrg: 1, PermPipe: 1})
	if err != nil {
		t.Fatalf("insert user_info: %v", err)
	}

	c, w := makeRuntimeGinContext(t, nil, admin)
	m := &hbtp.Map{}
	ctrl.pipelineVersions(c, m)
	if w.Code != http.StatusOK {
		t.Errorf("pipelineVersions(admin-all): status=%d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

// --- PipelineController.copy ---

func TestPipelineController_copy_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{}, admin)

	m := &hbtp.Map{}
	ctrl.copy(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("copy(empty): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_copy_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.copy(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("copy(notfound): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- PipelineController.rebuild ---

func TestPipelineController_rebuild_EmptyPvId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{}, admin)

	m := &hbtp.Map{}
	ctrl.rebuild(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("rebuild(empty): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_rebuild_PvNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("pipelineVersionId", "nonexistent")
	ctrl.rebuild(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("rebuild(notfound): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- PipelineController.run ---

func TestPipelineController_run_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{}, admin)

	m := &hbtp.Map{}
	ctrl.run(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("run(empty): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_run_PipelineNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("pipelineId", "nonexistent")
	ctrl.run(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("run(notfound): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- PipelineController.save ---

func TestPipelineController_save_EmptyPipelineId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{}, admin)

	m := &hbtp.Map{}
	ctrl.save(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("save(empty): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- PipelineController.new ---

func TestPipelineController_new_EmptyName(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	// new() takes *bean.NewPipeline, empty name/content should fail validation
	npipe := &bean.NewPipeline{
		Name:    "",
		Content: "",
		OrgId:   "",
	}
	ctrl.new(c, npipe)
	if w.Code != http.StatusBadRequest {
		t.Errorf("new(empty): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- PipelineController.pipelineVersion ---

func TestPipelineController_pipelineVersion_EmptyId(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, hbtp.Map{}, admin)

	m := &hbtp.Map{}
	ctrl.pipelineVersion(c, m)
	if w.Code != http.StatusBadRequest {
		t.Errorf("pipelineVersion(empty): status=%d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPipelineController_pipelineVersion_PvNotFound(t *testing.T) {
	setupPipelineTestDB(t)
	ctrl := PipelineController{}
	admin := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeRuntimeGinContext(t, nil, admin)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.pipelineVersion(c, m)
	if w.Code != http.StatusNotFound {
		t.Errorf("pipelineVersion(notfound): status=%d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- fillPipelineListBuildInfo ---

func TestFillPipelineListBuildInfo_EmptyList(t *testing.T) {
	setupPipelineTestDB(t)
	err := fillPipelineListBuildInfo(nil, nil)
	if err != nil {
		t.Errorf("fillPipelineListBuildInfo(empty): unexpected error: %v", err)
	}
}

// --- PipelineController_Routes_GetPath_duplicate ---

func TestPipelineController_GetPath_Unique(t *testing.T) {
	ctrl := PipelineController{}
	if ctrl.GetPath() != "/api/pipeline" {
		t.Errorf("GetPath() = %q, want %q", ctrl.GetPath(), "/api/pipeline")
	}
}
