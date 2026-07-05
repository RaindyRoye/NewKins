package service

import (
	"testing"

	"github.com/gokins/gokins/model"
)

// --- OrgPerm edge case tests ---

func TestOrgPerm_CanRead_OwnerNotPublic(t *testing.T) {
	// Owner of private org should be able to read
	user := &model.TUser{Id: "u1"}
	org := &model.TOrg{Id: "o1", Uid: "u1", Public: 0}
	c := &OrgPerm{lgusr: user, org: org}
	if !c.CanRead() {
		t.Error("owner of private org should be able to read")
	}
}

func TestOrgPerm_CanWrite_OwnerNotAdmin(t *testing.T) {
	// Owner is not automatically an admin unless they have PermAdm
	user := &model.TUser{Id: "u1"}
	org := &model.TOrg{Id: "o1", Uid: "u1", Public: 0}
	c := &OrgPerm{lgusr: user, org: org}
	if !c.IsOrgOwner() {
		t.Error("expected IsOrgOwner to be true")
	}
	// Owner should have admin access via IsOrgAdmin
	if !c.CanWrite() {
		t.Error("org owner should be able to write")
	}
}

func TestOrgPerm_CanExec_MemberWithPermAdm(t *testing.T) {
	// Member with PermAdm should have exec permission
	user := &model.TUser{Id: "u1"}
	org := &model.TOrg{Id: "o1", Uid: "u2"}
	userOrg := &model.TUserOrg{PermAdm: 1, PermExec: 0}
	c := &OrgPerm{lgusr: user, org: org, usrOrg: userOrg}
	if !c.IsOrgAdmin() {
		t.Error("member with PermAdm should be org admin")
	}
	if !c.CanExec() {
		t.Error("org admin should be able to exec")
	}
}

func TestOrgPerm_CanDownload_Owner(t *testing.T) {
	// Owner should have download permission via IsOrgAdmin
	user := &model.TUser{Id: "u1"}
	org := &model.TOrg{Id: "o1", Uid: "u1"}
	c := &OrgPerm{lgusr: user, org: org}
	if !c.CanDownload() {
		t.Error("org owner should be able to download")
	}
}

func TestOrgPerm_AdminOverride(t *testing.T) {
	// Global admin should have all permissions regardless of org membership
	admin := &model.TUser{Id: "admin"}
	org := &model.TOrg{Id: "o1", Uid: "u2", Public: 0}
	c := &OrgPerm{lgusr: admin, org: org, usrOrg: nil}
	
	if !c.IsAdmin() {
		t.Error("expected IsAdmin to be true")
	}
	if !c.IsOrgAdmin() {
		t.Error("global admin should be org admin")
	}
	if !c.CanRead() {
		t.Error("global admin should be able to read")
	}
	if !c.CanWrite() {
		t.Error("global admin should be able to write")
	}
	if !c.CanExec() {
		t.Error("global admin should be able to exec")
	}
	if !c.CanDownload() {
		t.Error("global admin should be able to download")
	}
}

func TestOrgPerm_NilUserOrg(t *testing.T) {
	// Non-member with nil usrOrg should not have member permissions
	user := &model.TUser{Id: "u1"}
	org := &model.TOrg{Id: "o1", Uid: "u2", Public: 0}
	c := &OrgPerm{lgusr: user, org: org, usrOrg: nil}
	
	if c.CanWrite() {
		t.Error("non-member should not be able to write")
	}
	if c.CanExec() {
		t.Error("non-member should not be able to exec")
	}
	if c.CanDownload() {
		t.Error("non-member should not be able to download")
	}
}

// --- PipePerm edge case tests ---

func TestPipePerm_CanRead_MultipleOrgs(t *testing.T) {
	// User in multiple orgs should be able to read if any org is public
	user := &model.TUser{Id: "u1"}
	pipe := &model.TPipeline{Id: "p1", Uid: "u2"}
	perms := []*UserPipeOrgPerm{
		{OrgId: "o1", OrgPublic: 0, CurUid: ""},
		{OrgId: "o2", OrgPublic: 1, CurUid: ""}, // public org
	}
	c := &PipePerm{lgusr: user, pipe: pipe, perms: perms}
	if !c.CanRead() {
		t.Error("should be able to read via public org")
	}
}

func TestPipePerm_CanWrite_OrgOwner(t *testing.T) {
	// User who is org owner (OrgUid matches) should have write permission
	user := &model.TUser{Id: "u1"}
	pipe := &model.TPipeline{Id: "p1", Uid: "u2"}
	perms := []*UserPipeOrgPerm{
		{OrgId: "o1", OrgUid: "u1"}, // user is org owner
	}
	c := &PipePerm{lgusr: user, pipe: pipe, perms: perms}
	if !c.CanWrite() {
		t.Error("org owner should be able to write")
	}
}

func TestPipePerm_CanExec_OrgOwner(t *testing.T) {
	// User who is org owner should have exec permission
	user := &model.TUser{Id: "u1"}
	pipe := &model.TPipeline{Id: "p1", Uid: "u2"}
	perms := []*UserPipeOrgPerm{
		{OrgId: "o1", OrgUid: "u1"},
	}
	c := &PipePerm{lgusr: user, pipe: pipe, perms: perms}
	if !c.CanExec() {
		t.Error("org owner should be able to exec")
	}
}

func TestPipePerm_CanWrite_MemberWithBothPerms(t *testing.T) {
	// Member with both PermAdm and PermRw should have write permission
	user := &model.TUser{Id: "u1"}
	pipe := &model.TPipeline{Id: "p1", Uid: "u2"}
	perms := []*UserPipeOrgPerm{
		{OrgId: "o1", CurUid: "u1", PermAdm: 1, PermRw: 1},
	}
	c := &PipePerm{lgusr: user, pipe: pipe, perms: perms}
	if !c.CanWrite() {
		t.Error("member with PermAdm and PermRw should be able to write")
	}
}

func TestPipePerm_CanExec_MemberWithPermAdm(t *testing.T) {
	// Member with PermAdm should have exec permission
	user := &model.TUser{Id: "u1"}
	pipe := &model.TPipeline{Id: "p1", Uid: "u2"}
	perms := []*UserPipeOrgPerm{
		{OrgId: "o1", CurUid: "u1", PermAdm: 1, PermExec: 0},
	}
	c := &PipePerm{lgusr: user, pipe: pipe, perms: perms}
	if !c.CanExec() {
		t.Error("member with PermAdm should be able to exec")
	}
}

func TestPipePerm_CanRead_NilUser(t *testing.T) {
	// Nil user should only be able to read if org is public
	pipe := &model.TPipeline{Id: "p1", Uid: "u2"}
	
	// Public org
	perms := []*UserPipeOrgPerm{{OrgPublic: 1}}
	c := &PipePerm{lgusr: nil, pipe: pipe, perms: perms}
	if !c.CanRead() {
		t.Error("nil user should be able to read public org pipeline")
	}
	
	// Private org
	perms2 := []*UserPipeOrgPerm{{OrgPublic: 0}}
	c2 := &PipePerm{lgusr: nil, pipe: pipe, perms: perms2}
	if c2.CanRead() {
		t.Error("nil user should not be able to read private org pipeline")
	}
}

func TestPipePerm_CanWrite_NilUser(t *testing.T) {
	// Nil user should not be able to write even to public org
	pipe := &model.TPipeline{Id: "p1", Uid: "u2"}
	perms := []*UserPipeOrgPerm{{OrgPublic: 1}}
	c := &PipePerm{lgusr: nil, pipe: pipe, perms: perms}
	if c.CanWrite() {
		t.Error("nil user should not be able to write")
	}
}

func TestPipePerm_CanExec_NilUser(t *testing.T) {
	// Nil user should not be able to exec even to public org
	pipe := &model.TPipeline{Id: "p1", Uid: "u2"}
	perms := []*UserPipeOrgPerm{{OrgPublic: 1}}
	c := &PipePerm{lgusr: nil, pipe: pipe, perms: perms}
	if c.CanExec() {
		t.Error("nil user should not be able to exec")
	}
}

func TestPipePerm_NilPipeline(t *testing.T) {
	// Nil pipeline should not grant any permissions except admin
	user := &model.TUser{Id: "u1"}
	c := &PipePerm{lgusr: user, pipe: nil, perms: nil}
	
	if c.IsPipeOwner() {
		t.Error("should not be pipe owner with nil pipeline")
	}
	if c.CanRead() {
		t.Error("should not be able to read with nil pipeline")
	}
	if c.CanWrite() {
		t.Error("should not be able to write with nil pipeline")
	}
	if c.CanExec() {
		t.Error("should not be able to exec with nil pipeline")
	}
}

func TestPipePerm_AdminWithNilPipeline(t *testing.T) {
	// Admin should have all permissions even with nil pipeline
	admin := &model.TUser{Id: "admin"}
	c := &PipePerm{lgusr: admin, pipe: nil, perms: nil}
	
	if !c.IsAdmin() {
		t.Error("expected IsAdmin to be true")
	}
	if !c.CanRead() {
		t.Error("admin should be able to read")
	}
	if !c.CanWrite() {
		t.Error("admin should be able to write")
	}
	if !c.CanExec() {
		t.Error("admin should be able to exec")
	}
}

// --- IsAdmin standalone tests ---

func TestIsAdmin_Standalone(t *testing.T) {
	admin := &model.TUser{Id: "admin"}
	if !IsAdmin(admin) {
		t.Error("user with Id='admin' should be admin")
	}
	
	user := &model.TUser{Id: "user1"}
	if IsAdmin(user) {
		t.Error("user with Id='user1' should not be admin")
	}
	
	empty := &model.TUser{Id: ""}
	if IsAdmin(empty) {
		t.Error("user with empty Id should not be admin")
	}
}


