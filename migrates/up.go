package migrates

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gokins/gokins/comm"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/sirupsen/logrus"
)

func UpMysqlMigrate(ul string) error {
	if ul == "" {
		return errors.New("database config not found")
	}
	db, err := sql.Open("mysql", ul)
	if err != nil {
		logrus.Errorf("mysql open db error: %v", err)
		return fmt.Errorf("open mysql database: %w", err)
	}
	defer func() { _ = db.Close() }()
	err = db.Ping()
	if err != nil {
		logrus.Errorf("mysql ping failed: %v", err)
		return fmt.Errorf("ping mysql database: %w", err)
	}

	// Run migrations
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		logrus.Errorf("mysql migration driver error: %v", err)
		return fmt.Errorf("init mysql migration driver: %w", err)
	}
	defer func() { _ = driver.Close() }()
	var nms []string
	tms := comm.AssetNames()
	for _, v := range tms {
		if strings.HasPrefix(v, "mysql") {
			nms = append(nms, strings.Replace(v, "mysql/", "", 1))
		}
	}
	s := bindata.Resource(nms, func(name string) ([]byte, error) {
		return comm.Asset("mysql/" + name)
	})
	sc, err := bindata.WithInstance(s)
	if err != nil {
		return fmt.Errorf("init mysql bindata source: %w", err)
	}
	defer func() { _ = sc.Close() }()
	mgt, err := migrate.NewWithInstance(
		"bindata", sc,
		"mysql", driver)
	if err != nil {
		return fmt.Errorf("create mysql migrate instance: %w", err)
	}
	defer func() { _, _ = mgt.Close() }()
	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		return fmt.Errorf("run mysql migration: %w", err)
	}

	return nil
}

func UpPostgresMigrate(ul string) error {
	if ul == "" {
		return errors.New("database config not found")
	}
	db, err := sql.Open("postgres", ul)
	if err != nil {
		logrus.Errorf("postgres open db error: %v", err)
		return fmt.Errorf("open postgres database: %w", err)
	}
	defer func() { _ = db.Close() }()
	err = db.Ping()
	if err != nil {
		logrus.Errorf("postgres ping failed: %v", err)
		return fmt.Errorf("ping postgres database: %w", err)
	}

	// Run migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logrus.Errorf("postgres migration driver error: %v", err)
		return fmt.Errorf("init postgres migration driver: %w", err)
	}
	defer func() { _ = driver.Close() }()
	var nms []string
	tms := comm.AssetNames()
	for _, v := range tms {
		if strings.HasPrefix(v, "postgres") {
			nms = append(nms, strings.Replace(v, "postgres/", "", 1))
		}
	}
	s := bindata.Resource(nms, func(name string) ([]byte, error) {
		return comm.Asset("postgres/" + name)
	})
	sc, err := bindata.WithInstance(s)
	if err != nil {
		return fmt.Errorf("init postgres bindata source: %w", err)
	}
	defer func() { _ = sc.Close() }()
	mgt, err := migrate.NewWithInstance(
		"bindata", sc,
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("create postgres migrate instance: %w", err)
	}
	defer func() { _, _ = mgt.Close() }()
	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		return fmt.Errorf("run postgres migration: %w", err)
	}

	return nil
}
func UpSqliteMigrate(ul string) error {
	if ul == "" {
		return errors.New("database config not found")
	}
	db, err := sql.Open("sqlite3", ul)
	if err != nil {
		logrus.Errorf("sqlite open db error: %v", err)
		return fmt.Errorf("open sqlite database: %w", err)
	}
	defer func() { _ = db.Close() }()
	err = db.Ping()
	if err != nil {
		logrus.Errorf("sqlite ping failed: %v", err)
		return fmt.Errorf("ping sqlite database: %w", err)
	}

	// Run migrations
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		logrus.Errorf("sqlite migration driver error: %v", err)
		return fmt.Errorf("init sqlite migration driver: %w", err)
	}
	defer func() { _ = driver.Close() }()
	var nms []string
	tms := comm.AssetNames()
	for _, v := range tms {
		if strings.HasPrefix(v, "sqlite") {
			nms = append(nms, strings.Replace(v, "sqlite/", "", 1))
		}
	}
	s := bindata.Resource(nms, func(name string) ([]byte, error) {
		return comm.Asset("sqlite/" + name)
	})
	sc, err := bindata.WithInstance(s)
	if err != nil {
		return fmt.Errorf("init sqlite bindata source: %w", err)
	}
	defer func() { _ = sc.Close() }()
	mgt, err := migrate.NewWithInstance(
		"bindata", sc,
		"sqlite3", driver)
	if err != nil {
		return fmt.Errorf("create sqlite migrate instance: %w", err)
	}
	defer func() { _, _ = mgt.Close() }()
	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		return fmt.Errorf("run sqlite migration: %w", err)
	}

	return nil
}
