package server

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gokins/core"
	"github.com/gokins/gokins/comm"
	"github.com/gokins/gokins/migrates"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"xorm.io/xorm"
)

func initDb() error {
	var err error
	dvs := comm.DatasourceDriverMySQL
	ul := comm.Cfg.Datasource.Url
	if comm.Cfg.Datasource.Driver != "" {
		dvs = comm.Cfg.Datasource.Driver
	}
	comm.IsMySQL = dvs == comm.DatasourceDriverMySQL
	if !comm.Installed {
		switch dvs {
		case comm.DatasourceDriverMySQL:
			err = migrates.UpMysqlMigrate(ul)
		case comm.DatasourceDriverPostgres:
			err = migrates.UpPostgresMigrate(ul)
		default:
			err = migrates.UpSqliteMigrate(ul)
		}
	}
	if err != nil {
		return fmt.Errorf("database migration: %w", err)
	}
	// xorm uses the registered sql driver name, which differs from config names.
	// "sqlite" in config → "sqlite3" driver; "postgres" → "postgres" driver.
	xormDriver := dvs
	if dvs == comm.DatasourceDriverSQLite {
		xormDriver = "sqlite3"
	}
	if ul == "" {
		return fmt.Errorf("database URL is empty for driver %q", dvs)
	}
	db, err := xorm.NewEngine(xormDriver, ul)
	if err != nil {
		return fmt.Errorf("open database (%s): %w", dvs, err)
	}
	db.ShowSQL(core.Debug)
	comm.Db = db
	return nil
}

func initCache() error {
	pth := filepath.Join(comm.WorkPath, "cache.dat")
	_ = os.Remove(pth)
	db, err := bolt.Open(pth, 0640, nil)
	if err != nil {
		logrus.Errorf("InitCache err:%v", err)
		return fmt.Errorf("open cache db at %s: %w", pth, err)
	}
	comm.BCache = db
	return nil
}
