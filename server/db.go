package server

import (
	"context"
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
		ctx := context.Background()
		switch dvs {
		case comm.DatasourceDriverMySQL:
			err = migrates.UpMysqlMigrate(ctx, ul)
		case comm.DatasourceDriverPostgres:
			err = migrates.UpPostgresMigrate(ctx, ul)
		default:
			err = migrates.UpSqliteMigrate(ctx, ul)
		}
	}
	if err != nil {
		return fmt.Errorf("database migration: %w", err)
	}
	db, err := xorm.NewEngine(dvs, comm.Cfg.Datasource.Url)
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
