package migrates

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/gokins/gokins/comm"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/sirupsen/logrus"
)

// InitMysqlMigrate initializes the MySQL database, creating it if necessary,
// then runs all pending migrations.
// Returns wait=true if the database server is reachable but the DB doesn't exist yet
// and needs creation; in that case the caller should retry.
func InitMysqlMigrate(host, dbs, user, pass string) (wait bool, rtul string, errs error) {
	wait = false
	if host == "" || dbs == "" || user == "" {
		errs = fmt.Errorf("%w: mysql requires host, database, and user", ErrDatabaseConfigMissing)
		return
	}
	wait = true
	ul := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&multiStatements=true",
		user, pass, host, dbs)
	db, err := sql.Open("mysql", ul)
	if err != nil {
		errs = fmt.Errorf("open mysql database: %w", err)
		return
	}
	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		// Database doesn't exist yet; try connecting without DB name and create it.
		_ = db.Close()
		uls := fmt.Sprintf("%s:%s@tcp(%s)/?parseTime=true&multiStatements=true",
			user, pass, host)
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

	// Delegate migration execution to the common helper.
	if err := runMigrationOnDB(db, &driverConfig{
		name:        "mysql",
		assetPrefix: "mysql",
		driverFunc: func(d *sql.DB) (database.Driver, error) {
			return mysql.WithInstance(d, &mysql.Config{})
		},
	}); err != nil {
		errs = err
		return
	}

	return false, ul, nil
}

// InitSqliteMigrate initializes the SQLite database and runs all pending migrations.
// The database file is stored at <WorkPath>/db.dat.
func InitSqliteMigrate() (rtul string, errs error) {
	ul := filepath.Join(comm.WorkPath, "db.dat")
	db, err := sql.Open("sqlite3", ul)
	if err != nil {
		errs = fmt.Errorf("open sqlite database: %w", err)
		return
	}
	defer func() { _ = db.Close() }()

	// Delegate migration execution to the common helper.
	if err := runMigrationOnDB(db, &driverConfig{
		name:        "sqlite3",
		assetPrefix: "sqlite",
		driverFunc: func(d *sql.DB) (database.Driver, error) {
			return sqlite3.WithInstance(d, &sqlite3.Config{})
		},
	}); err != nil {
		errs = err
		return
	}

	return ul, nil
}

// InitPostgresMigrate initializes the Postgres database and runs all pending migrations.
func InitPostgresMigrate(host, dbs, user, pass string) (wait bool, rtul string, errs error) {
	wait = false
	if host == "" || dbs == "" || user == "" {
		errs = fmt.Errorf("%w: postgres requires host, database, and user", ErrDatabaseConfigMissing)
		return
	}
	wait = true
	// Build connection URL: postgres://user:***@host/database?sslmode=disable
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

	// Delegate migration execution to the common helper.
	if err := runMigrationOnDB(db, &driverConfig{
		name:        "postgres",
		assetPrefix: "postgres",
		driverFunc: func(d *sql.DB) (database.Driver, error) {
			return postgres.WithInstance(d, &postgres.Config{})
		},
	}); err != nil {
		errs = err
		return
	}

	return false, ul, nil
}
