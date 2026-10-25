package migrates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gokins/gokins/comm"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/sirupsen/logrus"
)

// driverFactory creates a migrate database.Driver from an open *sql.DB.
type driverFactory func(db *sql.DB) (database.Driver, error)

// newMysqlDriver wraps the mysql migration driver constructor.
func newMysqlDriver(db *sql.DB) (database.Driver, error) {
	return mysql.WithInstance(db, &mysql.Config{})
}

// newPostgresDriver wraps the postgres migration driver constructor.
func newPostgresDriver(db *sql.DB) (database.Driver, error) {
	return postgres.WithInstance(db, &postgres.Config{})
}

// newSqlite3Driver wraps the sqlite3 migration driver constructor.
func newSqlite3Driver(db *sql.DB) (database.Driver, error) {
	return sqlite3.WithInstance(db, &sqlite3.Config{})
}

// filterAssetNames returns the asset file names (without prefix) for the given
// database driver. For example, filterAssetNames("mysql") returns
// ["1_init.sql", "2_add_users.sql"] from embedded assets like "mysql/1_init.sql".
func filterAssetNames(prefix string) []string {
	tms := comm.AssetNames()
	nms := make([]string, 0, len(tms)/3)
	pfx := prefix + "/"
	for _, v := range tms {
		if strings.HasPrefix(v, pfx) {
			nms = append(nms, strings.TrimPrefix(v, pfx))
		}
	}
	return nms
}

// runMigration is the shared core logic for all Up*Migrate functions.
// It opens a database, creates the migration driver, loads embedded migration
// assets, and runs the migrations.
//
// Parameters:
//   - dbDriver: the SQL driver name (e.g. "mysql", "postgres", "sqlite3")
//   - dbURL: the database connection string
//   - assetPrefix: the embedded asset directory prefix (e.g. "mysql")
//   - mkDriver: factory to create the migrate database.Driver
func runMigration(dbDriver, dbURL, assetPrefix string, mkDriver driverFactory) error {
	if dbURL == "" {
		return fmt.Errorf("%w: %s connection string is empty", ErrDatabaseConfigMissing, dbDriver)
	}
	db, err := sql.Open(dbDriver, dbURL)
	if err != nil {
		logrus.Errorf("%s open db error: %v", dbDriver, err)
		return fmt.Errorf("open %s database: %w", dbDriver, err)
	}
	defer func() { _ = db.Close() }()
	err = db.PingContext(context.Background())
	if err != nil {
		logrus.Errorf("%s ping failed: %v", dbDriver, err)
		return fmt.Errorf("ping %s database: %w", dbDriver, err)
	}

	// Create migration driver
	driver, err := mkDriver(db)
	if err != nil {
		logrus.Errorf("%s migration driver error: %v", dbDriver, err)
		return fmt.Errorf("init %s migration driver: %w", dbDriver, err)
	}
	defer func() { _ = driver.Close() }()

	// Build bindata source from embedded assets
	nms := filterAssetNames(assetPrefix)
	s := bindata.Resource(nms, func(name string) ([]byte, error) {
		return comm.Asset(assetPrefix + "/" + name)
	})
	sc, err := bindata.WithInstance(s)
	if err != nil {
		return fmt.Errorf("init %s bindata source: %w", dbDriver, err)
	}
	defer func() { _ = sc.Close() }()

	// Create and run migration
	mgt, err := migrate.NewWithInstance("bindata", sc, dbDriver, driver)
	if err != nil {
		return fmt.Errorf("create %s migrate instance: %w", dbDriver, err)
	}
	defer func() { _, _ = mgt.Close() }()
	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		return fmt.Errorf("run %s migration: %w", dbDriver, err)
	}

	return nil
}

func UpMysqlMigrate(ul string) error {
	return runMigration("mysql", ul, "mysql", newMysqlDriver)
}

func UpPostgresMigrate(ul string) error {
	return runMigration("postgres", ul, "postgres", newPostgresDriver)
}

func UpSqliteMigrate(ul string) error {
	return runMigration("sqlite3", ul, "sqlite", newSqlite3Driver)
}
