package psqltest

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"go.soon.build/kit/psql"
)

func createTestDb(name string, db *sql.DB) error {
	defer db.Close()
	_, err := db.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, name))
	if err != nil {
		return fmt.Errorf("error dropping existing test database: %v", err)
	}
	_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, name))
	if err != nil {
		return fmt.Errorf("error creating test database: %v", err)
	}
	return nil
}

// Open connects to the default database
// then creates a new database from the test name
func Open(ctx context.Context, t *testing.T, dbCfg psql.Config) *sql.DB {
	// Create test database
	db, err := psql.Open(ctx, dbCfg)
	if err != nil {
		t.Fatalf("err opening test db: %v", err)
	}
	testDbName := t.Name()
	t.Logf("creating test db: %s", testDbName)
	err = createTestDb(testDbName, db)
	if err != nil {
		t.Fatal(err)
	}
	// Open test database
	dbCfg.DBName = testDbName
	tdb, err := psql.Open(ctx, dbCfg)
	if err != nil {
		t.Fatalf("err opening test db: %v", err)
	}
	return tdb
}

// OpenAndMigrate opens a new test database and applies migrations
func OpenAndMigrate(ctx context.Context, t *testing.T, dbCfg psql.Config, migrationsSource string) *sql.DB {
	tdb := Open(ctx, t, dbCfg)
	// Run migrations
	m, err := psql.NewMigrator(ctx, tdb, migrationsSource)
	if err != nil {
		t.Fatalf("migrator err: %v", err)
	}
	t.Logf("running migrations against test db: %s", t.Name())
	if err := m.Migrate.Up(); err != nil {
		t.Fatal(err)
	}
	return tdb
}

// DropMigrations drops the migrations from the test database
func DropMigrations(ctx context.Context, t *testing.T, tdb *sql.DB, migrationsSource string) {
	m, err := psql.NewMigrator(ctx, tdb, migrationsSource)
	defer tdb.Close()
	if err != nil {
		t.Fatalf("migrator err: %v", err)
	}
	t.Logf("downing migrations against test db: %s", t.Name())
	if err := m.Migrate.Down(); err != nil {
		t.Fatal(err)
	}
}
