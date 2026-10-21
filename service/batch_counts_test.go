package service

import (
	"context"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupBatchCountTestDB(t *testing.T) *xorm.Engine {
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

	// Only sync the tables we actually need for batch counts
	err = eng.Sync2(
		&model.TOrgPipe{},
		&model.TUserOrg{},
	)
	if err != nil {
		t.Fatalf("failed to sync database schema: %v", err)
	}

	return eng
}

func TestBatchOrgPipeCounts_WithData(t *testing.T) {
	eng := setupBatchCountTestDB(t)
	ctx := context.Background()

	// Create org-pipe mappings directly (no need to insert TOrg rows)
	orgPipes := []model.TOrgPipe{
		{OrgId: "org1", PipeId: "pipe1"},
		{OrgId: "org1", PipeId: "pipe2"},
		{OrgId: "org1", PipeId: "pipe3"},
		{OrgId: "org2", PipeId: "pipe4"},
		{OrgId: "org2", PipeId: "pipe5"},
	}
	for _, op := range orgPipes {
		if _, err := eng.Insert(&op); err != nil {
			t.Fatalf("failed to insert org pipe: %v", err)
		}
	}

	// Test with all orgs
	counts, err := BatchOrgPipeCounts(ctx, []string{"org1", "org2", "org3"})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts failed: %v", err)
	}

	if counts["org1"] != 3 {
		t.Errorf("org1 count = %d, want 3", counts["org1"])
	}
	if counts["org2"] != 2 {
		t.Errorf("org2 count = %d, want 2", counts["org2"])
	}
	if _, exists := counts["org3"]; exists {
		t.Error("org3 should not be in the result (it has no pipes)")
	}

	// Test with subset of orgs
	counts, err = BatchOrgPipeCounts(ctx, []string{"org1", "org2"})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts failed: %v", err)
	}
	if len(counts) != 2 {
		t.Errorf("expected 2 entries, got %d", len(counts))
	}

	// Test with empty list
	counts, err = BatchOrgPipeCounts(ctx, []string{})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts failed: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("expected 0 entries for empty list, got %d", len(counts))
	}
}

func TestBatchOrgUserCounts_WithData(t *testing.T) {
	eng := setupBatchCountTestDB(t)
	ctx := context.Background()

	// Create user-org mappings directly
	userOrgs := []model.TUserOrg{
		{OrgId: "org1", Uid: "user1"},
		{OrgId: "org1", Uid: "user2"},
		{OrgId: "org1", Uid: "user3"},
		{OrgId: "org1", Uid: "user4"},
		{OrgId: "org2", Uid: "user5"},
		{OrgId: "org2", Uid: "user6"},
	}
	for _, uo := range userOrgs {
		if _, err := eng.Insert(&uo); err != nil {
			t.Fatalf("failed to insert user org: %v", err)
		}
	}

	// Test with all orgs
	counts, err := BatchOrgUserCounts(ctx, []string{"org1", "org2", "org3"})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts failed: %v", err)
	}

	if counts["org1"] != 4 {
		t.Errorf("org1 count = %d, want 4", counts["org1"])
	}
	if counts["org2"] != 2 {
		t.Errorf("org2 count = %d, want 2", counts["org2"])
	}
	if _, exists := counts["org3"]; exists {
		t.Error("org3 should not be in the result (it has no users)")
	}

	// Test with subset of orgs
	counts, err = BatchOrgUserCounts(ctx, []string{"org1", "org2"})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts failed: %v", err)
	}
	if len(counts) != 2 {
		t.Errorf("expected 2 entries, got %d", len(counts))
	}

	// Test with empty list
	counts, err = BatchOrgUserCounts(ctx, []string{})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts failed: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("expected 0 entries for empty list, got %d", len(counts))
	}
}

func TestBatchOrgPipeCounts_NonExistentOrgs(t *testing.T) {
	setupBatchCountTestDB(t)
	ctx := context.Background()

	// Test with non-existent orgs - should return empty map (not 0 values)
	counts, err := BatchOrgPipeCounts(ctx, []string{"nonexistent1", "nonexistent2"})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts failed: %v", err)
	}

	// Non-existent orgs should not appear in the result
	if len(counts) != 0 {
		t.Errorf("expected empty map for non-existent orgs, got %d entries", len(counts))
	}
}

func TestBatchOrgUserCounts_NonExistentOrgs(t *testing.T) {
	setupBatchCountTestDB(t)
	ctx := context.Background()

	// Test with non-existent orgs - should return empty map (not 0 values)
	counts, err := BatchOrgUserCounts(ctx, []string{"nonexistent1", "nonexistent2"})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts failed: %v", err)
	}

	// Non-existent orgs should not appear in the result
	if len(counts) != 0 {
		t.Errorf("expected empty map for non-existent orgs, got %d entries", len(counts))
	}
}
