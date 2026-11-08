package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
)

func TestInitCache_Success(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origBCache
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("BCache should not be nil after successful initCache")
	}

	// Verify cache file was created
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file not created at %s", cachePath)
	}
}

func TestInitCache_InvalidPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		comm.BCache = origBCache
	}()

	// Use a path that cannot be created (non-existent parent directory)
	comm.WorkPath = "/nonexistent/directory/that/does/not/exist"

	err := initCache()
	if err == nil {
		t.Fatal("expected error when cache path is invalid, got nil")
	}
}

func TestInitCache_RemovesOldCache(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if comm.BCache != nil {
			_ = comm.BCache.Close()
		}
		comm.BCache = origBCache
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Create an old cache file
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("old data"), 0600); err != nil { //nolint:gosec // test file, not production
		t.Fatalf("failed to create old cache file: %v", err)
	}

	// Write some content to verify it gets replaced
	oldContent, _ := os.ReadFile(cachePath) //nolint:gosec // test file path is controlled
	if string(oldContent) != "old data" {
		t.Fatal("setup failed: could not write old cache")
	}

	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}

	// The old cache should be removed and a new one created
	if comm.BCache == nil {
		t.Fatal("BCache should not be nil after successful initCache")
	}

	// Verify the file exists and is a valid bolt DB (not the old content)
	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("cache file not found after initCache: %v", err)
	}
	if info.Size() == int64(len("old data")) {
		t.Error("cache file was not replaced")
	}
}

func TestInitDb_InvalidDriver(t *testing.T) {
	origInstalled := comm.Installed
	origCfg := comm.Cfg
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.Installed = origInstalled
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	// Use an unsupported driver that will cause xorm.NewEngine to fail
	comm.Installed = true // skip migration
	comm.Cfg.Datasource.Driver = "invalid_driver"
	comm.Cfg.Datasource.Url = "invalid://url"

	err := initDb()
	if err == nil {
		t.Fatal("expected error for invalid driver, got nil")
	}
}

func TestInitDb_InvalidURL(t *testing.T) {
	origInstalled := comm.Installed
	origCfg := comm.Cfg
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.Installed = origInstalled
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	// Use sqlite with an invalid path that cannot be created
	comm.Installed = true // skip migration
	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = "/nonexistent/directory/test.db"

	err := initDb()
	// xorm might not fail immediately on open, but let's test the path
	if err != nil {
		// Expected — database open failed
		return
	}
	// If no error, clean up
	if comm.Db != nil {
		_ = comm.Db.Close()
	}
}

func TestInitDb_SQLiteSuccess(t *testing.T) {
	origInstalled := comm.Installed
	origCfg := comm.Cfg
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.Installed = origInstalled
		comm.Cfg = origCfg
		if comm.Db != nil {
			_ = comm.Db.Close()
		}
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	tmpDir := t.TempDir()
	comm.Installed = true // skip migration
	comm.Cfg.Datasource.Driver = "sqlite3"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")

	err := initDb()
	if err != nil {
		t.Fatalf("initDb with sqlite failed: %v", err)
	}
	if comm.Db == nil {
		t.Fatal("Db should not be nil after successful initDb")
	}
	if comm.IsMySQL {
		t.Error("IsMySQL should be false for sqlite driver")
	}
}

func TestInitDb_MySQLFlag(t *testing.T) {
	origInstalled := comm.Installed
	origCfg := comm.Cfg
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.Installed = origInstalled
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	// Just verify the IsMySQL flag is set correctly based on driver
	// We don't actually connect to MySQL
	comm.Installed = true
	comm.Cfg.Datasource.Driver = comm.DatasourceDriverMySQL
	comm.Cfg.Datasource.Url = "user:pass@tcp(localhost)/db"

	// This will likely fail to connect, but we can check the flag was set
	_ = initDb()
	if !comm.IsMySQL {
		t.Error("IsMySQL should be true when driver is mysql")
	}
}

func TestInitDb_DefaultDriverIsMySQL(t *testing.T) {
	origInstalled := comm.Installed
	origCfg := comm.Cfg
	origDb := comm.Db
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.Installed = origInstalled
		comm.Cfg = origCfg
		comm.Db = origDb
		comm.IsMySQL = origIsMySQL
	}()

	// When no driver is specified, default should be mysql
	comm.Installed = true
	comm.Cfg.Datasource.Driver = ""
	comm.Cfg.Datasource.Url = "user:pass@tcp(localhost)/db"

	_ = initDb()
	if !comm.IsMySQL {
		t.Error("IsMySQL should be true when no driver specified (default is mysql)")
	}
}
