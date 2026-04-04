package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations applies pending SQL migrations from the given directory.
// It skips rollback files (*_rollback.sql).
// Migrations are tracked in the schema_migrations table.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	// Ensure schema_migrations table exists
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	// Get already-applied migrations
	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return err
		}
		applied[v] = true
	}

	// Read migration files
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		if strings.HasSuffix(name, "_rollback.sql") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	// Apply pending migrations
	for _, file := range files {
		version := strings.TrimSuffix(file, ".sql")
		if applied[version] {
			continue
		}

		// Clean the filename to prevent directory traversal (e.g., "../secret.sql")
		cleanFile := filepath.Base(file)
		content, err := os.ReadFile(filepath.Join(migrationsDir, cleanFile)) // #nosec G304 -- path is sanitized via filepath.Base
		if err != nil {
			return fmt.Errorf("read migration %s: %w", cleanFile, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", file, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log.Printf("Warning: Failed to rollback transaction for migration %s: %v", file, rbErr)
			}
			return fmt.Errorf("execute migration %s: %w", file, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, $3)",
			version, file, time.Now(),
		); err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log.Printf("Warning: Failed to rollback transaction for migration %s: %v", file, rbErr)
			}
			return fmt.Errorf("record migration %s: %w", file, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", file, err)
		}

		log.Printf("Applied migration: %s", file)
	}

	log.Println("Migration check completed!")
	return nil
}
