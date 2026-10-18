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
	"github.com/golang-migrate/migrate/v4/source"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/sirupsen/logrus"
)

// driverFactory creates a database.Driver from an open *sql.DB.
// Each supported backend (mysql, postgres, sqlite3) provides its own factory.
type driverFactory func(db *sql.DB) (database.Driver, error)

// filterAssetNames returns migration file names for the given prefix,
// stripping the "prefix/" directory component from each asset name.
func filterAssetNames(prefix string) []string {
	var nms []string
	for _, v := range comm.AssetNames() {
		if strings.HasPrefix(v, prefix) {
			nms = append(nms, strings.Replace(v, prefix+"/", "", 1))
		}
	}
	return nms
}

// newBindataSource creates a bindata migration source filtered to the given prefix.
func newBindataSource(prefix string) (source.Driver, error) {
	nms := filterAssetNames(prefix)
	s := bindata.Resource(nms, func(name string) ([]byte, error) {
		return comm.Asset(prefix + "/" + name)
	})
	sc, err := bindata.WithInstance(s)
	if err != nil {
		return nil, fmt.Errorf("init %s bindata source: %w", prefix, err)
	}
	return sc, nil
}

// runMigrateUp opens a database, creates the migration driver, and runs
// migrations upward. It handles all resource cleanup via defers.
//
// Parameters:
//   - driverName: the Go SQL driver name (e.g., "mysql", "postgres", "sqlite3")
//   - url: the connection string or file path
//   - dbLabel: human-readable label for error messages (e.g., "mysql", "postgres", "sqlite")
//   - assetPrefix: the embedded asset directory prefix (e.g., "mysql", "postgres", "sqlite")
//   - makeDriver: factory that creates the golang-migrate database.Driver from *sql.DB
func runMigrateUp(driverName, url, dbLabel, assetPfx string, makeDriver driverFactory) error {
	if url == "" {
		return fmt.Errorf("%w: %s connection string is empty", ErrDatabaseConfigMissing, dbLabel)
	}
	db, err := sql.Open(driverName, url)
	if err != nil {
		logrus.Errorf("%s open db error: %v", dbLabel, err)
		return fmt.Errorf("open %s database: %w", dbLabel, err)
	}
	defer func() { _ = db.Close() }()

	err = db.PingContext(context.Background())
	if err != nil {
		logrus.Errorf("%s ping failed: %v", dbLabel, err)
		return fmt.Errorf("ping %s database: %w", dbLabel, err)
	}

	// Run migrations
	drv, err := makeDriver(db)
	if err != nil {
		logrus.Errorf("%s migration driver error: %v", dbLabel, err)
		return fmt.Errorf("init %s migration driver: %w", dbLabel, err)
	}
	defer func() { _ = drv.Close() }()

	sc, err := newBindataSource(assetPfx)
	if err != nil {
		return err
	}
	defer func() { _ = sc.Close() }()

	mgt, err := migrate.NewWithInstance(
		"bindata", sc,
		dbLabel, drv)
	if err != nil {
		return fmt.Errorf("create %s migrate instance: %w", dbLabel, err)
	}
	defer func() { _, _ = mgt.Close() }()

	err = mgt.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_ = mgt.Down()
		return fmt.Errorf("run %s migration: %w", dbLabel, err)
	}

	return nil
}
