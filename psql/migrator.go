package psql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// migrateLogger implements the logging interface for database migrations
type migrateLogger struct {
	logger *zerolog.Logger
}

// Printf passes the log message and arguments to our logger
func (l *migrateLogger) Printf(format string, v ...interface{}) {
	l.logger.Printf(strings.TrimSpace(format), v...)
}

// Verbose returns true so we can log what the migrator is doing
func (l *migrateLogger) Verbose() bool {
	return true
}

type Version struct {
	Version uint
	Dirty   bool
}

// Migrator
type Migrator struct {
	Db      *sql.DB
	Migrate *migrate.Migrate
}

// NewMigrator returns a new database migrator for the given connection using a file path for the migrations source
func NewMigrator(ctx context.Context, db *sql.DB, source string) (*Migrator, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return &Migrator{}, fmt.Errorf("cannot get postgres driver: %v", err)
	}
	// Create migrator using file path as migrations source
	m, err := migrate.NewWithDatabaseInstance("file://"+source, "postgres", driver)
	if err != nil {
		return &Migrator{}, fmt.Errorf("failed to get migrations instance: %v", err)
	}
	// Assign db logger to migrator
	m.Log = &migrateLogger{log.Ctx(ctx)}
	return &Migrator{
		Db:      db,
		Migrate: m,
	}, nil
}

// MigrateUp run up migrations
func (m *Migrator) Up(steps int) error {
	defer m.Db.Close()
	if steps == 0 {
		return m.Migrate.Up()
	}
	return m.Migrate.Steps(steps)
}

// MigrateDown run down migrations
func (m *Migrator) Down() error {
	defer m.Db.Close()
	return m.Migrate.Down()
}

// MigrateForce force version
func (m *Migrator) Force(v int) error {
	defer m.Db.Close()
	return m.Migrate.Force(v)
}

// MigrateVersion prints the current migration version
func (m *Migrator) Version() (*Version, error) {
	defer m.Db.Close()
	version, dirty, err := m.Migrate.Version()
	if err != nil {
		return nil, err
	}
	return &Version{
		Version: version,
		Dirty:   dirty,
	}, nil
}
