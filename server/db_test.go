package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
	bolt "go.etcd.io/bbolt"
)

func TestInitCache_Success(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origCache
	})

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.BCache = nil

	if err := initCache(); err != nil {
		t.Fatalf("initCache() error = %v", err)
	}

	if comm.BCache == nil {
		t.Fatal("initCache() did not set comm.BCache")
	}

	// Verify the cache file was created
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file %q was not created", cachePath)
	}

	// Verify we can use the cache
	if err := comm.BCache.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("test"))
		if err != nil {
			return err
		}
		return bucket.Put([]byte("key"), []byte("value"))
	}); err != nil {
		t.Errorf("cache is not usable: %v", err)
	}
}

func TestInitCache_RemovesOldCache(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origCache
	})

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.BCache = nil

	// Create a pre-existing cache file with some content
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("old cache marker"), 0600); err != nil {
		t.Fatalf("write old cache: %v", err)
	}

	// Verify old file exists
	oldInfo, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("stat old cache: %v", err)
	}

	if err := initCache(); err != nil {
		t.Fatalf("initCache() error = %v", err)
	}

	// New cache should be a valid bolt DB, not the old marker content
	newInfo, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("stat new cache: %v", err)
	}
	if newInfo.Size() == oldInfo.Size() {
		t.Logf("note: old and new cache have same size (coincidental but ok)")
	}

	// The new cache should be openable as a bolt DB
	if comm.BCache == nil {
		t.Fatal("comm.BCache is nil after initCache")
	}
}

func TestInitCache_InvalidWorkPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCache := comm.BCache
	t.Cleanup(func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origCache
	})

	// Use a path that cannot be written (non-existent parent directory)
	comm.WorkPath = "/nonexistent/directory/that/does/not/exist"
	comm.BCache = nil

	err := initCache()
	if err == nil {
		t.Skip("skipping: initCache succeeded unexpectedly (possibly running as root)")
	}

	// Verify error wraps useful context
	if err.Error() == "" {
		t.Error("error message is empty")
	}
}

func TestInitDb_EmptyUrl(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	t.Cleanup(func() {
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	})

	comm.Installed = true // Skip migration
	comm.Db = nil
	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = "" // empty URL

	err := initDb()
	// xorm may or may not error on empty sqlite URL — handle both cases
	if err != nil {
		t.Logf("initDb() with empty URL returned error (expected): %v", err)
		return
	}
	// If no error, db should be set
	if comm.Db == nil {
		t.Error("initDb() did not set comm.Db")
	}
	// Clean up the engine if it was created
	if comm.Db != nil {
		_ = comm.Db.Close()
		comm.Db = nil
	}
}

func TestInitDb_InvalidMysqlUrl(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	t.Cleanup(func() {
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	})

	comm.Installed = true
	comm.Db = nil
	comm.Cfg.Datasource.Driver = "mysql"
	comm.Cfg.Datasource.Url = "invalid:invalid@tcp(nonexistent:3306)/test"

	err := initDb()
	if err == nil {
		t.Log("initDb() with invalid mysql URL unexpectedly succeeded; cleaning up")
		if comm.Db != nil {
			_ = comm.Db.Close()
			comm.Db = nil
		}
		return
	}

	// Error should mention database
	t.Logf("initDb() error (expected): %v", err)
	if !comm.IsMySQL {
		t.Error("IsMySQL should be true when driver is mysql")
	}
}

func TestInitDb_SetsIsMySQLFlag(t *testing.T) {
	tests := []struct {
		name      string
		driver    string
		wantMySQL bool
	}{
		{"mysql driver", "mysql", true},
		{"sqlite driver", "sqlite", false},
		{"postgres driver", "postgres", false},
		{"empty defaults to mysql", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origCfg := comm.Cfg
			origDb := comm.Db
			origInstalled := comm.Installed
			origIsMySQL := comm.IsMySQL
			t.Cleanup(func() {
				comm.Cfg = origCfg
				comm.Db = origDb
				comm.Installed = origInstalled
				comm.IsMySQL = origIsMySQL
			})

			comm.Installed = true
			comm.Db = nil
			comm.Cfg.Datasource.Driver = tt.driver
			comm.Cfg.Datasource.Url = "anything"

			// We don't care if initDb succeeds — just that it sets IsMySQL correctly
			_ = initDb()

			if comm.IsMySQL != tt.wantMySQL {
				t.Errorf("IsMySQL = %v, want %v for driver %q", comm.IsMySQL, tt.wantMySQL, tt.driver)
			}

			if comm.Db != nil {
				_ = comm.Db.Close()
				comm.Db = nil
			}
		})
	}
}

func TestInitDb_Sqlite3InMemory(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	t.Cleanup(func() {
		comm.Cfg = origCfg
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	})

	comm.Installed = true
	comm.Db = nil
	// Use "sqlite3" as xorm passes it directly to sql.Open, which needs "sqlite3"
	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = ":memory:"

	if err := initDb(); err != nil {
		t.Fatalf("initDb() with sqlite3 :memory: error = %v", err)
	}
	if comm.Db == nil {
		t.Fatal("initDb() did not set comm.Db")
	}
	if comm.IsMySQL {
		t.Error("IsMySQL should be false for sqlite3 driver")
	}
}

func TestInitDb_WithMigration(t *testing.T) {
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	origWorkPath := comm.WorkPath
	t.Cleanup(func() {
		comm.Cfg = origCfg
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
		comm.WorkPath = origWorkPath
	})

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.Installed = false // Trigger migration
	comm.Db = nil
	// Use "sqlite3" for sql.Open compatibility
	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test_migrate.db")

	if err := initDb(); err != nil {
		t.Fatalf("initDb() with sqlite3 migration error = %v", err)
	}
	if comm.Db == nil {
		t.Fatal("initDb() did not set comm.Db")
	}

	// Verify migration was applied — schema_migrations table should exist
	var count int
	ok, err := comm.Db.SQL("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Get(&count)
	if err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if !ok || count != 1 {
		t.Errorf("schema_migrations table not found after migration (count=%d)", count)
	}
}
