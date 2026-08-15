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

// setupDBTest creates an isolated in-memory SQLite DB for db.go tests.
func setupDBTest(t *testing.T) *xorm.Engine {
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
	if err := eng.Sync2(&model.TUser{}, &model.TUserInfo{}); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// --- GetIdOrAidCtx ---

func TestGetIdOrAidCtx_IntegFindById(t *testing.T) {
	eng := setupDBTest(t)
	ctx := context.Background()

	// Insert a test user
	user := &model.TUser{
		Id:      "user-1",
		Aid:     1,
		Name:    "alice",
		Nick:    "Alice",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var found model.TUser
	ok := GetIdOrAidCtx(ctx, "user-1", &found)
	if !ok {
		t.Fatal("GetIdOrAidCtx should find user by id")
	}
	if found.Name != "alice" {
		t.Errorf("Name = %q, want %q", found.Name, "alice")
	}
	if found.Nick != "Alice" {
		t.Errorf("Nick = %q, want %q", found.Nick, "Alice")
	}
}

func TestGetIdOrAidCtx_IntegFindByAid(t *testing.T) {
	eng := setupDBTest(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "user-aid-test",
		Aid:     99,
		Name:    "bob",
		Nick:    "Bob",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var found model.TUser
	ok := GetIdOrAidCtx(ctx, int64(99), &found)
	if !ok {
		t.Fatal("GetIdOrAidCtx should find user by aid (fallback)")
	}
	if found.Id != "user-aid-test" {
		t.Errorf("Id = %q, want %q", found.Id, "user-aid-test")
	}
}

func TestGetIdOrAidCtx_IntegNotFound(t *testing.T) {
	setupDBTest(t)
	ctx := context.Background()

	var found model.TUser
	ok := GetIdOrAidCtx(ctx, "nonexistent", &found)
	if ok {
		t.Error("GetIdOrAidCtx should return false for nonexistent id")
	}
}

func TestGetIdOrAidCtx_IntegNilInputs(t *testing.T) {
	setupDBTest(t)
	ctx := context.Background()

	// nil id
	ok := GetIdOrAidCtx(ctx, nil, &model.TUser{})
	if ok {
		t.Error("nil id should return false")
	}

	// nil entity
	ok = GetIdOrAidCtx(ctx, "some-id", nil)
	if ok {
		t.Error("nil entity should return false")
	}

	// empty string id
	ok = GetIdOrAidCtx(ctx, "", &model.TUser{})
	if ok {
		t.Error("empty string id should return false")
	}

	// both nil
	ok = GetIdOrAidCtx(ctx, nil, nil)
	if ok {
		t.Error("both nil should return false")
	}
}

// --- GetIdOrAidECtx ---

func TestGetIdOrAidECtx_IntegFindById(t *testing.T) {
	eng := setupDBTest(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "user-e-1",
		Aid:     10,
		Name:    "charlie",
		Nick:    "Charlie",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var found model.TUser
	ok, err := GetIdOrAidECtx(ctx, "user-e-1", &found)
	if err != nil {
		t.Fatalf("GetIdOrAidECtx error: %v", err)
	}
	if !ok {
		t.Fatal("should find user by id")
	}
	if found.Name != "charlie" {
		t.Errorf("Name = %q, want %q", found.Name, "charlie")
	}
}

func TestGetIdOrAidECtx_IntegFindByAid(t *testing.T) {
	eng := setupDBTest(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "user-e-aid",
		Aid:     55,
		Name:    "dave",
		Nick:    "Dave",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var found model.TUser
	ok, err := GetIdOrAidECtx(ctx, int64(55), &found)
	if err != nil {
		t.Fatalf("GetIdOrAidECtx error: %v", err)
	}
	if !ok {
		t.Fatal("should find user by aid")
	}
	if found.Id != "user-e-aid" {
		t.Errorf("Id = %q, want %q", found.Id, "user-e-aid")
	}
}

func TestGetIdOrAidECtx_IntegNotFound(t *testing.T) {
	setupDBTest(t)
	ctx := context.Background()

	var found model.TUser
	ok, err := GetIdOrAidECtx(ctx, "missing", &found)
	if err != nil {
		t.Errorf("should not return error for not-found: %v", err)
	}
	if ok {
		t.Error("should return false for nonexistent id")
	}
}

func TestGetIdOrAidECtx_IntegNilInputs(t *testing.T) {
	setupDBTest(t)
	ctx := context.Background()

	ok, err := GetIdOrAidECtx(ctx, nil, &model.TUser{})
	if ok {
		t.Error("nil id should return false")
	}
	if err != nil {
		t.Errorf("nil id should not error: %v", err)
	}

	ok, err = GetIdOrAidECtx(ctx, "some-id", nil)
	if ok {
		t.Error("nil entity should return false")
	}
	if err != nil {
		t.Errorf("nil entity should not error: %v", err)
	}

	ok, err = GetIdOrAidECtx(ctx, "", &model.TUser{})
	if ok {
		t.Error("empty string should return false")
	}
	if err != nil {
		t.Errorf("empty string should not error: %v", err)
	}
}

// --- Global context wrappers ---

func TestGetIdOrAid_IntegFindById(t *testing.T) {
	eng := setupDBTest(t)
	ctx := context.Background()

	user := &model.TUser{
		Id:      "user-global-1",
		Aid:     20,
		Name:    "eve",
		Nick:    "Eve",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var found model.TUser
	ok := GetIdOrAid("user-global-1", &found)
	if !ok {
		t.Fatal("GetIdOrAid should find user by id")
	}
	if found.Name != "eve" {
		t.Errorf("Name = %q, want %q", found.Name, "eve")
	}

	// Also test aid fallback
	var foundByAid model.TUser
	ok = GetIdOrAid(int64(20), &foundByAid)
	if !ok {
		t.Fatal("GetIdOrAid should find user by aid")
	}
	if foundByAid.Id != "user-global-1" {
		t.Errorf("Id = %q, want %q", foundByAid.Id, "user-global-1")
	}
	_ = ctx // used by eng operations implicitly via comm.Ctx
}

func TestGetIdOrAidE_IntegFindById(t *testing.T) {
	eng := setupDBTest(t)

	user := &model.TUser{
		Id:      "user-globale-1",
		Aid:     30,
		Name:    "frank",
		Nick:    "Frank",
		Created: time.Now(),
	}
	if _, err := eng.Insert(user); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var found model.TUser
	ok, err := GetIdOrAidE("user-globale-1", &found)
	if err != nil {
		t.Fatalf("GetIdOrAidE error: %v", err)
	}
	if !ok {
		t.Fatal("should find user by id")
	}
	if found.Name != "frank" {
		t.Errorf("Name = %q, want %q", found.Name, "frank")
	}

	// aid fallback
	var foundByAid model.TUser
	ok, err = GetIdOrAidE(int64(30), &foundByAid)
	if err != nil {
		t.Fatalf("GetIdOrAidE aid error: %v", err)
	}
	if !ok {
		t.Fatal("should find user by aid")
	}
	if foundByAid.Id != "user-globale-1" {
		t.Errorf("Id = %q, want %q", foundByAid.Id, "user-globale-1")
	}
}

// --- BatchOrgPipeCounts / BatchOrgUserCounts with real data ---

func TestBatchOrgPipeCounts_WithRealData(t *testing.T) {
	t.Skip("Skipping: BatchOrgPipeCounts uses raw SQL with IN (?) which requires MySQL/PostgreSQL array parameter support")
}

func TestBatchOrgUserCounts_WithRealData(t *testing.T) {
	t.Skip("Skipping: BatchOrgUserCounts uses raw SQL with IN (?) which requires MySQL/PostgreSQL array parameter support")
}
