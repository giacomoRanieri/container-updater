package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"container-updater/backend/internal/logger"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Enable WAL mode and foreign keys
	if _, err := DB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	if _, err := DB.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := createSchemas(); err != nil {
		return fmt.Errorf("failed to run database schemas: %w", err)
	}

	logger.Log.Info("SQLite Database initialized and schemas verified", "path", dbPath)
	return nil
}

func createSchemas() error {
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS workloads (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			namespace_project TEXT NOT NULL,
			orchestrator_type TEXT NOT NULL,
			current_image TEXT NOT NULL,
			current_digest TEXT NOT NULL,
			new_image TEXT,
			new_digest TEXT,
			update_status TEXT NOT NULL DEFAULT 'up_to_date',
			last_checked_at DATETIME,
			last_updated_at DATETIME
		);`,

		`CREATE TABLE IF NOT EXISTS update_jobs (
			id TEXT PRIMARY KEY,
			workload_id TEXT NOT NULL,
			status TEXT NOT NULL,
			error_message TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(workload_id) REFERENCES workloads(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS notification_services (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			apprise_url TEXT NOT NULL,
			is_enabled BOOLEAN NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS registry_credentials (
			id TEXT PRIMARY KEY,
			server_address TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, schema := range schemas {
		if _, err := DB.Exec(schema); err != nil {
			return err
		}
	}

	return nil
}
