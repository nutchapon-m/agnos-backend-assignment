package migrate

import (
	"database/sql"
	"embed"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed sql/*.sql
var migrationFS embed.FS

func Run(db *sql.DB, action string) error {
	d, err := iofs.New(migrationFS, "sql")
	if err != nil {
		return err
	}

	driver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", d, "pgx5", driver)
	if err != nil {
		return err
	}

	switch action {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return err
		}
		return nil
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			return err
		}
		return nil
	default:
		return nil
	}
}
