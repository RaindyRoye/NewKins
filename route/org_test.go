package route

import (
	"net/http"
	"testing"

	"github.com/gokins/gokins/bean"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	hbtp "github.com/mgr9525/HyperByte-Transfer-Protocol"
)

func TestOrgController_list_Empty(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := OrgController{}
	m := &hbtp.Map{}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_list_WithPublicOrg(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Public Org", 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOrgController_list_WithSearch(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Alpha Org", 1)
	seedOrg(t, "org2", "u1", "Beta Org", 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"q": "Alpha"}
	ctrl.list(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOrgController_new_EmptyName(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := OrgController{}
	m := &hbtp.Map{"name": "", "desc": "test", "public": true}
	ctrl.new(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_new_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	// User needs perm_org=1 to create orgs
	ui := &model.TUserInfo{Id: "u1", PermOrg: 1}
	_, _ = comm.Db.InsertOne(ui)

	c, w := makeRouteGinCtx(t, user)

	ctrl := OrgController{}
	m := &hbtp.Map{"name": "New Org", "desc": "Description", "public": true}
	ctrl.new(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_new_NoPermission(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	// User without perm_org
	ui := &model.TUserInfo{Id: "u1", PermOrg: 0}
	_, _ = comm.Db.InsertOne(ui)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"name": "Test Org"}
	ctrl.new(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOrgController_info_EmptyId(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := OrgController{}
	m := &hbtp.Map{"id": ""}
	ctrl.info(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_info_NotFound(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := OrgController{}
	m := &hbtp.Map{"id": "nonexistent"}
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_info_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"id": "org1"}
	ctrl.info(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_info_DeletedOrg(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	org := seedOrg(t, "org1", "u1", "Test Org", 1)
	org.Deleted = 1
	_, _ = comm.Db.Where("id=?", org.Id).Cols("deleted").Update(org)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"id": "org1"}
	ctrl.info(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_users_EmptyId(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := OrgController{}
	m := &hbtp.Map{"id": ""}
	ctrl.users(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_users_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)
	seedUserOrg(t, "u1", "org1", 1, 1, 1, 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"id": "org1"}
	ctrl.users(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOrgController_save_NoPermission(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "other", "Test Org", 0)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"id": "org1", "name": "Updated"}
	ctrl.save(c, m)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestOrgController_save_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"id": "org1", "name": "Updated Org", "desc": "New desc", "public": true}
	ctrl.save(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestOrgController_rm_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"id": "org1"}
	ctrl.rm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOrgController_rm_NotFound(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := OrgController{}
	m := &hbtp.Map{"id": "nonexistent"}
	ctrl.rm(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestOrgController_vars_EmptyOrgId(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := OrgController{}
	m := &hbtp.Map{"orgId": ""}
	ctrl.vars(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_vars_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"orgId": "org1"}
	ctrl.vars(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOrgController_userEdit_SelfEdit(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)
	seedUserOrg(t, "u1", "org1", 1, 1, 1, 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"id": "org1", "uid": "u1", "adm": true}
	ctrl.userEdit(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (can't edit self)", w.Code, http.StatusConflict)
	}
}

func TestOrgController_userRm_SelfRemove(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)
	seedUserOrg(t, "u1", "org1", 1, 1, 1, 1)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"id": "org1", "uid": "u1"}
	ctrl.userRm(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (can't remove self)", w.Code, http.StatusConflict)
	}
}

func TestOrgController_pipeAdd_AlreadyExists(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	// Insert existing pipeline
	pipe := &model.TPipeline{Id: "pipe1", Name: "Test Pipe", Uid: "u1"}
	_, _ = comm.Db.InsertOne(pipe)

	// Add to org
	op := &model.TOrgPipe{OrgId: "org1", PipeId: "pipe1"}
	_, _ = comm.Db.InsertOne(op)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"id": "org1", "pipeId": "pipe1"}
	ctrl.pipeAdd(c, m)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestOrgController_pipeRm_Success(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	seedOrg(t, "org1", "u1", "Test Org", 1)

	op := &model.TOrgPipe{OrgId: "org1", PipeId: "pipe1"}
	_, _ = comm.Db.InsertOne(op)

	c, w := makeRouteGinCtx(t, user)
	ctrl := OrgController{}
	m := &hbtp.Map{"id": "org1", "pipeId": "pipe1"}
	ctrl.pipeRm(c, m)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestOrgController_varSave_EmptyParams(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := OrgController{}
	pv := &bean.OrgVar{OrgId: "", Name: "", Value: ""}
	ctrl.varSave(c, pv)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varDel_InvalidAid(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := OrgController{}
	m := &hbtp.Map{"aid": int64(0)}
	ctrl.varDel(c, m)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestOrgController_varDel_NotFound(t *testing.T) {
	setupRouteTestDB(t)
	user := seedUser(t, "u1", "user1", "User One", 1)
	c, w := makeRouteGinCtx(t, user)

	ctrl := OrgController{}
	m := &hbtp.Map{"aid": int64(999)}
	ctrl.varDel(c, m)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
