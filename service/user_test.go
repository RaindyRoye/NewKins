package service

import (
	"testing"

	"github.com/gokins/gokins/model"
)

// --- OrgPerm Tests ---

func TestOrgPerm_IsAdmin(t *testing.T) {
	tests := []struct {
		name string
		user *model.TUser
		want bool
	}{
		{"admin user returns true", &model.TUser{Id: "admin"}, true},
		{"non-admin user returns false", &model.TUser{Id: "user1"}, false},
		{"nil user returns false", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OrgPerm{lgusr: tt.user}
			if got := c.IsAdmin(); got != tt.want {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPerm_IsOrgOwner(t *testing.T) {
	tests := []struct {
		name string
		user *model.TUser
		org  *model.TOrg
		want bool
	}{
		{
			"owner returns true",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u1"},
			true,
		},
		{
			"non-owner returns false",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u2"},
			false,
		},
		{"nil org returns false", &model.TUser{Id: "u1"}, nil, false},
		{"nil user returns false", nil, &model.TOrg{Id: "o1", Uid: "u1"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OrgPerm{lgusr: tt.user, org: tt.org}
			if got := c.IsOrgOwner(); got != tt.want {
				t.Errorf("IsOrgOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPerm_IsOrgPublic(t *testing.T) {
	tests := []struct {
		name string
		org  *model.TOrg
		want bool
	}{
		{"public org", &model.TOrg{Public: 1}, true},
		{"private org", &model.TOrg{Public: 0}, false},
		{"nil org", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OrgPerm{org: tt.org}
			if got := c.IsOrgPublic(); got != tt.want {
				t.Errorf("IsOrgPublic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPerm_IsOrgAdmin(t *testing.T) {
	tests := []struct {
		name   string
		user   *model.TUser
		org    *model.TOrg
		usrOrg *model.TUserOrg
		want   bool
	}{
		{
			"admin user is org admin",
			&model.TUser{Id: "admin"},
			&model.TOrg{Id: "o1"},
			nil,
			true,
		},
		{
			"org owner is org admin",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u1"},
			nil,
			true,
		},
		{
			"user with perm_adm is org admin",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u2"},
			&model.TUserOrg{PermAdm: 1},
			true,
		},
		{
			"regular user is not org admin",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u2"},
			&model.TUserOrg{PermAdm: 0},
			false,
		},
		{
			"user with no membership is not org admin",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u2"},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OrgPerm{lgusr: tt.user, org: tt.org, usrOrg: tt.usrOrg}
			if got := c.IsOrgAdmin(); got != tt.want {
				t.Errorf("IsOrgAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPerm_CanRead(t *testing.T) {
	tests := []struct {
		name   string
		user   *model.TUser
		org    *model.TOrg
		usrOrg *model.TUserOrg
		want   bool
	}{
		{
			"public org allows read for anyone",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Public: 1, Uid: "u2"},
			nil,
			true,
		},
		{
			"org admin can read",
			&model.TUser{Id: "admin"},
			&model.TOrg{Id: "o1", Public: 0},
			nil,
			true,
		},
		{
			"member can read",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Public: 0, Uid: "u2"},
			&model.TUserOrg{PermRw: 0},
			true,
		},
		{
			"non-member of private org cannot read",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Public: 0, Uid: "u2"},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OrgPerm{lgusr: tt.user, org: tt.org, usrOrg: tt.usrOrg}
			if got := c.CanRead(); got != tt.want {
				t.Errorf("CanRead() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPerm_CanWrite(t *testing.T) {
	tests := []struct {
		name   string
		user   *model.TUser
		org    *model.TOrg
		usrOrg *model.TUserOrg
		want   bool
	}{
		{
			"org admin can write",
			&model.TUser{Id: "admin"},
			&model.TOrg{Id: "o1"},
			nil,
			true,
		},
		{
			"member with perm_rw can write",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u2"},
			&model.TUserOrg{PermRw: 1},
			true,
		},
		{
			"member without perm_rw cannot write",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u2"},
			&model.TUserOrg{PermRw: 0},
			false,
		},
		{
			"public does not grant write",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Public: 1, Uid: "u2"},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OrgPerm{lgusr: tt.user, org: tt.org, usrOrg: tt.usrOrg}
			if got := c.CanWrite(); got != tt.want {
				t.Errorf("CanWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPerm_CanDownload(t *testing.T) {
	tests := []struct {
		name   string
		user   *model.TUser
		org    *model.TOrg
		usrOrg *model.TUserOrg
		want   bool
	}{
		{
			"org admin can download",
			&model.TUser{Id: "admin"},
			&model.TOrg{Id: "o1"},
			nil,
			true,
		},
		{
			"member with perm_down can download",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u2"},
			&model.TUserOrg{PermDown: 1},
			true,
		},
		{
			"member without perm_down cannot download",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u2"},
			&model.TUserOrg{PermDown: 0},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OrgPerm{lgusr: tt.user, org: tt.org, usrOrg: tt.usrOrg}
			if got := c.CanDownload(); got != tt.want {
				t.Errorf("CanDownload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPerm_CanExec(t *testing.T) {
	tests := []struct {
		name   string
		user   *model.TUser
		org    *model.TOrg
		usrOrg *model.TUserOrg
		want   bool
	}{
		{
			"org admin can exec",
			&model.TUser{Id: "admin"},
			&model.TOrg{Id: "o1"},
			nil,
			true,
		},
		{
			"org owner can exec",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u1"},
			nil,
			true,
		},
		{
			"member with perm_exec can exec",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u2"},
			&model.TUserOrg{PermExec: 1},
			true,
		},
		{
			"member without perm_exec cannot exec",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u2"},
			&model.TUserOrg{PermExec: 0},
			false,
		},
		{
			"non-member cannot exec",
			&model.TUser{Id: "u1"},
			&model.TOrg{Id: "o1", Uid: "u2"},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OrgPerm{lgusr: tt.user, org: tt.org, usrOrg: tt.usrOrg}
			if got := c.CanExec(); got != tt.want {
				t.Errorf("CanExec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgPerm_Accessors(t *testing.T) {
	usr := &model.TUser{Id: "u1"}
	org := &model.TOrg{Id: "o1"}
	usrOrg := &model.TUserOrg{PermAdm: 1}
	c := &OrgPerm{lgusr: usr, org: org, usrOrg: usrOrg}

	if c.LgUser() != usr {
		t.Errorf("LgUser() returned wrong user")
	}
	if c.Org() != org {
		t.Errorf("Org() returned wrong org")
	}
	if c.UserOrg() != usrOrg {
		t.Errorf("UserOrg() returned wrong userOrg")
	}
}

// --- PipePerm Tests ---

func TestPipePerm_IsAdmin(t *testing.T) {
	tests := []struct {
		name string
		user *model.TUser
		want bool
	}{
		{"admin user", &model.TUser{Id: "admin"}, true},
		{"non-admin user", &model.TUser{Id: "user1"}, false},
		{"nil user", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PipePerm{lgusr: tt.user}
			if got := c.IsAdmin(); got != tt.want {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipePerm_IsPipeOwner(t *testing.T) {
	tests := []struct {
		name string
		user *model.TUser
		pipe *model.TPipeline
		want bool
	}{
		{
			"pipe owner",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u1"},
			true,
		},
		{
			"not pipe owner",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			false,
		},
		{"nil pipe", &model.TUser{Id: "u1"}, nil, false},
		{"nil user", nil, &model.TPipeline{Id: "p1", Uid: "u1"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PipePerm{lgusr: tt.user, pipe: tt.pipe}
			if got := c.IsPipeOwner(); got != tt.want {
				t.Errorf("IsPipeOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipePerm_CanRead(t *testing.T) {
	tests := []struct {
		name  string
		user  *model.TUser
		pipe  *model.TPipeline
		perms []*UserPipeOrgPerm
		want  bool
	}{
		{
			"admin can read",
			&model.TUser{Id: "admin"},
			&model.TPipeline{Id: "p1"},
			nil,
			true,
		},
		{
			"pipe owner can read",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u1"},
			nil,
			true,
		},
		{
			"public org pipeline can be read",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			[]*UserPipeOrgPerm{{OrgPublic: 1}},
			true,
		},
		{
			"org member can read",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			[]*UserPipeOrgPerm{{CurUid: "u1"}},
			true,
		},
		{
			"non-member of private org cannot read",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			nil,
			false,
		},
		{
			"user matching OrgUid in perms can read",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			[]*UserPipeOrgPerm{{OrgUid: "u1"}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PipePerm{lgusr: tt.user, pipe: tt.pipe, perms: tt.perms}
			if got := c.CanRead(); got != tt.want {
				t.Errorf("CanRead() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipePerm_CanWrite(t *testing.T) {
	tests := []struct {
		name  string
		user  *model.TUser
		pipe  *model.TPipeline
		perms []*UserPipeOrgPerm
		want  bool
	}{
		{
			"admin can write",
			&model.TUser{Id: "admin"},
			&model.TPipeline{Id: "p1"},
			nil,
			true,
		},
		{
			"pipe owner can write",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u1"},
			nil,
			true,
		},
		{
			"user with PermRw can write",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			[]*UserPipeOrgPerm{{CurUid: "u1", PermRw: 1}},
			true,
		},
		{
			"user with PermAdm can write",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			[]*UserPipeOrgPerm{{CurUid: "u1", PermAdm: 1}},
			true,
		},
		{
			"public org does not grant write",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			[]*UserPipeOrgPerm{{OrgPublic: 1}},
			false,
		},
		{
			"member without write perm cannot write",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			[]*UserPipeOrgPerm{{CurUid: "u1", PermRw: 0, PermAdm: 0}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PipePerm{lgusr: tt.user, pipe: tt.pipe, perms: tt.perms}
			if got := c.CanWrite(); got != tt.want {
				t.Errorf("CanWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipePerm_CanExec(t *testing.T) {
	tests := []struct {
		name  string
		user  *model.TUser
		pipe  *model.TPipeline
		perms []*UserPipeOrgPerm
		want  bool
	}{
		{
			"admin can exec",
			&model.TUser{Id: "admin"},
			&model.TPipeline{Id: "p1"},
			nil,
			true,
		},
		{
			"pipe owner can exec",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u1"},
			nil,
			true,
		},
		{
			"user with PermExec can exec",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			[]*UserPipeOrgPerm{{CurUid: "u1", PermExec: 1}},
			true,
		},
		{
			"user with PermAdm can exec",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			[]*UserPipeOrgPerm{{CurUid: "u1", PermAdm: 1}},
			true,
		},
		{
			"member without exec perm cannot exec",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			[]*UserPipeOrgPerm{{CurUid: "u1", PermExec: 0}},
			false,
		},
		{
			"public org does not grant exec",
			&model.TUser{Id: "u1"},
			&model.TPipeline{Id: "p1", Uid: "u2"},
			[]*UserPipeOrgPerm{{OrgPublic: 1}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PipePerm{lgusr: tt.user, pipe: tt.pipe, perms: tt.perms}
			if got := c.CanExec(); got != tt.want {
				t.Errorf("CanExec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPipePerm_Accessors(t *testing.T) {
	usr := &model.TUser{Id: "u1"}
	pipe := &model.TPipeline{Id: "p1"}
	c := &PipePerm{lgusr: usr, pipe: pipe}

	if c.LgUser() != usr {
		t.Errorf("LgUser() returned wrong user")
	}
	if c.Pipeline() != pipe {
		t.Errorf("Pipeline() returned wrong pipeline")
	}
}
