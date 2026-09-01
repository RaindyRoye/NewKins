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

// setupUserTestDB creates an in-memory SQLite database with user-related tables.
func setupUserTestDB(t *testing.T) *xorm.Engine {
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

// --- GetUserCtx ---

func TestGetUserCtx_Found(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "usr-1",
		Aid:     1,
		Name:    "alice",
		Nick:    "Alice Wonderland",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	got, ok := GetUserCtx(ctx, "usr-1")
	if !ok {
		t.Fatal("GetUserCtx should find existing user")
	}
	if got.Name != "alice" {
		t.Errorf("Name = %q, want %q", got.Name, "alice")
	}
	if got.Nick != "Alice Wonderland" {
		t.Errorf("Nick = %q, want %q", got.Nick, "Alice Wonderland")
	}
}

func TestGetUserCtx_NotFound(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	got, ok := GetUserCtx(ctx, "nonexistent")
	if ok {
		t.Error("GetUserCtx should return false for nonexistent user")
	}
	// got is a pointer to an empty struct, not nil (xorm returns it)
	if got == nil {
		t.Error("GetUserCtx should return a non-nil empty struct on not-found")
	}
}

func TestGetUserCtx_EmptyID(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	_, ok := GetUserCtx(ctx, "")
	if ok {
		t.Error("GetUserCtx should return false for empty ID")
	}
}

func TestGetUser_DelegatesToCtx(t *testing.T) {
	eng := setupUserTestDB(t)
	user := &model.TUser{
		Id:   "usr-global",
		Aid:  2,
		Name: "bob",
		Nick: "Bob",
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// GetUser uses comm.Ctx which is the background context by default
	got, ok := GetUser("usr-global")
	if !ok {
		t.Fatal("GetUser should find existing user")
	}
	if got.Name != "bob" {
		t.Errorf("Name = %q, want %q", got.Name, "bob")
	}
}

// --- GetUserInfoCtx ---

func TestGetUserInfoCtx_Found(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	info := &model.TUserInfo{
		Id:    "usr-1",
		Phone: "555-0100",
		Email: "alice@example.com",
	}
	if _, err := eng.Insert(info); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	got, ok := GetUserInfoCtx(ctx, "usr-1")
	if !ok {
		t.Fatal("GetUserInfoCtx should find existing user info")
	}
	if got.Phone != "555-0100" {
		t.Errorf("Phone = %q, want %q", got.Phone, "555-0100")
	}
	if got.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "alice@example.com")
	}
}

func TestGetUserInfoCtx_NotFound(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	_, ok := GetUserInfoCtx(ctx, "nonexistent")
	if ok {
		t.Error("GetUserInfoCtx should return false for nonexistent user")
	}
}

func TestGetUserInfoCtx_EmptyID(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	_, ok := GetUserInfoCtx(ctx, "")
	if ok {
		t.Error("GetUserInfoCtx should return false for empty ID")
	}
}

func TestGetUserInfo_DelegatesToCtx(t *testing.T) {
	eng := setupUserTestDB(t)
	info := &model.TUserInfo{
		Id:    "usr-info-global",
		Phone: "555-0200",
	}
	if _, err := eng.Insert(info); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	got, ok := GetUserInfo("usr-info-global")
	if !ok {
		t.Fatal("GetUserInfo should find existing user info")
	}
	if got.Phone != "555-0200" {
		t.Errorf("Phone = %q, want %q", got.Phone, "555-0200")
	}
}

// --- FindUserNameCtx ---

func TestFindUserNameCtx_Found(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:   "usr-name-1",
		Aid:  10,
		Name: "charlie",
		Nick: "Charlie",
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	got, ok := FindUserNameCtx(ctx, "charlie")
	if !ok {
		t.Fatal("FindUserNameCtx should find user by name")
	}
	if got.Id != "usr-name-1" {
		t.Errorf("Id = %q, want %q", got.Id, "usr-name-1")
	}
}

func TestFindUserNameCtx_NotFound(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	_, ok := FindUserNameCtx(ctx, "nonexistent")
	if ok {
		t.Error("FindUserNameCtx should return false for nonexistent name")
	}
}

func TestFindUserName_DelegatesToCtx(t *testing.T) {
	eng := setupUserTestDB(t)
	user := &model.TUser{
		Id:   "usr-fname",
		Aid:  11,
		Name: "dave",
		Nick: "Dave",
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	got, ok := FindUserName("dave")
	if !ok {
		t.Fatal("FindUserName should find user by name")
	}
	if got.Id != "usr-fname" {
		t.Errorf("Id = %q, want %q", got.Id, "usr-fname")
	}
}

// --- GetUserCacheCtx (without cache - falls back to DB) ---

func TestGetUserCacheCtx_FallsBackToDB(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// BCache is nil, so cache operations fail and function falls back to DB
	user := &model.TUser{
		Id:   "usr-cache-1",
		Aid:  20,
		Name: "eve",
		Nick: "Eve",
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	got, ok := GetUserCacheCtx(ctx, "usr-cache-1")
	if !ok {
		t.Fatal("GetUserCacheCtx should find user via DB fallback")
	}
	if got.Name != "eve" {
		t.Errorf("Name = %q, want %q", got.Name, "eve")
	}
}

func TestGetUserCacheCtx_NotFound(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	_, ok := GetUserCacheCtx(ctx, "nonexistent")
	if ok {
		t.Error("GetUserCacheCtx should return false for nonexistent user")
	}
}

func TestGetUserCache_DelegatesToCtx(t *testing.T) {
	eng := setupUserTestDB(t)
	user := &model.TUser{
		Id:   "usr-cache-global",
		Aid:  21,
		Name: "frank",
		Nick: "Frank",
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	got, ok := GetUserCache("usr-cache-global")
	if !ok {
		t.Fatal("GetUserCache should find user via DB fallback")
	}
	if got.Name != "frank" {
		t.Errorf("Name = %q, want %q", got.Name, "frank")
	}
}

// --- IsOrgAdminCtx ---

func TestIsOrgAdminCtx_UserIsOrgAdmin(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-1", Aid: 1, Name: "TestOrg", Uid: "creator"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	userOrg := &model.TUserOrg{
		Uid:     "user-1",
		OrgId:   "org-1",
		PermAdm: 1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	got := IsOrgAdminCtx(ctx, "user-1", "org-1")
	if !got {
		t.Error("IsOrgAdminCtx should return true for org admin")
	}
}

func TestIsOrgAdminCtx_UserNotOrgAdmin(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-2", Aid: 2, Name: "TestOrg2", Uid: "creator"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	userOrg := &model.TUserOrg{
		Uid:     "user-2",
		OrgId:   "org-2",
		PermAdm: 0,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	got := IsOrgAdminCtx(ctx, "user-2", "org-2")
	if got {
		t.Error("IsOrgAdminCtx should return false for non-admin member")
	}
}

func TestIsOrgAdminCtx_NonMember(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-3", Aid: 3, Name: "TestOrg3", Uid: "creator"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	got := IsOrgAdminCtx(ctx, "nonmember", "org-3")
	if got {
		t.Error("IsOrgAdminCtx should return false for non-member")
	}
}

func TestIsOrgAdminCtx_OrgNotFound(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	got := IsOrgAdminCtx(ctx, "user-1", "nonexistent-org")
	if got {
		t.Error("IsOrgAdminCtx should return false for nonexistent org")
	}
}

// --- HasOrgExecCtx ---

func TestHasOrgExecCtx_UserHasExec(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-exec-1", Aid: 10, Name: "ExecOrg", Uid: "creator"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	userOrg := &model.TUserOrg{
		Uid:      "user-exec",
		OrgId:    "org-exec-1",
		PermExec: 1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	got := HasOrgExecCtx(ctx, "user-exec", "org-exec-1")
	if !got {
		t.Error("HasOrgExecCtx should return true for user with exec permission")
	}
}

func TestHasOrgExecCtx_UserLacksExec(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-exec-2", Aid: 11, Name: "NoExecOrg", Uid: "creator"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	userOrg := &model.TUserOrg{
		Uid:      "user-noexec",
		OrgId:    "org-exec-2",
		PermExec: 0,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	got := HasOrgExecCtx(ctx, "user-noexec", "org-exec-2")
	if got {
		t.Error("HasOrgExecCtx should return false for user without exec permission")
	}
}

// --- GetUsePermRwrCtx ---

func TestGetUsePermRwrCtx_HasWritePermission(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-rw-1", Aid: 20, Name: "RwOrg", Uid: "creator"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	userOrg := &model.TUserOrg{
		Uid:    "user-rw",
		OrgId:  "org-rw-1",
		PermRw: 1,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	got := GetUsePermRwrCtx(ctx, "user-rw", "org-rw-1")
	if got != 1 {
		t.Errorf("GetUsePermRwrCtx = %d, want 1", got)
	}
}

func TestGetUsePermRwrCtx_NoWritePermission(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-rw-2", Aid: 21, Name: "NoRwOrg", Uid: "creator"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	userOrg := &model.TUserOrg{
		Uid:    "user-norw",
		OrgId:  "org-rw-2",
		PermRw: 0,
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	got := GetUsePermRwrCtx(ctx, "user-norw", "org-rw-2")
	if got != 0 {
		t.Errorf("GetUsePermRwrCtx = %d, want 0", got)
	}
}

func TestGetUsePermRwrCtx_NonMember(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-rw-3", Aid: 22, Name: "NonMemberOrg", Uid: "creator"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	got := GetUsePermRwrCtx(ctx, "stranger", "org-rw-3")
	if got != 0 {
		t.Errorf("GetUsePermRwrCtx for non-member = %d, want 0", got)
	}
}

// --- ClearUserCache ---

func TestClearUserCache_EmptyID(t *testing.T) {
	// Should not panic with empty ID
	ClearUserCache("")
}

func TestClearUserCache_NilCache(t *testing.T) {
	// BCache is nil; should log warning but not panic
	ClearUserCache("some-user-id")
}

// --- NewOrgPermCtx ---

func TestNewOrgPermCtx_OrgOwnerCanWrite(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-owner-1", Aid: 100, Name: "OwnerOrg", Uid: "owner-user"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	owner := &model.TUser{Id: "owner-user", Aid: 50, Name: "owner"}
	if _, err := eng.Insert(owner); err != nil {
		t.Fatalf("insert owner: %v", err)
	}

	op := NewOrgPermCtx(ctx, owner, "org-owner-1")
	if op.Org() == nil {
		t.Fatal("NewOrgPermCtx should resolve org")
	}
	if !op.IsOrgOwner() {
		t.Error("owner-user should be org owner")
	}
	if !op.CanWrite() {
		t.Error("org owner should have write permission")
	}
	if !op.CanExec() {
		t.Error("org owner should have exec permission")
	}
}

func TestNewOrgPermCtx_DeletedOrg(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-deleted", Aid: 101, Name: "DeletedOrg", Uid: "u1", Deleted: 1}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	user := &model.TUser{Id: "u1", Aid: 51, Name: "user1"}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	op := NewOrgPermCtx(ctx, user, "org-deleted")
	// Org is deleted, so org should be nil (deleted org is not resolved)
	if op.Org() != nil {
		t.Error("NewOrgPermCtx should not resolve deleted org")
	}
	if op.CanRead() {
		t.Error("should not be able to read deleted org")
	}
}

func TestNewOrgPermCtx_EmptyOrgId(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	user := &model.TUser{Id: "u1", Aid: 51, Name: "user1"}
	op := NewOrgPermCtx(ctx, user, "")
	if op.Org() != nil {
		t.Error("NewOrgPermCtx with empty orgId should not resolve org")
	}
	if op.IsAdmin() {
		t.Error("regular user should not be admin")
	}
}

func TestNewOrgPermCtx_NilUser(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-nil-user", Aid: 102, Name: "NilUserOrg", Uid: "creator", Public: 1}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	op := NewOrgPermCtx(ctx, nil, "org-nil-user")
	if op.Org() == nil {
		t.Fatal("NewOrgPermCtx should resolve org even with nil user")
	}
	if !op.IsOrgPublic() {
		t.Error("org should be public")
	}
	// nil user but public org => can read
	if !op.CanRead() {
		t.Error("anyone should be able to read public org")
	}
	// but not write
	if op.CanWrite() {
		t.Error("nil user should not be able to write even on public org")
	}
}

func TestNewOrgPermCtx_AdminUser(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "org-admin-test", Aid: 103, Name: "AdminTestOrg", Uid: "someone"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	admin := &model.TUser{Id: "admin", Aid: 60, Name: "admin"}
	if _, err := eng.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	op := NewOrgPermCtx(ctx, admin, "org-admin-test")
	if !op.IsAdmin() {
		t.Error("admin user should be IsAdmin")
	}
	if !op.IsOrgAdmin() {
		t.Error("admin user should be IsOrgAdmin")
	}
	if !op.CanWrite() {
		t.Error("admin should be able to write")
	}
	if !op.CanExec() {
		t.Error("admin should be able to exec")
	}
	if !op.CanDownload() {
		t.Error("admin should be able to download")
	}
}

// --- NewPipePermCtx ---

func TestNewPipePermCtx_PipeOwner(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Need to set IsMySQL = false since we use sqlite (default is false, but be explicit)
	pipe := &model.TPipeline{Id: "pipe-owner", Uid: "pipe-user", Name: "test-pipe"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	user := &model.TUser{Id: "pipe-user", Aid: 70, Name: "pipeuser"}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	pp := NewPipePermCtx(ctx, user, "pipe-owner")
	if pp.Pipeline() == nil {
		t.Fatal("NewPipePermCtx should resolve pipeline")
	}
	if !pp.IsPipeOwner() {
		t.Error("pipe-user should be pipe owner")
	}
	if !pp.CanRead() {
		t.Error("pipe owner should be able to read")
	}
	if !pp.CanWrite() {
		t.Error("pipe owner should be able to write")
	}
	if !pp.CanExec() {
		t.Error("pipe owner should be able to exec")
	}
}

func TestNewPipePermCtx_AdminCanDoEverything(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{Id: "pipe-admin", Uid: "someone-else", Name: "admin-pipe"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	admin := &model.TUser{Id: "admin", Aid: 80, Name: "admin"}
	if _, err := eng.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	pp := NewPipePermCtx(ctx, admin, "pipe-admin")
	if pp.Pipeline() == nil {
		t.Fatal("NewPipePermCtx should resolve pipeline")
	}
	if !pp.IsAdmin() {
		t.Error("admin should be IsAdmin")
	}
	if !pp.CanRead() {
		t.Error("admin should be able to read")
	}
	if !pp.CanWrite() {
		t.Error("admin should be able to write")
	}
	if !pp.CanExec() {
		t.Error("admin should be able to exec")
	}
}

func TestNewPipePermCtx_EmptyPipeId(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	user := &model.TUser{Id: "u1", Aid: 81, Name: "user1"}
	pp := NewPipePermCtx(ctx, user, "")
	if pp.Pipeline() != nil {
		t.Error("NewPipePermCtx with empty pipeId should not resolve pipeline")
	}
	if pp.CanRead() {
		t.Error("should not be able to read unresolved pipeline")
	}
}

func TestNewPipePermCtx_NonexistentPipe(t *testing.T) {
	setupUserTestDB(t)
	ctx := context.Background()

	user := &model.TUser{Id: "u1", Aid: 82, Name: "user1"}
	pp := NewPipePermCtx(ctx, user, "nonexistent-pipe-id")
	if pp.Pipeline() != nil {
		t.Error("NewPipePermCtx should not resolve nonexistent pipeline")
	}
}

func TestNewPipePermCtx_NilUser(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{Id: "pipe-nil-user", Uid: "someone", Name: "nil-user-pipe"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	pp := NewPipePermCtx(ctx, nil, "pipe-nil-user")
	if pp.Pipeline() == nil {
		t.Fatal("NewPipePermCtx should resolve pipeline even with nil user")
	}
	// nil user, no org perms, not owner => cannot read
	if pp.CanRead() {
		t.Error("nil user should not be able to read private pipeline")
	}
}

func TestNewPipePermCtx_Accessors(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	pipe := &model.TPipeline{Id: "pipe-accessor", Uid: "u1", Name: "accessor-pipe"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}
	user := &model.TUser{Id: "u1", Aid: 83, Name: "user1"}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	pp := NewPipePermCtx(ctx, user, "pipe-accessor")
	if pp.LgUser() == nil {
		t.Error("LgUser() should not be nil")
	}
	if pp.Pipeline() == nil {
		t.Error("Pipeline() should not be nil")
	}
}

// --- Canceled context ---

func TestGetUserCtx_CancelledContext(t *testing.T) {
	eng := setupUserTestDB(t)

	user := &model.TUser{
		Id:   "usr-cancel",
		Aid:  30,
		Name: "canceluser",
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// With a canceled context, the DB query may still succeed (sqlite in-memory
	// is synchronous) or fail depending on timing. Either way, it should not panic.
	_, _ = GetUserCtx(ctx, "usr-cancel")
}

// --- Global context wrapper functions ---

func TestIsOrgAdmin_Global(t *testing.T) {
	eng := setupUserTestDB(t)
	org := &model.TOrg{Id: "o1", Aid: 1, Name: "test org"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{Aid: 1, Uid: "u1", OrgId: "o1", PermAdm: 1}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	if !IsOrgAdmin("u1", "o1") {
		t.Error("expected user to be org admin")
	}
}

func TestGetUsePermRwr_Global(t *testing.T) {
	eng := setupUserTestDB(t)
	org := &model.TOrg{Id: "o1", Aid: 1, Name: "test org"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{Aid: 1, Uid: "u1", OrgId: "o1", PermRw: 2}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	if perm := GetUsePermRwr("u1", "o1"); perm != 2 {
		t.Errorf("expected perm 2, got %d", perm)
	}
}

func TestHasOrgExec_Global(t *testing.T) {
	eng := setupUserTestDB(t)
	org := &model.TOrg{Id: "o1", Aid: 1, Name: "test org"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{Aid: 1, Uid: "u1", OrgId: "o1", PermExec: 1}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	if !HasOrgExec("u1", "o1") {
		t.Error("expected user to have exec permission")
	}
}

func TestGetUserOrg_Global(t *testing.T) {
	eng := setupUserTestDB(t)
	org := &model.TOrg{Id: "o1", Aid: 1, Name: "test org"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{Aid: 1, Uid: "u1", OrgId: "o1", PermRw: 1, PermExec: 1}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	result, ok := GetUserOrg("u1", "o1")
	if !ok {
		t.Fatal("expected to find user org")
	}
	if result.PermRw != 1 || result.PermExec != 1 {
		t.Error("user org permissions mismatch")
	}
}

func TestNewOrgPerm_Global(t *testing.T) {
	eng := setupUserTestDB(t)

	user := &model.TUser{Id: "u1", Aid: 1, Name: "test"}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	org := &model.TOrg{Id: "o1", Aid: 1, Name: "test org", Uid: "u1"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	op := NewOrgPerm(user, "o1")
	if op == nil {
		t.Fatal("expected NewOrgPerm to return non-nil")
	}
	if !op.IsOrgOwner() {
		t.Error("expected user to be org owner")
	}
}

func TestNewPipePerm_Global(t *testing.T) {
	eng := setupUserTestDB(t)

	user := &model.TUser{Id: "u1", Aid: 1, Name: "test"}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	pipe := &model.TPipeline{Id: "p1", Name: "test pipeline", Uid: "u1"}
	if _, err := eng.Insert(pipe); err != nil {
		t.Fatalf("insert pipeline: %v", err)
	}

	pp := NewPipePerm(user, "p1")
	if pp == nil {
		t.Fatal("expected NewPipePerm to return non-nil")
	}
	if !pp.IsPipeOwner() {
		t.Error("expected user to be pipeline owner")
	}
}
