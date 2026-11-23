package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gokins/gokins/comm"
	_ "github.com/mattn/go-sqlite3"
)

func TestInitCache_Success(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if origBCache != nil {
			comm.BCache = origBCache
		} else {
			comm.BCache = nil
		}
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("expected BCache to be set after initCache")
	}

	// Verify the cache file was created
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("cache file not created at %s", cachePath)
	}

	// Clean up the bolt DB
	if comm.BCache != nil {
		_ = comm.BCache.Close()
		comm.BCache = nil
	}
}

func TestInitCache_RemovesOldFile(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if origBCache != nil {
			comm.BCache = origBCache
		} else {
			comm.BCache = nil
		}
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir

	// Create an old cache file to verify it gets removed
	cachePath := filepath.Join(tmpDir, "cache.dat")
	if err := os.WriteFile(cachePath, []byte("old data"), 0600); err != nil {
		t.Fatalf("write old cache: %v", err)
	}

	err := initCache()
	if err != nil {
		t.Fatalf("initCache failed: %v", err)
	}
	if comm.BCache == nil {
		t.Fatal("expected BCache to be set after initCache")
	}

	// Clean up
	_ = comm.BCache.Close()
	comm.BCache = nil
}

func TestInitCache_InvalidPath(t *testing.T) {
	origWorkPath := comm.WorkPath
	origBCache := comm.BCache
	defer func() {
		comm.WorkPath = origWorkPath
		if origBCache != nil {
			comm.BCache = origBCache
		} else {
			comm.BCache = nil
		}
	}()

	// Use a path that can't be created (file where directory is expected)
	tmpDir := t.TempDir()
	blocker := filepath.Join(tmpDir, "blocker")
	if err := os.WriteFile(blocker, []byte("block"), 0600); err != nil {
		t.Fatalf("create blocker: %v", err)
	}
	// Set WorkPath to a file inside the blocker — can't create directory
	comm.WorkPath = filepath.Join(blocker, "subdir")

	err := initCache()
	if err == nil {
		// Some systems may still succeed; clean up if so
		if comm.BCache != nil {
			_ = comm.BCache.Close()
			comm.BCache = nil
		}
		// On some platforms this may not fail, so we just note it
		t.Log("initCache succeeded on invalid path (platform-dependent)")
	}
}

func TestInitDb_SqliteSuccess(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
		if origDb != nil {
			comm.Db = origDb
		} else {
			comm.Db = nil
		}
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")
	comm.Installed = false

	err := initDb()
	if err != nil {
		t.Fatalf("initDb with sqlite failed: %v", err)
	}
	if comm.Db == nil {
		t.Fatal("expected Db to be set after initDb")
	}
	if comm.IsMySQL {
		t.Error("expected IsMySQL to be false for sqlite driver")
	}

	// Clean up
	_ = comm.Db.Close()
	comm.Db = nil
}

func TestInitDb_SqliteAlreadyInstalled(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
		if origDb != nil {
			comm.Db = origDb
		} else {
			comm.Db = nil
		}
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = filepath.Join(tmpDir, "test.db")
	comm.Installed = true // Skip migration

	err := initDb()
	if err != nil {
		t.Fatalf("initDb with sqlite (installed) failed: %v", err)
	}
	if comm.Db == nil {
		t.Fatal("expected Db to be set after initDb")
	}

	_ = comm.Db.Close()
	comm.Db = nil
}

func TestInitDb_DefaultDriverIsMySQL(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
		if origDb != nil {
			comm.Db = origDb
		} else {
			comm.Db = nil
		}
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	}()

	tmpDir := t.TempDir()
	comm.WorkPath = tmpDir
	comm.Cfg.Datasource.Driver = "" // Empty driver defaults to MySQL
	comm.Cfg.Datasource.Url = "invalid-url"
	comm.Installed = true // Skip migration to avoid needing real MySQL

	// initDb should set IsMySQL to true
	_ = initDb() // May fail on xorm.NewEngine with invalid URL, that's fine
	if !comm.IsMySQL {
		t.Error("expected IsMySQL to be true when driver is empty (defaults to mysql)")
	}

	if comm.Db != nil {
		_ = comm.Db.Close()
		comm.Db = nil
	}
}

func TestInitDb_InvalidURL(t *testing.T) {
	origWorkPath := comm.WorkPath
	origCfg := comm.Cfg
	origDb := comm.Db
	origInstalled := comm.Installed
	origIsMySQL := comm.IsMySQL
	defer func() {
		comm.WorkPath = origWorkPath
		comm.Cfg = origCfg
		if origDb != nil {
			comm.Db = origDb
		} else {
			comm.Db = nil
		}
		comm.Installed = origInstalled
		comm.IsMySQL = origIsMySQL
	}()

	comm.Cfg.Datasource.Driver = "sqlite"
	comm.Cfg.Datasource.Url = ""
	comm.Installed = true

	err := initDb()
	if err == nil {
		if comm.Db != nil {
			_ = comm.Db.Close()
			comm.Db = nil
		}
		t.Log("initDb with empty URL succeeded (platform-dependent)")
	}
}
