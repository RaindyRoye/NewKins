package engine

import (
	"container/list"
	"context"
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

func setupBuildEgnTestDB(t *testing.T) {
	t.Helper()
	origDb := comm.Db
	t.Cleanup(func() { comm.Db = origDb })

	db, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	tables := []string{
		`CREATE TABLE t_build (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			status VARCHAR(50),
			error TEXT
		)`,
		`CREATE TABLE t_stage (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			status VARCHAR(50),
			error TEXT
		)`,
		`CREATE TABLE t_step (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			status VARCHAR(50),
			error TEXT
		)`,
		`CREATE TABLE t_cmd_line (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			status VARCHAR(50)
		)`,
	}
	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}
	comm.Db = db
}

func TestBuildEngineInitWithContext(t *testing.T) {
	setupBuildEgnTestDB(t)
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	ctx := context.Background()
	c.initWithContext(ctx)
	// initWithContext should complete without panicking and cancel pending builds
}

func TestBuildEngineInitWithContext_Canceled(t *testing.T) {
	setupBuildEgnTestDB(t)
	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	// initWithContext should handle canceled context gracefully without panicking
	c.initWithContext(ctx)
}

func TestBuildEngineInitWithContext_PendingBuilds(t *testing.T) {
	setupBuildEgnTestDB(t)
	// Insert a pending build
	_, err := comm.Db.Exec("INSERT INTO t_build (id, status, error) VALUES ('b1', 'running', '')")
	if err != nil {
		t.Fatalf("failed to insert test build: %v", err)
	}
	// Insert a pending stage
	_, err = comm.Db.Exec("INSERT INTO t_stage (id, status, error) VALUES ('s1', 'running', '')")
	if err != nil {
		t.Fatalf("failed to insert test stage: %v", err)
	}

	c := &BuildEngine{
		taskw: list.New(),
		tasks: make(map[string]*BuildTask),
	}
	c.initWithContext(context.Background())

	// Verify build was canceled
	var status string
	ok, err := comm.Db.SQL("SELECT status FROM t_build WHERE id='b1'").Get(&status)
	if err != nil || !ok {
		t.Fatalf("failed to query build status: %v", err)
	}
	if status != "cancel" {
		t.Errorf("build status = %q, want 'cancel'", status)
	}

	// Verify stage was canceled
	ok, err = comm.Db.SQL("SELECT status FROM t_stage WHERE id='s1'").Get(&status)
	if err != nil || !ok {
		t.Fatalf("failed to query stage status: %v", err)
	}
	if status != "cancel" {
		t.Errorf("stage status = %q, want 'cancel'", status)
	}
}
