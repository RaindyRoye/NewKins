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

// setupBatchTestDB creates an in-memory SQLite database with test tables for batch operations.
func setupBatchTestDB(t *testing.T) *xorm.Engine {
	t.Helper()
	eng, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Set the global Db so service functions can use it
	oldDb := comm.Db
	comm.Db = eng
	t.Cleanup(func() {
		comm.Db = oldDb
		_ = eng.Close()
	})

	// Create tables using xorm's schema sync (auto-generates from model structs)
	err = eng.Sync2(
		&model.TBuild{},
		&model.TUser{},
		&model.TUserInfo{},
		&model.TArtifactPackage{},
		&model.TArtifactVersion{},
		&model.TOrg{},
		&model.TUserOrg{},
		&model.TPipeline{},
		&model.TPipelineConf{},
		&model.TPipelineVersion{},
		&model.TStage{},
		&model.TStep{},
		&model.TTrigger{},
		&model.TTriggerRun{},
		&model.TOrgVar{},
		&model.TPipelineVar{},
		&model.TParam{},
	)
	if err != nil {
		t.Fatalf("failed to sync database schema: %v", err)
	}

	return eng
}

func TestBatchBuildCounts_WithData(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	// Insert test builds
	builds := []model.TBuild{
		{Id: "b1", PipelineId: "p1", Created: time.Now()},
		{Id: "b2", PipelineId: "p1", Created: time.Now()},
		{Id: "b3", PipelineId: "p1", Created: time.Now()},
		{Id: "b4", PipelineId: "p2", Created: time.Now()},
		{Id: "b5", PipelineId: "p2", Created: time.Now()},
	}
	for _, b := range builds {
		if _, err := eng.Insert(&b); err != nil {
			t.Fatalf("failed to insert build: %v", err)
		}
	}

	result, err := BatchBuildCounts(ctx, []string{"p1", "p2", "p3"})
	if err != nil {
		t.Fatalf("BatchBuildCounts failed: %v", err)
	}

	if result["p1"] != 3 {
		t.Errorf("p1 count = %d, want 3", result["p1"])
	}
	if result["p2"] != 2 {
		t.Errorf("p2 count = %d, want 2", result["p2"])
	}
	if _, exists := result["p3"]; exists {
		t.Error("p3 should not exist in result (no builds)")
	}
}

func TestBatchLatestBuilds_WithData(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	now := time.Now()
	builds := []model.TBuild{
		{Id: "b1", PipelineId: "p1", Created: now.Add(-2 * time.Hour)},
		{Id: "b2", PipelineId: "p1", Created: now.Add(-1 * time.Hour)}, // Latest for p1
		{Id: "b3", PipelineId: "p2", Created: now.Add(-3 * time.Hour)},
		{Id: "b4", PipelineId: "p2", Created: now}, // Latest for p2
	}
	for _, b := range builds {
		if _, err := eng.Insert(&b); err != nil {
			t.Fatalf("failed to insert build: %v", err)
		}
	}

	result, err := BatchLatestBuilds(ctx, []string{"p1", "p2"})
	if err != nil {
		t.Fatalf("BatchLatestBuilds failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result["p1"].Id != "b2" {
		t.Errorf("p1 latest build = %s, want b2", result["p1"].Id)
	}
	if result["p2"].Id != "b4" {
		t.Errorf("p2 latest build = %s, want b4", result["p2"].Id)
	}
}

func TestBatchLatestBuildsForVersions_WithData(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	now := time.Now()
	builds := []model.TBuild{
		{Id: "b1", PipelineVersionId: "v1", Created: now.Add(-2 * time.Hour)},
		{Id: "b2", PipelineVersionId: "v1", Created: now},                     // Latest for v1
		{Id: "b3", PipelineVersionId: "v2", Created: now.Add(-1 * time.Hour)}, // Latest for v2
	}
	for _, b := range builds {
		if _, err := eng.Insert(&b); err != nil {
			t.Fatalf("failed to insert build: %v", err)
		}
	}

	result, err := BatchLatestBuildsForVersions(ctx, []string{"v1", "v2"})
	if err != nil {
		t.Fatalf("BatchLatestBuildsForVersions failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result["v1"].Id != "b2" {
		t.Errorf("v1 latest build = %s, want b2", result["v1"].Id)
	}
	if result["v2"].Id != "b3" {
		t.Errorf("v2 latest build = %s, want b3", result["v2"].Id)
	}
}

func TestBatchGetUsers_WithData(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	users := []model.TUser{
		{Id: "u1", Aid: 1, Name: "Alice", Nick: "alice@example.com"},
		{Id: "u2", Aid: 2, Name: "Bob", Nick: "bob@example.com"},
		{Id: "u3", Aid: 3, Name: "Charlie", Nick: "charlie@example.com"},
	}
	for _, u := range users {
		if _, err := eng.Insert(&u); err != nil {
			t.Fatalf("failed to insert user: %v", err)
		}
	}

	result, err := BatchGetUsers(ctx, []string{"u1", "u3", "u999"})
	if err != nil {
		t.Fatalf("BatchGetUsers failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	if result["u1"].Name != "Alice" {
		t.Errorf("u1 name = %s, want Alice", result["u1"].Name)
	}
	if result["u3"].Name != "Charlie" {
		t.Errorf("u3 name = %s, want Charlie", result["u3"].Name)
	}
	if _, exists := result["u999"]; exists {
		t.Error("u999 should not exist in result")
	}
}

func TestBatchCountArtifactPackages_WithData(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	packages := []model.TArtifactPackage{
		{Id: "pkg1", Aid: 1, RepoId: "r1"},
		{Id: "pkg2", Aid: 2, RepoId: "r1"},
		{Id: "pkg3", Aid: 3, RepoId: "r1"},
		{Id: "pkg4", Aid: 4, RepoId: "r2"},
		{Id: "pkg5", Aid: 5, RepoId: "r2"},
	}
	for _, p := range packages {
		if _, err := eng.Insert(&p); err != nil {
			t.Fatalf("failed to insert package: %v", err)
		}
	}

	result, err := BatchCountArtifactPackages(ctx, []string{"r1", "r2", "r3"})
	if err != nil {
		t.Fatalf("BatchCountArtifactPackages failed: %v", err)
	}

	if result["r1"] != 3 {
		t.Errorf("r1 count = %d, want 3", result["r1"])
	}
	if result["r2"] != 2 {
		t.Errorf("r2 count = %d, want 2", result["r2"])
	}
	if _, exists := result["r3"]; exists {
		t.Error("r3 should not exist in result")
	}
}

func TestBatchCountArtifactVersions_WithData(t *testing.T) {
	eng := setupBatchTestDB(t)
	ctx := context.Background()

	versions := []model.TArtifactVersion{
		{Id: "ver1", Aid: 1, PackageId: "pkg1"},
		{Id: "ver2", Aid: 2, PackageId: "pkg1"},
		{Id: "ver3", Aid: 3, PackageId: "pkg2"},
		{Id: "ver4", Aid: 4, PackageId: "pkg2"},
		{Id: "ver5", Aid: 5, PackageId: "pkg2"},
	}
	for _, v := range versions {
		if _, err := eng.Insert(&v); err != nil {
			t.Fatalf("failed to insert version: %v", err)
		}
	}

	result, err := BatchCountArtifactVersions(ctx, []string{"pkg1", "pkg2", "pkg3"})
	if err != nil {
		t.Fatalf("BatchCountArtifactVersions failed: %v", err)
	}

	if result["pkg1"] != 2 {
		t.Errorf("pkg1 count = %d, want 2", result["pkg1"])
	}
	if result["pkg2"] != 3 {
		t.Errorf("pkg2 count = %d, want 3", result["pkg2"])
	}
	if _, exists := result["pkg3"]; exists {
		t.Error("pkg3 should not exist in result")
	}
}
