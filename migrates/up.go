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

// newDriver returns a migrate database driver for the given driver name and
// open database connection. It consolidates the per-database driver
// construction that was previously duplicated across Up*Migrate functions.
func newDriver(driverName string, db *sql.DB) (database.Driver, error) {
	switch driverName {
	case "mysql":
		return mysql.WithInstance(db, &mysql.Config{})
	case "postgres":
		return postgres.WithInstance(db, &postgres.Config{})
	case "sqlite3":
		return sqlite3.WithInstance(db, &sqlite3.Config{})
	default:
		return nil, fmt.Errorf("%w: unsupported driver %q", ErrDatabaseConfigMissing, driverName)
	}
}

// runMigrate is the shared core for Up*Migrate functions. It opens the
// database, builds the bindata source, creates a migrate instance and runs
// migrations up. On non-trivial errors it rolls back (Down) before returning.
func runMigrate(ul, driverName, assetPrefix string) error {
	if ul == "" {
		return fmt.Errorf("%w: %s connection string is empty", ErrDatabaseConfigMissing, driverName)
	}
	db, err := sql.Open(driverName, ul)
	if err != nil {
		logrus.Errorf("%s open db error: %v", driverName, err)
		return fmt.Errorf("open %s database: %w", driverName, err)
	}
	defer func() { _ = db.Close() }()

	err = db.PingContext(context.Background())
	if err != nil {
		logrus.Errorf("%s ping failed: %v", driverName, err)
		return fmt.Errorf("ping %s database: %w", driverName, err)
	}

	driver, err := newDriver(driverName, db)
	if err != nil {
		logrus.Errorf("%s migration driver error: %v", driverName, err)
		return fmt.Errorf("init %s migration driver: %w", driverName, err)
	}
	defer func() { _ = driver.Close() }()

	var nms []string
	tms := comm.AssetNames()
	for _, v := range tms {
		if strings.HasPrefix(v, assetPrefix) {
			nms = append(nms, strings.Replace(v, assetPrefix+"/", "", 1))
		}
	}
	s := bindata.Resource(nms, func(name string) ([]byte, error) {
		return comm.Asset(assetPrefix + "/" + name)
	})
	sc, err := bindata.WithInstance(s)
	if err != nil {
		return fmt.Errorf("init %s bindata source: %w", driverName, err)
	}
	defer func() { _ = sc.Close() }()

	mgt, err := migrate.NewWithInstance("bindata", sc, driverName, driver)
	if err != nil {
		return fmt.Errorf("create %s migrate instance: %w", driverName, err)
	}
	defer func() { _, _ = mgt.Close() }()

	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		return fmt.Errorf("run %s migration: %w", driverName, err)
	}
	return nil
}

func UpMysqlMigrate(ul string) error {
	return runMigrate(ul, "mysql", "mysql")
}

func UpPostgresMigrate(ul string) error {
	return runMigrate(ul, "postgres", "postgres")
}

func UpSqliteMigrate(ul string) error {
	return runMigrate(ul, "sqlite3", "sqlite")
}
