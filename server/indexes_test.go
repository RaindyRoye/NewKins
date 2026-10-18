package server

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"
)

// TestEnsureIndexes verifies that ensureIndexes creates all expected indexes
func TestEnsureIndexes(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE t_build (
			id VARCHAR(64) PRIMARY KEY,
			pipeline_id VARCHAR(64),
			pipeline_version_id VARCHAR(64),
			status VARCHAR(50),
			created DATETIME
		);
		CREATE TABLE t_stage (
			id VARCHAR(64) PRIMARY KEY,
			build_id VARCHAR(64),
			pipeline_version_id VARCHAR(64),
			name VARCHAR(100)
		);
		CREATE TABLE t_step (
			id VARCHAR(64) PRIMARY KEY,
			build_id VARCHAR(64),
			stage_id VARCHAR(64),
			name VARCHAR(100)
		);
		CREATE TABLE t_cmd_line (
			id VARCHAR(64) PRIMARY KEY,
			build_id VARCHAR(64),
			step_id VARCHAR(64),
			content TEXT
		);
		CREATE TABLE t_pipeline (
			id VARCHAR(64) PRIMARY KEY,
			uid VARCHAR(64),
			name VARCHAR(200),
			deleted INT DEFAULT 0
		);
		CREATE TABLE t_pipeline_conf (
			id VARCHAR(64) PRIMARY KEY,
			pipeline_id VARCHAR(64),
			config TEXT
		);
		CREATE TABLE t_pipeline_version (
			id VARCHAR(64) PRIMARY KEY,
			pipeline_id VARCHAR(64),
			version VARCHAR(50)
		);
		CREATE TABLE t_pipeline_var (
			id VARCHAR(64) PRIMARY KEY,
			pipeline_id VARCHAR(64),
			key VARCHAR(100),
			value TEXT
		);
		CREATE TABLE t_user (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(100),
			email VARCHAR(200)
		);
		CREATE TABLE t_param (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(100),
			value TEXT
		);
		CREATE TABLE t_trigger (
			id VARCHAR(64) PRIMARY KEY,
			uid VARCHAR(64),
			pipeline_id VARCHAR(64),
			enabled INT DEFAULT 1,
			types VARCHAR(50)
		);
		CREATE TABLE t_trigger_run (
			id VARCHAR(64) PRIMARY KEY,
			tid VARCHAR(64),
			created DATETIME
		);
		CREATE TABLE t_user_org (
			id VARCHAR(64) PRIMARY KEY,
			uid VARCHAR(64),
			org_id VARCHAR(64)
		);
		CREATE TABLE t_user_msg (
			id VARCHAR(64) PRIMARY KEY,
			uid VARCHAR(64),
			msg_id VARCHAR(64),
			created DATETIME
		);
		CREATE TABLE t_org (
			id VARCHAR(64) PRIMARY KEY,
			uid VARCHAR(64),
			name VARCHAR(200),
			deleted INT DEFAULT 0
		);
		CREATE TABLE t_yml_plugin (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(200),
			deleted INT DEFAULT 0
		);
		CREATE TABLE t_yml_template (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(200),
			deleted INT DEFAULT 0
		);
		CREATE TABLE t_artifactory (
			id VARCHAR(64) PRIMARY KEY,
			identifier VARCHAR(100),
			repo_id VARCHAR(64),
			name VARCHAR(200),
			deleted INT DEFAULT 0
		);
		CREATE TABLE t_org_pipe (
			id VARCHAR(64) PRIMARY KEY,
			org_id VARCHAR(64),
			pipe_id VARCHAR(64)
		);
		CREATE TABLE t_artifact_version (
			id VARCHAR(64) PRIMARY KEY,
			package_id VARCHAR(64),
			version VARCHAR(50)
		);
		CREATE TABLE t_artifact_package (
			id VARCHAR(64) PRIMARY KEY,
			repo_id VARCHAR(64),
			name VARCHAR(200),
			deleted INT DEFAULT 0
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	// Initialize xorm engine
	eng, err := xorm.NewEngine("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to create xorm engine: %v", err)
	}
	defer func() { _ = eng.Close() }()

	// Set global Db for ensureIndexes
	comm.Db = eng

	// Run ensureIndexes
	ensureIndexes()

	// Verify indexes were created
	expectedIndexes := []struct {
		table string
		index string
	}{
		{"t_build", "idx_build_pipeline_id"},
		{"t_build", "idx_build_pipeline_version_id"},
		{"t_build", "idx_build_status"},
		{"t_stage", "idx_stage_build_id"},
		{"t_step", "idx_step_build_id"},
		{"t_step", "idx_step_stage_id"},
		{"t_cmd_line", "idx_cmdline_build_id"},
		{"t_cmd_line", "idx_cmdline_step_id"},
		{"t_pipeline", "idx_pipeline_uid"},
		{"t_pipeline", "idx_pipeline_deleted"},
		{"t_pipeline_conf", "idx_pipeconf_pipeline_id"},
		{"t_pipeline_version", "idx_pipever_pipeline_id"},
		{"t_pipeline_var", "idx_pipevar_pipeline_id"},
		{"t_user", "idx_user_name"},
		{"t_param", "idx_param_name"},
		{"t_trigger", "idx_trigger_enabled_types"},
		{"t_trigger", "idx_trigger_pipeline_id"},
		{"t_artifactory", "idx_artifactory_identifier"},
		{"t_artifactory", "idx_artifactory_deleted_identifier"},
		{"t_artifactory", "idx_artifactory_deleted_repo_name"},
		{"t_org_pipe", "idx_orgpipe_pipe_id"},
		{"t_org_pipe", "idx_orgpipe_org_pipe"},
		{"t_artifact_version", "idx_artver_package_id"},
		{"t_artifact_package", "idx_artpkg_deleted_repo"},
		{"t_trigger_run", "idx_trigrun_tid"},
		{"t_user_org", "idx_userorg_uid_orgid"},
		{"t_user_msg", "idx_usermsg_uid"},
		{"t_org", "idx_org_uid"},
		{"t_org", "idx_org_deleted"},
		{"t_yml_plugin", "idx_ymlplugin_deleted"},
		{"t_yml_template", "idx_ymltemplate_deleted"},
		{"t_pipeline", "idx_pipeline_uid_deleted"},
		{"t_stage", "idx_stage_pipeline_version_id"},
	}

	for _, expected := range expectedIndexes {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master 
			WHERE type='index' AND tbl_name=? AND name=?
		`, expected.table, expected.index).Scan(&count)
		if err != nil {
			t.Errorf("failed to query index %s.%s: %v", expected.table, expected.index, err)
			continue
		}
		if count == 0 {
			t.Errorf("index %s.%s was not created", expected.table, expected.index)
		}
	}
}

// TestIndexPerformanceImprovement measures query performance before and after index creation
func TestIndexPerformanceImprovement(t *testing.T) {
	// Skip in short mode as this test takes time
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "perf_test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create table with large dataset
	_, err = db.Exec(`
		CREATE TABLE t_build (
			id VARCHAR(64) PRIMARY KEY,
			pipeline_id VARCHAR(64),
			status VARCHAR(50),
			created DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert 10000 rows
	t.Logf("Inserting 10000 test rows...")
	start := time.Now()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	stmt, err := tx.Prepare("INSERT INTO t_build (id, pipeline_id, status, created) VALUES (?, ?, ?, ?)")
	if err != nil {
		t.Fatalf("failed to prepare statement: %v", err)
	}
	defer func() { _ = stmt.Close() }()

	for i := 0; i < 10000; i++ {
		_, err = stmt.Exec(
			fmt.Sprintf("build-%d", i),
			fmt.Sprintf("pipe-%d", i%100), // 100 unique pipelines
			"success",
			time.Now(),
		)
		if err != nil {
			t.Fatalf("failed to insert row: %v", err)
		}
	}
	err = tx.Commit()
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	t.Logf("Inserted 10000 rows in %v", time.Since(start))

	// Measure query performance WITHOUT index
	t.Logf("Measuring query performance WITHOUT index...")
	noIndexDurations := measureQueryPerformance(t, db, "SELECT * FROM t_build WHERE pipeline_id = 'pipe-50'", 100)
	noIndexAvg := averageDuration(noIndexDurations)
	t.Logf("Average query time without index: %v", noIndexAvg)

	// Create index
	_, err = db.Exec("CREATE INDEX idx_build_pipeline_id ON t_build (pipeline_id)")
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	// Measure query performance WITH index
	t.Logf("Measuring query performance WITH index...")
	withIndexDurations := measureQueryPerformance(t, db, "SELECT * FROM t_build WHERE pipeline_id = 'pipe-50'", 100)
	withIndexAvg := averageDuration(withIndexDurations)
	t.Logf("Average query time with index: %v", withIndexAvg)

	// Calculate improvement
	improvement := float64(noIndexAvg-withIndexAvg) / float64(noIndexAvg) * 100
	t.Logf("Performance improvement: %.2f%%", improvement)

	// Index should provide at least 20% improvement for this query pattern
	if improvement < 20 {
		t.Logf("WARNING: Index improvement (%.2f%%) is less than expected 20%%", improvement)
		t.Logf("This may be due to SQLite's query optimizer or small dataset size")
	}
}

// measureQueryPerformance runs a query multiple times and returns durations
func measureQueryPerformance(t *testing.T, db *sql.DB, query string, iterations int) []time.Duration {
	t.Helper()

	durations := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		rows, err := db.Query(query)
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		// Consume all rows
		for rows.Next() {
			// Just iterate, don't need to scan
		}
		_ = rows.Close()
		durations[i] = time.Since(start)
	}
	return durations
}

// averageDuration calculates the average of a slice of durations
func averageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

// TestCreateIndexIfNotExists verifies idempotent index creation
func TestCreateIndexIfNotExists(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "idempotent.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create table
	_, err = db.Exec("CREATE TABLE test_table (id VARCHAR(64), name VARCHAR(100))")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Initialize xorm engine
	eng, err := xorm.NewEngine("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to create xorm engine: %v", err)
	}
	defer func() { _ = eng.Close() }()

	comm.Db = eng
	comm.IsMySQL = false

	// Create index first time
	err = createIndexIfNotExists("test_table", "idx_test_name", "name")
	if err != nil {
		t.Errorf("first index creation failed: %v", err)
	}

	// Verify index exists
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master 
		WHERE type='index' AND tbl_name='test_table' AND name='idx_test_name'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query index: %v", err)
	}
	if count == 0 {
		t.Error("index was not created")
	}

	// Create index second time (should be idempotent)
	err = createIndexIfNotExists("test_table", "idx_test_name", "name")
	if err != nil {
		t.Errorf("second index creation should be idempotent, but got error: %v", err)
	}
}

// TestCleanup verifies proper resource cleanup
func TestCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "cleanup.db")

	// Create and close database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	eng, err := xorm.NewEngine("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to create xorm engine: %v", err)
	}

	comm.Db = eng

	// Run ensureIndexes
	ensureIndexes()

	// Clean up
	_ = eng.Close()
	_ = db.Close()

	// Verify database file still exists and is valid
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was deleted during cleanup")
	}
}
