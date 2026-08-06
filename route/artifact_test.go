package route

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func setupArtifactTestDb(t *testing.T) *xorm.Engine {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create sqlite engine: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	comm.Db = db

	_, err = db.Exec(`CREATE TABLE t_org (
		id VARCHAR(64) NOT NULL,
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		name VARCHAR(200),
		"desc" TEXT,
		public INT DEFAULT 0,
		created DATETIME,
		updated DATETIME,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create org table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		name VARCHAR(100),
		pass VARCHAR(255),
		nick VARCHAR(100),
		avatar VARCHAR(500),
		created DATETIME,
		login_time DATETIME,
		active INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create user table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_org (
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		created DATETIME,
		perm_adm INT DEFAULT 0,
		perm_rw INT DEFAULT 0,
		perm_exec INT DEFAULT 0,
		perm_down INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create user_org table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifactory (
		id VARCHAR(64) NOT NULL,
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		identifier VARCHAR(50),
		name VARCHAR(200),
		disabled INT DEFAULT 0,
		source VARCHAR(50),
		"desc" VARCHAR(500),
		logo VARCHAR(255),
		created DATETIME,
		updated DATETIME,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create artifactory table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifact_package (
		id VARCHAR(64) NOT NULL,
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		repo_id VARCHAR(64),
		name VARCHAR(200),
		display_name VARCHAR(200),
		"desc" VARCHAR(500),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create artifact_package table: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifact_version (
		id VARCHAR(64) NOT NULL,
		aid INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		repo_id VARCHAR(64),
		package_id VARCHAR(64),
		name VARCHAR(100),
		version VARCHAR(100),
		sha VARCHAR(100),
		"desc" VARCHAR(500),
		preview INT DEFAULT 0,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create artifact_version table: %v", err)
	}

	return db
}

func makeArtifactGinCtx(t *testing.T, user *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if user != nil {
		c.Set(service.LgUserKey, user)
	}
	return c, w
}

func TestArtifact_PackageList_MissingRepoId(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("repoId", "")
	ctrl.packageList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing repoId, got %d", w.Code)
	}
}

func TestArtifact_PackageList_Success(t *testing.T) {
	db := setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	// Insert test packages
	pkgs := []*model.TArtifactPackage{
		{Id: "pkg-1", RepoId: "repo-1", Name: "package-alpha", Created: time.Now()},
		{Id: "pkg-2", RepoId: "repo-1", Name: "package-beta", Created: time.Now()},
	}
	for _, p := range pkgs {
		if _, err := db.InsertOne(p); err != nil {
			t.Fatalf("insert package: %v", err)
		}
	}

	c, w := makeArtifactGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("repoId", "repo-1")
	m.Set("page", int64(1))
	ctrl.packageList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestArtifact_PackageList_WithSearch(t *testing.T) {
	db := setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	pkgs := []*model.TArtifactPackage{
		{Id: "pkg-1", RepoId: "repo-1", Name: "frontend-lib", Created: time.Now()},
		{Id: "pkg-2", RepoId: "repo-1", Name: "backend-lib", Created: time.Now()},
	}
	for _, p := range pkgs {
		if _, err := db.InsertOne(p); err != nil {
			t.Fatalf("insert package: %v", err)
		}
	}

	c, w := makeArtifactGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("repoId", "repo-1")
	m.Set("q", "frontend")
	m.Set("page", int64(1))
	ctrl.packageList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestArtifact_VersionList_MissingPackId(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("packId", "")
	ctrl.versionList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing packId, got %d", w.Code)
	}
}

func TestArtifact_VersionList_Success(t *testing.T) {
	db := setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	versions := []*model.TArtifactVersion{
		{Id: "ver-1", PackageId: "pkg-1", Name: "v1.0.0", Version: "1.0.0", Created: time.Now()},
		{Id: "ver-2", PackageId: "pkg-1", Name: "v1.1.0", Version: "1.1.0", Created: time.Now()},
	}
	for _, v := range versions {
		if _, err := db.InsertOne(v); err != nil {
			t.Fatalf("insert version: %v", err)
		}
	}

	c, w := makeArtifactGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("packId", "pkg-1")
	m.Set("page", int64(1))
	ctrl.versionList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestArtifact_VersionInfos_NotFound(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.versionInfos(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestArtifact_VersionInfos_RepoNotFound(t *testing.T) {
	db := setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ver := &model.TArtifactVersion{Id: "ver-1", RepoId: "nonexistent-repo", Name: "v1.0.0", Created: time.Now()}
	if _, err := db.InsertOne(ver); err != nil {
		t.Fatalf("insert version: %v", err)
	}

	c, w := makeArtifactGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	ctrl.versionInfos(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing repo, got %d", w.Code)
	}
}

func TestArtifact_VersionUrl_NotFound(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("path", "file.txt")
	ctrl.versionUrl(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestArtifact_VersionUrl_RepoNotFound(t *testing.T) {
	db := setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ver := &model.TArtifactVersion{Id: "ver-1", RepoId: "nonexistent-repo", Name: "v1.0.0", Created: time.Now()}
	if _, err := db.InsertOne(ver); err != nil {
		t.Fatalf("insert version: %v", err)
	}

	c, w := makeArtifactGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	m.Set("path", "file.txt")
	ctrl.versionUrl(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing repo, got %d", w.Code)
	}
}

func TestArtifact_VersionSave_NotFound(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("version", "1.0.0")
	ctrl.versionSave(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestArtifact_VersionSave_RepoNotFound(t *testing.T) {
	db := setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ver := &model.TArtifactVersion{Id: "ver-1", RepoId: "nonexistent-repo", Name: "v1.0.0", Created: time.Now()}
	if _, err := db.InsertOne(ver); err != nil {
		t.Fatalf("insert version: %v", err)
	}

	c, w := makeArtifactGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	m.Set("version", "2.0.0")
	ctrl.versionSave(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing repo, got %d", w.Code)
	}
}

func TestArtifact_VersionRm_NotFound(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.versionRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestArtifact_VersionRm_RepoNotFound(t *testing.T) {
	db := setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	ver := &model.TArtifactVersion{Id: "ver-1", RepoId: "nonexistent-repo", Name: "v1.0.0", Created: time.Now()}
	if _, err := db.InsertOne(ver); err != nil {
		t.Fatalf("insert version: %v", err)
	}

	c, w := makeArtifactGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	ctrl.versionRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing repo, got %d", w.Code)
	}
}

func TestArtifact_Info_NotFound(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestArtifact_Info_Deleted(t *testing.T) {
	db := setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}

	art := &model.TArtifactory{Id: "art-1", Name: "deleted-art", Deleted: 1, Created: time.Now()}
	if _, err := db.InsertOne(art); err != nil {
		t.Fatalf("insert artifactory: %v", err)
	}

	c, w := makeArtifactGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "art-1")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted artifact, got %d", w.Code)
	}
}

func TestArtifact_Rm_NotFound(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestArtifact_Edit_MissingName(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("name", "")
	ctrl.edit(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", w.Code)
	}
}

func TestArtifact_Edit_OrgNotFound(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	m.Set("name", "new-art")
	ctrl.edit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing org, got %d", w.Code)
	}
}

func TestArtifact_OrgList_MissingOrgId(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "")
	ctrl.orgList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing orgId, got %d", w.Code)
	}
}

func TestArtifact_OrgList_OrgNotFound(t *testing.T) {
	setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	c, w := makeArtifactGinCtx(t, user)

	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	m.Set("page", int64(1))
	ctrl.orgList(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing org, got %d", w.Code)
	}
}

func TestArtifact_OrgList_Success(t *testing.T) {
	db := setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	org := &model.TOrg{Id: "org-1", Uid: "admin", Name: "test-org", Public: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	arts := []*model.TArtifactory{
		{Id: "art-1", OrgId: "org-1", Uid: "admin", Name: "art-alpha", Created: time.Now()},
		{Id: "art-2", OrgId: "org-1", Uid: "admin", Name: "art-beta", Created: time.Now()},
	}
	for _, a := range arts {
		if _, err := db.InsertOne(a); err != nil {
			t.Fatalf("insert artifactory: %v", err)
		}
	}

	c, w := makeArtifactGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("orgId", "org-1")
	m.Set("page", int64(1))
	ctrl.orgList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

func TestArtifact_Info_Success(t *testing.T) {
	db := setupArtifactTestDb(t)
	ctrl := ArtifactController{}
	user := &model.TUser{Id: "admin", Name: "admin", Active: 1}
	if _, err := db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	org := &model.TOrg{Id: "org-1", Uid: "admin", Name: "test-org", Public: 1, Created: time.Now(), Updated: time.Now()}
	if _, err := db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	art := &model.TArtifactory{Id: "art-1", OrgId: "org-1", Uid: "admin", Name: "test-art", Created: time.Now()}
	if _, err := db.InsertOne(art); err != nil {
		t.Fatalf("insert artifactory: %v", err)
	}

	c, w := makeArtifactGinCtx(t, user)
	m := &hbtp.Map{}
	m.Set("id", "art-1")
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["arty"] == nil {
		t.Error("expected arty in response")
	}
	if resp["user"] == nil {
		t.Error("expected user in response")
	}
	if resp["perm"] == nil {
		t.Error("expected perm in response")
	}
}
