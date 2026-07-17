package route

import (
	"context"
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

func setupArtifactTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	origCtx := comm.Ctx
	t.Cleanup(func() {
		comm.Db = origDb
		comm.Ctx = origCtx
	})

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_org (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		uid VARCHAR(64),
		name VARCHAR(255),
		desc TEXT,
		public INT DEFAULT 0,
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_org: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifactory (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		name VARCHAR(255),
		identifier VARCHAR(100),
		desc VARCHAR(500),
		source VARCHAR(50),
		logo VARCHAR(255),
		public INT DEFAULT 0,
		disabled INT DEFAULT 0,
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_artifactory: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifact_package (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		repo_id VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		desc VARCHAR(500),
		deleted INT DEFAULT 0,
		deleted_time DATETIME,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_artifact_package: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_artifact_version (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		repo_id VARCHAR(64),
		package_id VARCHAR(64),
		uid VARCHAR(64),
		name VARCHAR(255),
		display_name VARCHAR(255),
		version VARCHAR(100),
		sha VARCHAR(100),
		desc VARCHAR(500),
		preview INT DEFAULT 0,
		created DATETIME,
		updated DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_artifact_version: %v", err)
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
		t.Fatalf("create t_user: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_org (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		perm_adm INT DEFAULT 0,
		perm_rw INT DEFAULT 0,
		perm_exec INT DEFAULT 0,
		perm_down INT DEFAULT 0,
		created DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_user_org: %v", err)
	}

	comm.Db = db
	comm.Ctx = context.Background()
}

func makeArtifactGinContext(t *testing.T, lgusr *model.TUser) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	if lgusr != nil {
		c.Set(service.LgUserKey, lgusr)
	}
	return c, w
}

func TestArtifactController_orgList_EmptyOrgId(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "")
	ctrl.orgList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestArtifactController_orgList_OrgNotFound(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "nonexistent")
	ctrl.orgList(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_packageList_EmptyRepoId(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("repoId", "")
	ctrl.packageList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestArtifactController_versionList_EmptyPackId(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("packId", "")
	ctrl.versionList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestArtifactController_info_NotFound(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_edit_EmptyName(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("name", "")
	m.Set("orgId", "org1")
	ctrl.edit(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestArtifactController_edit_OrgNotFound(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("name", "test-artifact")
	m.Set("orgId", "nonexistent")
	ctrl.edit(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_rm_NotFound(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_versionInfos_NotFound(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.versionInfos(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_versionUrl_NotFound(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("path", "file.txt")
	ctrl.versionUrl(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_versionSave_NotFound(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	m.Set("version", "v1.0")
	ctrl.versionSave(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_versionRm_NotFound(t *testing.T) {
	setupArtifactTestDB(t)
	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "nonexistent")
	ctrl.versionRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_orgList_Success(t *testing.T) {
	setupArtifactTestDB(t)
	// Create org
	_, err := comm.Db.Insert(&model.TOrg{
		Id:      "org1",
		Uid:     "u1",
		Name:    "Test Org",
		Public:  1,
		Deleted: 0,
		Created: time.Now(),
		Updated: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Create artifact
	_, err = comm.Db.Insert(&model.TArtifactory{
		Id:      "art1",
		Uid:     "u1",
		OrgId:   "org1",
		Name:    "test-artifact",
		Deleted: 0,
		Created: time.Now(),
		Updated: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "org1")
	m.Set("page", int64(1))
	ctrl.orgList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestArtifactController_packageList_Success(t *testing.T) {
	setupArtifactTestDB(t)
	_, err := comm.Db.Insert(&model.TArtifactPackage{
		Id:      "pkg1",
		RepoId:  "repo1",
		Name:    "test-package",
		Deleted: 0,
		Created: time.Now(),
	})
	if err != nil {
		t.Fatalf("insert package: %v", err)
	}

	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("repoId", "repo1")
	m.Set("page", int64(1))
	ctrl.packageList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestArtifactController_versionList_Success(t *testing.T) {
	setupArtifactTestDB(t)
	_, err := comm.Db.Insert(&model.TArtifactVersion{
		Id:        "ver1",
		PackageId: "pkg1",
		Name:      "v1.0",
		Version:   "1.0.0",
		Created:   time.Now(),
	})
	if err != nil {
		t.Fatalf("insert version: %v", err)
	}

	c, w := makeArtifactGinContext(t, &model.TUser{Id: "u1", Active: 1})
	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("packId", "pkg1")
	m.Set("page", int64(1))
	ctrl.versionList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}
