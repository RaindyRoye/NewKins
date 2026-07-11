package migrates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
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

func InitMysqlMigrate(host, dbs, user, pass string) (wait bool, rtul string, errs error) {
	wait = false
	if host == "" || dbs == "" || user == "" {
		errs = ErrDatabaseConfigMissing
		return
	}
	wait = true
	ul := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&multiStatements=true",
		user,
		pass,
		host,
		dbs)
	db, err := sql.Open("mysql", ul)
	if err != nil {
		errs = fmt.Errorf("%w: %w", ErrOpenDatabase, err)
		return
	}
	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		_ = db.Close()
		uls := fmt.Sprintf("%s:%s@tcp(%s)/?parseTime=true&multiStatements=true",
			user,
			pass,
			host)
		db, err = sql.Open("mysql", uls)
		if err != nil {
			logrus.Errorf("InitMysqlMigrate: open dbs err: %v", err)
			errs = fmt.Errorf("%w: %w", ErrOpenDatabase, err)
			return
		}
		defer func() { _ = db.Close() }()
		_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s` DEFAULT CHARACTER SET utf8mb4;", dbs))
		if err != nil {
			logrus.Errorf("InitMysqlMigrate: create dbs err: %v", err)
			errs = fmt.Errorf("%w %q: %w", ErrCreateDatabase, dbs, err)
			return
		}
		_, _ = db.ExecContext(ctx, fmt.Sprintf("USE `%s`;", dbs))
		err = db.PingContext(ctx)
	}
	defer func() { _ = db.Close() }()
	wait = false
	if err != nil {
		errs = fmt.Errorf("%w: %w", ErrPingDatabase, err)
		return
	}

	// Run migrations
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		logrus.Errorf("InitMysqlMigrate: could not start sql migration: %v", err)
		errs = fmt.Errorf("%w: %w", ErrInitDriver, err)
		return
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
		errs = fmt.Errorf("%w: %w", ErrInitSource, err)
		return
	}
	defer func() { _ = sc.Close() }()
	mgt, err := migrate.NewWithInstance(
		"bindata", sc,
		"mysql", driver)
	if err != nil {
		errs = fmt.Errorf("%w: %w", ErrCreateMigrateInstance, err)
		return
	}
	defer func() { _, _ = mgt.Close() }()
	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		errs = fmt.Errorf("%w: %w", ErrRunMigration, err)
		return
	}

	return false, ul, nil
}

func InitSqliteMigrate() (rtul string, errs error) {
	ul := filepath.Join(comm.WorkPath, "db.dat")
	db, err := sql.Open("sqlite3", ul)
	if err != nil {
		errs = fmt.Errorf("%w: %w", ErrOpenDatabase, err)
		return
	}
	defer func() { _ = db.Close() }()

	// Run migrations
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		logrus.Errorf("InitSqliteMigrate: could not start sql migration: %v", err)
		errs = fmt.Errorf("%w: %w", ErrInitDriver, err)
		return
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
		errs = fmt.Errorf("%w: %w", ErrInitSource, err)
		return
	}
	defer func() { _ = sc.Close() }()
	mgt, err := migrate.NewWithInstance(
		"bindata", sc,
		"sqlite3", driver)
	if err != nil {
		errs = fmt.Errorf("%w: %w", ErrCreateMigrateInstance, err)
		return
	}
	defer func() { _, _ = mgt.Close() }()
	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		errs = fmt.Errorf("%w: %w", ErrRunMigration, err)
		return
	}

	return ul, nil
}

func InitPostgresMigrate(host, dbs, user, pass string) (wait bool, rtul string, errs error) {
	wait = false
	if host == "" || dbs == "" || user == "" {
		errs = ErrDatabaseConfigMissing
		return
	}
	wait = true
	// Build connection URL: postgres://user:***@host/database?sslmode=disable
	ul := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, pass, host, dbs)
	db, err := sql.Open("postgres", ul)
	if err != nil {
		errs = fmt.Errorf("%w: %w", ErrOpenDatabase, err)
		return
	}
	err = db.PingContext(context.Background())
	if err != nil {
		_ = db.Close()
		errs = fmt.Errorf("%w: %w", ErrPingDatabase, err)
		return
	}
	defer func() { _ = db.Close() }()
	wait = false

	// Run migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logrus.Errorf("InitPostgresMigrate: could not start sql migration: %v", err)
		errs = fmt.Errorf("%w: %w", ErrInitDriver, err)
		return
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
		errs = fmt.Errorf("%w: %w", ErrInitSource, err)
		return
	}
	defer func() { _ = sc.Close() }()
	mgt, err := migrate.NewWithInstance(
		"bindata", sc,
		"postgres", driver)
	if err != nil {
		errs = fmt.Errorf("%w: %w", ErrCreateMigrateInstance, err)
		return
	}
	defer func() { _, _ = mgt.Close() }()
	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		errs = fmt.Errorf("%w: %w", ErrRunMigration, err)
		return
	}

	return false, ul, nil
}
