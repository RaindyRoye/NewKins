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

func setupArtifactTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Create tables needed by artifact tests
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
		`CREATE TABLE t_artifactory (
			id VARCHAR(64) NOT NULL,
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id VARCHAR(64),
			uid VARCHAR(64),
			identifier VARCHAR(100),
			name VARCHAR(200),
			"desc" VARCHAR(500),
			source VARCHAR(50),
			logo VARCHAR(255),
			disabled INT DEFAULT 0,
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			created DATETIME,
			updated DATETIME
		)`,
		`CREATE TABLE t_artifact_package (
			id VARCHAR(64) NOT NULL,
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id VARCHAR(64),
			name VARCHAR(100),
			display_name VARCHAR(255),
			"desc" VARCHAR(500),
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			created DATETIME,
			updated DATETIME
		)`,
		`CREATE TABLE t_artifact_version (
			id VARCHAR(64) NOT NULL,
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id VARCHAR(64),
			package_id VARCHAR(64),
			uid VARCHAR(64),
			name VARCHAR(100),
			display_name VARCHAR(255),
			version VARCHAR(100),
			sha VARCHAR(100),
			"desc" VARCHAR(500),
			preview INT DEFAULT 0,
			deleted INT DEFAULT 0,
			deleted_time DATETIME,
			created DATETIME,
			updated DATETIME
		)`,
		`CREATE TABLE t_org (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			aid BIGINT,
			uid VARCHAR(64),
			name VARCHAR(100),
			desc TEXT,
			public INT DEFAULT 0,
			created DATETIME,
			updated DATETIME,
			deleted INT DEFAULT 0,
			deleted_time DATETIME
		)`,
		`CREATE TABLE t_user_org (
			aid INTEGER PRIMARY KEY AUTOINCREMENT,
			uid VARCHAR(64),
			org_id VARCHAR(64),
			perm_adm INT DEFAULT 0,
			perm_rw INT DEFAULT 0,
			perm_exec INT DEFAULT 0,
			perm_down INT DEFAULT 0,
			created DATETIME
		)`,
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	comm.Db = db
}

func makeArtifactTestContext(t *testing.T, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
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

	// Set a test user to bypass MidUserCheck
	c.Set(service.LgUserKey, &model.TUser{
		Id:     "test-user",
		Name:   "tester",
		Active: 1,
	})
	return c, w
}

func TestArtifactController_GetPathFromArtifact(t *testing.T) {
	c := &ArtifactController{}
	if got := c.GetPath(); got != "/api/art" {
		t.Errorf("GetPath() = %q, want %q", got, "/api/art")
	}
}

func TestArtifactController_Routes(t *testing.T) {
	setupArtifactTestDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ac := &ArtifactController{}
	ac.Routes(r.Group("/api/art"))
	// Routes registered successfully
}

func TestArtifactInfo_EmptyId(t *testing.T) {
	setupArtifactTestDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "")
	c, w := makeArtifactTestContext(t, m)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for empty id, got %d", w.Code)
	}
}

func TestArtifactInfo_NonexistentId(t *testing.T) {
	setupArtifactTestDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent-id")
	c, w := makeArtifactTestContext(t, m)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent artifact, got %d", w.Code)
	}
}

func TestArtifactInfo_DeletedArtifact(t *testing.T) {
	setupArtifactTestDB(t)

	// Insert a deleted artifact
	_, err := comm.Db.Exec(`INSERT INTO t_artifactory (id, aid, org_id, uid, name, deleted) 
		VALUES ('art-deleted', 1, 'org-1', 'user-1', 'Deleted Art', 1)`)
	if err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "art-deleted")
	c, w := makeArtifactTestContext(t, m)
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted artifact, got %d", w.Code)
	}
}

func TestArtifactEdit_EmptyName(t *testing.T) {
	setupArtifactTestDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("id", "")
	m.Set("name", "")
	c, w := makeArtifactTestContext(t, m)
	ctrl.edit(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", w.Code)
	}
}

func TestArtifactRm_NonexistentId(t *testing.T) {
	setupArtifactTestDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeArtifactTestContext(t, m)
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent artifact, got %d", w.Code)
	}
}

func TestArtifactPackageList_EmptyRepoId(t *testing.T) {
	setupArtifactTestDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("repoId", "")
	c, w := makeArtifactTestContext(t, m)
	ctrl.packageList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty repoId, got %d", w.Code)
	}
}

func TestArtifactVersionList_EmptyPackId(t *testing.T) {
	setupArtifactTestDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("packId", "")
	c, w := makeArtifactTestContext(t, m)
	ctrl.versionList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty packId, got %d", w.Code)
	}
}

func TestArtifactVersionInfos_NonexistentId(t *testing.T) {
	setupArtifactTestDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeArtifactTestContext(t, m)
	ctrl.versionInfos(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent version, got %d", w.Code)
	}
}

func TestArtifactVersionUrl_NonexistentId(t *testing.T) {
	setupArtifactTestDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("path", "/test")
	c, w := makeArtifactTestContext(t, m)
	ctrl.versionUrl(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent version, got %d", w.Code)
	}
}

func TestArtifactVersionSave_NonexistentId(t *testing.T) {
	setupArtifactTestDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("version", "1.0.0")
	c, w := makeArtifactTestContext(t, m)
	ctrl.versionSave(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent version, got %d", w.Code)
	}
}

func TestArtifactVersionRm_NonexistentId(t *testing.T) {
	setupArtifactTestDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	c, w := makeArtifactTestContext(t, m)
	ctrl.versionRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent version, got %d", w.Code)
	}
}

func TestArtifactOrgList_EmptyOrgId(t *testing.T) {
	setupArtifactTestDB(t)
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	c, w := makeArtifactTestContext(t, m)
	ctrl.orgList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty orgId, got %d", w.Code)
	}
}

func TestArtifactPackageList_WithSearch(t *testing.T) {
	setupArtifactTestDB(t)

	// Insert test packages
	_, err := comm.Db.Exec(`INSERT INTO t_artifact_package (id, aid, repo_id, name, display_name) 
		VALUES ('pkg-1', 1, 'repo-1', 'test-package', 'Test Package')`)
	if err != nil {
		t.Fatalf("insert package: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("repoId", "repo-1")
	m.Set("q", "test")
	m.Set("page", int64(1))
	c, w := makeArtifactTestContext(t, m)
	ctrl.packageList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid request, got %d", w.Code)
	}
}

func TestArtifactVersionList_WithSearch(t *testing.T) {
	setupArtifactTestDB(t)

	// Insert test version
	_, err := comm.Db.Exec(`INSERT INTO t_artifact_version (id, aid, package_id, name, version) 
		VALUES ('ver-1', 1, 'pkg-1', 'test-version', '1.0.0')`)
	if err != nil {
		t.Fatalf("insert version: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("packId", "pkg-1")
	m.Set("q", "test")
	m.Set("page", int64(1))
	c, w := makeArtifactTestContext(t, m)
	ctrl.versionList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for valid request, got %d", w.Code)
	}
}

// ============ Happy path and permission tests ============

//nolint:unparam // test helper designed for reusability
func createArtifactTestOrg(t *testing.T, orgId string, name string) {
	t.Helper()
	_, err := comm.Db.Exec(`INSERT INTO t_org (id, name, uid, deleted, created, updated) 
		VALUES (?, ?, 'other-user', 0, datetime('now'), datetime('now'))`, orgId, name)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
}

//nolint:unparam // test helper designed for reusability
func createArtifactTestUserOrg(t *testing.T, uid string, orgId string, permAdm, permRw, permExec, permDown int) {
	t.Helper()
	_, err := comm.Db.Exec(`INSERT INTO t_user_org (uid, org_id, perm_adm, perm_rw, perm_exec, perm_down, created) 
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`, uid, orgId, permAdm, permRw, permExec, permDown)
	if err != nil {
		t.Fatalf("create user org: %v", err)
	}
}

func TestArtifactInfo_Success(t *testing.T) {
	setupArtifactTestDB(t)

	// Insert the test user first
	_, err := comm.Db.Exec(`INSERT INTO t_user (id, name, nick, active, created, login_time) 
		VALUES ('test-user', 'test-user', 'Test User', 1, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	createArtifactTestOrg(t, "org-1", "Test Org")
	createArtifactTestUserOrg(t, "test-user", "org-1", 0, 1, 0, 1)

	// Create an active artifact
	_, err = comm.Db.Exec(`INSERT INTO t_artifactory (id, aid, org_id, uid, name, deleted, created, updated) 
		VALUES ('art-1', 1, 'org-1', 'test-user', 'My Artifact', 0, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "art-1")
	c, w := makeArtifactTestContext(t, m)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestArtifactEdit_Create_Success(t *testing.T) {
	setupArtifactTestDB(t)
	createArtifactTestOrg(t, "org-1", "Test Org")
	createArtifactTestUserOrg(t, "test-user", "org-1", 1, 1, 1, 1)

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("id", "")
	m.Set("name", "new-artifact")
	m.Set("desc", "A new artifact")
	m.Set("disabled", false)
	c, w := makeArtifactTestContext(t, m)
	ctrl.edit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for create, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify it was created
	var count int64
	count, err := comm.Db.Where("org_id=? AND name=?", "org-1", "new-artifact").Count(&model.TArtifactory{})
	if err != nil {
		t.Fatalf("count artifacts: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 artifact, got %d", count)
	}
}

func TestArtifactEdit_Update_Success(t *testing.T) {
	setupArtifactTestDB(t)
	createArtifactTestOrg(t, "org-1", "Test Org")
	createArtifactTestUserOrg(t, "test-user", "org-1", 1, 1, 1, 1)

	// Create an existing artifact
	_, err := comm.Db.Exec(`INSERT INTO t_artifactory (id, aid, org_id, uid, name, "desc", deleted, created, updated) 
		VALUES ('art-1', 1, 'org-1', 'test-user', 'Old Name', 'old desc', 0, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("id", "art-1")
	m.Set("name", "Updated Name")
	m.Set("desc", "Updated desc")
	m.Set("disabled", true)
	c, w := makeArtifactTestContext(t, m)
	ctrl.edit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for update, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify update
	var art model.TArtifactory
	ok, err := comm.Db.Where("id=?", "art-1").Get(&art)
	if err != nil || !ok {
		t.Fatalf("query artifact: %v", err)
	}
	if art.Name != "Updated Name" {
		t.Errorf("expected name='Updated Name', got %q", art.Name)
	}
	if art.Disabled != 1 {
		t.Errorf("expected disabled=1, got %d", art.Disabled)
	}
}

func TestArtifactEdit_NoPermission(t *testing.T) {
	setupArtifactTestDB(t)

	// Create org owned by a different user
	_, err := comm.Db.Exec(`INSERT INTO t_org (id, name, uid, deleted, created, updated) 
		VALUES ('org-1', 'Test Org', 'other-user', 0, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	// test-user is NOT a member of org-1, so should have no permission
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("id", "")
	m.Set("name", "new-artifact")
	c, w := makeArtifactTestContext(t, m)
	ctrl.edit(c, m)

	// Should get 405 (Method Not Allowed) since user is not in the org
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for no permission, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestArtifactRm_Success(t *testing.T) {
	setupArtifactTestDB(t)
	createArtifactTestOrg(t, "org-1", "Test Org")
	createArtifactTestUserOrg(t, "test-user", "org-1", 1, 1, 1, 1)

	// Create an artifact
	_, err := comm.Db.Exec(`INSERT INTO t_artifactory (id, aid, org_id, uid, name, deleted, created, updated) 
		VALUES ('art-rm', 1, 'org-1', 'test-user', 'To Remove', 0, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "art-rm")
	c, w := makeArtifactTestContext(t, m)
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify soft delete
	var art model.TArtifactory
	ok, err := comm.Db.Where("id=?", "art-rm").Get(&art)
	if err != nil || !ok {
		t.Fatalf("query artifact: %v", err)
	}
	if art.Deleted != 1 {
		t.Errorf("expected deleted=1, got %d", art.Deleted)
	}
}

func TestArtifactPackageList_Success(t *testing.T) {
	setupArtifactTestDB(t)

	// Insert test packages
	_, err := comm.Db.Exec(`INSERT INTO t_artifact_package (id, aid, repo_id, name, display_name, deleted, created, updated) 
		VALUES ('pkg-1', 1, 'repo-1', 'my-pkg', 'My Package', 0, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert package: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("repoId", "repo-1")
	m.Set("page", int64(1))
	c, w := makeArtifactTestContext(t, m)
	ctrl.packageList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["data"] == nil {
		t.Error("expected data field in response")
	}
}

func TestArtifactVersionList_Success(t *testing.T) {
	setupArtifactTestDB(t)

	// Insert test versions
	_, err := comm.Db.Exec(`INSERT INTO t_artifact_version (id, aid, package_id, repo_id, name, version, created, updated) 
		VALUES ('ver-1', 1, 'pkg-1', 'repo-1', 'v1.0.0', '1.0.0', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert version: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("packId", "pkg-1")
	m.Set("page", int64(1))
	c, w := makeArtifactTestContext(t, m)
	ctrl.versionList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["data"] == nil {
		t.Error("expected data field in response")
	}
}

func TestArtifactVersionSave_Success(t *testing.T) {
	setupArtifactTestDB(t)
	createArtifactTestOrg(t, "org-1", "Test Org")
	createArtifactTestUserOrg(t, "test-user", "org-1", 1, 1, 1, 1)

	// Create artifact and version
	_, err := comm.Db.Exec(`INSERT INTO t_artifactory (id, aid, org_id, uid, name, deleted, created, updated) 
		VALUES ('art-1', 1, 'org-1', 'test-user', 'My Art', 0, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	_, err = comm.Db.Exec(`INSERT INTO t_artifact_version (id, aid, package_id, repo_id, name, version, created, updated) 
		VALUES ('ver-1', 1, 'pkg-1', 'art-1', 'v1.0.0', '1.0.0', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert version: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	m.Set("version", "1.0.1")
	m.Set("desc", "Updated description")
	m.Set("ispre", false)
	c, w := makeArtifactTestContext(t, m)
	ctrl.versionSave(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify update
	var ver model.TArtifactVersion
	ok, err := comm.Db.Where("id=?", "ver-1").Get(&ver)
	if err != nil || !ok {
		t.Fatalf("query version: %v", err)
	}
	if ver.Version != "1.0.1" {
		t.Errorf("expected version='1.0.1', got %q", ver.Version)
	}
}

func TestArtifactVersionRm_Success(t *testing.T) {
	setupArtifactTestDB(t)
	createArtifactTestOrg(t, "org-1", "Test Org")
	createArtifactTestUserOrg(t, "test-user", "org-1", 1, 1, 1, 1)

	// Create artifact and version
	_, err := comm.Db.Exec(`INSERT INTO t_artifactory (id, aid, org_id, uid, name, deleted, created, updated) 
		VALUES ('art-1', 1, 'org-1', 'test-user', 'My Art', 0, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	_, err = comm.Db.Exec(`INSERT INTO t_artifact_version (id, aid, package_id, repo_id, name, version, created, updated) 
		VALUES ('ver-rm', 1, 'pkg-1', 'art-1', 'v1.0.0', '1.0.0', datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert version: %v", err)
	}

	origWorkPath := comm.WorkPath
	comm.WorkPath = t.TempDir()
	t.Cleanup(func() { comm.WorkPath = origWorkPath })

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-rm")
	c, w := makeArtifactTestContext(t, m)
	ctrl.versionRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify deletion
	count, err := comm.Db.Where("id=?", "ver-rm").Count(&model.TArtifactVersion{})
	if err != nil {
		t.Fatalf("count versions: %v", err)
	}
	if count != 0 {
		t.Error("expected version to be deleted")
	}
}

func TestArtifactOrgList_Success(t *testing.T) {
	setupArtifactTestDB(t)
	createArtifactTestOrg(t, "org-1", "Test Org")
	createArtifactTestUserOrg(t, "test-user", "org-1", 1, 1, 1, 1)

	// Create artifacts
	_, err := comm.Db.Exec(`INSERT INTO t_artifactory (id, aid, org_id, uid, name, deleted, created, updated) 
		VALUES ('art-1', 1, 'org-1', 'test-user', 'Art 1', 0, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("page", int64(1))
	c, w := makeArtifactTestContext(t, m)
	ctrl.orgList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["data"] == nil {
		t.Error("expected data field in response")
	}
}
