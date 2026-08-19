package database

import (
	"os"
	"strings"
	"testing"
)

// TestLoadMigrationsReadsEveryEmbeddedFile is the regression test for the
// defect where embed.FS paths were joined with path/filepath. Keys in an
// embed.FS are always slash-separated, so on Windows the join produced
// "migrations\001_create_tenants.up.sql" and every lookup missed, making the
// migrator fail at startup on the development machine while continuing to pass
// inside the Linux container.
//
// It needs no database, so it runs on every commit in CI on any platform.
func TestLoadMigrationsReadsEveryEmbeddedFile(t *testing.T) {
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations returned an error: %v", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		t.Fatalf("could not read the embedded migrations directory: %v", err)
	}

	want := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			want++
		}
	}

	if want == 0 {
		t.Fatal("no .up.sql files are embedded; the go:embed directive is not matching")
	}
	if len(migrations) != want {
		t.Fatalf("loaded %d migrations, want %d", len(migrations), want)
	}

	for _, m := range migrations {
		if strings.TrimSpace(m.sql) == "" {
			t.Errorf("migration %q loaded with empty SQL", m.version)
		}
	}
}

// TestLoadMigrationsDerivesVersionsInOrder pins the tracking key format and the
// ordering guarantee. The version is what schema_migrations stores, so a change
// in how it is derived would silently re-apply migrations against a database
// that already has them.
func TestLoadMigrationsDerivesVersionsInOrder(t *testing.T) {
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations returned an error: %v", err)
	}

	for _, m := range migrations {
		if strings.Contains(m.version, ".") {
			t.Errorf("version %q still carries a file extension", m.version)
		}
		if strings.ContainsAny(m.version, `/\`) {
			t.Errorf("version %q contains a path separator", m.version)
		}
	}

	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].version >= migrations[i].version {
			t.Errorf("migrations are not in ascending order: %q precedes %q",
				migrations[i-1].version, migrations[i].version)
		}
	}
}

// TestEveryUpMigrationHasADownMigration guards the rollback path. A missing
// down file is invisible until the moment someone needs it, which is the worst
// moment to discover it.
//
// This reads the source directory rather than migrationFS, because the embed
// directive deliberately captures only .up.sql; the down files never enter the
// binary and so cannot be checked through the embedded FS.
func TestEveryUpMigrationHasADownMigration(t *testing.T) {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		t.Fatalf("could not read the migrations source directory: %v", err)
	}

	ups := map[string]bool{}
	downs := map[string]bool{}
	for _, e := range entries {
		switch {
		case strings.HasSuffix(e.Name(), ".up.sql"):
			ups[strings.TrimSuffix(e.Name(), ".up.sql")] = true
		case strings.HasSuffix(e.Name(), ".down.sql"):
			downs[strings.TrimSuffix(e.Name(), ".down.sql")] = true
		}
	}

	if len(ups) == 0 {
		t.Fatal("no .up.sql files found in the migrations directory")
	}

	for name := range ups {
		if !downs[name] {
			t.Errorf("migration %q has no matching .down.sql", name)
		}
	}
	for name := range downs {
		if !ups[name] {
			t.Errorf("orphan down migration %q has no matching .up.sql", name)
		}
	}
}
