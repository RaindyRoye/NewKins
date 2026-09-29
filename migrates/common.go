package migrates

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gokins/gokins/comm"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/sirupsen/logrus"
)

// driverConfig holds the configuration for a database migration driver.
type driverConfig struct {
	// name is the driver name used by the migrate library (e.g., "mysql", "postgres", "sqlite3").
	name string
	// dsn is the database connection string. Used only by runMigration; ignored by runMigrationOnDB.
	dsn string
	// assetPrefix is the bindata asset prefix (e.g., "mysql", "postgres", "sqlite").
	assetPrefix string
	// driverFunc creates a migration driver instance wrapping the given sql.DB.
	driverFunc func(db *sql.DB) (database.Driver, error)
}

// runMigration opens a fresh DB connection using cfg.dsn and runs all pending migrations.
// It returns an error if any step fails.
func runMigration(cfg *driverConfig) error {
	if cfg.dsn == "" {
		return fmt.Errorf("%w: %s connection string is empty", ErrDatabaseConfigMissing, cfg.name)
	}

	// Open database connection
	db, err := sql.Open(cfg.name, cfg.dsn)
	if err != nil {
		logrus.Errorf("%s open db error: %v", cfg.name, err)
		return fmt.Errorf("open %s database: %w", cfg.name, err)
	}
	defer func() { _ = db.Close() }()

	// Verify connection
	if err := db.Ping(); err != nil {
		logrus.Errorf("%s ping failed: %v", cfg.name, err)
		return fmt.Errorf("ping %s database: %w", cfg.name, err)
	}

	return runMigrationOnDB(db, cfg)
}

// runMigrationOnDB runs all pending migrations on a pre-opened sql.DB.
// The caller retains ownership of the db handle and is responsible for closing it.
// This helper is used by the Init* functions that need to set up the DB
// (e.g., create the database) before running migrations.
func runMigrationOnDB(db *sql.DB, cfg *driverConfig) error {
	if db == nil {
		return fmt.Errorf("%w: %s database connection is nil", ErrDatabaseConfigMissing, cfg.name)
	}

	// Create migration driver
	driver, err := cfg.driverFunc(db)
	if err != nil {
		logrus.Errorf("%s migration driver error: %v", cfg.name, err)
		return fmt.Errorf("init %s migration driver: %w", cfg.name, err)
	}
	defer func() { _ = driver.Close() }()

	// Collect migration assets
	var nms []string
	tms := comm.AssetNames()
	for _, v := range tms {
		if strings.HasPrefix(v, cfg.assetPrefix) {
			nms = append(nms, strings.Replace(v, cfg.assetPrefix+"/", "", 1))
		}
	}

	// Create bindata source
	s := bindata.Resource(nms, func(name string) ([]byte, error) {
		return comm.Asset(cfg.assetPrefix + "/" + name)
	})
	sc, err := bindata.WithInstance(s)
	if err != nil {
		return fmt.Errorf("init %s bindata source: %w", cfg.name, err)
	}
	defer func() { _ = sc.Close() }()

	// Create migrate instance
	mgt, err := migrate.NewWithInstance("bindata", sc, cfg.name, driver)
	if err != nil {
		return fmt.Errorf("create %s migrate instance: %w", cfg.name, err)
	}
	defer func() { _, _ = mgt.Close() }()

	// Run migrations
	if err := mgt.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		return fmt.Errorf("run %s migration: %w", cfg.name, err)
	}

	return nil
}
