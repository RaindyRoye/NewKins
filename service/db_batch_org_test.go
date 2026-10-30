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

// setupOrgBatchTestDB creates an in-memory SQLite database with org-related tables
// for testing BatchOrgPipeCounts and BatchOrgUserCounts.
func setupOrgBatchTestDB(t *testing.T) *xorm.Engine {
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
		&model.TOrg{},
		&model.TOrgPipe{},
		&model.TUser{},
		&model.TUserOrg{},
	); err != nil {
		t.Fatalf("failed to sync schema: %v", err)
	}
	return eng
}

// --- BatchOrgPipeCounts integration tests ---

func TestBatchOrgPipeCounts_WithData(t *testing.T) {
	eng := setupOrgBatchTestDB(t)
	ctx := context.Background()

	// Insert orgs
	orgs := []model.TOrg{
		{Id: "org-1", Aid: 1, Name: "org1", Created: time.Now()},
		{Id: "org-2", Aid: 2, Name: "org2", Created: time.Now()},
		{Id: "org-3", Aid: 3, Name: "org3", Created: time.Now()},
	}
	for _, o := range orgs {
		if _, err := eng.Insert(&o); err != nil {
			t.Fatalf("insert org: %v", err)
		}
	}

	// Insert org-pipe associations
	pipes := []model.TOrgPipe{
		{Aid: 1, OrgId: "org-1", PipeId: "pipe-a", Created: time.Now()},
		{Aid: 2, OrgId: "org-1", PipeId: "pipe-b", Created: time.Now()},
		{Aid: 3, OrgId: "org-1", PipeId: "pipe-c", Created: time.Now()},
		{Aid: 4, OrgId: "org-2", PipeId: "pipe-d", Created: time.Now()},
		{Aid: 5, OrgId: "org-2", PipeId: "pipe-e", Created: time.Now()},
		// org-3 has no pipelines
	}
	for _, p := range pipes {
		if _, err := eng.Insert(&p); err != nil {
			t.Fatalf("insert org pipe: %v", err)
		}
	}

	result, err := BatchOrgPipeCounts(ctx, []string{"org-1", "org-2", "org-3"})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts error: %v", err)
	}

	if result["org-1"] != 3 {
		t.Errorf("org-1 count = %d, want 3", result["org-1"])
	}
	if result["org-2"] != 2 {
		t.Errorf("org-2 count = %d, want 2", result["org-2"])
	}
	// org-3 has no pipelines, so it should not be in the map
	if _, exists := result["org-3"]; exists {
		t.Errorf("org-3 should not be in result map, got %d", result["org-3"])
	}
}

func TestBatchOrgPipeCounts_SingleOrg(t *testing.T) {
	eng := setupOrgBatchTestDB(t)
	ctx := context.Background()

	if _, err := eng.Insert(&model.TOrg{Id: "org-solo", Aid: 1, Name: "solo", Created: time.Now()}); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgPipe{Aid: 1, OrgId: "org-solo", PipeId: "p1", Created: time.Now()}); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	result, err := BatchOrgPipeCounts(ctx, []string{"org-solo"})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts error: %v", err)
	}
	if result["org-solo"] != 1 {
		t.Errorf("org-solo count = %d, want 1", result["org-solo"])
	}
}

func TestBatchOrgPipeCounts_NonexistentOrg(t *testing.T) {
	setupOrgBatchTestDB(t)
	ctx := context.Background()

	result, err := BatchOrgPipeCounts(ctx, []string{"nonexistent-org"})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result for nonexistent org, got %v", result)
	}
}

func TestBatchOrgPipeCounts_MixedExistingAndNonexisting(t *testing.T) {
	eng := setupOrgBatchTestDB(t)
	ctx := context.Background()

	if _, err := eng.Insert(&model.TOrg{Id: "org-mix", Aid: 1, Name: "mix", Created: time.Now()}); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := eng.Insert(&model.TOrgPipe{Aid: 1, OrgId: "org-mix", PipeId: "p1", Created: time.Now()}); err != nil {
		t.Fatalf("insert org pipe: %v", err)
	}

	result, err := BatchOrgPipeCounts(ctx, []string{"org-mix", "ghost-org"})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts error: %v", err)
	}
	if result["org-mix"] != 1 {
		t.Errorf("org-mix count = %d, want 1", result["org-mix"])
	}
	if _, exists := result["ghost-org"]; exists {
		t.Error("ghost-org should not be in result")
	}
}

// --- BatchOrgUserCounts integration tests ---

func TestBatchOrgUserCounts_WithData(t *testing.T) {
	eng := setupOrgBatchTestDB(t)
	ctx := context.Background()

	// Insert orgs
	orgs := []model.TOrg{
		{Id: "org-a", Aid: 1, Name: "orgA", Created: time.Now()},
		{Id: "org-b", Aid: 2, Name: "orgB", Created: time.Now()},
		{Id: "org-c", Aid: 3, Name: "orgC", Created: time.Now()},
	}
	for _, o := range orgs {
		if _, err := eng.Insert(&o); err != nil {
			t.Fatalf("insert org: %v", err)
		}
	}

	// Insert users
	users := []model.TUser{
		{Id: "user-1", Aid: 1, Name: "u1", Created: time.Now()},
		{Id: "user-2", Aid: 2, Name: "u2", Created: time.Now()},
		{Id: "user-3", Aid: 3, Name: "u3", Created: time.Now()},
		{Id: "user-4", Aid: 4, Name: "u4", Created: time.Now()},
	}
	for _, u := range users {
		if _, err := eng.Insert(&u); err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}

	// Insert user-org associations
	userOrgs := []model.TUserOrg{
		{Aid: 1, Uid: "user-1", OrgId: "org-a", Created: time.Now()},
		{Aid: 2, Uid: "user-2", OrgId: "org-a", Created: time.Now()},
		{Aid: 3, Uid: "user-3", OrgId: "org-a", Created: time.Now()},
		{Aid: 4, Uid: "user-1", OrgId: "org-b", Created: time.Now()},
		// org-c has no users
	}
	for _, uo := range userOrgs {
		if _, err := eng.Insert(&uo); err != nil {
			t.Fatalf("insert user org: %v", err)
		}
	}

	result, err := BatchOrgUserCounts(ctx, []string{"org-a", "org-b", "org-c"})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts error: %v", err)
	}

	if result["org-a"] != 3 {
		t.Errorf("org-a count = %d, want 3", result["org-a"])
	}
	if result["org-b"] != 1 {
		t.Errorf("org-b count = %d, want 1", result["org-b"])
	}
	if _, exists := result["org-c"]; exists {
		t.Errorf("org-c should not be in result map, got %d", result["org-c"])
	}
}

func TestBatchOrgUserCounts_SingleOrg(t *testing.T) {
	eng := setupOrgBatchTestDB(t)
	ctx := context.Background()

	if _, err := eng.Insert(&model.TOrg{Id: "org-solo", Aid: 1, Name: "solo", Created: time.Now()}); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := eng.Insert(&model.TUser{Id: "user-solo", Aid: 1, Name: "solo", Created: time.Now()}); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := eng.Insert(&model.TUserOrg{Aid: 1, Uid: "user-solo", OrgId: "org-solo", Created: time.Now()}); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	result, err := BatchOrgUserCounts(ctx, []string{"org-solo"})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts error: %v", err)
	}
	if result["org-solo"] != 1 {
		t.Errorf("org-solo count = %d, want 1", result["org-solo"])
	}
}

func TestBatchOrgUserCounts_NonexistentOrg(t *testing.T) {
	setupOrgBatchTestDB(t)
	ctx := context.Background()

	result, err := BatchOrgUserCounts(ctx, []string{"nonexistent-org"})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result for nonexistent org, got %v", result)
	}
}

func TestBatchOrgUserCounts_MixedExistingAndNonexisting(t *testing.T) {
	eng := setupOrgBatchTestDB(t)
	ctx := context.Background()

	if _, err := eng.Insert(&model.TOrg{Id: "org-mix", Aid: 1, Name: "mix", Created: time.Now()}); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	if _, err := eng.Insert(&model.TUser{Id: "user-mix", Aid: 1, Name: "mix", Created: time.Now()}); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := eng.Insert(&model.TUserOrg{Aid: 1, Uid: "user-mix", OrgId: "org-mix", Created: time.Now()}); err != nil {
		t.Fatalf("insert user org: %v", err)
	}

	result, err := BatchOrgUserCounts(ctx, []string{"org-mix", "ghost-org"})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts error: %v", err)
	}
	if result["org-mix"] != 1 {
		t.Errorf("org-mix count = %d, want 1", result["org-mix"])
	}
	if _, exists := result["ghost-org"]; exists {
		t.Error("ghost-org should not be in result")
	}
}
