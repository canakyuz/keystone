package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"nexspaces-api/internal/config"
)

type DB struct {
	*sql.DB
}

func NewPostgreSQL(cfg *config.DatabaseConfig) (*DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Println("✅ Connected to PostgreSQL database")
	return &DB{db}, nil
}

func (db *DB) Close() error {
	log.Println("🔌 Closing database connection")
	return db.DB.Close()
}

// Health check for database
func (db *DB) Health() error {
	return db.Ping()
}