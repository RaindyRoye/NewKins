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
	app := comm.GetApp()
	var err error
	dvs := comm.DatasourceDriverMySQL
	ul := app.Cfg.Datasource.Url
	if app.Cfg.Datasource.Driver != "" {
		dvs = app.Cfg.Datasource.Driver
	}
	app.IsMySQL = dvs == comm.DatasourceDriverMySQL
	if !app.Installed {
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
	db, err := xorm.NewEngine(dvs, app.Cfg.Datasource.Url)
	if err != nil {
		return fmt.Errorf("open database (%s): %w", dvs, err)
	}
	db.ShowSQL(core.Debug)
	app.Db = db
	// Sync to globals for backward compatibility
	comm.SyncToGlobals()
	return nil
}

func initCache() error {
	app := comm.GetApp()
	pth := filepath.Join(app.WorkPath, "cache.dat")
	_ = os.Remove(pth)
	db, err := bolt.Open(pth, 0640, nil)
	if err != nil {
		logrus.Errorf("InitCache err:%v", err)
		return fmt.Errorf("open cache db at %s: %w", pth, err)
	}
	app.BCache = db
	// Sync to globals for backward compatibility
	comm.SyncToGlobals()
	return nil
}
