package service

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"github.com/gokins/gokins/util"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// setupCoverageTestDB creates an in-memory SQLite database with all relevant tables
// for coverage-focused tests in this file.
func setupCoverageTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	oldDb := comm.Db
	comm.Db = eng
	oldCtx := comm.Ctx
	comm.Ctx = context.Background()
	t.Cleanup(func() {
		comm.Db = oldDb
		comm.Ctx = oldCtx
		_ = eng.Close()
	})
	err = eng.Sync2(
		&model.TParam{},
		&model.TUser{},
		&model.TUserInfo{},
		&model.TOrg{},
		&model.TOrgPipe{},
		&model.TUserOrg{},
		&model.TPipeline{},
		&model.TPipelineConf{},
		&model.TPipelineVersion{},
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TTrigger{},
		&model.TOrgVar{},
		&model.TPipelineVar{},
	)
	if err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// TestGetsParamCacheCtx_CacheMiss verifies GetsParamCacheCtx fetches from DB on cache miss.
func TestGetsParamCacheCtx_CacheMiss(t *testing.T) {
	setupCoverageTestDB(t)
	ctx := context.Background()

	// Insert a param into DB
	data := map[string]string{"host": "localhost"}
	if err := SetsParamCtx(ctx, "cache.miss.param", data); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	var result map[string]string
	if err := GetsParamCacheCtx(ctx, "cache.miss.param", &result, time.Minute); err != nil {
		t.Fatalf("GetsParamCacheCtx: %v", err)
	}
	if result["host"] != "localhost" {
		t.Errorf("result[host] = %q, want %q", result["host"], "localhost")
	}
}

// TestGetsParamCacheCtx_CacheHit verifies that on second call, GetsParamCacheCtx hits cache.
func TestGetsParamCacheCtx_CacheHit(t *testing.T) {
	setupCoverageTestDB(t)
	ctx := context.Background()

	data := map[string]int{"count": 42}
	if err := SetsParamCtx(ctx, "cache.hit.param", data); err != nil {
		t.Fatalf("SetsParamCtx: %v", err)
	}

	var r1 map[string]int
	if err := GetsParamCacheCtx(ctx, "cache.hit.param", &r1, time.Minute); err != nil {
		t.Fatalf("first GetsParamCacheCtx: %v", err)
	}

	// Second call — should hit in-memory cache
	var r2 map[string]int
	if err := GetsParamCacheCtx(ctx, "cache.hit.param", &r2, time.Minute); err != nil {
		t.Fatalf("second GetsParamCacheCtx: %v", err)
	}
	if r2["count"] != 42 {
		t.Errorf("r2[count] = %d, want 42", r2["count"])
	}
}

// TestGetsParamCache_NonCtxVersion tests the non-context wrapper GetsParamCache.
func TestGetsParamCache_NonCtxVersion(t *testing.T) {
	setupCoverageTestDB(t)

	data := map[string]bool{"active": true}
	if err := SetsParam("nocache.param", data); err != nil {
		t.Fatalf("SetsParam: %v", err)
	}

	var result map[string]bool
	if err := GetsParamCache("nocache.param", &result); err != nil {
		t.Fatalf("GetsParamCache: %v", err)
	}
	if !result["active"] {
		t.Errorf("result[active] = %v, want true", result["active"])
	}
}

// TestGetsParamCacheCtx_NotFound verifies proper error when param does not exist.
func TestGetsParamCacheCtx_NotFound(t *testing.T) {
	setupCoverageTestDB(t)
	ctx := context.Background()

	var result map[string]string
	err := GetsParamCacheCtx(ctx, "nonexistent.param", &result)
	if err == nil {
		t.Fatal("GetsParamCacheCtx should return error for nonexistent param")
	}
}

// TestCheckCurrPermission tests the gin-context based permission check.
func TestCheckCurrPermission(t *testing.T) {
	eng := setupCoverageTestDB(t)
	gin.SetMode(gin.TestMode)

	// Set a known login key
	comm.Cfg.Server.LoginKey = "test-key-coverage"

	// Insert users (set Aid explicitly as it's part of composite PK)
	admin := &model.TUser{Id: "admin", Aid: 1, Name: "admin", Active: 1, Created: time.Now(), LoginTime: time.Now()}
	regular := &model.TUser{Id: "regular", Aid: 2, Name: "regular", Active: 1, Created: time.Now(), LoginTime: time.Now()}
	for _, u := range []*model.TUser{admin, regular} {
		if _, err := eng.Insert(u); err != nil {
			t.Fatalf("insert user %s: %v", u.Id, err)
		}
	}

	// Helper to create a valid token
	makeToken := func(uid string) string {
		tok, err := util.CreateToken(jwt.MapClaims{"uid": uid}, comm.Cfg.Server.LoginKey, time.Hour)
		if err != nil {
			t.Fatalf("create token for %s: %v", uid, err)
		}
		return tok
	}

	tests := []struct {
		name   string
		uid    string // empty means no token
		perms  string
		expect bool
	}{
		{"admin with common perm", "admin", PermCommon, true},
		{"admin with admin perm", "admin", PermAdmin, true},
		{"regular with common perm", "regular", PermCommon, true},
		{"regular with admin perm", "regular", PermAdmin, false},
		{"no user (empty token)", "", PermCommon, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest("GET", "/test", nil)

			if tt.uid != "" {
				token := makeToken(tt.uid)
				req.Header.Set("Authorization", "TOKEN "+token)
			}
			c.Request = req

			got := CheckCurrPermission(c, tt.perms)
			if got != tt.expect {
				t.Errorf("CheckCurrPermission() = %v, want %v", got, tt.expect)
			}
		})
	}
}

// TestCheckCurrPermission_NilRequest tests with a nil gin request (edge case).
func TestCheckCurrPermission_NilRequest(t *testing.T) {
	setupCoverageTestDB(t)
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = nil

	got := CheckCurrPermission(c, PermCommon)
	if got {
		t.Error("CheckCurrPermission with nil request should return false")
	}
}

// TestBatchOrgPipeCounts tests batch counting of pipelines per organization.
func TestBatchOrgPipeCounts(t *testing.T) {
	eng := setupCoverageTestDB(t)
	ctx := context.Background()

	// Insert orgs (set Aid explicitly for composite PK)
	for _, org := range []*model.TOrg{
		{Id: "org1", Aid: 1, Name: "Org 1"},
		{Id: "org2", Aid: 2, Name: "Org 2"},
		{Id: "org3", Aid: 3, Name: "Org 3"},
	} {
		if _, err := eng.Insert(org); err != nil {
			t.Fatalf("insert org %s: %v", org.Id, err)
		}
	}

	// Insert org-pipe associations
	assoc := []model.TOrgPipe{
		{OrgId: "org1", PipeId: "p1"},
		{OrgId: "org1", PipeId: "p2"},
		{OrgId: "org1", PipeId: "p3"},
		{OrgId: "org2", PipeId: "p4"},
	}
	for i := range assoc {
		if _, err := eng.Insert(&assoc[i]); err != nil {
			t.Fatalf("insert org pipe: %v", err)
		}
	}

	counts, err := BatchOrgPipeCounts(ctx, []string{"org1", "org2", "org3"})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts: %v", err)
	}
	if counts["org1"] != 3 {
		t.Errorf("counts[org1] = %d, want 3", counts["org1"])
	}
	if counts["org2"] != 1 {
		t.Errorf("counts[org2] = %d, want 1", counts["org2"])
	}
	if _, ok := counts["org3"]; ok {
		t.Errorf("counts[org3] should not exist, got %d", counts["org3"])
	}
}

// TestBatchOrgPipeCounts_WithEmptySlice tests with empty input slice.
func TestBatchOrgPipeCounts_WithEmptySlice(t *testing.T) {
	setupCoverageTestDB(t)
	ctx := context.Background()

	counts, err := BatchOrgPipeCounts(ctx, []string{})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("expected empty map, got %v", counts)
	}
}

// TestBatchOrgUserCounts tests batch counting of users per organization.
func TestBatchOrgUserCounts(t *testing.T) {
	eng := setupCoverageTestDB(t)
	ctx := context.Background()

	// Insert orgs (set Aid explicitly for composite PK)
	for _, org := range []*model.TOrg{
		{Id: "org1", Aid: 1, Name: "Org 1"},
		{Id: "org2", Aid: 2, Name: "Org 2"},
		{Id: "org3", Aid: 3, Name: "Org 3"},
	} {
		if _, err := eng.Insert(org); err != nil {
			t.Fatalf("insert org %s: %v", org.Id, err)
		}
	}

	// Insert user-org associations
	assoc := []model.TUserOrg{
		{Uid: "u1", OrgId: "org1"},
		{Uid: "u2", OrgId: "org1"},
		{Uid: "u3", OrgId: "org2"},
		{Uid: "u4", OrgId: "org2"},
		{Uid: "u5", OrgId: "org2"},
	}
	for i := range assoc {
		if _, err := eng.Insert(&assoc[i]); err != nil {
			t.Fatalf("insert user org: %v", err)
		}
	}

	counts, err := BatchOrgUserCounts(ctx, []string{"org1", "org2", "org3"})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts: %v", err)
	}
	if counts["org1"] != 2 {
		t.Errorf("counts[org1] = %d, want 2", counts["org1"])
	}
	if counts["org2"] != 3 {
		t.Errorf("counts[org2] = %d, want 3", counts["org2"])
	}
	if _, ok := counts["org3"]; ok {
		t.Errorf("counts[org3] should not exist, got %d", counts["org3"])
	}
}

// TestBatchOrgUserCounts_WithEmptySlice tests with empty input slice.
func TestBatchOrgUserCounts_WithEmptySlice(t *testing.T) {
	setupCoverageTestDB(t)
	ctx := context.Background()

	counts, err := BatchOrgUserCounts(ctx, []string{})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("expected empty map, got %v", counts)
	}
}

// TestCheckUPermission_EdgeCases covers edge cases not in perms_test.go.
func TestCheckUPermission_EdgeCases(t *testing.T) {
	user := &model.TUser{Id: "user1", Name: "user1"}

	tests := []struct {
		name  string
		perms string
		want  bool
	}{
		{"empty permission string", "", false},
		{"unknown permission level", "superuser", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckUPermission(user, tt.perms)
			if got != tt.want {
				t.Errorf("CheckUPermission(%q) = %v, want %v", tt.perms, got, tt.want)
			}
		})
	}
}

// TestGetMidLgUser tests the GetMidLgUser function for retrieving the logged-in user from gin context.
func TestGetMidLgUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("no lguser key set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		got := GetMidLgUser(c)
		if got != nil {
			t.Errorf("GetMidLgUser() = %v, want nil", got)
		}
	})

	t.Run("lguser key set to correct type", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		usr := &model.TUser{Id: "u1", Name: "alice"}
		c.Set(LgUserKey, usr)
		got := GetMidLgUser(c)
		if got == nil {
			t.Fatal("GetMidLgUser() = nil, want user")
		}
		if got.Id != "u1" {
			t.Errorf("got.Id = %q, want %q", got.Id, "u1")
		}
	})

	t.Run("lguser key set to wrong type", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(LgUserKey, "not-a-user")
		got := GetMidLgUser(c)
		if got != nil {
			t.Errorf("GetMidLgUser() = %v, want nil (wrong type)", got)
		}
	})
}
