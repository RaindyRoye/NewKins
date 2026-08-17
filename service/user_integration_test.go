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

	// Sync all user-related tables
	err = eng.Sync2(
		&model.TUser{},
		&model.TUserInfo{},
		&model.TUserOrg{},
		&model.TOrg{},
	)
	if err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// TestGetUserCtx_Integ tests fetching a user by ID with context.
func TestGetUserCtx_Integ(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Insert test user
	user := &model.TUser{
		Id:      "test-user-1",
		Aid:     1,
		Name:    "alice",
		Nick:    "Alice",
		Avatar:  "avatar1.png",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Test finding existing user
	found, ok := GetUserCtx(ctx, "test-user-1")
	if !ok {
		t.Fatal("GetUserCtx should find existing user")
	}
	if found.Name != "alice" {
		t.Errorf("Name = %q, want %q", found.Name, "alice")
	}
	if found.Nick != "Alice" {
		t.Errorf("Nick = %q, want %q", found.Nick, "Alice")
	}
	if found.Avatar != "avatar1.png" {
		t.Errorf("Avatar = %q, want %q", found.Avatar, "avatar1.png")
	}

	// Test not found
	notFound, ok := GetUserCtx(ctx, "nonexistent")
	if ok {
		t.Error("GetUserCtx should return false for nonexistent user")
	}
	if notFound == nil {
		t.Error("GetUserCtx should return empty struct even when not found")
	}

	// Test empty uid
	empty, ok := GetUserCtx(ctx, "")
	if ok {
		t.Error("GetUserCtx should return false for empty uid")
	}
	if empty != nil {
		t.Error("GetUserCtx should return nil for empty uid")
	}
}

// TestGetUserInfoCtx_Integ tests fetching user info with context.
func TestGetUserInfoCtx_Integ(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Insert user and user info
	user := &model.TUser{
		Id:      "test-user-2",
		Aid:     2,
		Name:    "bob",
		Nick:    "Bob",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	userInfo := &model.TUserInfo{
		Id:       "test-user-2",
		Phone:    "123-456-7890",
		Email:    "bob@example.com",
		Remark:   "Test remark",
		PermUser: 1,
		PermOrg:  0,
		PermPipe: 0,
	}
	if _, err := eng.Insert(userInfo); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	// Test finding existing user info
	found, ok := GetUserInfoCtx(ctx, "test-user-2")
	if !ok {
		t.Fatal("GetUserInfoCtx should find existing user info")
	}
	if found.Phone != "123-456-7890" {
		t.Errorf("Phone = %q, want %q", found.Phone, "123-456-7890")
	}
	if found.Email != "bob@example.com" {
		t.Errorf("Email = %q, want %q", found.Email, "bob@example.com")
	}
	if found.Remark != "Test remark" {
		t.Errorf("Remark = %q, want %q", found.Remark, "Test remark")
	}
	if found.PermUser != 1 {
		t.Errorf("PermUser = %d, want 1", found.PermUser)
	}

	// Test not found
	notFound, ok := GetUserInfoCtx(ctx, "nonexistent")
	if ok {
		t.Error("GetUserInfoCtx should return false for nonexistent user")
	}
	if notFound == nil {
		t.Error("GetUserInfoCtx should return empty struct even when not found")
	}

	// Test empty uid
	empty, ok := GetUserInfoCtx(ctx, "")
	if ok {
		t.Error("GetUserInfoCtx should return false for empty uid")
	}
	if empty != nil {
		t.Error("GetUserInfoCtx should return nil for empty uid")
	}
}

// TestFindUserNameCtx_Integ tests finding a user by name.
func TestFindUserNameCtx_Integ(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Insert test user
	user := &model.TUser{
		Id:      "test-user-3",
		Aid:     3,
		Name:    "charlie",
		Nick:    "Charlie",
		Avatar:  "charlie.png",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Test finding by name
	found, ok := FindUserNameCtx(ctx, "charlie")
	if !ok {
		t.Fatal("FindUserNameCtx should find user by name")
	}
	if found.Id != "test-user-3" {
		t.Errorf("Id = %q, want %q", found.Id, "test-user-3")
	}
	if found.Nick != "Charlie" {
		t.Errorf("Nick = %q, want %q", found.Nick, "Charlie")
	}
	if found.Avatar != "charlie.png" {
		t.Errorf("Avatar = %q, want %q", found.Avatar, "charlie.png")
	}

	// Test not found
	notFound, ok := FindUserNameCtx(ctx, "nonexistent")
	if ok {
		t.Error("FindUserNameCtx should return false for nonexistent name")
	}
	if notFound == nil {
		t.Error("FindUserNameCtx should return empty struct even when not found")
	}
}

// TestIsOrgAdminCtx_Integ tests checking if a user is an org admin.
func TestIsOrgAdminCtx_Integ(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Create org and user
	org := &model.TOrg{
		Id:      "org-1",
		Aid:     1,
		Name:    "Test Org",
		Uid:     "owner-1",
		Public:  0,
		Deleted: 0,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{
		Id:      "admin-user",
		Aid:     10,
		Name:    "admin",
		Nick:    "Admin",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Create user-org relationship with admin permission
	userOrg := &model.TUserOrg{
		Aid:      1,
		Uid:      "admin-user",
		OrgId:    "org-1",
		PermAdm:  1,
		PermRw:   0,
		PermExec: 0,
		Created:  time.Now(),
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	// Test admin user
	if !IsOrgAdminCtx(ctx, "admin-user", "org-1") {
		t.Error("IsOrgAdminCtx should return true for user with PermAdm=1")
	}

	// Test non-admin user
	regularUser := &model.TUser{
		Id:      "regular-user",
		Aid:     11,
		Name:    "regular",
		Nick:    "Regular",
		Created: time.Now(),
	}
	if _, err := eng.Insert(regularUser); err != nil {
		t.Fatalf("insert regular user: %v", err)
	}

	regularUserOrg := &model.TUserOrg{
		Aid:      2,
		Uid:      "regular-user",
		OrgId:    "org-1",
		PermAdm:  0,
		PermRw:   1,
		PermExec: 1,
		Created:  time.Now(),
	}
	if _, err := eng.Insert(regularUserOrg); err != nil {
		t.Fatalf("insert regular user org: %v", err)
	}

	if IsOrgAdminCtx(ctx, "regular-user", "org-1") {
		t.Error("IsOrgAdminCtx should return false for user with PermAdm=0")
	}

	// Test nonexistent org
	if IsOrgAdminCtx(ctx, "admin-user", "nonexistent-org") {
		t.Error("IsOrgAdminCtx should return false for nonexistent org")
	}

	// Test nonexistent user
	if IsOrgAdminCtx(ctx, "nonexistent-user", "org-1") {
		t.Error("IsOrgAdminCtx should return false for nonexistent user")
	}
}

// TestGetUsePermRwrCtx_Integ tests getting read-write permission level.
func TestGetUsePermRwrCtx_Integ(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Create org and user
	org := &model.TOrg{
		Id:      "org-2",
		Aid:     2,
		Name:    "Test Org 2",
		Uid:     "owner-2",
		Public:  0,
		Deleted: 0,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{
		Id:      "rw-user",
		Aid:     20,
		Name:    "rwuser",
		Nick:    "RW User",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Create user-org relationship with read-write permission
	userOrg := &model.TUserOrg{
		Aid:      10,
		Uid:      "rw-user",
		OrgId:    "org-2",
		PermAdm:  0,
		PermRw:   1,
		PermExec: 0,
		Created:  time.Now(),
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	// Test read-write permission
	perm := GetUsePermRwrCtx(ctx, "rw-user", "org-2")
	if perm != 1 {
		t.Errorf("GetUsePermRwrCtx = %d, want 1", perm)
	}

	// Test user without permission
	regularUser := &model.TUser{
		Id:      "noread-user",
		Aid:     21,
		Name:    "noreaduser",
		Nick:    "No Read User",
		Created: time.Now(),
	}
	if _, err := eng.Insert(regularUser); err != nil {
		t.Fatalf("insert regular user: %v", err)
	}

	noPermUserOrg := &model.TUserOrg{
		Aid:      11,
		Uid:      "noread-user",
		OrgId:    "org-2",
		PermAdm:  0,
		PermRw:   0,
		PermExec: 0,
		Created:  time.Now(),
	}
	if _, err := eng.Insert(noPermUserOrg); err != nil {
		t.Fatalf("insert no perm user org: %v", err)
	}

	perm = GetUsePermRwrCtx(ctx, "noread-user", "org-2")
	if perm != 0 {
		t.Errorf("GetUsePermRwrCtx = %d, want 0", perm)
	}

	// Test nonexistent user
	perm = GetUsePermRwrCtx(ctx, "nonexistent", "org-2")
	if perm != 0 {
		t.Errorf("GetUsePermRwrCtx for nonexistent user = %d, want 0", perm)
	}
}

// TestHasOrgExecCtx_Integ tests checking if a user has exec permission.
func TestHasOrgExecCtx_Integ(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Create org and user
	org := &model.TOrg{
		Id:      "org-3",
		Aid:     3,
		Name:    "Test Org 3",
		Uid:     "owner-3",
		Public:  0,
		Deleted: 0,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{
		Id:      "exec-user",
		Aid:     30,
		Name:    "execuser",
		Nick:    "Exec User",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Create user-org relationship with exec permission
	userOrg := &model.TUserOrg{
		Aid:      20,
		Uid:      "exec-user",
		OrgId:    "org-3",
		PermAdm:  0,
		PermRw:   0,
		PermExec: 1,
		Created:  time.Now(),
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	// Test exec permission
	if !HasOrgExecCtx(ctx, "exec-user", "org-3") {
		t.Error("HasOrgExecCtx should return true for user with PermExec=1")
	}

	// Test user without exec permission
	noExecUser := &model.TUser{
		Id:      "noexec-user",
		Aid:     31,
		Name:    "noexecuser",
		Nick:    "No Exec User",
		Created: time.Now(),
	}
	if _, err := eng.Insert(noExecUser); err != nil {
		t.Fatalf("insert no exec user: %v", err)
	}

	noExecUserOrg := &model.TUserOrg{
		Aid:      21,
		Uid:      "noexec-user",
		OrgId:    "org-3",
		PermAdm:  0,
		PermRw:   1,
		PermExec: 0,
		Created:  time.Now(),
	}
	if _, err := eng.Insert(noExecUserOrg); err != nil {
		t.Fatalf("insert no exec user org: %v", err)
	}

	if HasOrgExecCtx(ctx, "noexec-user", "org-3") {
		t.Error("HasOrgExecCtx should return false for user with PermExec=0")
	}

	// Test nonexistent org
	if HasOrgExecCtx(ctx, "exec-user", "nonexistent-org") {
		t.Error("HasOrgExecCtx should return false for nonexistent org")
	}
}

// TestGetUserOrgCtx_Integ tests fetching user-org relationship.
func TestGetUserOrgCtx_Integ(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Create org and user
	org := &model.TOrg{
		Id:      "org-4",
		Aid:     4,
		Name:    "Test Org 4",
		Uid:     "owner-4",
		Public:  1,
		Deleted: 0,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{
		Id:      "orguser-1",
		Aid:     40,
		Name:    "orguser1",
		Nick:    "Org User 1",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Create user-org relationship
	userOrg := &model.TUserOrg{
		Aid:      30,
		Uid:      "orguser-1",
		OrgId:    "org-4",
		PermAdm:  1,
		PermRw:   1,
		PermExec: 1,
		Created:  time.Now(),
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	// Test finding user org
	found, ok := GetUserOrgCtx(ctx, "orguser-1", "org-4")
	if !ok {
		t.Fatal("GetUserOrgCtx should find user org relationship")
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

	// Test not found - wrong user
	notFound, ok := GetUserOrgCtx(ctx, "wrong-user", "org-4")
	if ok {
		t.Error("GetUserOrgCtx should return false for wrong user")
	}
	if notFound != nil {
		t.Error("GetUserOrgCtx should return nil when not found")
	}

	// Test not found - wrong org
	notFound, ok = GetUserOrgCtx(ctx, "orguser-1", "wrong-org")
	if ok {
		t.Error("GetUserOrgCtx should return false for wrong org")
	}
	if notFound != nil {
		t.Error("GetUserOrgCtx should return nil when org not found")
	}
}

// TestNewOrgPermCtx_Integ tests the OrgPerm constructor with context.
func TestNewOrgPermCtx_Integ(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Create org and user
	org := &model.TOrg{
		Id:      "org-5",
		Aid:     5,
		Name:    "Test Org 5",
		Uid:     "owner-5",
		Public:  0,
		Deleted: 0,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{
		Id:      "orgperm-user",
		Aid:     50,
		Name:    "orgpermuser",
		Nick:    "Org Perm User",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Create user-org relationship
	userOrg := &model.TUserOrg{
		Aid:      40,
		Uid:      "orgperm-user",
		OrgId:    "org-5",
		PermAdm:  1,
		PermRw:   1,
		PermExec: 1,
		Created:  time.Now(),
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	// Test NewOrgPermCtx
	op := NewOrgPermCtx(ctx, user, "org-5")
	if op == nil {
		t.Fatal("NewOrgPermCtx should return non-nil OrgPerm")
	}
	if op.Org() == nil {
		t.Error("OrgPerm.Org() should return non-nil org")
	}
	if op.LgUser() == nil {
		t.Error("OrgPerm.LgUser() should return non-nil user")
	}
	if op.UserOrg() == nil {
		t.Error("OrgPerm.UserOrg() should return non-nil userOrg")
	}
	if !op.IsOrgAdmin() {
		t.Error("OrgPerm.IsOrgAdmin() should return true for admin user")
	}
	if !op.CanRead() {
		t.Error("OrgPerm.CanRead() should return true for admin user")
	}
	if !op.CanWrite() {
		t.Error("OrgPerm.CanWrite() should return true for admin user")
	}
	if !op.CanExec() {
		t.Error("OrgPerm.CanExec() should return true for admin user")
	}

	// Test with nonexistent org
	op2 := NewOrgPermCtx(ctx, user, "nonexistent-org")
	if op2 == nil {
		t.Fatal("NewOrgPermCtx should return non-nil OrgPerm even for nonexistent org")
	}
	if op2.Org() != nil {
		t.Error("OrgPerm.Org() should return nil for nonexistent org")
	}
	if op2.IsOrgAdmin() {
		t.Error("OrgPerm.IsOrgAdmin() should return false for nonexistent org")
	}

	// Test with nil user
	op3 := NewOrgPermCtx(ctx, nil, "org-5")
	if op3 == nil {
		t.Fatal("NewOrgPermCtx should return non-nil OrgPerm even for nil user")
	}
	if op3.LgUser() != nil {
		t.Error("OrgPerm.LgUser() should return nil for nil user")
	}

	// Test with empty orgId
	op4 := NewOrgPermCtx(ctx, user, "")
	if op4 == nil {
		t.Fatal("NewOrgPermCtx should return non-nil OrgPerm even for empty orgId")
	}
	if op4.Org() != nil {
		t.Error("OrgPerm.Org() should return nil for empty orgId")
	}
}

// TestNewOrgPermCtx_DeletedOrg tests that deleted orgs are not accessible.
func TestNewOrgPermCtx_DeletedOrg(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Create deleted org
	org := &model.TOrg{
		Id:      "org-deleted",
		Aid:     6,
		Name:    "Deleted Org",
		Uid:     "owner-6",
		Public:  0,
		Deleted: 1,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{
		Id:      "del-user",
		Aid:     60,
		Name:    "deluser",
		Nick:    "Del User",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Test with deleted org
	op := NewOrgPermCtx(ctx, user, "org-deleted")
	if op == nil {
		t.Fatal("NewOrgPermCtx should return non-nil OrgPerm")
	}
	if op.Org() != nil {
		t.Error("OrgPerm.Org() should return nil for deleted org")
	}
	if op.CanRead() {
		t.Error("OrgPerm.CanRead() should return false for deleted org")
	}
}

// TestNewOrgPermCtx_PublicOrg tests that public orgs allow read access.
func TestNewOrgPermCtx_PublicOrg(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Create public org
	org := &model.TOrg{
		Id:      "org-public",
		Aid:     7,
		Name:    "Public Org",
		Uid:     "owner-7",
		Public:  1,
		Deleted: 0,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{
		Id:      "pub-user",
		Aid:     70,
		Name:    "pubuser",
		Nick:    "Pub User",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Test with public org - user is not a member
	op := NewOrgPermCtx(ctx, user, "org-public")
	if op == nil {
		t.Fatal("NewOrgPermCtx should return non-nil OrgPerm")
	}
	if op.Org() == nil {
		t.Error("OrgPerm.Org() should return non-nil for public org")
	}
	if !op.IsOrgPublic() {
		t.Error("OrgPerm.IsOrgPublic() should return true for public org")
	}
	if !op.CanRead() {
		t.Error("OrgPerm.CanRead() should return true for public org")
	}
	if op.CanWrite() {
		t.Error("OrgPerm.CanWrite() should return false for non-member of public org")
	}
	if op.CanExec() {
		t.Error("OrgPerm.CanExec() should return false for non-member of public org")
	}
}

// TestCheckPermissionCtx_Integ tests permission checking with context.
func TestCheckPermissionCtx_Integ(t *testing.T) {
	eng := setupUserTestDB(t)
	ctx := context.Background()

	// Insert admin user
	adminUser := &model.TUser{
		Id:      "admin",
		Aid:     100,
		Name:    "admin",
		Nick:    "Admin",
		Created: time.Now(),
	}
	if _, err := eng.Insert(adminUser); err != nil {
		t.Fatalf("insert admin user: %v", err)
	}

	// Insert regular user
	regularUser := &model.TUser{
		Id:      "user1",
		Aid:     101,
		Name:    "alice",
		Nick:    "Alice",
		Created: time.Now(),
	}
	if _, err := eng.Insert(regularUser); err != nil {
		t.Fatalf("insert regular user: %v", err)
	}

	// Test admin permission for admin user
	if !CheckPermissionCtx(ctx, "admin", "admin") {
		t.Error("CheckPermissionCtx should return true for admin user with admin permission")
	}

	// Test common permission for regular user
	if !CheckPermissionCtx(ctx, "user1", "common") {
		t.Error("CheckPermissionCtx should return true for regular user with common permission")
	}

	// Test admin permission for regular user
	if CheckPermissionCtx(ctx, "user1", "admin") {
		t.Error("CheckPermissionCtx should return false for regular user with admin permission")
	}

	// Test nonexistent user
	if CheckPermissionCtx(ctx, "nonexistent", "common") {
		t.Error("CheckPermissionCtx should return false for nonexistent user")
	}
}

// TestGlobalWrappers_Integ tests global context wrappers with real DB.
func TestGlobalWrappers_Integ(t *testing.T) {
	eng := setupUserTestDB(t)

	// Insert test user
	user := &model.TUser{
		Id:      "global-user-1",
		Aid:     100,
		Name:    "globaluser",
		Nick:    "Global User",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Test GetUser
	found, ok := GetUser("global-user-1")
	if !ok {
		t.Fatal("GetUser should find existing user")
	}
	if found.Name != "globaluser" {
		t.Errorf("Name = %q, want %q", found.Name, "globaluser")
	}

	// Test FindUserName
	found2, ok := FindUserName("globaluser")
	if !ok {
		t.Fatal("FindUserName should find user by name")
	}
	if found2.Id != "global-user-1" {
		t.Errorf("Id = %q, want %q", found2.Id, "global-user-1")
	}

	// Create org and user-org relationship
	org := &model.TOrg{
		Id:      "global-org-1",
		Aid:     200,
		Name:    "Global Org",
		Uid:     "global-owner",
		Public:  0,
		Deleted: 0,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:      100,
		Uid:      "global-user-1",
		OrgId:    "global-org-1",
		PermAdm:  1,
		PermRw:   1,
		PermExec: 1,
		Created:  time.Now(),
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	// Test IsOrgAdmin
	if !IsOrgAdmin("global-user-1", "global-org-1") {
		t.Error("IsOrgAdmin should return true for admin user")
	}

	// Test GetUsePermRwr
	perm := GetUsePermRwr("global-user-1", "global-org-1")
	if perm != 1 {
		t.Errorf("GetUsePermRwr = %d, want 1", perm)
	}

	// Test HasOrgExec
	if !HasOrgExec("global-user-1", "global-org-1") {
		t.Error("HasOrgExec should return true for user with exec permission")
	}

	// Test GetUserOrg
	found3, ok := GetUserOrg("global-user-1", "global-org-1")
	if !ok {
		t.Fatal("GetUserOrg should find user org relationship")
	}
	if found3.PermAdm != 1 {
		t.Errorf("PermAdm = %d, want 1", found3.PermAdm)
	}

	// Test CheckPermission
	if !CheckPermission("global-user-1", "common") {
		t.Error("CheckPermission should return true for common permission")
	}
}

// TestNewOrgPermGlobal_Integ tests the global context wrapper for NewOrgPerm.
func TestNewOrgPermGlobal_Integ(t *testing.T) {
	eng := setupUserTestDB(t)

	// Create org and user
	org := &model.TOrg{
		Id:      "global-org-2",
		Aid:     202,
		Name:    "Global Org 2",
		Uid:     "global-owner-2",
		Public:  0,
		Deleted: 0,
		Created: time.Now(),
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{
		Id:      "global-orgperm-user",
		Aid:     203,
		Name:    "globalorgpermuser",
		Nick:    "Global Org Perm User",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	userOrg := &model.TUserOrg{
		Aid:      101,
		Uid:      "global-orgperm-user",
		OrgId:    "global-org-2",
		PermAdm:  1,
		PermRw:   1,
		PermExec: 1,
		Created:  time.Now(),
	}
	if _, err := eng.Insert(userOrg); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	// Test NewOrgPerm
	op := NewOrgPerm(user, "global-org-2")
	if op == nil {
		t.Fatal("NewOrgPerm should return non-nil OrgPerm")
	}
	if !op.IsOrgAdmin() {
		t.Error("NewOrgPerm should create OrgPerm with admin access")
	}
}
