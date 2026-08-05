package migrates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
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

func InitMysqlMigrate(ctx context.Context, host, dbs, user, pass string) (wait bool, rtul string, errs error) {
	wait = false
	if host == "" || dbs == "" || user == "" {
		errs = fmt.Errorf("%w: mysql requires host, database, and user", ErrDatabaseConfigMissing)
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
		errs = fmt.Errorf("open mysql database: %w", err)
		return
	}
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
			errs = fmt.Errorf("open database: %w", err)
			return
		}
		defer func() { _ = db.Close() }()
		_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s` DEFAULT CHARACTER SET utf8mb4;", dbs))
		if err != nil {
			logrus.Errorf("InitMysqlMigrate: create dbs err: %v", err)
			errs = fmt.Errorf("create database %q: %w", dbs, err)
			return
		}
		_, _ = db.ExecContext(ctx, fmt.Sprintf("USE `%s`;", dbs))
		err = db.PingContext(ctx)
	}
	defer func() { _ = db.Close() }()
	wait = false
	if err != nil {
		errs = fmt.Errorf("ping mysql database: %w", err)
		return
	}

	// Run migrations
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		logrus.Errorf("InitMysqlMigrate: could not start sql migration: %v", err)
		errs = fmt.Errorf("init mysql migration driver: %w", err)
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
		errs = fmt.Errorf("init mysql bindata source: %w", err)
		return
	}
	defer func() { _ = sc.Close() }()
	mgt, err := migrate.NewWithInstance(
		"bindata", sc,
		"mysql", driver)
	if err != nil {
		errs = fmt.Errorf("create mysql migrate instance: %w", err)
		return
	}
	defer func() { _, _ = mgt.Close() }()
	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		errs = fmt.Errorf("run mysql migration: %w", err)
		return
	}

	return false, ul, nil
}

func InitSqliteMigrate(ctx context.Context) (rtul string, errs error) {
	ul := filepath.Join(comm.WorkPath, "db.dat")
	db, err := sql.Open("sqlite3", ul)
	if err != nil {
		errs = fmt.Errorf("open sqlite database: %w", err)
		return
	}
	defer func() { _ = db.Close() }()

	// Run migrations
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		logrus.Errorf("InitSqliteMigrate: could not start sql migration: %v", err)
		errs = fmt.Errorf("init sqlite migration driver: %w", err)
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
		errs = fmt.Errorf("init sqlite bindata source: %w", err)
		return
	}
	defer func() { _ = sc.Close() }()
	mgt, err := migrate.NewWithInstance(
		"bindata", sc,
		"sqlite3", driver)
	if err != nil {
		errs = fmt.Errorf("create sqlite migrate instance: %w", err)
		return
	}
	defer func() { _, _ = mgt.Close() }()
	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		errs = fmt.Errorf("run sqlite migration: %w", err)
		return
	}

	return ul, nil
}

// BuildPostgresURL constructs a PostgreSQL connection URL using net/url
// to properly escape special characters in credentials. This prevents the
// bug where literal "***" in a format string caused the password to be
// interpreted as the host.
//
//nolint:gosec // G101: function builds connection URLs, not hardcoded credentials
func BuildPostgresURL(user, pass, host, dbs string) string {
	u := &url.URL{
		Scheme: "postgres",
		Host:   host,
		Path:   dbs,
	}
	if user != "" {
		u.User = url.UserPassword(user, pass)
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	return u.String()
}

func InitPostgresMigrate(ctx context.Context, host, dbs, user, pass string) (wait bool, rtul string, errs error) {
	wait = false
	if host == "" || dbs == "" || user == "" {
		errs = fmt.Errorf("%w: postgres requires host, database, and user", ErrDatabaseConfigMissing)
		return
	}
	wait = true
	ul := BuildPostgresURL(user, pass, host, dbs)
	db, err := sql.Open("postgres", ul)
	if err != nil {
		errs = fmt.Errorf("open postgres database: %w", err)
		return
	}
	err = db.PingContext(ctx)
	if err != nil {
		_ = db.Close()
		errs = fmt.Errorf("ping postgres database: %w", err)
		return
	}
	defer func() { _ = db.Close() }()
	wait = false

	// Run migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logrus.Errorf("InitPostgresMigrate: could not start sql migration: %v", err)
		errs = fmt.Errorf("init postgres migration driver: %w", err)
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
		errs = fmt.Errorf("init postgres bindata source: %w", err)
		return
	}
	defer func() { _ = sc.Close() }()
	mgt, err := migrate.NewWithInstance(
		"bindata", sc,
		"postgres", driver)
	if err != nil {
		errs = fmt.Errorf("create postgres migrate instance: %w", err)
		return
	}
	defer func() { _, _ = mgt.Close() }()
	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		errs = fmt.Errorf("run postgres migration: %w", err)
		return
	}

	return false, ul, nil
}
