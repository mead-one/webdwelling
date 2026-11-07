package database

import (
	"database/sql"
	"io/fs"
	"strconv"
	"testing"
)

// Open a fresh in-memory database as DB, without running migrations.
//
// Each test gets its own empty database. Tests using this must not call
// t.Parallel(), because they share the package-level DB variable.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file::memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open in-memory database: %v", err)
	}
	// Every connection to :memory: gets its own empty database, so keep the
	// pool to one connection or tables will seem to vanish between queries
	db.SetMaxOpenConns(1)

	previous := DB
	DB = db
	t.Cleanup(func() {
		db.Close()
		DB = previous
	})

	return db
}

// Open a fresh in-memory database as DB, with all migrations applied
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db := openTestDB(t)
	if err := migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return db
}

// The schema version the embedded migrations should bring a database up to
func latestVersion(t *testing.T) int {
	t.Helper()

	files, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no embedded migrations found")
	}

	return len(files)
}

func schemaVersion(t *testing.T, db *sql.DB) int {
	t.Helper()

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}

	return version
}

func setSchemaVersion(t *testing.T, db *sql.DB, version int) {
	t.Helper()

	// PRAGMA doesn't accept bound parameters; version is an int, so this is safe
	if _, err := db.Exec("PRAGMA user_version = " + strconv.Itoa(version)); err != nil {
		t.Fatalf("set user_version: %v", err)
	}
}

func TestMigrateFreshDatabase(t *testing.T) {
	db := setupTestDB(t)

	if got, want := schemaVersion(t, db), latestVersion(t); got != want {
		t.Errorf("user_version = %d, want %d", got, want)
	}

	for _, table := range []string{"users", "bookmark_folders", "bookmarks"} {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %s missing after migration: %v", table, err)
		}
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := setupTestDB(t)

	if err := migrate(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	if got, want := schemaVersion(t, db), latestVersion(t); got != want {
		t.Errorf("user_version after second migrate = %d, want %d", got, want)
	}
}

func TestMigrateRefusesNewerDatabase(t *testing.T) {
	db := openTestDB(t)
	newer := latestVersion(t) + 1
	setSchemaVersion(t, db, newer)

	if err := migrate(); err == nil {
		t.Fatal("migrate succeeded on a database newer than the latest migration, want error")
	}

	if got := schemaVersion(t, db); got != newer {
		t.Errorf("user_version = %d after refused migrate, want it left at %d", got, newer)
	}
}
