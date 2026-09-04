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

// openTestDB, gerçek bir PostgreSQL bağlantısı açar. Sunucu erişilemiyorsa nil
// döner ve çağıran test kendini atlar; CI dışında da çalışabilsin diye.
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
	require.NoErrorf(t, err, "hazırlık sorgusu başarısız: %s", query)
}

// openBenchDB, benchmark'lar için gerçek bağlantı açar.
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
