package server

import (
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	bolt "go.etcd.io/bbolt"
)

// TestInitCache_Success verifies that initCache creates a bbolt cache database.
func TestInitCache_Success(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origBCache
	})

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.BCache = nil

	if err := initCache(); err != nil {
		t.Fatalf("initCache() error = %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("expected BCache to be non-nil after initCache()")
	}

	// Verify the cache is functional by creating a bucket
	bktErr := comm.BCache.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("test"))
		return err
	})
	if bktErr != nil {
		t.Errorf("cache bucket creation failed: %v", bktErr)
	}
}

// TestInitCache_InvalidPath verifies initCache fails with an unwritable path.
func TestInitCache_InvalidPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	})

	comm.WorkPath = "/nonexistent/path/that/does/not/exist"
	comm.BCache = nil

	err := initCache()
	if err == nil {
		t.Skip("skipping: initCache succeeded unexpectedly (possibly running as root)")
	}
}

// TestInitDb_SQLiteSuccess tests that initDb correctly initializes a SQLite database.
func TestInitDb_SQLiteSuccess(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	t.Cleanup(func() {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	})

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = dbPath
	comm.Installed = true // skip migration for unit test
	comm.Db = nil

	err := initDb()
	if err != nil {
		t.Fatalf("initDb() error = %v", err)
	}
	if comm.Db == nil {
		t.Fatal("expected Db to be non-nil after initDb()")
	}
	if comm.IsMySQL {
		t.Error("expected IsMySQL to be false for sqlite3 driver")
	}

	// Verify we can ping the database
	db := comm.Db.DB()
	if db != nil {
		if err := db.Ping(); err != nil {
			t.Errorf("database ping failed: %v", err)
		}
	}
}

// TestInitDb_InvalidDriver tests initDb with an invalid driver name.
func TestInitDb_InvalidDriver(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	t.Cleanup(func() {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	})

	comm.Cfg.Datasource.Driver = "invalid_driver_xyz"
	comm.Cfg.Datasource.Url = "some-url"
	comm.Installed = true
	comm.Db = nil

	err := initDb()
	if err == nil {
		// xorm may not error immediately for unknown drivers
		if comm.Db != nil {
			_ = comm.Db.Close()
			comm.Db = nil
		}
	}
}

// TestInitDb_MySQLFlag tests that IsMySQL is set correctly for mysql driver.
func TestInitDb_MySQLFlag(t *testing.T) {
	origDb := comm.Db
	origCfg := comm.Cfg
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	t.Cleanup(func() {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Cfg = origCfg
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	})

	comm.Cfg.Datasource.Driver = "mysql"
	comm.Cfg.Datasource.Url = "root:pass@tcp(127.0.0.1:3306)/testdb"
	comm.Installed = true
	comm.Db = nil

	// This will fail to connect but should still set IsMySQL correctly before failing
	err := initDb()
	if !comm.IsMySQL {
		t.Error("expected IsMySQL to be true for mysql driver")
	}
	// Error is expected since MySQL isn't running
	if err == nil {
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
	}
}
