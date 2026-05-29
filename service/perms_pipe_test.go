package service

import (
	"testing"

	"github.com/gokins/gokins/model"
)

func TestPipePermIsAdmin(t *testing.T) {
	adminUser := &model.TUser{Id: "admin"}
	regularUser := &model.TUser{Id: "user1"}

	tests := []struct {
		name string
		pp   *PipePerm
		want bool
	}{
		{
			name: "admin user",
			pp:   &PipePerm{lgusr: adminUser},
			want: true,
		},
		{
			name: "regular user",
			pp:   &PipePerm{lgusr: regularUser},
			want: false,
		},
		{
			name: "nil user",
			pp:   &PipePerm{lgusr: nil},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pp.IsAdmin()
			if got != tt.want {
				t.Errorf("PipePerm.IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipePermIsPipeOwner(t *testing.T) {
	user := &model.TUser{Id: "user1"}
	otherUser := &model.TUser{Id: "user2"}

	tests := []struct {
		name string
		pp   *PipePerm
		want bool
	}{
		{
			name: "user is pipe owner",
			pp: &PipePerm{
				lgusr: user,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "user1"},
			},
			want: true,
		},
		{
			name: "user is not pipe owner",
			pp: &PipePerm{
				lgusr: user,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "user2"},
			},
			want: false,
		},
		{
			name: "nil pipe",
			pp: &PipePerm{
				lgusr: user,
				pipe:  nil,
			},
			want: false,
		},
		{
			name: "nil user",
			pp: &PipePerm{
				lgusr: nil,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "user1"},
			},
			want: false,
		},
		{
			name: "different user",
			pp: &PipePerm{
				lgusr: otherUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "user1"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pp.IsPipeOwner()
			if got != tt.want {
				t.Errorf("PipePerm.IsPipeOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipePermCanRead(t *testing.T) {
	adminUser := &model.TUser{Id: "admin"}
	regularUser := &model.TUser{Id: "user1"}
	orgMember := &model.TUser{Id: "member1"}

	tests := []struct {
		name string
		pp   *PipePerm
		want bool
	}{
		{
			name: "admin can read",
			pp: &PipePerm{
				lgusr: adminUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "someone"},
			},
			want: true,
		},
		{
			name: "pipe owner can read",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "user1"},
			},
			want: true,
		},
		{
			name: "org member can read (via CurUid)",
			pp: &PipePerm{
				lgusr: orgMember,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgId: "org1", CurUid: "member1"},
				},
			},
			want: true,
		},
		{
			name: "public org allows read",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgId: "org1", OrgPublic: 1},
				},
			},
			want: true,
		},
		{
			name: "non-member private pipe cannot read",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgId: "org1", OrgPublic: 0, OrgUid: "someone-else"},
				},
			},
			want: false,
		},
		{
			name: "nil user with no perms cannot read",
			pp: &PipePerm{
				lgusr: nil,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: nil,
			},
			want: false,
		},
		{
			name: "empty perms list, non-owner non-admin",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pp.CanRead()
			if got != tt.want {
				t.Errorf("PipePerm.CanRead() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipePermCanWrite(t *testing.T) {
	adminUser := &model.TUser{Id: "admin"}
	regularUser := &model.TUser{Id: "user1"}

	tests := []struct {
		name string
		pp   *PipePerm
		want bool
	}{
		{
			name: "admin can write",
			pp: &PipePerm{
				lgusr: adminUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "someone"},
			},
			want: true,
		},
		{
			name: "pipe owner can write",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "user1"},
			},
			want: true,
		},
		{
			name: "member with admin perm can write",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgId: "org1", CurUid: "user1", PermAdm: 1},
				},
			},
			want: true,
		},
		{
			name: "member with rw perm can write",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgId: "org1", CurUid: "user1", PermRw: 1},
				},
			},
			want: true,
		},
		{
			name: "member without write perms cannot write",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgId: "org1", CurUid: "user1", PermAdm: 0, PermRw: 0},
				},
			},
			want: false,
		},
		{
			name: "non-member cannot write even if public",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgId: "org1", OrgPublic: 1, OrgUid: "someone"},
				},
			},
			want: false,
		},
		{
			name: "nil user cannot write",
			pp: &PipePerm{
				lgusr: nil,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pp.CanWrite()
			if got != tt.want {
				t.Errorf("PipePerm.CanWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipePermCanExec(t *testing.T) {
	adminUser := &model.TUser{Id: "admin"}
	regularUser := &model.TUser{Id: "user1"}

	tests := []struct {
		name string
		pp   *PipePerm
		want bool
	}{
		{
			name: "admin can exec",
			pp: &PipePerm{
				lgusr: adminUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "someone"},
			},
			want: true,
		},
		{
			name: "pipe owner can exec",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "user1"},
			},
			want: true,
		},
		{
			name: "member with exec perm can exec",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgId: "org1", CurUid: "user1", PermExec: 1},
				},
			},
			want: true,
		},
		{
			name: "member with admin perm can exec",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgId: "org1", CurUid: "user1", PermAdm: 1, PermExec: 0},
				},
			},
			want: true,
		},
		{
			name: "member without exec perm cannot exec",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgId: "org1", CurUid: "user1", PermAdm: 0, PermExec: 0},
				},
			},
			want: false,
		},
		{
			name: "non-member cannot exec",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgId: "org1", OrgPublic: 1, OrgUid: "someone"},
				},
			},
			want: false,
		},
		{
			name: "nil user cannot exec",
			pp: &PipePerm{
				lgusr: nil,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pp.CanExec()
			if got != tt.want {
				t.Errorf("PipePerm.CanExec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipePermAccessors(t *testing.T) {
	user := &model.TUser{Id: "user1"}
	pipe := &model.TPipeline{Id: "pipe1"}

	pp := &PipePerm{
		lgusr: user,
		pipe:  pipe,
	}

	if pp.LgUser() != user {
		t.Error("LgUser() returned wrong user")
	}
	if pp.Pipeline() != pipe {
		t.Error("Pipeline() returned wrong pipeline")
	}

	// Test nil accessors
	ppNil := &PipePerm{}
	if ppNil.LgUser() != nil {
		t.Error("LgUser() should return nil")
	}
	if ppNil.Pipeline() != nil {
		t.Error("Pipeline() should return nil")
	}
}
