package route

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/service"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupArtifactDeepDB builds a comprehensive in-memory SQLite database
// covering all tables that the artifact controller exercises: users, orgs,
// user_orgs, artifactory, artifact_package, and artifact_version.
func setupArtifactDeepDB(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	t.Cleanup(func() {
		comm.Db = origDb
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
	})

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	stmts := []string{
		`CREATE TABLE t_user (
			id VARCHAR(64) NOT NULL, aid BIGINT NOT NULL,
			name VARCHAR(100), pass VARCHAR(255), nick VARCHAR(100),
			avatar VARCHAR(500), created DATETIME, login_time DATETIME,
			active INT DEFAULT 0, PRIMARY KEY (id, aid))`,
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL, aid BIGINT NOT NULL,
			uid VARCHAR(64), name VARCHAR(200), "desc" TEXT,
			public INT DEFAULT 0, created DATETIME, updated DATETIME,
			deleted INT DEFAULT 0, deleted_time DATETIME,
			PRIMARY KEY (id, aid))`,
		`CREATE TABLE t_user_org (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64), org_id VARCHAR(64), created DATETIME,
			perm_adm INT DEFAULT 0, perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0, perm_down INT DEFAULT 0)`,
		`CREATE TABLE t_artifactory (
			id VARCHAR(64) NOT NULL, aid BIGINT DEFAULT 0,
			uid VARCHAR(64), org_id VARCHAR(64), identifier VARCHAR(50),
			name VARCHAR(200), "desc" VARCHAR(500), source VARCHAR(50),
			logo VARCHAR(255), disabled INT DEFAULT 0, deleted INT DEFAULT 0,
			deleted_time DATETIME, created DATETIME, updated DATETIME,
			PRIMARY KEY (id))`,
		`CREATE TABLE t_artifact_package (
			id VARCHAR(64) NOT NULL, aid BIGINT NOT NULL,
			repo_id VARCHAR(64), name VARCHAR(100), display_name VARCHAR(255),
			"desc" VARCHAR(500), deleted INT DEFAULT 0,
			deleted_time DATETIME, created DATETIME, updated DATETIME,
			PRIMARY KEY (id, aid))`,
		`CREATE TABLE t_artifact_version (
			id VARCHAR(64) NOT NULL, aid BIGINT NOT NULL,
			repo_id VARCHAR(64), package_id VARCHAR(64), uid VARCHAR(64),
			name VARCHAR(100), display_name VARCHAR(255),
			version VARCHAR(100), sha VARCHAR(100), "desc" VARCHAR(500),
			preview INT DEFAULT 0, deleted INT DEFAULT 0,
			deleted_time DATETIME, created DATETIME, updated DATETIME,
			PRIMARY KEY (id, aid))`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("exec %q: %v", s[:40], err)
		}
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	seed := []string{
		// Users: admin (owner), regular member, and outsider
		`INSERT INTO t_user (id, aid, name, nick, active) VALUES ('u-admin', 1, 'admin', 'Admin', 1)`,
		`INSERT INTO t_user (id, aid, name, nick, active) VALUES ('u-member', 2, 'member', 'Member', 1)`,
		`INSERT INTO t_user (id, aid, name, nick, active) VALUES ('u-outsider', 3, 'outsider', 'Outsider', 1)`,
		// Org: owned by admin
		`INSERT INTO t_org (id, aid, uid, name, public, created, updated) VALUES ('org-1', 1, 'u-admin', 'Test Org', 1, '` + now + `','` + now + `')`,
		`INSERT INTO t_org (id, aid, uid, name, public, created, updated, deleted) VALUES ('org-del', 2, 'u-admin', 'Deleted Org', 1, '` + now + `','` + now + `', 1)`,
		// User-org membership: member with rw+down perm
		`INSERT INTO t_user_org (uid, org_id, perm_rw, perm_down) VALUES ('u-member', 'org-1', 1, 1)`,
		// Artifacts
		`INSERT INTO t_artifactory (id, aid, uid, org_id, identifier, name, "desc", created, updated) VALUES ('art-1', 1, 'u-admin', 'org-1', 'abcdefgh', 'Repo One', 'desc 1', '` + now + `','` + now + `')`,
		`INSERT INTO t_artifactory (id, aid, uid, org_id, identifier, name, created, updated) VALUES ('art-2', 2, 'u-member', 'org-1', 'xyz12345', 'Repo Two', '` + now + `','` + now + `')`,
		// Packages
		`INSERT INTO t_artifact_package (id, aid, repo_id, name, display_name) VALUES ('pkg-1', 1, 'art-1', 'my-pkg', 'My Pkg')`,
		`INSERT INTO t_artifact_package (id, aid, repo_id, name, display_name) VALUES ('pkg-2', 2, 'art-1', 'other-pkg', 'Other')`,
		// Versions
		`INSERT INTO t_artifact_version (id, aid, repo_id, package_id, uid, name, version) VALUES ('ver-1', 1, 'art-1', 'pkg-1', 'u-admin', 'my-pkg', '1.0.0')`,
		`INSERT INTO t_artifact_version (id, aid, repo_id, package_id, uid, name, version) VALUES ('ver-2', 2, 'art-1', 'pkg-1', 'u-admin', 'my-pkg', '2.0.0')`,
	}
	for _, s := range seed {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed %q: %v", s[:60], err)
		}
	}

	comm.Db = db
	comm.WorkPath = t.TempDir()
	comm.Cfg.Server.DownToken = "test-token-123"
	comm.Cfg.Server.Host = "http://localhost"
	return db
}

// makeArtDeepCtx creates a gin context with a logged-in user already set
// (bypassing MidUserCheck).
func makeArtDeepCtx(t *testing.T, userId string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

	usr := &model.TUser{}
	ok := service.GetIdOrAidCtx(c.Request.Context(), userId, usr)
	if !ok {
		t.Fatalf("seed user %q not found", userId)
	}
	c.Set(service.LgUserKey, usr)
	return c, w
}

// ---------- orgList ----------

func TestOrgList_WithPermission(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("q", "Repo")
	m.Set("page", int64(1))

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.orgList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "art-1") {
		t.Errorf("response should contain art-1, got %s", w.Body.String())
	}
}

func TestOrgList_DeletedOrg(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-del")
	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.orgList(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestOrgList_NoPermission(t *testing.T) {
	setupArtifactDeepDB(t)
	// Create a private org where outsider has no membership
	_, _ = comm.Db.Exec(`INSERT INTO t_org (id, aid, uid, name, public) VALUES ('org-priv', 99, 'u-admin', 'Private', 0)`)

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-priv")
	c, w := makeArtDeepCtx(t, "u-outsider", m)
	ctrl.orgList(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

// ---------- info ----------

func TestInfo_Success(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "art-1")

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"arty"`) {
		t.Errorf("response should contain arty key, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"perm"`) {
		t.Errorf("response should contain perm key, got %s", w.Body.String())
	}
}

func TestInfo_NoReadPerm(t *testing.T) {
	setupArtifactDeepDB(t)
	// outsider has no membership on org-1 which is public (read ok), so create private one
	_, _ = comm.Db.Exec(`INSERT INTO t_org (id, aid, uid, name, public) VALUES ('org-x', 50, 'u-admin', 'X', 0)`)
	_, _ = comm.Db.Exec(`INSERT INTO t_artifactory (id, aid, uid, org_id, identifier, name) VALUES ('art-x', 50, 'u-admin', 'org-x', 'private1', 'X Repo')`)

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "art-x")
	c, w := makeArtDeepCtx(t, "u-outsider", m)
	ctrl.info(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405 (no read perm)", w.Code)
	}
}

// ---------- edit ----------

func TestEdit_CreateNew(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("id", "")
	m.Set("name", "New Repo")
	m.Set("desc", "New description")
	m.Set("disabled", true)

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.edit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	// Should return the newly created id
	body := w.Body.String()
	if len(body) < 5 {
		t.Errorf("expected new id in body, got %q", body)
	}

	// Verify the record exists
	var count int64
	count, _ = comm.Db.Where("name=?", "New Repo").Count(&model.TArtifactory{})
	if count != 1 {
		t.Errorf("expected 1 new repo, found %d", count)
	}
}

func TestEdit_UpdateExisting(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("id", "art-1")
	m.Set("name", "Updated Name")
	m.Set("desc", "Updated desc")

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.edit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	art := &model.TArtifactory{}
	ok := service.GetIdOrAidCtx(c.Request.Context(), "art-1", art)
	if !ok {
		t.Fatal("art-1 should still exist")
	}
	if art.Name != "Updated Name" {
		t.Errorf("name = %q, want 'Updated Name'", art.Name)
	}
}

func TestEdit_NoWritePerm(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("id", "")
	m.Set("name", "Should Fail")

	// outsider has no membership
	c, w := makeArtDeepCtx(t, "u-outsider", m)
	ctrl.edit(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestEdit_UpdateCrossOrg(t *testing.T) {
	setupArtifactDeepDB(t)
	// Try to update art-1 (owned by org-1) while passing a different orgId
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	_, _ = comm.Db.Exec(`INSERT INTO t_org (id, aid, uid, name, public) VALUES ('org-other', 100, 'u-admin', 'Other', 1)`)
	m.Set("orgId", "org-other")
	m.Set("id", "art-1")
	m.Set("name", "Cross-Org")

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.edit(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405 (cross-org update blocked)", w.Code)
	}
}

// ---------- rm ----------

func TestRm_Success(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "art-2")

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	art := &model.TArtifactory{}
	ok := service.GetIdOrAidCtx(c.Request.Context(), "art-2", art)
	if !ok {
		t.Fatal("art-2 should still exist (soft-deleted)")
	}
	if art.Deleted != 1 {
		t.Errorf("deleted flag = %d, want 1", art.Deleted)
	}
}

func TestRm_DeletedOrg(t *testing.T) {
	setupArtifactDeepDB(t)
	// Create artifact in the deleted org
	_, _ = comm.Db.Exec(`INSERT INTO t_artifactory (id, aid, uid, org_id, identifier, name) VALUES ('art-in-del', 60, 'u-admin', 'org-del', 'deldel', 'In Del')`)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "art-in-del")
	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (deleted org)", w.Code)
	}
}

// ---------- versionInfos ----------

func TestVersionInfos_Success(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.versionInfos(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"info"`) {
		t.Errorf("response should contain info key, got %s", w.Body.String())
	}
}

func TestVersionInfos_RepoNotFound(t *testing.T) {
	setupArtifactDeepDB(t)
	// Version pointing at a non-existent repo
	_, _ = comm.Db.Exec(`INSERT INTO t_artifact_version (id, aid, repo_id, package_id, name, version) VALUES ('ver-orphan', 50, 'no-such-repo', 'pkg-x', 'x', '0.1')`)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-orphan")
	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.versionInfos(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (repo not found)", w.Code)
	}
}

func TestVersionInfos_RepoDeleted(t *testing.T) {
	setupArtifactDeepDB(t)
	// Soft-delete art-1
	_, _ = comm.Db.Exec(`UPDATE t_artifactory SET deleted=1 WHERE id='art-1'`)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.versionInfos(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (repo deleted)", w.Code)
	}
}

// ---------- versionUrl ----------

func TestVersionUrl_Success(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	m.Set("path", "/dist/app.jar")

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.versionUrl(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	for _, key := range []string{"id", "times", "random", "sign", "url"} {
		if _, ok := resp[key]; !ok {
			t.Errorf("response missing key %q", key)
		}
	}
	if u, _ := resp["url"].(string); !strings.Contains(u, "app.jar") {
		t.Errorf("url should contain path, got %q", u)
	}
}

func TestVersionUrl_NoDownloadPerm(t *testing.T) {
	setupArtifactDeepDB(t)
	// outsider has no membership → no download perm on org-1 (even though public read)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	m.Set("path", "/x")

	c, w := makeArtDeepCtx(t, "u-outsider", m)
	ctrl.versionUrl(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

// ---------- versionSave ----------

func TestVersionSave_Success(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	m.Set("version", "1.1.0")
	m.Set("desc", "patch release")
	m.Set("ispre", true)

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.versionSave(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	v := &model.TArtifactVersion{}
	ok := service.GetIdOrAidCtx(c.Request.Context(), "ver-1", v)
	if !ok {
		t.Fatal("ver-1 should exist")
	}
	if v.Version != "1.1.0" {
		t.Errorf("version = %q, want '1.1.0'", v.Version)
	}
	if v.Preview != 1 {
		t.Errorf("preview = %d, want 1", v.Preview)
	}
	if v.Desc != "patch release" {
		t.Errorf("desc = %q, want 'patch release'", v.Desc)
	}
}

func TestVersionSave_NoWritePerm(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	m.Set("version", "x")
	c, w := makeArtDeepCtx(t, "u-outsider", m)
	ctrl.versionSave(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

// ---------- versionRm ----------

func TestVersionRm_Success(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-2")

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.versionRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestVersionRm_NoWritePerm(t *testing.T) {
	setupArtifactDeepDB(t)
	// member has rw on org-1 but let's try with outsider
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	c, w := makeArtDeepCtx(t, "u-outsider", m)
	ctrl.versionRm(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestVersionRm_RepoDeleted(t *testing.T) {
	setupArtifactDeepDB(t)
	_, _ = comm.Db.Exec(`UPDATE t_artifactory SET deleted=1 WHERE id='art-1'`)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.versionRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ---------- packageList with pagination ----------

func TestPackageList_WithPagination(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("repoId", "art-1")
	m.Set("page", int64(1))

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.packageList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "my-pkg") {
		t.Errorf("response should include 'my-pkg', got %s", w.Body.String())
	}
}

// ---------- versionList with pagination ----------

func TestVersionList_WithPagination(t *testing.T) {
	setupArtifactDeepDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("packId", "pkg-1")
	m.Set("page", int64(1))

	c, w := makeArtDeepCtx(t, "u-admin", m)
	ctrl.versionList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "1.0.0") {
		t.Errorf("response should include '1.0.0', got %s", w.Body.String())
	}
}
