package testutil

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("skipping integration test: TEST_DATABASE_URL or DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping test db: %v", err)
	}

	for _, migrationPath := range migrationFilePaths(t) {
		sqlBytes, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("read migration file %s: %v", migrationPath, err)
		}
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("apply schema %s: %v", migrationPath, err)
		}
	}

	cleanupDB(t, db)
	t.Cleanup(func() {
		cleanupDB(t, db)
		_ = db.Close()
	})

	return db
}

func cleanupDB(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statements := []string{
		`TRUNCATE TABLE audit_logs, webhook_deliveries, user_sessions, tenant_users, users, webhooks, usage, subscriptions, api_keys, messages, whatsapp_sessions, tenants RESTART IDENTITY CASCADE`,
		`DO $$
		DECLARE
			r RECORD;
		BEGIN
			FOR r IN (
				SELECT tablename
				FROM pg_tables
				WHERE schemaname = 'public' AND tablename LIKE 'whatsmeow_%'
			) LOOP
				EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;`,
	}

	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("cleanup test db: %v", err)
		}
	}
}

func migrationFilePaths(t *testing.T) []string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve test helper path")
	}
	base := filepath.Dir(filename)
	matches, err := filepath.Glob(filepath.Join(base, "..", "..", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("list migration files: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no migration files found")
	}
	sort.Strings(matches)
	for i := range matches {
		matches[i] = filepath.Clean(matches[i])
	}
	return matches
}

func MustHeaderContains(t *testing.T, value, want string) {
	t.Helper()
	if !strings.Contains(value, want) {
		t.Fatalf("expected %q to contain %q", value, want)
	}
}
