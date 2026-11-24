package service

import (
	"context"
	"testing"

	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/model"
	"xorm.io/xorm"
)

func setupPipelineTestDB(t *testing.T) *xorm.Engine {
	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	err = db.Sync2(
		&model.TPipeline{},
		&model.TPipelineConf{},
		&model.TPipelineVersion{},
		&model.TBuild{},
		&model.TStage{},
		&model.TStep{},
		&model.TOrgVar{},
		&model.TPipelineVar{},
	)
	if err != nil {
		t.Fatalf("failed to sync database schema: %v", err)
	}

	return db
}

func TestRun_PipelineNotFound(t *testing.T) {
	db := setupPipelineTestDB(t)
	comm.Db = db
	defer func() { comm.Db = nil }()

	ctx := context.Background()
	_, _, err := Run(ctx, "user1", "nonexistent", "sha123", "push")

	if err == nil {
		t.Fatal("Run should return error when pipeline not found")
	}
	if err != ErrPipelineNotFound {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestRun_EmptyYmlContent(t *testing.T) {
	db := setupPipelineTestDB(t)
	comm.Db = db
	defer func() { comm.Db = nil }()

	// Insert pipeline config with empty YAML
	conf := &model.TPipelineConf{
		PipelineId: "pipe1",
		YmlContent: "",
	}
	_, err := db.Insert(conf)
	if err != nil {
		t.Fatalf("failed to insert pipeline config: %v", err)
	}

	ctx := context.Background()
	_, _, err = Run(ctx, "user1", "pipe1", "sha123", "push")

	if err == nil {
		t.Fatal("Run should return error when YAML content is empty")
	}
	if err != ErrPipelineYmlEmpty {
		t.Errorf("expected ErrPipelineYmlEmpty, got: %v", err)
	}
}

func TestRun_InvalidYml(t *testing.T) {
	db := setupPipelineTestDB(t)
	comm.Db = db
	defer func() { comm.Db = nil }()

	// Insert pipeline config with invalid YAML
	conf := &model.TPipelineConf{
		PipelineId: "pipe1",
		YmlContent: "invalid: yaml: content: [unclosed",
	}
	_, err := db.Insert(conf)
	if err != nil {
		t.Fatalf("failed to insert pipeline config: %v", err)
	}

	ctx := context.Background()
	_, _, err = Run(ctx, "user1", "pipe1", "sha123", "push")

	if err == nil {
		t.Fatal("Run should return error when YAML is invalid")
	}
}

func TestReBuild_PipelineNotFound(t *testing.T) {
	db := setupPipelineTestDB(t)
	comm.Db = db
	defer func() { comm.Db = nil }()

	ctx := context.Background()
	tvp := &model.TPipelineVersion{
		PipelineId: "nonexistent",
		Content:    "stages: []",
	}
	_, _, err := ReBuild(ctx, "user1", tvp)

	if err == nil {
		t.Fatal("ReBuild should return error when pipeline not found")
	}
	if err != ErrPipelineNotFound {
		t.Errorf("expected ErrPipelineNotFound, got: %v", err)
	}
}

func TestReBuild_EmptyContent(t *testing.T) {
	db := setupPipelineTestDB(t)
	comm.Db = db
	defer func() { comm.Db = nil }()

	// Insert pipeline config
	conf := &model.TPipelineConf{
		PipelineId: "pipe1",
		YmlContent: "stages: []",
	}
	_, err := db.Insert(conf)
	if err != nil {
		t.Fatalf("failed to insert pipeline config: %v", err)
	}

	ctx := context.Background()
	tvp := &model.TPipelineVersion{
		PipelineId: "pipe1",
		Content:    "",
	}
	_, _, err = ReBuild(ctx, "user1", tvp)

	if err == nil {
		t.Fatal("ReBuild should return error when content is empty")
	}
	if err != ErrPipelineYmlEmpty {
		t.Errorf("expected ErrPipelineYmlEmpty, got: %v", err)
	}
}

func TestReBuild_InvalidYml(t *testing.T) {
	db := setupPipelineTestDB(t)
	comm.Db = db
	defer func() { comm.Db = nil }()

	// Insert pipeline config
	conf := &model.TPipelineConf{
		PipelineId: "pipe1",
		YmlContent: "stages: []",
	}
	_, err := db.Insert(conf)
	if err != nil {
		t.Fatalf("failed to insert pipeline config: %v", err)
	}

	ctx := context.Background()
	tvp := &model.TPipelineVersion{
		PipelineId: "pipe1",
		Content:    "invalid: yaml: [unclosed",
	}
	_, _, err = ReBuild(ctx, "user1", tvp)

	if err == nil {
		t.Fatal("ReBuild should return error when YAML is invalid")
	}
}
