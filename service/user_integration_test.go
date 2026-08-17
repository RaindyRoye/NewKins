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

func setupUserIntegDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE t_user (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		name VARCHAR(100),
		pass VARCHAR(255),
		nick VARCHAR(100),
		avatar VARCHAR(500),
		created DATETIME,
		login_time DATETIME,
		active INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_user: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_info (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		phone VARCHAR(100),
		email VARCHAR(200),
		birthday DATETIME,
		remark TEXT,
		perm_user INT,
		perm_org INT,
		perm_pipe INT
	)`)
	if err != nil {
		t.Fatalf("create t_user_info: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_org (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		aid BIGINT,
		uid VARCHAR(64),
		name VARCHAR(200),
		"desc" TEXT,
		public INT DEFAULT 0,
		created DATETIME,
		updated DATETIME,
		deleted INT DEFAULT 0,
		deleted_time DATETIME
	)`)
	if err != nil {
		t.Fatalf("create t_org: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE t_user_org (
		aid INTEGER PRIMARY KEY AUTOINCREMENT,
		uid VARCHAR(64),
		org_id VARCHAR(64),
		created DATETIME,
		perm_adm INT DEFAULT 0,
		perm_rw INT DEFAULT 0,
		perm_exec INT DEFAULT 0,
		perm_down INT DEFAULT 0
	)`)
	if err != nil {
		t.Fatalf("create t_user_org: %v", err)
	}

	comm.Db = db
}

func insertTestUser(t *testing.T, id, name, nick string) {
	t.Helper()
	u := &model.TUser{
		Id:        id,
		Name:      name,
		Nick:      nick,
		Pass:      "hashed",
		Active:    1,
		Created:   time.Now(),
		LoginTime: time.Now(),
	}
	_, err := comm.Db.InsertOne(u)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetUserCtx / GetUser
// ---------------------------------------------------------------------------

func TestGetUserCtx_Found(t *testing.T) {
	setupUserIntegDB(t)
	insertTestUser(t, "u1", "alice", "Alice")

	u, ok := GetUserCtx(context.Background(), "u1")
	if !ok {
		t.Fatal("expected user found")
	}
	if u.Name != "alice" {
		t.Errorf("name = %q, want alice", u.Name)
	}
}

func TestGetUserCtx_NotFound(t *testing.T) {
	setupUserIntegDB(t)
	_, ok := GetUserCtx(context.Background(), "nonexistent")
	if ok {
		t.Error("expected user not found")
	}
}

func TestGetUserCtx_EmptyUid(t *testing.T) {
	setupUserIntegDB(t)
	_, ok := GetUserCtx(context.Background(), "")
	if ok {
		t.Error("expected empty uid returns false")
	}
}

func TestGetUser_Found(t *testing.T) {
	setupUserIntegDB(t)
	insertTestUser(t, "u2", "bob", "Bob")

	u, ok := GetUser("u2")
	if !ok {
		t.Fatal("expected user found via GetUser")
	}
	if u.Nick != "Bob" {
		t.Errorf("nick = %q, want Bob", u.Nick)
	}
}

// ---------------------------------------------------------------------------
// GetUserInfoCtx / GetUserInfo
// ---------------------------------------------------------------------------

func TestGetUserInfoCtx_Found(t *testing.T) {
	setupUserIntegDB(t)
	// Insert user info
	_, err := comm.Db.Exec(
		`INSERT INTO t_user_info (id, phone, email, perm_user, perm_org, perm_pipe) VALUES (?, ?, ?, ?, ?, ?)`,
		"u1", "123", "alice@test.com", 1, 0, 1,
	)
	if err != nil {
		t.Fatalf("insert user_info: %v", err)
	}

	info, ok := GetUserInfoCtx(context.Background(), "u1")
	if !ok {
		t.Fatal("expected user info found")
	}
	if info.Phone != "123" {
		t.Errorf("phone = %q, want 123", info.Phone)
	}
	if info.PermUser != 1 {
		t.Errorf("perm_user = %d, want 1", info.PermUser)
	}
	if info.PermPipe != 1 {
		t.Errorf("perm_pipe = %d, want 1", info.PermPipe)
	}
}

func TestGetUserInfoCtx_NotFound(t *testing.T) {
	setupUserIntegDB(t)
	_, ok := GetUserInfoCtx(context.Background(), "no_such_user")
	if ok {
		t.Error("expected user info not found")
	}
}

func TestGetUserInfoCtx_EmptyUid(t *testing.T) {
	setupUserIntegDB(t)
	_, ok := GetUserInfoCtx(context.Background(), "")
	if ok {
		t.Error("expected empty uid returns false")
	}
}

// ---------------------------------------------------------------------------
// FindUserNameCtx / FindUserName
// ---------------------------------------------------------------------------

func TestFindUserNameCtx_Found(t *testing.T) {
	setupUserIntegDB(t)
	insertTestUser(t, "u3", "charlie", "Charlie")

	u, ok := FindUserNameCtx(context.Background(), "charlie")
	if !ok {
		t.Fatal("expected user found by name")
	}
	if u.Id != "u3" {
		t.Errorf("id = %q, want u3", u.Id)
	}
}

func TestFindUserNameCtx_NotFound(t *testing.T) {
	setupUserIntegDB(t)
	_, ok := FindUserNameCtx(context.Background(), "nobody")
	if ok {
		t.Error("expected user not found by name")
	}
}

func TestFindUserName_Found(t *testing.T) {
	setupUserIntegDB(t)
	insertTestUser(t, "u4", "dave", "Dave")

	u, ok := FindUserName("dave")
	if !ok {
		t.Fatal("expected user found by FindUserName")
	}
	if u.Nick != "Dave" {
		t.Errorf("nick = %q, want Dave", u.Nick)
	}
}

// ---------------------------------------------------------------------------
// IsAdmin
// ---------------------------------------------------------------------------

// Note: IsAdmin is already covered by TestIsAdmin in perms_test.go.
// Skipping duplicate coverage here.

// ---------------------------------------------------------------------------
// IsOrgAdminCtx / IsOrgAdmin
// ---------------------------------------------------------------------------

func TestIsOrgAdminCtx_True(t *testing.T) {
	setupUserIntegDB(t)
	// Create org
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org1", Uid: "owner", Name: "Test Org",
		Created: time.Now(), Updated: time.Now(),
	})
	// Create user_org with perm_adm=1
	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid: "u1", OrgId: "org1", PermAdm: 1, Created: time.Now(),
	})

	if !IsOrgAdminCtx(context.Background(), "u1", "org1") {
		t.Error("expected u1 to be org admin")
	}
}

func TestIsOrgAdminCtx_False(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org2", Uid: "owner", Name: "Test Org 2",
		Created: time.Now(), Updated: time.Now(),
	})
	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid: "u1", OrgId: "org2", PermAdm: 0, Created: time.Now(),
	})

	if IsOrgAdminCtx(context.Background(), "u1", "org2") {
		t.Error("expected u1 to NOT be org admin")
	}
}

func TestIsOrgAdminCtx_NoMembership(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org3", Uid: "owner", Name: "Test Org 3",
		Created: time.Now(), Updated: time.Now(),
	})

	if IsOrgAdminCtx(context.Background(), "u_nonmember", "org3") {
		t.Error("expected non-member to not be org admin")
	}
}

func TestIsOrgAdmin_Wrapper(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org4", Uid: "owner", Name: "Test Org 4",
		Created: time.Now(), Updated: time.Now(),
	})
	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid: "u_admin", OrgId: "org4", PermAdm: 1, Created: time.Now(),
	})

	if !IsOrgAdmin("u_admin", "org4") {
		t.Error("expected IsOrgAdmin wrapper to work")
	}
}

// ---------------------------------------------------------------------------
// GetUsePermRwrCtx / GetUsePermRwr
// ---------------------------------------------------------------------------

func TestGetUsePermRwrCtx(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org_rw", Uid: "owner", Name: "RW Org",
		Created: time.Now(), Updated: time.Now(),
	})
	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid: "u1", OrgId: "org_rw", PermRw: 2, Created: time.Now(),
	})

	got := GetUsePermRwrCtx(context.Background(), "u1", "org_rw")
	if got != 2 {
		t.Errorf("perm_rw = %d, want 2", got)
	}

	got2 := GetUsePermRwrCtx(context.Background(), "nonmember", "org_rw")
	if got2 != 0 {
		t.Errorf("non-member perm_rw = %d, want 0", got2)
	}
}

func TestGetUsePermRwr(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org_rw2", Uid: "owner", Name: "RW2 Org",
		Created: time.Now(), Updated: time.Now(),
	})
	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid: "u1", OrgId: "org_rw2", PermRw: 1, Created: time.Now(),
	})

	got := GetUsePermRwr("u1", "org_rw2")
	if got != 1 {
		t.Errorf("perm_rw = %d, want 1", got)
	}
}

// ---------------------------------------------------------------------------
// HasOrgExecCtx / HasOrgExec
// ---------------------------------------------------------------------------

func TestHasOrgExecCtx(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org_ex", Uid: "owner", Name: "Exec Org",
		Created: time.Now(), Updated: time.Now(),
	})
	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid: "u1", OrgId: "org_ex", PermExec: 1, Created: time.Now(),
	})

	if !HasOrgExecCtx(context.Background(), "u1", "org_ex") {
		t.Error("expected u1 to have exec")
	}
	if HasOrgExecCtx(context.Background(), "u_noexec", "org_ex") {
		t.Error("expected non-member to not have exec")
	}
}

func TestHasOrgExec(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org_ex2", Uid: "owner", Name: "Exec2 Org",
		Created: time.Now(), Updated: time.Now(),
	})
	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid: "u1", OrgId: "org_ex2", PermExec: 1, Created: time.Now(),
	})

	if !HasOrgExec("u1", "org_ex2") {
		t.Error("expected u1 to have exec via wrapper")
	}
}

// ---------------------------------------------------------------------------
// GetUserOrgCtx / GetUserOrg
// ---------------------------------------------------------------------------

func TestGetUserOrgCtx_Found(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org_uo", Uid: "owner", Name: "UO Org",
		Created: time.Now(), Updated: time.Now(),
	})
	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid: "u1", OrgId: "org_uo", PermAdm: 1, PermRw: 1, PermExec: 1, Created: time.Now(),
	})

	uo, ok := GetUserOrgCtx(context.Background(), "u1", "org_uo")
	if !ok {
		t.Fatal("expected user org found")
	}
	if uo.PermAdm != 1 {
		t.Errorf("perm_adm = %d, want 1", uo.PermAdm)
	}
}

func TestGetUserOrgCtx_NotFound(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org_uo2", Uid: "owner", Name: "UO2 Org",
		Created: time.Now(), Updated: time.Now(),
	})

	_, ok := GetUserOrgCtx(context.Background(), "no_member", "org_uo2")
	if ok {
		t.Error("expected user org not found")
	}
}

func TestGetUserOrg_Wrapper(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org_uo3", Uid: "owner", Name: "UO3 Org",
		Created: time.Now(), Updated: time.Now(),
	})
	_, _ = comm.Db.InsertOne(&model.TUserOrg{
		Uid: "u1", OrgId: "org_uo3", Created: time.Now(),
	})

	_, ok := GetUserOrg("u1", "org_uo3")
	if !ok {
		t.Error("expected GetUserOrg wrapper to work")
	}
}

// ---------------------------------------------------------------------------
// NewOrgPermCtx / NewOrgPerm
// ---------------------------------------------------------------------------

func TestNewOrgPermCtx_AdminUser(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org_np", Uid: "other", Name: "NP Org", Public: 0,
		Created: time.Now(), Updated: time.Now(),
	})

	admin := &model.TUser{Id: "admin"}
	perm := NewOrgPermCtx(context.Background(), admin, "org_np")
	if perm.Org() == nil {
		t.Fatal("expected org to be loaded")
	}
	if !perm.IsAdmin() {
		t.Error("expected admin user to be admin")
	}
	if !perm.CanRead() {
		t.Error("admin should be able to read")
	}
	if !perm.CanWrite() {
		t.Error("admin should be able to write")
	}
}

func TestNewOrgPermCtx_DeletedOrg(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org_del", Uid: "other", Name: "Del Org", Deleted: 1,
		Created: time.Now(), Updated: time.Now(),
	})

	usr := &model.TUser{Id: "u1"}
	perm := NewOrgPermCtx(context.Background(), usr, "org_del")
	if perm.Org() != nil {
		t.Error("expected org to be nil for deleted org")
	}
}

func TestNewOrgPermCtx_EmptyOrgId(t *testing.T) {
	setupUserIntegDB(t)
	usr := &model.TUser{Id: "u1"}
	perm := NewOrgPermCtx(context.Background(), usr, "")
	if perm.Org() != nil {
		t.Error("expected org to be nil for empty orgId")
	}
}

func TestNewOrgPermCtx_NilUser(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org_nil", Uid: "other", Name: "Nil User Org", Public: 1,
		Created: time.Now(), Updated: time.Now(),
	})

	perm := NewOrgPermCtx(context.Background(), nil, "org_nil")
	if perm.Org() == nil {
		t.Error("expected org to be loaded even with nil user")
	}
	if perm.LgUser() != nil {
		t.Error("expected LgUser to be nil")
	}
}

func TestNewOrgPerm_Wrapper(t *testing.T) {
	setupUserIntegDB(t)
	_, _ = comm.Db.InsertOne(&model.TOrg{
		Id: "org_npw", Uid: "other", Name: "NPW Org", Public: 1,
		Created: time.Now(), Updated: time.Now(),
	})

	usr := &model.TUser{Id: "u1"}
	perm := NewOrgPerm(usr, "org_npw")
	if perm.Org() == nil {
		t.Error("expected org to be loaded via wrapper")
	}
}

// ---------------------------------------------------------------------------
// ClearUserCache (smoke test - just ensure no panic)
// ---------------------------------------------------------------------------

func TestClearUserCache_NoPanic(t *testing.T) {
	// ClearUserCache calls comm.CacheSet which may fail without BCache,
	// but should not panic
	ClearUserCache("u1")
	ClearUserCache("")
}

// ---------------------------------------------------------------------------
// GetUserCacheCtx / GetUserCache (smoke tests - without real cache)
// ---------------------------------------------------------------------------

func TestGetUserCacheCtx_WithoutCache(t *testing.T) {
	setupUserIntegDB(t)
	insertTestUser(t, "cache_u1", "cacheuser", "Cache User")

	// Without BCache initialized, GetUserCacheCtx should fall back to DB
	u, ok := GetUserCacheCtx(context.Background(), "cache_u1")
	if !ok {
		t.Fatal("expected user found via cache fallback")
	}
	if u.Name != "cacheuser" {
		t.Errorf("name = %q, want cacheuser", u.Name)
	}
}

func TestGetUserCache_WithoutCache(t *testing.T) {
	setupUserIntegDB(t)
	insertTestUser(t, "cache_u2", "cacheuser2", "Cache User 2")

	u, ok := GetUserCache("cache_u2")
	if !ok {
		t.Fatal("expected user found via GetUserCache")
	}
	if u.Nick != "Cache User 2" {
		t.Errorf("nick = %q, want 'Cache User 2'", u.Nick)
	}
}
