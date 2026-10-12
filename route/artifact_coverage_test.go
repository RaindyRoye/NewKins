package route

import (
	"net/http"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

// setupArtifactExtendedTestDB creates tables needed for artifact routes with org validation
func setupArtifactExtendedTestDB(t *testing.T) {
	setupArtifactTestDB(t)

	// Drop and recreate t_artifactory with auto-increment aid
	_, err := comm.Db.Exec(`DROP TABLE IF EXISTS t_artifactory`)
	if err != nil {
		t.Fatalf("drop artifactory table: %v", err)
	}

	_, err = comm.Db.Exec(`CREATE TABLE t_artifactory (
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
	)`)
	if err != nil {
		t.Fatalf("create artifactory table: %v", err)
	}

	// Add t_org table - aid is INTEGER PRIMARY KEY for auto-increment
	_, err = comm.Db.Exec(`CREATE TABLE t_org (
		id VARCHAR(64) NOT NULL,
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
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

	// Add t_user table with login_time column
	_, err = comm.Db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL,
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(100),
		nick VARCHAR(100),
		avatar VARCHAR(255),
		email VARCHAR(200),
		phone VARCHAR(50),
		pass VARCHAR(200),
		salt VARCHAR(100),
		admin INT DEFAULT 0,
		active INT DEFAULT 0,
		login_time DATETIME,
		created DATETIME,
		updated DATETIME,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create user table: %v", err)
	}

	// Recreate t_artifact_package with auto-increment aid
	_, err = comm.Db.Exec(`DROP TABLE IF EXISTS t_artifact_package`)
	if err != nil {
		t.Fatalf("drop artifact_package table: %v", err)
	}

	_, err = comm.Db.Exec(`CREATE TABLE t_artifact_package (
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
	)`)
	if err != nil {
		t.Fatalf("create artifact_package table: %v", err)
	}

	// Recreate t_artifact_version with auto-increment aid
	_, err = comm.Db.Exec(`DROP TABLE IF EXISTS t_artifact_version`)
	if err != nil {
		t.Fatalf("drop artifact_version table: %v", err)
	}

	_, err = comm.Db.Exec(`CREATE TABLE t_artifact_version (
		id VARCHAR(64) NOT NULL,
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		repo_id VARCHAR(64),
		package_id VARCHAR(64),
		name VARCHAR(255),
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
}

// TestArtifactOrgList_SuccessExtended tests the successful path of orgList
func TestArtifactOrgList_SuccessExtended(t *testing.T) {
	setupArtifactExtendedTestDB(t)

	// Create test organization
	org := &model.TOrg{
		Id: "test-org", Name: "Test Org", Uid: "test-user",
		Public: 1, Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Create test artifacts
	art1 := &model.TArtifactory{
		Id: "art-1", Name: "Artifact 1", Desc: "Test artifact 1",
		OrgId: "test-org", Uid: "test-user", Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(art1); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "test-org")
	m.Set("page", int64(1))

	c, w := makeArtifactTestContext(t, m)
	ctrl.orgList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for successful org list, got %d, body: %s", w.Code, w.Body.String())
	}
}

// TestArtifactOrgList_WithSearchExtended tests orgList with search query
func TestArtifactOrgList_WithSearchExtended(t *testing.T) {
	setupArtifactExtendedTestDB(t)

	org := &model.TOrg{
		Id: "test-org", Name: "Test Org", Uid: "test-user",
		Public: 1, Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	art1 := &model.TArtifactory{
		Id: "art-1", Name: "Searchable Artifact", Desc: "Test artifact",
		OrgId: "test-org", Uid: "test-user", Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(art1); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "test-org")
	m.Set("q", "Searchable")
	m.Set("page", int64(1))

	c, w := makeArtifactTestContext(t, m)
	ctrl.orgList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for org list with search, got %d, body: %s", w.Code, w.Body.String())
	}
}

// TestArtifactInfo_SuccessExtended tests the successful path of info
func TestArtifactInfo_SuccessExtended(t *testing.T) {
	setupArtifactExtendedTestDB(t)

	org := &model.TOrg{
		Id: "test-org", Name: "Test Org", Uid: "test-user",
		Public: 1, Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{
		Id: "test-user", Name: "Test User", Nick: "Tester",
		Active: 1, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	art := &model.TArtifactory{
		Id: "art-1", Name: "Test Artifact", Desc: "Test description",
		OrgId: "test-org", Uid: "test-user", Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(art); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "art-1")

	c, w := makeArtifactTestContext(t, m)
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for successful artifact info, got %d, body: %s", w.Code, w.Body.String())
	}
}

// TestArtifactEdit_CreateExtended tests creating a new artifact
func TestArtifactEdit_CreateExtended(t *testing.T) {
	setupArtifactExtendedTestDB(t)

	org := &model.TOrg{
		Id: "test-org", Name: "Test Org", Uid: "test-user",
		Public: 1, Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "test-org")
	m.Set("name", "New Artifact")
	m.Set("desc", "New description")
	m.Set("disabled", false)

	c, w := makeArtifactTestContext(t, m)
	ctrl.edit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for creating artifact, got %d, body: %s", w.Code, w.Body.String())
	}
}

// TestArtifactEdit_UpdateExtended tests updating an existing artifact
func TestArtifactEdit_UpdateExtended(t *testing.T) {
	setupArtifactExtendedTestDB(t)

	org := &model.TOrg{
		Id: "test-org", Name: "Test Org", Uid: "test-user",
		Public: 1, Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	art := &model.TArtifactory{
		Id: "art-1", Name: "Original Name", Desc: "Original description",
		OrgId: "test-org", Uid: "test-user", Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(art); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("orgId", "test-org")
	m.Set("id", "art-1")
	m.Set("name", "Updated Name")
	m.Set("desc", "Updated description")
	m.Set("disabled", true)

	c, w := makeArtifactTestContext(t, m)
	ctrl.edit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for updating artifact, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify the update
	updatedArt := &model.TArtifactory{}
	has, err := comm.Db.Where("id = ?", "art-1").Get(updatedArt)
	if err != nil {
		t.Fatalf("query updated artifact: %v", err)
	}
	if !has {
		t.Fatal("artifact not found after update")
	}
	if updatedArt.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", updatedArt.Name)
	}
	if updatedArt.Disabled != 1 {
		t.Errorf("expected disabled=1, got %d", updatedArt.Disabled)
	}
}

// TestArtifactRm_SuccessExtended tests the successful path of rm
func TestArtifactRm_SuccessExtended(t *testing.T) {
	setupArtifactExtendedTestDB(t)

	org := &model.TOrg{
		Id: "test-org", Name: "Test Org", Uid: "test-user",
		Public: 1, Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	art := &model.TArtifactory{
		Id: "art-1", Name: "Test Artifact", OrgId: "test-org",
		Uid: "test-user", Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(art); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "art-1")

	c, w := makeArtifactTestContext(t, m)
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for removing artifact, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify the artifact is marked as deleted
	deletedArt := &model.TArtifactory{}
	has, err := comm.Db.Where("id = ?", "art-1").Get(deletedArt)
	if err != nil {
		t.Fatalf("query deleted artifact: %v", err)
	}
	if !has {
		t.Fatal("artifact not found after deletion")
	}
	if deletedArt.Deleted != 1 {
		t.Errorf("expected deleted=1, got %d", deletedArt.Deleted)
	}
}

// TestArtifactVersionInfos_SuccessExtended tests the successful path of versionInfos
func TestArtifactVersionInfos_SuccessExtended(t *testing.T) {
	setupArtifactExtendedTestDB(t)

	org := &model.TOrg{
		Id: "test-org", Name: "Test Org", Uid: "test-user",
		Public: 1, Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	art := &model.TArtifactory{
		Id: "art-1", Name: "Test Artifact", OrgId: "test-org",
		Uid: "test-user", Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(art); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ver := &model.TArtifactVersion{
		Id: "ver-1", RepoId: "art-1", Version: "1.0.0",
		Desc: "Test version", Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(ver); err != nil {
		t.Fatalf("insert version: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")

	c, w := makeArtifactTestContext(t, m)
	ctrl.versionInfos(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for version infos, got %d, body: %s", w.Code, w.Body.String())
	}
}

// TestArtifactVersionUrl_SuccessExtended tests the successful path of versionUrl
func TestArtifactVersionUrl_SuccessExtended(t *testing.T) {
	setupArtifactExtendedTestDB(t)

	org := &model.TOrg{
		Id: "test-org", Name: "Test Org", Uid: "test-user",
		Public: 1, Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	art := &model.TArtifactory{
		Id: "art-1", Name: "Test Artifact", OrgId: "test-org",
		Uid: "test-user", Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(art); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ver := &model.TArtifactVersion{
		Id: "ver-1", RepoId: "art-1", Version: "1.0.0",
		Desc: "Test version", Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(ver); err != nil {
		t.Fatalf("insert version: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")
	m.Set("path", "/test/path")

	c, w := makeArtifactTestContext(t, m)
	ctrl.versionUrl(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for version url, got %d, body: %s", w.Code, w.Body.String())
	}
}

// TestArtifactVersionSave_SuccessExtended tests the successful path of versionSave (update operation)
func TestArtifactVersionSave_SuccessExtended(t *testing.T) {
	setupArtifactExtendedTestDB(t)

	org := &model.TOrg{
		Id: "test-org", Name: "Test Org", Uid: "test-user",
		Public: 1, Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	art := &model.TArtifactory{
		Id: "art-1", Name: "Test Artifact", OrgId: "test-org",
		Uid: "test-user", Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(art); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	// Create an existing version that we'll update
	ver := &model.TArtifactVersion{
		Id: "ver-1", RepoId: "art-1", PackageId: "pkg-1",
		Name: "v1.0.0", Version: "1.0.0", Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(ver); err != nil {
		t.Fatalf("insert version: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")  // The existing version ID to update
	m.Set("version", "1.0.1")
	m.Set("desc", "Updated version description")
	m.Set("ispre", false)

	c, w := makeArtifactTestContext(t, m)
	ctrl.versionSave(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for updating version, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verify the version was updated
	updatedVer := &model.TArtifactVersion{}
	has, err := comm.Db.Where("id = ?", "ver-1").Get(updatedVer)
	if err != nil {
		t.Fatalf("query updated version: %v", err)
	}
	if !has {
		t.Fatal("version not found after update")
	}
	if updatedVer.Version != "1.0.1" {
		t.Errorf("expected version '1.0.1', got '%s'", updatedVer.Version)
	}
	if updatedVer.Desc != "Updated version description" {
		t.Errorf("expected desc 'Updated version description', got '%s'", updatedVer.Desc)
	}
}

// TestArtifactVersionRm_SuccessExtended tests the successful path of versionRm
func TestArtifactVersionRm_SuccessExtended(t *testing.T) {
	setupArtifactExtendedTestDB(t)

	org := &model.TOrg{
		Id: "test-org", Name: "Test Org", Uid: "test-user",
		Public: 1, Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	art := &model.TArtifactory{
		Id: "art-1", Name: "Test Artifact", OrgId: "test-org",
		Uid: "test-user", Deleted: 0, Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(art); err != nil {
		t.Fatalf("insert artifact: %v", err)
	}

	ver := &model.TArtifactVersion{
		Id: "ver-1", RepoId: "art-1", Version: "1.0.0",
		Desc: "Test version", Created: time.Now(),
	}
	if _, err := comm.Db.InsertOne(ver); err != nil {
		t.Fatalf("insert version: %v", err)
	}

	ctrl := ArtifactController{}
	m := &hbtp.Map{}
	m.Set("id", "ver-1")

	c, w := makeArtifactTestContext(t, m)
	ctrl.versionRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for removing version, got %d, body: %s", w.Code, w.Body.String())
	}
}
