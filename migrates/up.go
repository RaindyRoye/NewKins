package migrates

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
)

// UpMysqlMigrate runs migrations for an existing MySQL database.
func UpMysqlMigrate(ul string) error {
	return runMigration(&driverConfig{
		name:        "mysql",
		dsn:         ul,
		assetPrefix: "mysql",
		driverFunc: func(db *sql.DB) (database.Driver, error) {
			return mysql.WithInstance(db, &mysql.Config{})
		},
	})
}

// UpPostgresMigrate runs migrations for an existing Postgres database.
func UpPostgresMigrate(ul string) error {
	return runMigration(&driverConfig{
		name:        "postgres",
		dsn:         ul,
		assetPrefix: "postgres",
		driverFunc: func(db *sql.DB) (database.Driver, error) {
			return postgres.WithInstance(db, &postgres.Config{})
		},
	})
}

// UpSqliteMigrate runs migrations for an existing SQLite database.
func UpSqliteMigrate(ul string) error {
	return runMigration(&driverConfig{
		name:        "sqlite3",
		dsn:         ul,
		assetPrefix: "sqlite",
		driverFunc: func(db *sql.DB) (database.Driver, error) {
			return sqlite3.WithInstance(db, &sqlite3.Config{})
		},
	})
}
