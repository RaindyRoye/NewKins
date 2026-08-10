package route

import (
	"net/http"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

func TestArtifactController_orgList_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	_ = seedOrg(t, "org1", "u1", "Test Org", 1)

	// Create artifact repo
	repo := &model.TArtifactory{
		Id:      "repo1",
		Uid:     "u1",
		OrgId:   "org1",
		Name:    "test-repo",
		Deleted: 0,
	}
	_, _ = comm.Db.InsertOne(repo)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{"orgId": "org1"}
	ctrl.orgList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestArtifactController_orgList_EmptyOrgId(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := ArtifactController{}
	m := &hbtp.Map{"orgId": ""}
	ctrl.orgList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestArtifactController_orgList_NotFound(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := ArtifactController{}
	m := &hbtp.Map{"orgId": "nonexistent"}
	ctrl.orgList(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_orgList_WithSearch(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	repo := &model.TArtifactory{
		Id:    "repo1",
		Uid:   "u1",
		OrgId: "org1",
		Name:  "test-repo",
	}
	_, _ = comm.Db.InsertOne(repo)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{"orgId": "org1", "q": "test"}
	ctrl.orgList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestArtifactController_info_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	repo := &model.TArtifactory{
		Id:      "repo1",
		Uid:     "u1",
		OrgId:   "org1",
		Name:    "test-repo",
		Deleted: 0,
	}
	_, _ = comm.Db.InsertOne(repo)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "repo1"}
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestArtifactController_info_NotFound(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "nonexistent"}
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_info_Deleted(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	repo := &model.TArtifactory{
		Id:      "repo1",
		Uid:     "u1",
		OrgId:   "org1",
		Name:    "test-repo",
		Deleted: 1,
	}
	_, _ = comm.Db.InsertOne(repo)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "repo1"}
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_edit_Create(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{
		"orgId": "org1",
		"name":  "new-repo",
		"desc":  "New repository",
	}
	ctrl.edit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestArtifactController_edit_Update(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	repo := &model.TArtifactory{
		Id:    "repo1",
		Uid:   "u1",
		OrgId: "org1",
		Name:  "test-repo",
	}
	_, _ = comm.Db.InsertOne(repo)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{
		"id":    "repo1",
		"orgId": "org1",
		"name":  "updated-repo",
		"desc":  "Updated description",
	}
	ctrl.edit(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestArtifactController_edit_EmptyName(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{
		"orgId": "org1",
		"name":  "",
	}
	ctrl.edit(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestArtifactController_edit_NoPermission(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "other", "Test Org", 0)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{
		"orgId": "org1",
		"name":  "test-repo",
	}
	ctrl.edit(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestArtifactController_rm_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	repo := &model.TArtifactory{
		Id:    "repo1",
		Uid:   "u1",
		OrgId: "org1",
		Name:  "test-repo",
	}
	_, _ = comm.Db.InsertOne(repo)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "repo1"}
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestArtifactController_rm_NotFound(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "nonexistent"}
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_packageList_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	repo := &model.TArtifactory{
		Id:    "repo1",
		Uid:   "u1",
		OrgId: "org1",
		Name:  "test-repo",
	}
	_, _ = comm.Db.InsertOne(repo)

	pkg := &model.TArtifactPackage{
		Id:      "pkg1",
		RepoId:  "repo1",
		Name:    "test-package",
		Deleted: 0,
	}
	_, _ = comm.Db.InsertOne(pkg)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{"repoId": "repo1"}
	ctrl.packageList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestArtifactController_packageList_EmptyRepoId(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := ArtifactController{}
	m := &hbtp.Map{"repoId": ""}
	ctrl.packageList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestArtifactController_versionList_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)

	pkg := &model.TArtifactPackage{
		Id:     "pkg1",
		RepoId: "repo1",
		Name:   "test-package",
	}
	_, _ = comm.Db.InsertOne(pkg)

	ver := &model.TArtifactVersion{
		Id:        "ver1",
		PackageId: "pkg1",
		Name:      "1.0.0",
	}
	_, _ = comm.Db.InsertOne(ver)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{"packId": "pkg1"}
	ctrl.versionList(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestArtifactController_versionList_EmptyPackId(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := ArtifactController{}
	m := &hbtp.Map{"packId": ""}
	ctrl.versionList(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestArtifactController_versionInfos_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	repo := &model.TArtifactory{
		Id:    "repo1",
		Uid:   "u1",
		OrgId: "org1",
		Name:  "test-repo",
	}
	_, _ = comm.Db.InsertOne(repo)

	ver := &model.TArtifactVersion{
		Id:        "ver1",
		RepoId:    "repo1",
		PackageId: "pkg1",
		Name:      "1.0.0",
	}
	_, _ = comm.Db.InsertOne(ver)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "ver1"}
	ctrl.versionInfos(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestArtifactController_versionInfos_NotFound(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "nonexistent"}
	ctrl.versionInfos(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_versionUrl_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	repo := &model.TArtifactory{
		Id:    "repo1",
		Uid:   "u1",
		OrgId: "org1",
		Name:  "test-repo",
	}
	_, _ = comm.Db.InsertOne(repo)

	ver := &model.TArtifactVersion{
		Id:        "ver1",
		RepoId:    "repo1",
		PackageId: "pkg1",
		Name:      "1.0.0",
	}
	_, _ = comm.Db.InsertOne(ver)

	seedUserOrg(t, "u1", "org1", 1, 1, 1, 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "ver1", "path": "test.txt"}
	ctrl.versionUrl(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestArtifactController_versionUrl_NotFound(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "nonexistent"}
	ctrl.versionUrl(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_versionSave_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	repo := &model.TArtifactory{
		Id:    "repo1",
		Uid:   "u1",
		OrgId: "org1",
		Name:  "test-repo",
	}
	_, _ = comm.Db.InsertOne(repo)

	ver := &model.TArtifactVersion{
		Id:        "ver1",
		RepoId:    "repo1",
		PackageId: "pkg1",
		Name:      "1.0.0",
		Version:   "1.0.0",
	}
	_, _ = comm.Db.InsertOne(ver)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{
		"id":      "ver1",
		"version": "1.0.1",
		"desc":    "Updated description",
		"ispre":   false,
	}
	ctrl.versionSave(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestArtifactController_versionSave_NotFound(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "nonexistent"}
	ctrl.versionSave(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestArtifactController_versionRm_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	repo := &model.TArtifactory{
		Id:    "repo1",
		Uid:   "u1",
		OrgId: "org1",
		Name:  "test-repo",
	}
	_, _ = comm.Db.InsertOne(repo)

	ver := &model.TArtifactVersion{
		Id:        "ver1",
		RepoId:    "repo1",
		PackageId: "pkg1",
		Name:      "1.0.0",
	}
	_, _ = comm.Db.InsertOne(ver)

	c, w := makeRouteGinCtx(t, user)
	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "ver1"}
	ctrl.versionRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestArtifactController_versionRm_NotFound(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := ArtifactController{}
	m := &hbtp.Map{"id": "nonexistent"}
	ctrl.versionRm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
