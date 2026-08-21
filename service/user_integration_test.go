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

// setupUserDBTest creates an isolated in-memory SQLite DB for user.go tests.
func setupUserDBTest(t *testing.T) *xorm.Engine {
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

// --- GetUserCtx Tests ---

func TestGetUserCtx_IntegFound(t *testing.T) {
	eng := setupUserDBTest(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "user-get-1",
		Aid:     1,
		Name:    "alice",
		Nick:    "Alice",
		Created: time.Now(),
		Active:  1,
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	found, ok := GetUserCtx(ctx, "user-get-1")
	if !ok {
		t.Fatal("GetUserCtx should find user")
	}
	if found.Name != "alice" {
		t.Errorf("Name = %q, want %q", found.Name, "alice")
	}
	if found.Nick != "Alice" {
		t.Errorf("Nick = %q, want %q", found.Nick, "Alice")
	}
	if found.Active != 1 {
		t.Errorf("Active = %d, want 1", found.Active)
	}
}

func TestGetUserCtx_IntegNotFound(t *testing.T) {
	setupUserDBTest(t)
	ctx := context.Background()

	_, ok := GetUserCtx(ctx, "nonexistent")
	if ok {
		t.Error("GetUserCtx should return false for nonexistent user")
	}
}

func TestGetUserCtx_IntegEmpty(t *testing.T) {
	setupUserDBTest(t)
	ctx := context.Background()

	_, ok := GetUserCtx(ctx, "")
	if ok {
		t.Error("GetUserCtx should return false for empty uid")
	}
}

// --- GetUserInfoCtx Tests ---

func TestGetUserInfoCtx_IntegFound(t *testing.T) {
	eng := setupUserDBTest(t)
	ctx := context.Background()

	info := &model.TUserInfo{
		Id:    "user-info-1",
		Phone: "1234567890",
		Email: "alice@example.com",
	}
	if _, err := eng.Insert(info); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	found, ok := GetUserInfoCtx(ctx, "user-info-1")
	if !ok {
		t.Fatal("GetUserInfoCtx should find user info")
	}
	if found.Phone != "1234567890" {
		t.Errorf("Phone = %q, want %q", found.Phone, "1234567890")
	}
	if found.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", found.Email, "alice@example.com")
	}
}

func TestGetUserInfoCtx_IntegNotFound(t *testing.T) {
	setupUserDBTest(t)
	ctx := context.Background()

	_, ok := GetUserInfoCtx(ctx, "nonexistent")
	if ok {
		t.Error("GetUserInfoCtx should return false for nonexistent user")
	}
}

func TestGetUserInfoCtx_IntegEmpty(t *testing.T) {
	setupUserDBTest(t)
	ctx := context.Background()

	_, ok := GetUserInfoCtx(ctx, "")
	if ok {
		t.Error("GetUserInfoCtx should return false for empty uid")
	}
}

// --- FindUserNameCtx Tests ---

func TestFindUserNameCtx_IntegFound(t *testing.T) {
	eng := setupUserDBTest(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "user-find-1",
		Aid:     10,
		Name:    "bob",
		Nick:    "Bob",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	found, ok := FindUserNameCtx(ctx, "bob")
	if !ok {
		t.Fatal("FindUserNameCtx should find user by name")
	}
	if found.Id != "user-find-1" {
		t.Errorf("Id = %q, want %q", found.Id, "user-find-1")
	}
	if found.Nick != "Bob" {
		t.Errorf("Nick = %q, want %q", found.Nick, "Bob")
	}
}

func TestFindUserNameCtx_IntegNotFound(t *testing.T) {
	setupUserDBTest(t)
	ctx := context.Background()

	_, ok := FindUserNameCtx(ctx, "nonexistent")
	if ok {
		t.Error("FindUserNameCtx should return false for nonexistent name")
	}
}

// --- GetUserOrgCtx Tests ---

func TestGetUserOrgCtx_IntegFound(t *testing.T) {
	eng := setupUserDBTest(t)
	ctx := context.Background()

	// Insert org
	org := &model.TOrg{
		Id:      "org-1",
		Aid:     1,
		Uid:     "user-1",
		Name:    "Test Org",
		Public:  0,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Insert user org membership
	userOrg := &model.TUserOrg{
		Aid:      1,
		Uid:      "user-1",
		OrgId:    "org-1",
		Created:  time.Now(),
		PermAdm:  1,
		PermRw:   1,
		PermExec: 1,
		PermDown: 1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	found, ok := GetUserOrgCtx(ctx, "user-1", "org-1")
	if !ok {
		t.Fatal("GetUserOrgCtx should find user org")
	}
	if found.PermAdm != 1 {
		t.Errorf("PermAdm = %d, want 1", found.PermAdm)
	}
	if found.PermRw != 1 {
		t.Errorf("PermRw = %d, want 1", found.PermRw)
	}
	if found.PermExec != 1 {
		t.Errorf("PermExec = %d, want 1", found.PermExec)
	}
}

func TestGetUserOrgCtx_IntegOrgNotFound(t *testing.T) {
	setupUserDBTest(t)
	ctx := context.Background()

	_, ok := GetUserOrgCtx(ctx, "user-1", "nonexistent-org")
	if ok {
		t.Error("GetUserOrgCtx should return false when org doesn't exist")
	}
}

func TestGetUserOrgCtx_IntegUserNotMember(t *testing.T) {
	eng := setupUserDBTest(t)
	ctx := context.Background()

	// Insert org but no user membership
	org := &model.TOrg{
		Id:      "org-2",
		Aid:     2,
		Uid:     "user-2",
		Name:    "Another Org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	_, ok := GetUserOrgCtx(ctx, "user-3", "org-2")
	if ok {
		t.Error("GetUserOrgCtx should return false when user is not a member")
	}
}

// --- IsOrgAdminCtx Tests ---

func TestIsOrgAdminCtx_IntegTrue(t *testing.T) {
	eng := setupUserDBTest(t)
	ctx := context.Background()

	org := &model.TOrg{
		Id:      "org-admin-1",
		Aid:     10,
		Uid:     "admin-user",
		Name:    "Admin Org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:     10,
		Uid:     "admin-user",
		OrgId:   "org-admin-1",
		Created: time.Now(),
		PermAdm: 1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	if !IsOrgAdminCtx(ctx, "admin-user", "org-admin-1") {
		t.Error("IsOrgAdminCtx should return true for org admin")
	}
}

func TestIsOrgAdminCtx_IntegFalse(t *testing.T) {
	eng := setupUserDBTest(t)
	ctx := context.Background()

	org := &model.TOrg{
		Id:      "org-admin-2",
		Aid:     20,
		Uid:     "owner-user",
		Name:    "Not Admin Org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:     20,
		Uid:     "regular-user",
		OrgId:   "org-admin-2",
		Created: time.Now(),
		PermAdm: 0,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	if IsOrgAdminCtx(ctx, "regular-user", "org-admin-2") {
		t.Error("IsOrgAdminCtx should return false for non-admin")
	}
}

func TestIsOrgAdminCtx_IntegOrgNotFound(t *testing.T) {
	setupUserDBTest(t)
	ctx := context.Background()

	if IsOrgAdminCtx(ctx, "user", "nonexistent") {
		t.Error("IsOrgAdminCtx should return false when org doesn't exist")
	}
}

// --- HasOrgExecCtx Tests ---

func TestHasOrgExecCtx_IntegTrue(t *testing.T) {
	eng := setupUserDBTest(t)
	ctx := context.Background()

	org := &model.TOrg{
		Id:      "org-exec-1",
		Aid:     30,
		Uid:     "exec-user",
		Name:    "Exec Org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:      30,
		Uid:      "exec-user",
		OrgId:    "org-exec-1",
		Created:  time.Now(),
		PermExec: 1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	if !HasOrgExecCtx(ctx, "exec-user", "org-exec-1") {
		t.Error("HasOrgExecCtx should return true for user with exec permission")
	}
}

func TestHasOrgExecCtx_IntegFalse(t *testing.T) {
	eng := setupUserDBTest(t)
	ctx := context.Background()

	org := &model.TOrg{
		Id:      "org-exec-2",
		Aid:     40,
		Uid:     "no-exec-user",
		Name:    "No Exec Org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:      40,
		Uid:      "no-exec-user",
		OrgId:    "org-exec-2",
		Created:  time.Now(),
		PermExec: 0,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	if HasOrgExecCtx(ctx, "no-exec-user", "org-exec-2") {
		t.Error("HasOrgExecCtx should return false for user without exec permission")
	}
}

// --- GetUsePermRwrCtx Tests ---

func TestGetUsePermRwrCtx_IntegWithPermission(t *testing.T) {
	eng := setupUserDBTest(t)
	ctx := context.Background()

	org := &model.TOrg{
		Id:      "org-rw-1",
		Aid:     50,
		Uid:     "rw-user",
		Name:    "RW Org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:     50,
		Uid:     "rw-user",
		OrgId:   "org-rw-1",
		Created: time.Now(),
		PermRw:  1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	result := GetUsePermRwrCtx(ctx, "rw-user", "org-rw-1")
	if result != 1 {
		t.Errorf("GetUsePermRwrCtx = %d, want 1", result)
	}
}

func TestGetUsePermRwrCtx_IntegNoPermission(t *testing.T) {
	eng := setupUserDBTest(t)
	ctx := context.Background()

	org := &model.TOrg{
		Id:      "org-rw-2",
		Aid:     60,
		Uid:     "no-rw-user",
		Name:    "No RW Org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:     60,
		Uid:     "no-rw-user",
		OrgId:   "org-rw-2",
		Created: time.Now(),
		PermRw:  0,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	result := GetUsePermRwrCtx(ctx, "no-rw-user", "org-rw-2")
	if result != 0 {
		t.Errorf("GetUsePermRwrCtx = %d, want 0", result)
	}
}

func TestGetUsePermRwrCtx_IntegOrgNotFound(t *testing.T) {
	setupUserDBTest(t)
	ctx := context.Background()

	result := GetUsePermRwrCtx(ctx, "user", "nonexistent")
	if result != 0 {
		t.Errorf("GetUsePermRwrCtx = %d, want 0 for nonexistent org", result)
	}
}

// --- Global context wrapper tests ---

func TestGetUser_IntegFound(t *testing.T) {
	eng := setupUserDBTest(t)

	user := &model.TUser{
		Id:      "user-global-get",
		Aid:     100,
		Name:    "charlie",
		Nick:    "Charlie",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	found, ok := GetUser("user-global-get")
	if !ok {
		t.Fatal("GetUser should find user")
	}
	if found.Name != "charlie" {
		t.Errorf("Name = %q, want %q", found.Name, "charlie")
	}
}

func TestFindUserName_IntegFound(t *testing.T) {
	eng := setupUserDBTest(t)

	user := &model.TUser{
		Id:      "user-global-find",
		Aid:     110,
		Name:    "dave",
		Nick:    "Dave",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	found, ok := FindUserName("dave")
	if !ok {
		t.Fatal("FindUserName should find user by name")
	}
	if found.Id != "user-global-find" {
		t.Errorf("Id = %q, want %q", found.Id, "user-global-find")
	}
}

func TestGetUserOrg_IntegFound(t *testing.T) {
	eng := setupUserDBTest(t)

	org := &model.TOrg{
		Id:      "org-global-1",
		Aid:     100,
		Uid:     "global-user",
		Name:    "Global Org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:     100,
		Uid:     "global-user",
		OrgId:   "org-global-1",
		Created: time.Now(),
		PermAdm: 1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	found, ok := GetUserOrg("global-user", "org-global-1")
	if !ok {
		t.Fatal("GetUserOrg should find user org")
	}
	if found.PermAdm != 1 {
		t.Errorf("PermAdm = %d, want 1", found.PermAdm)
	}
}

func TestIsOrgAdmin_IntegTrue(t *testing.T) {
	eng := setupUserDBTest(t)

	org := &model.TOrg{
		Id:      "org-global-admin",
		Aid:     110,
		Uid:     "global-admin",
		Name:    "Global Admin Org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:     110,
		Uid:     "global-admin",
		OrgId:   "org-global-admin",
		Created: time.Now(),
		PermAdm: 1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	if !IsOrgAdmin("global-admin", "org-global-admin") {
		t.Error("IsOrgAdmin should return true for org admin")
	}
}

func TestHasOrgExec_IntegTrue(t *testing.T) {
	eng := setupUserDBTest(t)

	org := &model.TOrg{
		Id:      "org-global-exec",
		Aid:     120,
		Uid:     "global-exec",
		Name:    "Global Exec Org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:      120,
		Uid:      "global-exec",
		OrgId:    "org-global-exec",
		Created:  time.Now(),
		PermExec: 1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	if !HasOrgExec("global-exec", "org-global-exec") {
		t.Error("HasOrgExec should return true for user with exec permission")
	}
}

func TestGetUsePermRwr_IntegWithPermission(t *testing.T) {
	eng := setupUserDBTest(t)

	org := &model.TOrg{
		Id:      "org-global-rw",
		Aid:     130,
		Uid:     "global-rw",
		Name:    "Global RW Org",
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:     130,
		Uid:     "global-rw",
		OrgId:   "org-global-rw",
		Created: time.Now(),
		PermRw:  1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	result := GetUsePermRwr("global-rw", "org-global-rw")
	if result != 1 {
		t.Errorf("GetUsePermRwr = %d, want 1", result)
	}
}
