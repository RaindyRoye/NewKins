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
	otherUser := &model.TUser{Id: "user2"}

	tests := []struct {
		name string
		pp   *PipePerm
		want bool
	}{
		{
			name: "admin can always read",
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
			name: "org member can read",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgUid: "someone", CurUid: "user1"},
				},
			},
			want: true,
		},
		{
			name: "public org pipeline allows read",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgPublic: 1},
				},
			},
			want: true,
		},
		{
			name: "private org, non-member cannot read",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgPublic: 0, OrgUid: "orgowner"},
				},
			},
			want: false,
		},
		{
			name: "no perms and not owner",
			pp: &PipePerm{
				lgusr: otherUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "user1"},
				perms: nil,
			},
			want: false,
		},
		{
			name: "nil pipe",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  nil,
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
			name: "admin can always write",
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
			name: "org admin can write",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{CurUid: "user1", PermAdm: 1},
				},
			},
			want: true,
		},
		{
			name: "org member with write perm can write",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{CurUid: "user1", PermRw: 1},
				},
			},
			want: true,
		},
		{
			name: "org member without write perm cannot write",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{CurUid: "user1", PermRw: 0, PermAdm: 0},
				},
			},
			want: false,
		},
		{
			name: "non-member cannot write",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: nil,
			},
			want: false,
		},
		{
			name: "public org does not grant write",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{OrgPublic: 1, CurUid: ""},
				},
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
			name: "admin can always exec",
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
			name: "org admin can exec",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{CurUid: "user1", PermAdm: 1, PermExec: 0},
				},
			},
			want: true,
		},
		{
			name: "org member with exec perm can exec",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{CurUid: "user1", PermExec: 1},
				},
			},
			want: true,
		},
		{
			name: "org member without exec perm cannot exec",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{CurUid: "user1", PermExec: 0, PermAdm: 0},
				},
			},
			want: false,
		},
		{
			name: "non-member cannot exec",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: nil,
			},
			want: false,
		},
		{
			name: "member with write but no exec cannot exec",
			pp: &PipePerm{
				lgusr: regularUser,
				pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
				perms: []*UserPipeOrgPerm{
					{CurUid: "user1", PermRw: 1, PermExec: 0, PermAdm: 0},
				},
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

	pp := &PipePerm{lgusr: user, pipe: pipe}

	if pp.LgUser() != user {
		t.Error("LgUser() should return the user")
	}
	if pp.Pipeline() != pipe {
		t.Error("Pipeline() should return the pipeline")
	}

	// Test nil accessors
	pp2 := &PipePerm{}
	if pp2.LgUser() != nil {
		t.Error("LgUser() should return nil")
	}
	if pp2.Pipeline() != nil {
		t.Error("Pipeline() should return nil")
	}
}

func TestPipePermMultipleOrgPerms(t *testing.T) {
	regularUser := &model.TUser{Id: "user1"}

	// Test with multiple org permissions entries
	pp := &PipePerm{
		lgusr: regularUser,
		pipe:  &model.TPipeline{Id: "pipe1", Uid: "other"},
		perms: []*UserPipeOrgPerm{
			{OrgUid: "org1owner", OrgPublic: 0, CurUid: ""},
			{OrgUid: "org2owner", OrgPublic: 0, CurUid: "user1", PermRw: 1, PermExec: 0},
			{OrgUid: "org3owner", OrgPublic: 1, CurUid: ""},
		},
	}

	// Should be able to read (org2 has membership, org3 is public)
	if !pp.CanRead() {
		t.Error("should be able to read via org membership or public org")
	}

	// Should be able to write (org2 has PermRw)
	if !pp.CanWrite() {
		t.Error("should be able to write via org2 PermRw")
	}

	// Should NOT be able to exec (no org has PermExec)
	if pp.CanExec() {
		t.Error("should not be able to exec without PermExec")
	}
}
