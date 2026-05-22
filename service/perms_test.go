package service

import (
	"testing"

	"github.com/gokins/gokins/model"
)

func TestIsAdmin(t *testing.T) {
	tests := []struct {
		name string
		user *model.TUser
		want bool
	}{
		{
			name: "admin user",
			user: &model.TUser{Id: "admin"},
			want: true,
		},
		{
			name: "non-admin user",
			user: &model.TUser{Id: "regular-user"},
			want: false,
		},
		{
			name: "empty id",
			user: &model.TUser{Id: ""},
			want: false,
		},
		{
			name: "Admin with capital A is not admin",
			user: &model.TUser{Id: "Admin"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAdmin(tt.user)
			if got != tt.want {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckUPermission(t *testing.T) {
	adminUser := &model.TUser{Id: "admin", Name: "admin"}
	regularUser := &model.TUser{Id: "user1", Name: "alice"}

	tests := []struct {
		name  string
		user  *model.TUser
		perms string
		want  bool
	}{
		{
			name:  "nil user returns false",
			user:  nil,
			perms: "common",
			want:  false,
		},
		{
			name:  "common permission always granted",
			user:  regularUser,
			perms: "common",
			want:  true,
		},
		{
			name:  "admin permission for admin user",
			user:  adminUser,
			perms: "admin",
			want:  true,
		},
		{
			name:  "admin permission for non-admin user",
			user:  regularUser,
			perms: "admin",
			want:  false,
		},
		{
			name:  "unknown permission",
			user:  regularUser,
			perms: "superuser",
			want:  false,
		},
		{
			name:  "admin user with common permission",
			user:  adminUser,
			perms: "common",
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckUPermission(tt.user, tt.perms)
			if got != tt.want {
				t.Errorf("CheckUPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPermIsAdmin(t *testing.T) {
	adminUser := &model.TUser{Id: "admin"}
	regularUser := &model.TUser{Id: "user1"}

	tests := []struct {
		name string
		op   *OrgPerm
		want bool
	}{
		{
			name: "admin user",
			op:   &OrgPerm{lgusr: adminUser},
			want: true,
		},
		{
			name: "regular user",
			op:   &OrgPerm{lgusr: regularUser},
			want: false,
		},
		{
			name: "nil user",
			op:   &OrgPerm{lgusr: nil},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.op.IsAdmin()
			if got != tt.want {
				t.Errorf("OrgPerm.IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPermIsOrgOwner(t *testing.T) {
	user := &model.TUser{Id: "user1"}
	otherUser := &model.TUser{Id: "user2"}

	tests := []struct {
		name string
		op   *OrgPerm
		want bool
	}{
		{
			name: "user is org owner",
			op: &OrgPerm{
				lgusr: user,
				org:   &model.TOrg{Id: "org1", Uid: "user1"},
			},
			want: true,
		},
		{
			name: "user is not org owner",
			op: &OrgPerm{
				lgusr: user,
				org:   &model.TOrg{Id: "org1", Uid: "user2"},
			},
			want: false,
		},
		{
			name: "nil org",
			op: &OrgPerm{
				lgusr: user,
				org:   nil,
			},
			want: false,
		},
		{
			name: "nil user",
			op: &OrgPerm{
				lgusr: nil,
				org:   &model.TOrg{Id: "org1", Uid: "user1"},
			},
			want: false,
		},
		{
			name: "different user",
			op: &OrgPerm{
				lgusr: otherUser,
				org:   &model.TOrg{Id: "org1", Uid: "user1"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.op.IsOrgOwner()
			if got != tt.want {
				t.Errorf("OrgPerm.IsOrgOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPermIsOrgPublic(t *testing.T) {
	tests := []struct {
		name string
		op   *OrgPerm
		want bool
	}{
		{
			name: "public org",
			op:   &OrgPerm{org: &model.TOrg{Public: 1}},
			want: true,
		},
		{
			name: "private org",
			op:   &OrgPerm{org: &model.TOrg{Public: 0}},
			want: false,
		},
		{
			name: "nil org",
			op:   &OrgPerm{org: nil},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.op.IsOrgPublic()
			if got != tt.want {
				t.Errorf("OrgPerm.IsOrgPublic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPermIsOrgAdmin(t *testing.T) {
	adminUser := &model.TUser{Id: "admin"}
	regularUser := &model.TUser{Id: "user1"}
	orgOwner := &model.TUser{Id: "owner1"}

	tests := []struct {
		name string
		op   *OrgPerm
		want bool
	}{
		{
			name: "system admin",
			op: &OrgPerm{
				lgusr: adminUser,
				org:   &model.TOrg{Id: "org1", Uid: "someone"},
			},
			want: true,
		},
		{
			name: "org owner",
			op: &OrgPerm{
				lgusr: orgOwner,
				org:   &model.TOrg{Id: "org1", Uid: "owner1"},
			},
			want: true,
		},
		{
			name: "org admin via user_org",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "someone"},
				usrOrg: &model.TUserOrg{PermAdm: 1},
			},
			want: true,
		},
		{
			name: "regular member",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "someone"},
				usrOrg: &model.TUserOrg{PermAdm: 0},
			},
			want: false,
		},
		{
			name: "non-member",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "someone"},
				usrOrg: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.op.IsOrgAdmin()
			if got != tt.want {
				t.Errorf("OrgPerm.IsOrgAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPermCanRead(t *testing.T) {
	regularUser := &model.TUser{Id: "user1"}

	tests := []struct {
		name string
		op   *OrgPerm
		want bool
	}{
		{
			name: "public org allows read",
			op: &OrgPerm{
				lgusr: regularUser,
				org:   &model.TOrg{Id: "org1", Public: 1},
			},
			want: true,
		},
		{
			name: "org admin can read",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "user1"},
				usrOrg: &model.TUserOrg{},
			},
			want: true,
		},
		{
			name: "member can read",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Public: 0},
				usrOrg: &model.TUserOrg{},
			},
			want: true,
		},
		{
			name: "non-member private org cannot read",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Public: 0, Uid: "other"},
				usrOrg: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.op.CanRead()
			if got != tt.want {
				t.Errorf("OrgPerm.CanRead() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPermCanWrite(t *testing.T) {
	adminUser := &model.TUser{Id: "admin"}
	regularUser := &model.TUser{Id: "user1"}

	tests := []struct {
		name string
		op   *OrgPerm
		want bool
	}{
		{
			name: "admin can write",
			op: &OrgPerm{
				lgusr: adminUser,
				org:   &model.TOrg{Id: "org1"},
			},
			want: true,
		},
		{
			name: "member with write permission",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "other"},
				usrOrg: &model.TUserOrg{PermRw: 1},
			},
			want: true,
		},
		{
			name: "member without write permission",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "other"},
				usrOrg: &model.TUserOrg{PermRw: 0},
			},
			want: false,
		},
		{
			name: "non-member cannot write",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "other", Public: 1},
				usrOrg: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.op.CanWrite()
			if got != tt.want {
				t.Errorf("OrgPerm.CanWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPermCanExec(t *testing.T) {
	regularUser := &model.TUser{Id: "user1"}

	tests := []struct {
		name string
		op   *OrgPerm
		want bool
	}{
		{
			name: "member with exec permission",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "other"},
				usrOrg: &model.TUserOrg{PermExec: 1},
			},
			want: true,
		},
		{
			name: "member without exec permission",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "other"},
				usrOrg: &model.TUserOrg{PermExec: 0},
			},
			want: false,
		},
		{
			name: "org admin can exec",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "user1"},
				usrOrg: &model.TUserOrg{PermExec: 0},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.op.CanExec()
			if got != tt.want {
				t.Errorf("OrgPerm.CanExec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPermCanDownload(t *testing.T) {
	regularUser := &model.TUser{Id: "user1"}

	tests := []struct {
		name string
		op   *OrgPerm
		want bool
	}{
		{
			name: "member with download permission",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "other"},
				usrOrg: &model.TUserOrg{PermDown: 1},
			},
			want: true,
		},
		{
			name: "member without download permission",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "other"},
				usrOrg: &model.TUserOrg{PermDown: 0},
			},
			want: false,
		},
		{
			name: "org admin can download",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "user1"},
				usrOrg: &model.TUserOrg{PermDown: 0},
			},
			want: true,
		},
		{
			name: "non-member cannot download",
			op: &OrgPerm{
				lgusr:  regularUser,
				org:    &model.TOrg{Id: "org1", Uid: "other"},
				usrOrg: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.op.CanDownload()
			if got != tt.want {
				t.Errorf("OrgPerm.CanDownload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPermAccessors(t *testing.T) {
	user := &model.TUser{Id: "user1"}
	org := &model.TOrg{Id: "org1"}
	userOrg := &model.TUserOrg{Uid: "user1", OrgId: "org1"}

	op := &OrgPerm{
		lgusr:  user,
		org:    org,
		usrOrg: userOrg,
	}

	if op.LgUser() != user {
		t.Error("LgUser() returned wrong user")
	}
	if op.Org() != org {
		t.Error("Org() returned wrong org")
	}
	if op.UserOrg() != userOrg {
		t.Error("UserOrg() returned wrong userOrg")
	}

	// Test nil accessors
	opNil := &OrgPerm{}
	if opNil.LgUser() != nil {
		t.Error("LgUser() should return nil")
	}
	if opNil.Org() != nil {
		t.Error("Org() should return nil")
	}
	if opNil.UserOrg() != nil {
		t.Error("UserOrg() should return nil")
	}
}
