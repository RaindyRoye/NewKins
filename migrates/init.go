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
	"github.com/golang-migrate/migrate/v4/source"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/sirupsen/logrus"
)

// filterAssetNames returns asset names from comm.AssetNames() that start with
// the given prefix, stripping the prefix and trailing slash from each name.
// For example, with prefix "mysql" and assets ["mysql/001_init.up.sql", "sqlite/001_init.up.sql"],
// it returns ["001_init.up.sql"].
func filterAssetNames(prefix string) []string {
	var names []string
	for _, v := range comm.AssetNames() {
		if strings.HasPrefix(v, prefix) {
			names = append(names, strings.Replace(v, prefix+"/", "", 1))
		}
	}
	return names
}

// newBindataSource creates a bindata source instance for the given database prefix.
// It reads migration SQL files embedded in the binary via comm.Asset.
// Returns a source.Driver suitable for use with migrate.NewWithInstance.
func newBindataSource(prefix string) (source.Driver, error) {
	names := filterAssetNames(prefix)
	s := bindata.Resource(names, func(name string) ([]byte, error) {
		return comm.Asset(prefix + "/" + name)
	})
	return bindata.WithInstance(s)
}

// runMigration executes an Up migration and handles the ErrNoChange sentinel.
// If the migration fails with a non-sentinel error, it attempts to roll back.
func runMigration(mgt *migrate.Migrate) error {
	err := mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		return fmt.Errorf("run migration: %w", err)
	}
	return nil
}

func InitMysqlMigrate(host, dbs, user, pass string) (wait bool, rtul string, errs error) {
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
	sc, err := newBindataSource("mysql")
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
	err = runMigration(mgt)
	if err != nil {
		errs = fmt.Errorf("run mysql migration: %w", err)
		return
	}

	return false, ul, nil
}

func InitSqliteMigrate() (rtul string, errs error) {
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
	sc, err := newBindataSource("sqlite")
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
	err = runMigration(mgt)
	if err != nil {
		errs = fmt.Errorf("run sqlite migration: %w", err)
		return
	}

	return ul, nil
}

func InitPostgresMigrate(host, dbs, user, pass string) (wait bool, rtul string, errs error) {
	wait = false
	if host == "" || dbs == "" || user == "" {
		errs = fmt.Errorf("%w: postgres requires host, database, and user", ErrDatabaseConfigMissing)
		return
	}
	wait = true
	ul := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, pass, host, dbs)
	db, err := sql.Open("postgres", ul)
	if err != nil {
		errs = fmt.Errorf("open postgres database: %w", err)
		return
	}
	err = db.PingContext(context.Background())
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
	sc, err := newBindataSource("postgres")
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
	err = runMigration(mgt)
	if err != nil {
		errs = fmt.Errorf("run postgres migration: %w", err)
		return
	}

	return false, ul, nil
}
