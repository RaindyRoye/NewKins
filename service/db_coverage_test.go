package service

import (
	"context"
	"testing"

	"github.com/gokins/gokins/model"
)

// TestGetIdOrAid_ByID tests GetIdOrAid lookup by primary ID.
func TestGetIdOrAid_ByID(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert test org
	org := &model.TOrg{
		Id:   "org-lookup-test",
		Aid:  100,
		Name: "lookup-org",
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("failed to insert org: %v", err)
	}

	// Test lookup by ID
	result := &model.TOrg{}
	found := GetIdOrAidCtx(ctx, "org-lookup-test", result)
	if !found {
		t.Error("GetIdOrAidCtx should find org by ID")
	}
	if result.Name != "lookup-org" {
		t.Errorf("GetIdOrAidCtx found wrong org: name = %q, want %q", result.Name, "lookup-org")
	}
}

// TestGetIdOrAid_ByAid tests GetIdOrAid fallback lookup by aid.
func TestGetIdOrAid_ByAid(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert test org with specific aid
	org := &model.TOrg{
		Id:   "org-aid-test",
		Aid:  200,
		Name: "aid-org",
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("failed to insert org: %v", err)
	}

	// Test lookup by aid (numeric)
	result := &model.TOrg{}
	found := GetIdOrAidCtx(ctx, 200, result)
	if !found {
		t.Error("GetIdOrAidCtx should find org by aid")
	}
	if result.Name != "aid-org" {
		t.Errorf("GetIdOrAidCtx found wrong org: name = %q, want %q", result.Name, "aid-org")
	}
}

// TestGetIdOrAid_NotFound tests GetIdOrAid when record doesn't exist.
func TestGetIdOrAid_NotFound(t *testing.T) {
	setupBatchTestDB(t)
	ctx := context.Background()

	result := &model.TOrg{}
	found := GetIdOrAidCtx(ctx, "nonexistent-id", result)
	if found {
		t.Error("GetIdOrAidCtx should return false for nonexistent ID")
	}
}

// TestGetIdOrAid_NilInputs tests GetIdOrAid with nil inputs.
func TestGetIdOrAid_NilInputs(t *testing.T) {
	ctx := context.Background()

	// nil id
	found := GetIdOrAidCtx(ctx, nil, &model.TOrg{})
	if found {
		t.Error("GetIdOrAidCtx(nil id) should return false")
	}

	// nil entity
	found = GetIdOrAidCtx(ctx, "some-id", nil)
	if found {
		t.Error("GetIdOrAidCtx(nil entity) should return false")
	}

	// empty string id
	found = GetIdOrAidCtx(ctx, "", &model.TOrg{})
	if found {
		t.Error("GetIdOrAidCtx(empty string) should return false")
	}
}

// TestGetIdOrAid_Wrapper tests the non-context wrapper delegates correctly.
func TestGetIdOrAid_Wrapper(t *testing.T) {
	setupBatchTestDB(t)

	// Should return false for nonexistent ID
	result := &model.TOrg{}
	found := GetIdOrAid("nonexistent", result)
	if found {
		t.Error("GetIdOrAid should return false for nonexistent ID")
	}
}

// TestGetIdOrAidE_Success tests GetIdOrAidE with successful lookup.
func TestGetIdOrAidE_Success(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert test org
	org := &model.TOrg{
		Id:   "org-e-test",
		Aid:  300,
		Name: "e-org",
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("failed to insert org: %v", err)
	}

	// Test lookup by ID
	result := &model.TOrg{}
	found, err := GetIdOrAidECtx(ctx, "org-e-test", result)
	if err != nil {
		t.Fatalf("GetIdOrAidECtx error: %v", err)
	}
	if !found {
		t.Error("GetIdOrAidECtx should find org by ID")
	}
	if result.Name != "e-org" {
		t.Errorf("GetIdOrAidECtx found wrong org: name = %q, want %q", result.Name, "e-org")
	}
}

// TestGetIdOrAidE_ByAid tests GetIdOrAidE fallback lookup by aid.
func TestGetIdOrAidE_ByAid(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert test org with specific aid
	org := &model.TOrg{
		Id:   "org-eaid-test",
		Aid:  400,
		Name: "eaid-org",
	}
	if _, err := eng.Insert(org); err != nil {
		t.Fatalf("failed to insert org: %v", err)
	}

	// Test lookup by aid
	result := &model.TOrg{}
	found, err := GetIdOrAidECtx(ctx, 400, result)
	if err != nil {
		t.Fatalf("GetIdOrAidECtx error: %v", err)
	}
	if !found {
		t.Error("GetIdOrAidECtx should find org by aid")
	}
	if result.Name != "eaid-org" {
		t.Errorf("GetIdOrAidECtx found wrong org: name = %q, want %q", result.Name, "eaid-org")
	}
}

// TestGetIdOrAidE_NotFound tests GetIdOrAidE when record doesn't exist.
func TestGetIdOrAidE_NotFound(t *testing.T) {
	setupBatchTestDB(t)
	ctx := context.Background()

	result := &model.TOrg{}
	found, err := GetIdOrAidECtx(ctx, "nonexistent-id", result)
	if err != nil {
		t.Fatalf("GetIdOrAidECtx should not return error for not-found: %v", err)
	}
	if found {
		t.Error("GetIdOrAidECtx should return false for nonexistent ID")
	}
}

// TestGetIdOrAidE_NilInputs tests GetIdOrAidE with nil inputs.
func TestGetIdOrAidE_NilInputs(t *testing.T) {
	ctx := context.Background()

	// nil id
	found, err := GetIdOrAidECtx(ctx, nil, &model.TOrg{})
	if err != nil {
		t.Errorf("GetIdOrAidECtx(nil id) should not error: %v", err)
	}
	if found {
		t.Error("GetIdOrAidECtx(nil id) should return false")
	}

	// nil entity
	found, err = GetIdOrAidECtx(ctx, "some-id", nil)
	if err != nil {
		t.Errorf("GetIdOrAidECtx(nil entity) should not error: %v", err)
	}
	if found {
		t.Error("GetIdOrAidECtx(nil entity) should return false")
	}

	// empty string id
	found, err = GetIdOrAidECtx(ctx, "", &model.TOrg{})
	if err != nil {
		t.Errorf("GetIdOrAidECtx(empty string) should not error: %v", err)
	}
	if found {
		t.Error("GetIdOrAidECtx(empty string) should return false")
	}
}

// TestGetIdOrAidE_Wrapper tests the non-context wrapper delegates correctly.
func TestGetIdOrAidE_Wrapper(t *testing.T) {
	setupBatchTestDB(t)

	// Should return false for nonexistent ID without error
	result := &model.TOrg{}
	found, err := GetIdOrAidE("nonexistent", result)
	if err != nil {
		t.Errorf("GetIdOrAidE should not error for nonexistent: %v", err)
	}
	if found {
		t.Error("GetIdOrAidE should return false for nonexistent ID")
	}
}

// TestBatchOrgPipeCounts_WithData tests BatchOrgPipeCounts with actual data.
func TestBatchOrgPipeCounts_WithData(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert test orgs
	orgs := []model.TOrg{
		{Id: "org1", Aid: 1, Name: "org1"},
		{Id: "org2", Aid: 2, Name: "org2"},
	}
	for _, o := range orgs {
		if _, err := eng.Insert(&o); err != nil {
			t.Fatalf("failed to insert org: %v", err)
		}
	}

	// Insert org-pipe associations
	type orgPipe struct {
		OrgId    string `xorm:"org_id"`
		PipelineId string `xorm:"pipe_id"`
	}
	// Manually create the t_org_pipe table
	_, err := eng.Exec(`CREATE TABLE IF NOT EXISTS t_org_pipe (
		org_id TEXT,
		pipe_id TEXT
	)`)
	if err != nil {
		t.Fatalf("failed to create t_org_pipe table: %v", err)
	}

	// Insert test data
	_, err = eng.Exec(`INSERT INTO t_org_pipe (org_id, pipe_id) VALUES (?, ?)`, "org1", "p1")
	if err != nil {
		t.Fatalf("failed to insert org_pipe: %v", err)
	}
	_, err = eng.Exec(`INSERT INTO t_org_pipe (org_id, pipe_id) VALUES (?, ?)`, "org1", "p2")
	if err != nil {
		t.Fatalf("failed to insert org_pipe: %v", err)
	}
	_, err = eng.Exec(`INSERT INTO t_org_pipe (org_id, pipe_id) VALUES (?, ?)`, "org2", "p3")
	if err != nil {
		t.Fatalf("failed to insert org_pipe: %v", err)
	}

	// Test batch counts
	result, err := BatchOrgPipeCounts(ctx, []string{"org1", "org2", "org3"})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts error: %v", err)
	}

	if result["org1"] != 2 {
		t.Errorf("org1 count = %d, want 2", result["org1"])
	}
	if result["org2"] != 1 {
		t.Errorf("org2 count = %d, want 1", result["org2"])
	}
	if _, exists := result["org3"]; exists {
		t.Error("org3 should not exist in result (no pipes)")
	}
}

// TestBatchOrgUserCounts_WithData tests BatchOrgUserCounts with actual data.
func TestBatchOrgUserCounts_WithData(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert test orgs
	orgs := []model.TOrg{
		{Id: "org1", Aid: 1, Name: "org1"},
		{Id: "org2", Aid: 2, Name: "org2"},
	}
	for _, o := range orgs {
		if _, err := eng.Insert(&o); err != nil {
			t.Fatalf("failed to insert org: %v", err)
		}
	}

	// Insert user-org associations
	userOrgs := []model.TUserOrg{
		{Uid: "u1", OrgId: "org1"},
		{Uid: "u2", OrgId: "org1"},
		{Uid: "u3", OrgId: "org1"},
		{Uid: "u4", OrgId: "org2"},
	}
	for _, uo := range userOrgs {
		if _, err := eng.Insert(&uo); err != nil {
			t.Fatalf("failed to insert user_org: %v", err)
		}
	}

	// Test batch counts - SQLite requires explicit IN clause expansion
	// The actual implementation uses xorm's In() which should handle this
	result, err := BatchOrgUserCounts(ctx, []string{"org1", "org2"})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts error: %v", err)
	}

	if result["org1"] != 3 {
		t.Errorf("org1 count = %d, want 3", result["org1"])
	}
	if result["org2"] != 1 {
		t.Errorf("org2 count = %d, want 1", result["org2"])
	}
}

// TestBatchOrgPipeCounts_EmptyInput tests BatchOrgPipeCounts with empty input.
func TestBatchOrgPipeCounts_EmptyInput(t *testing.T) {
	ctx := context.Background()

	result, err := BatchOrgPipeCounts(ctx, nil)
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts(nil) error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchOrgPipeCounts(nil) should return empty map, got %v", result)
	}

	result, err = BatchOrgPipeCounts(ctx, []string{})
	if err != nil {
		t.Fatalf("BatchOrgPipeCounts([]) error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchOrgPipeCounts([]) should return empty map, got %v", result)
	}
}

// TestBatchOrgUserCounts_EmptyInput tests BatchOrgUserCounts with empty input.
func TestBatchOrgUserCounts_EmptyInput(t *testing.T) {
	ctx := context.Background()

	result, err := BatchOrgUserCounts(ctx, nil)
	if err != nil {
		t.Fatalf("BatchOrgUserCounts(nil) error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchOrgUserCounts(nil) should return empty map, got %v", result)
	}

	result, err = BatchOrgUserCounts(ctx, []string{})
	if err != nil {
		t.Fatalf("BatchOrgUserCounts([]) error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("BatchOrgUserCounts([]) should return empty map, got %v", result)
	}
}
