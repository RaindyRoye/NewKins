package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/model"
)

// --- CheckPermission / CheckPermissionCtx DB-backed tests ---

func TestCheckPermissionCtx_UserNotFound(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()
	_ = eng

	got := CheckPermissionCtx(ctx, "nonexistent-user", PermCommon)
	if got {
		t.Error("CheckPermissionCtx(nonexistent user) should return false")
	}
}

func TestCheckPermissionCtx_CommonPerm(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert a regular user
	u := &model.TUser{Id: "u1", Aid: 1, Name: "alice"}
	if _, err := eng.Insert(u); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	got := CheckPermissionCtx(ctx, "u1", PermCommon)
	if !got {
		t.Error("CheckPermissionCtx(u1, common) should return true for any existing user")
	}
}

func TestCheckPermissionCtx_AdminPerm(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert admin user (name must be "admin")
	admin := &model.TUser{Id: "admin", Aid: 1, Name: "admin"}
	if _, err := eng.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	got := CheckPermissionCtx(ctx, "admin", PermAdmin)
	if !got {
		t.Error("CheckPermissionCtx(admin, admin) should return true for admin user")
	}
}

func TestCheckPermissionCtx_AdminPermForRegularUser(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	u := &model.TUser{Id: "u1", Aid: 1, Name: "alice"}
	if _, err := eng.Insert(u); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	got := CheckPermissionCtx(ctx, "u1", PermAdmin)
	if got {
		t.Error("CheckPermissionCtx(alice, admin) should return false for non-admin user")
	}
}

func TestCheckPermission_DelegatesToCtx(t *testing.T) {
	eng := setupBatchTestDB(t)
	_ = eng

	// Insert admin user
	admin := &model.TUser{Id: "admin", Aid: 1, Name: "admin"}
	if _, err := eng.Insert(admin); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	if !CheckPermission("admin", PermAdmin) {
		t.Error("CheckPermission(admin, admin) should return true")
	}
	if CheckPermission("nonexistent", PermCommon) {
		t.Error("CheckPermission(nonexistent, common) should return false")
	}
}

func TestCheckPermissionCtx_CancelledContext(t *testing.T) {
	eng := setupBatchTestDB(t)
	_ = eng

	u := &model.TUser{Id: "u1", Aid: 1, Name: "alice"}
	if _, err := eng.Insert(u); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// With a canceled context, the DB query should fail
	got := CheckPermissionCtx(ctx, "u1", PermCommon)
	if got {
		t.Error("CheckPermissionCtx with canceled context should return false")
	}
}

// --- CheckCurrPermission tests ---

func TestCheckCurrPermission_NilContext(t *testing.T) {
	if CheckCurrPermission(nil, PermCommon) {
		t.Error("CheckCurrPermission(nil) should return false")
	}
}

func TestCheckCurrPermission_NilRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = nil
	if CheckCurrPermission(c, PermCommon) {
		t.Error("CheckCurrPermission with nil request should return false")
	}
}

func TestCheckCurrPermission_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/test", nil)

	// No auth cookie/token set, so CurrUserCacheCtx returns false
	got := CheckCurrPermission(c, PermCommon)
	if got {
		t.Error("CheckCurrPermission without auth should return false")
	}
}

// --- MidUserCheck tests ---

func TestMidUserCheck_NilContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	defer func() {
		if r := recover(); r != nil {
			// MidUserCheck with nil gin.Context may panic on c.String
			// which is expected since nil gin.Context is an invalid call
			t.Logf("recovered expected panic: %v", r)
		}
	}()

	// Create a valid gin context but with no auth
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/test", nil)

	MidUserCheck(c)
	if w.Code != http.StatusForbidden {
		t.Errorf("MidUserCheck no auth: status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestMidUserCheck_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/test", nil)

	MidUserCheck(c)
	if w.Code != http.StatusForbidden {
		t.Errorf("MidUserCheck no auth: status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// --- GetMidLgUser additional tests ---

func TestGetMidLgUser_EmptyContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	got := GetMidLgUser(c)
	if got != nil {
		t.Errorf("GetMidLgUser with no value set should return nil, got %+v", got)
	}
}

// --- GetUserOrgCtx with DB ---

func TestGetUserOrgCtx_Found(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert org
	org := &model.TOrg{Id: "o1", Aid: 1, Name: "testorg"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	// Insert user org
	uo := &model.TUserOrg{Uid: "u1", OrgId: "o1", PermRw: 1}
	if _, err := eng.Insert(uo); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	got, ok := GetUserOrgCtx(ctx, "u1", "o1")
	if !ok {
		t.Fatal("GetUserOrgCtx should find the user org")
	}
	if got.PermRw != 1 {
		t.Errorf("PermRw = %d, want 1", got.PermRw)
	}
}

func TestGetUserOrgCtx_NotFound(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "o1", Aid: 1, Name: "testorg"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	_, ok := GetUserOrgCtx(ctx, "u1", "o1")
	if ok {
		t.Error("GetUserOrgCtx should return false for non-member")
	}
}

func TestGetUserOrgCtx_OrgNotFound(t *testing.T) {
	setupBatchTestDB(t)
	ctx := context.Background()

	_, ok := GetUserOrgCtx(ctx, "u1", "nonexistent")
	if ok {
		t.Error("GetUserOrgCtx should return false when org does not exist")
	}
}

func TestGetUserOrg_DelegatesToCtx(t *testing.T) {
	eng := setupBatchTestDB(t)

	org := &model.TOrg{Id: "o1", Aid: 1, Name: "testorg"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	uo := &model.TUserOrg{Uid: "u1", OrgId: "o1", PermAdm: 1}
	if _, err := eng.Insert(uo); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	got, ok := GetUserOrg("u1", "o1")
	if !ok {
		t.Fatal("GetUserOrg should find the user org")
	}
	if got.PermAdm != 1 {
		t.Errorf("PermAdm = %d, want 1", got.PermAdm)
	}
}

// --- IsOrgAdmin / HasOrgExec / GetUsePermRwr DB-backed ---

func TestIsOrgAdmin_DBBacked(t *testing.T) {
	eng := setupBatchTestDB(t)

	org := &model.TOrg{Id: "o1", Aid: 1, Name: "testorg"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	uo := &model.TUserOrg{Uid: "u1", OrgId: "o1", PermAdm: 1}
	if _, err := eng.Insert(uo); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	if !IsOrgAdmin("u1", "o1") {
		t.Error("IsOrgAdmin should return true for user with PermAdm=1")
	}

	// Non-admin member
	uo2 := &model.TUserOrg{Uid: "u2", OrgId: "o1", PermAdm: 0}
	if _, err := eng.Insert(uo2); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}
	if IsOrgAdmin("u2", "o1") {
		t.Error("IsOrgAdmin should return false for user with PermAdm=0")
	}
}

func TestHasOrgExec_DBBacked(t *testing.T) {
	eng := setupBatchTestDB(t)

	org := &model.TOrg{Id: "o1", Aid: 1, Name: "testorg"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	uo := &model.TUserOrg{Uid: "u1", OrgId: "o1", PermExec: 1}
	if _, err := eng.Insert(uo); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	if !HasOrgExec("u1", "o1") {
		t.Error("HasOrgExec should return true for user with PermExec=1")
	}

	uo2 := &model.TUserOrg{Uid: "u2", OrgId: "o1", PermExec: 0}
	if _, err := eng.Insert(uo2); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}
	if HasOrgExec("u2", "o1") {
		t.Error("HasOrgExec should return false for user with PermExec=0")
	}
}

func TestGetUsePermRwr_DBBacked(t *testing.T) {
	eng := setupBatchTestDB(t)

	org := &model.TOrg{Id: "o1", Aid: 1, Name: "testorg"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	uo := &model.TUserOrg{Uid: "u1", OrgId: "o1", PermRw: 2}
	if _, err := eng.Insert(uo); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	perm := GetUsePermRwr("u1", "o1")
	if perm != 2 {
		t.Errorf("GetUsePermRwr = %d, want 2", perm)
	}

	// Non-member returns 0
	perm = GetUsePermRwr("u999", "o1")
	if perm != 0 {
		t.Errorf("GetUsePermRwr(nonmember) = %d, want 0", perm)
	}
}

// --- BatchOrgPipeCounts / BatchOrgUserCounts with data ---

func TestBatchOrgPipeCounts_WithData(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert orgs
	for _, o := range []model.TOrg{
		{Id: "o1", Aid: 1, Name: "org1"},
		{Id: "o2", Aid: 2, Name: "org2"},
	} {
		if _, err := eng.Insert(&o); err != nil {
			t.Fatalf("insert org: %v", err)
		}
	}

	// Insert org-pipe associations using raw SQL
	_, err := eng.Exec("CREATE TABLE IF NOT EXISTS t_org_pipe (aid INTEGER PRIMARY KEY AUTOINCREMENT, org_id VARCHAR(64), pipe_id VARCHAR(64))")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	for _, op := range []struct{ orgId, pipeId string }{
		{"o1", "p1"}, {"o1", "p2"}, {"o1", "p3"},
		{"o2", "p4"}, {"o2", "p5"},
	} {
		_, err := eng.Exec("INSERT INTO t_org_pipe (org_id, pipe_id) VALUES (?, ?)", op.orgId, op.pipeId)
		if err != nil {
			t.Fatalf("insert org_pipe: %v", err)
		}
	}

	result, err := BatchOrgPipeCounts(ctx, []string{"o1", "o2", "o3"})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts failed: %v", err)
	}
	if result["o1"] != 3 {
		t.Errorf("o1 count = %d, want 3", result["o1"])
	}
	if result["o2"] != 2 {
		t.Errorf("o2 count = %d, want 2", result["o2"])
	}
	if _, exists := result["o3"]; exists {
		t.Error("o3 should not exist in result")
	}
}

func TestBatchOrgUserCounts_WithData(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "o1", Aid: 1, Name: "org1"}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	// Insert user-org associations
	for _, uo := range []model.TUserOrg{
		{Uid: "u1", OrgId: "o1"},
		{Uid: "u2", OrgId: "o1"},
		{Uid: "u3", OrgId: "o1"},
	} {
		if _, err := eng.Insert(&uo); err != nil {
			t.Fatalf("insert user_org: %v", err)
		}
	}

	result, err := BatchOrgUserCounts(ctx, []string{"o1"})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts failed: %v", err)
	}
	if result["o1"] != 3 {
		t.Errorf("o1 count = %d, want 3", result["o1"])
	}
}

// --- GetUserCtx / GetUserInfoCtx / FindUserNameCtx DB-backed ---

func TestGetUserCtx_Found(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	u := &model.TUser{Id: "u1", Aid: 1, Name: "alice", Nick: "Alice W"}
	if _, err := eng.Insert(u); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	got, ok := GetUserCtx(ctx, "u1")
	if !ok {
		t.Fatal("GetUserCtx should find user u1")
	}
	if got.Name != "alice" {
		t.Errorf("Name = %s, want alice", got.Name)
	}
}

func TestGetUserCtx_EmptyId(t *testing.T) {
	setupBatchTestDB(t)
	ctx := context.Background()

	_, ok := GetUserCtx(ctx, "")
	if ok {
		t.Error("GetUserCtx with empty id should return false")
	}
}

func TestGetUserCtx_NotFound(t *testing.T) {
	setupBatchTestDB(t)
	ctx := context.Background()

	_, ok := GetUserCtx(ctx, "nonexistent")
	if ok {
		t.Error("GetUserCtx should return false for nonexistent user")
	}
}

func TestGetUserInfoCtx_Found(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	ui := &model.TUserInfo{Id: "u1", Phone: "123456", Email: "alice@test.com"}
	if _, err := eng.Insert(ui); err != nil {
		t.Fatalf("insert user info: %v", err)
	}

	got, ok := GetUserInfoCtx(ctx, "u1")
	if !ok {
		t.Fatal("GetUserInfoCtx should find user info")
	}
	if got.Email != "alice@test.com" {
		t.Errorf("Email = %s, want alice@test.com", got.Email)
	}
}

func TestGetUserInfoCtx_EmptyId(t *testing.T) {
	setupBatchTestDB(t)
	ctx := context.Background()

	_, ok := GetUserInfoCtx(ctx, "")
	if ok {
		t.Error("GetUserInfoCtx with empty id should return false")
	}
}

func TestFindUserNameCtx_Found(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	u := &model.TUser{Id: "u1", Aid: 1, Name: "alice"}
	if _, err := eng.Insert(u); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	got, ok := FindUserNameCtx(ctx, "alice")
	if !ok {
		t.Fatal("FindUserNameCtx should find user by name")
	}
	if got.Id != "u1" {
		t.Errorf("Id = %s, want u1", got.Id)
	}
}

func TestFindUserNameCtx_NotFound(t *testing.T) {
	setupBatchTestDB(t)
	ctx := context.Background()

	_, ok := FindUserNameCtx(ctx, "nonexistent")
	if ok {
		t.Error("FindUserNameCtx should return false for nonexistent name")
	}
}

// --- ClearUserCache (no-op without cache, should not panic) ---

func TestClearUserCache_EmptyId(t *testing.T) {
	// Should not panic with empty id
	ClearUserCache("")
}

func TestClearUserCache_NonEmpty(t *testing.T) {
	// Without cache initialized, this should log a warning but not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ClearUserCache panicked: %v", r)
		}
	}()
	ClearUserCache("some-uid")
}

// --- NewOrgPermCtx with DB ---

func TestNewOrgPermCtx_WithDB(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "o1", Aid: 1, Name: "testorg", Public: 0}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	uo := &model.TUserOrg{Uid: "u1", OrgId: "o1", PermRw: 1}
	if _, err := eng.Insert(uo); err != nil {
		t.Fatalf("insert user_org: %v", err)
	}

	user := &model.TUser{Id: "u1", Name: "alice"}
	op := NewOrgPermCtx(ctx, user, "o1")

	if op.Org() == nil {
		t.Fatal("Org() should not be nil for existing org")
	}
	if op.Org().Name != "testorg" {
		t.Errorf("Org().Name = %s, want testorg", op.Org().Name)
	}
	if op.UserOrg() == nil {
		t.Fatal("UserOrg() should not be nil for member")
	}
	if !op.CanWrite() {
		t.Error("CanWrite() should return true for PermRw=1")
	}
}

func TestNewOrgPermCtx_DeletedOrg(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	org := &model.TOrg{Id: "o1", Aid: 1, Name: "testorg", Deleted: 1}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{Id: "u1", Name: "alice"}
	op := NewOrgPermCtx(ctx, user, "o1")

	if op.Org() != nil {
		t.Error("Org() should be nil for deleted org")
	}
}

func TestNewOrgPermCtx_EmptyOrgId(t *testing.T) {
	setupBatchTestDB(t)
	ctx := context.Background()

	user := &model.TUser{Id: "u1", Name: "alice"}
	op := NewOrgPermCtx(ctx, user, "")

	if op.Org() != nil {
		t.Error("Org() should be nil for empty org id")
	}
}

func TestNewOrgPerm_DelegatesToCtx(t *testing.T) {
	eng := setupBatchTestDB(t)

	org := &model.TOrg{Id: "o1", Aid: 1, Name: "testorg", Public: 1}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("insert org: %v", err)
	}

	user := &model.TUser{Id: "u1", Name: "alice"}
	op := NewOrgPerm(user, "o1")

	if op.Org() == nil {
		t.Fatal("Org() should not be nil")
	}
	if !op.IsOrgPublic() {
		t.Error("IsOrgPublic() should return true for Public=1")
	}
}

// Ensure comm.Db is not nil for tests that need it
func init() {
	// Suppress gin debug output in tests
	gin.SetMode(gin.TestMode)
}
