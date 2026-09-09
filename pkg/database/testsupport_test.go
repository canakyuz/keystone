package database

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// openTestDB opens a real PostgreSQL connection. If the server is unreachable it
// returns nil and the calling test skips itself, so the suite also runs outside CI.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOr("TEST_DB_HOST", "localhost"),
		envOr("TEST_DB_PORT", "5432"),
		envOr("TEST_DB_USER", "postgres"),
		envOr("TEST_DB_PASSWORD", "postgres"),
		envOr("TEST_DB_ADMIN_NAME", "postgres"),
		envOr("TEST_DB_SSLMODE", "disable"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil
	}

	t.Cleanup(func() { db.Close() })
	return db
}

func mustExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	_, err := db.Exec(query)
	require.NoErrorf(t, err, "setup query failed: %s", query)
}

// openBenchDB opens a real connection for benchmarks.
func openBenchDB(b *testing.B) *sql.DB {
	b.Helper()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOr("TEST_DB_HOST", "localhost"),
		envOr("TEST_DB_PORT", "5432"),
		envOr("TEST_DB_USER", "postgres"),
		envOr("TEST_DB_PASSWORD", "postgres"),
		envOr("TEST_DB_ADMIN_NAME", "postgres"),
		envOr("TEST_DB_SSLMODE", "disable"),
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil
	}

	b.Cleanup(func() { db.Close() })
	return db
}
