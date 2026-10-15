package migrates

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
)

func UpMysqlMigrate(ul string) error {
	return runMigrateUp("mysql", ul, "mysql", "mysql", func(db *sql.DB) (database.Driver, error) {
		return mysql.WithInstance(db, &mysql.Config{})
	})
}

func UpPostgresMigrate(ul string) error {
	return runMigrateUp("postgres", ul, "postgres", "postgres", func(db *sql.DB) (database.Driver, error) {
		return postgres.WithInstance(db, &postgres.Config{})
	})
}

func UpSqliteMigrate(ul string) error {
	return runMigrateUp("sqlite3", ul, "sqlite", "sqlite", func(db *sql.DB) (database.Driver, error) {
		return sqlite3.WithInstance(db, &sqlite3.Config{})
	})
}
