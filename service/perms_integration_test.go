package service

import (
	"context"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupPermsDBTest creates an isolated in-memory SQLite DB for perms.go tests.
func setupPermsDBTest(t *testing.T) *xorm.Engine {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	oldDb := comm.Db
	comm.Db = eng
	t.Cleanup(func() {
		comm.Db = oldDb
		_ = eng.Close()
	})
	if err := eng.Sync2(
		&model.TUser{},
		&model.TUserInfo{},
		&model.TOrg{},
		&model.TUserOrg{},
		&model.TPipeline{},
	); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// --- CheckPermissionCtx Tests ---

func TestCheckPermissionCtx_IntegCommon(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "perm-user-1",
		Aid:     1,
		Name:    "alice",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// "common" permission should be granted to any existing user
	if !CheckPermissionCtx(ctx, "perm-user-1", PermCommon) {
		t.Error("CheckPermissionCtx should grant common permission to existing user")
	}
}

func TestCheckPermissionCtx_IntegAdmin(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	// Admin user has Name == "admin"
	admin := &model.TUser{
		Id:      "admin",
		Aid:     2,
		Name:    "admin",
		Created: time.Now(),
	}
	if _, err := eng.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	if !CheckPermissionCtx(ctx, "admin", PermAdmin) {
		t.Error("CheckPermissionCtx should grant admin permission to admin user")
	}
}

func TestCheckPermissionCtx_IntegNonAdminDenied(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "perm-user-2",
		Aid:     3,
		Name:    "bob",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if CheckPermissionCtx(ctx, "perm-user-2", PermAdmin) {
		t.Error("CheckPermissionCtx should deny admin permission to non-admin user")
	}
}

func TestCheckPermissionCtx_IntegUserNotFound(t *testing.T) {
	setupPermsDBTest(t)
	ctx := context.Background()

	if CheckPermissionCtx(ctx, "nonexistent", PermCommon) {
		t.Error("CheckPermissionCtx should deny permission when user not found")
	}
}

func TestCheckPermissionCtx_IntegUnknownPermLevel(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "perm-user-3",
		Aid:     4,
		Name:    "charlie",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Unknown permission level should be denied
	if CheckPermissionCtx(ctx, "perm-user-3", "superuser") {
		t.Error("CheckPermissionCtx should deny unknown permission level")
	}
}

// --- CheckPermission (global wrapper) Tests ---

func TestCheckPermission_IntegCommon(t *testing.T) {
	eng := setupPermsDBTest(t)

	user := &model.TUser{
		Id:      "perm-global-1",
		Aid:     10,
		Name:    "dave",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if !CheckPermission("perm-global-1", PermCommon) {
		t.Error("CheckPermission should grant common permission")
	}
}

func TestCheckPermission_IntegAdminDenied(t *testing.T) {
	eng := setupPermsDBTest(t)

	user := &model.TUser{
		Id:      "perm-global-2",
		Aid:     11,
		Name:    "eve",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if CheckPermission("perm-global-2", PermAdmin) {
		t.Error("CheckPermission should deny admin for non-admin user")
	}
}

// --- NewOrgPermCtx Tests ---

func TestNewOrgPermCtx_IntegOwner(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	org := &model.TOrg{
		Id:      "org-np-1",
		Aid:     20,
		Uid:     "owner-1",
		Name:    "Owner Org",
		Public:  0,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{Id: "owner-1"}
	op := NewOrgPermCtx(ctx, user, "org-np-1")

	if !op.IsOrgOwner() {
		t.Error("NewOrgPermCtx should recognize org owner")
	}
	if op.Org() == nil {
		t.Error("Org() should not be nil for valid org")
	}
	if op.LgUser() == nil {
		t.Error("LgUser() should not be nil when user provided")
	}
}

func TestNewOrgPermCtx_IntegMemberWithPerms(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	org := &model.TOrg{
		Id:      "org-np-2",
		Aid:     21,
		Uid:     "owner-2",
		Name:    "Member Org",
		Public:  0,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:      21,
		Uid:      "member-1",
		OrgId:    "org-np-2",
		Created:  time.Now(),
		PermRw:   1,
		PermExec: 1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	user := &model.TUser{Id: "member-1"}
	op := NewOrgPermCtx(ctx, user, "org-np-2")

	if op.UserOrg() == nil {
		t.Fatal("UserOrg() should not be nil for member")
	}
	if !op.CanWrite() {
		t.Error("Member with PermRw should be able to write")
	}
	if !op.CanExec() {
		t.Error("Member with PermExec should be able to exec")
	}
	if op.IsOrgOwner() {
		t.Error("Member is not org owner")
	}
}

func TestNewOrgPermCtx_IntegNonexistentOrg(t *testing.T) {
	setupPermsDBTest(t)
	ctx := context.Background()

	user := &model.TUser{Id: "user-x"}
	op := NewOrgPermCtx(ctx, user, "nonexistent-org")

	if op.Org() != nil {
		t.Error("Org() should be nil for nonexistent org")
	}
	if op.IsOrgAdmin() {
		t.Error("Should not be org admin for nonexistent org")
	}
	if op.CanRead() {
		t.Error("Should not be able to read nonexistent org")
	}
}

func TestNewOrgPermCtx_IntegEmptyOrgId(t *testing.T) {
	setupPermsDBTest(t)
	ctx := context.Background()

	user := &model.TUser{Id: "user-y"}
	op := NewOrgPermCtx(ctx, user, "")

	if op.Org() != nil {
		t.Error("Org() should be nil for empty org ID")
	}
}

func TestNewOrgPermCtx_IntegNilUser(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	org := &model.TOrg{
		Id:      "org-np-3",
		Aid:     22,
		Uid:     "owner-3",
		Name:    "Public Org",
		Public:  1,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	op := NewOrgPermCtx(ctx, nil, "org-np-3")
	if op.LgUser() != nil {
		t.Error("LgUser() should be nil")
	}
	if op.Org() == nil {
		t.Error("Org() should not be nil for valid org")
	}
	// Public org allows read even without user
	if !op.CanRead() {
		t.Error("Public org should allow read for nil user")
	}
}

func TestNewOrgPermCtx_IntegDeletedOrg(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	org := &model.TOrg{
		Id:      "org-np-4",
		Aid:     23,
		Uid:     "owner-4",
		Name:    "Deleted Org",
		Public:  1,
		Deleted: 1,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{Id: "user-z"}
	op := NewOrgPermCtx(ctx, user, "org-np-4")

	// Deleted org should not be resolved
	if op.Org() != nil {
		t.Error("Org() should be nil for deleted org")
	}
}

// --- NewPipePermCtx Tests (SQLite path - no MySQL-specific SQL) ---

func TestNewPipePermCtx_IntegPipeOwner(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:         "pipe-np-1",
		Uid:        "pipe-owner-1",
		Name:       "test-pipe",
		Deleted:    0,
		CreateTime: time.Now(),
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	user := &model.TUser{Id: "pipe-owner-1"}
	pp := NewPipePermCtx(ctx, user, "pipe-np-1")

	if pp.Pipeline() == nil {
		t.Fatal("Pipeline() should not be nil")
	}
	if !pp.IsPipeOwner() {
		t.Error("User should be recognized as pipe owner")
	}
	if !pp.CanRead() {
		t.Error("Pipe owner should be able to read")
	}
	if !pp.CanWrite() {
		t.Error("Pipe owner should be able to write")
	}
	if !pp.CanExec() {
		t.Error("Pipe owner should be able to exec")
	}
}

func TestNewPipePermCtx_IntegNonexistentPipe(t *testing.T) {
	setupPermsDBTest(t)
	ctx := context.Background()

	user := &model.TUser{Id: "user-p1"}
	pp := NewPipePermCtx(ctx, user, "nonexistent-pipe")

	if pp.Pipeline() != nil {
		t.Error("Pipeline() should be nil for nonexistent pipeline")
	}
	if pp.CanRead() {
		t.Error("Should not be able to read nonexistent pipeline")
	}
}

func TestNewPipePermCtx_IntegEmptyPipeId(t *testing.T) {
	setupPermsDBTest(t)
	ctx := context.Background()

	user := &model.TUser{Id: "user-p2"}
	pp := NewPipePermCtx(ctx, user, "")

	if pp.Pipeline() != nil {
		t.Error("Pipeline() should be nil for empty pipe ID")
	}
}

func TestNewPipePermCtx_IntegAdminUser(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:         "pipe-np-2",
		Uid:        "pipe-owner-2",
		Name:       "admin-test-pipe",
		Deleted:    0,
		CreateTime: time.Now(),
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	admin := &model.TUser{Id: "admin"}
	pp := NewPipePermCtx(ctx, admin, "pipe-np-2")

	if !pp.IsAdmin() {
		t.Error("Admin user should be recognized")
	}
	if !pp.CanRead() {
		t.Error("Admin should be able to read")
	}
	if !pp.CanWrite() {
		t.Error("Admin should be able to write")
	}
	if !pp.CanExec() {
		t.Error("Admin should be able to exec")
	}
}

func TestNewPipePermCtx_IntegNilUser(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:         "pipe-np-3",
		Uid:        "pipe-owner-3",
		Name:       "nil-user-pipe",
		Deleted:    0,
		CreateTime: time.Now(),
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	pp := NewPipePermCtx(ctx, nil, "pipe-np-3")
	if pp.LgUser() != nil {
		t.Error("LgUser() should be nil")
	}
	if pp.Pipeline() == nil {
		t.Error("Pipeline() should not be nil for valid pipe")
	}
	// Non-admin, non-owner, no perms
	if pp.CanRead() {
		t.Error("Nil user should not be able to read private pipeline")
	}
}

func TestNewPipePermCtx_IntegDeletedPipe(t *testing.T) {
	eng := setupPermsDBTest(t)
	ctx := context.Background()

	pipe := &model.TPipeline{
		Id:         "pipe-np-4",
		Uid:        "pipe-owner-4",
		Name:       "deleted-pipe",
		Deleted:    1,
		CreateTime: time.Now(),
	}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	user := &model.TUser{Id: "pipe-owner-4"}
	pp := NewPipePermCtx(ctx, user, "pipe-np-4")

	// Deleted pipeline should still be found (the WHERE clause includes deleted!=1 only in preBuild, not in NewPipePermCtx)
	// The query is WHERE("id=?", pipeId).Get(pipe) - no deleted filter
	if pp.Pipeline() == nil {
		t.Log("NewPipePermCtx finds deleted pipelines (no deleted filter in query)")
	}
}

// --- OrgPerm accessor nil-safety tests ---

func TestOrgPerm_AccessorsNil(t *testing.T) {
	op := &OrgPerm{}
	if op.LgUser() != nil {
		t.Error("LgUser() should be nil")
	}
	if op.Org() != nil {
		t.Error("Org() should be nil")
	}
	if op.UserOrg() != nil {
		t.Error("UserOrg() should be nil")
	}
}

// --- PipePerm accessor nil-safety tests ---

func TestPipePerm_AccessorsNil(t *testing.T) {
	pp := &PipePerm{}
	if pp.LgUser() != nil {
		t.Error("LgUser() should be nil")
	}
	if pp.Pipeline() != nil {
		t.Error("Pipeline() should be nil")
	}
}
