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

	_, err = db.Exec(`CREATE TABLE t_artifactory (
		id VARCHAR(64) NOT NULL,
		aid BIGINT NOT NULL,
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
		updated DATETIME,
		PRIMARY KEY (id, aid)
	)`)
	if err != nil {
		t.Fatalf("create artifactory table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifact_package (
		id VARCHAR(64) NOT NULL,
		aid BIGINT NOT NULL,
		repo_id VARCHAR(64),
		name VARCHAR(100),
		display_name VARCHAR(255),
		"desc" VARCHAR(500),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		created DATETIME,
		updated DATETIME,
		PRIMARY KEY (id, aid)
	)`)
	if err != nil {
		t.Fatalf("create package table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifact_version (
		id VARCHAR(64) NOT NULL,
		aid BIGINT NOT NULL,
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
		updated DATETIME,
		PRIMARY KEY (id, aid)
	)`)
	if err != nil {
		t.Fatalf("create version table: %v", err)
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
