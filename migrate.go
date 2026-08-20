package core

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunCoreMigrations applies all go-core built-in migrations from the embedded
// migrations directory. Consumers should call this before RunMigrations to
// ensure the core schema (users, sessions, etc.) exists.
func RunCoreMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	return runMigrationsFS(ctx, pool, coreMigrationsFS, "migrations")
}

// RunMigrations applies pending SQL migrations from the given directory on disk.
// It skips rollback files (*_rollback.sql) and down migration files (*.down.sql).
// Migrations are tracked in the schema_migrations table.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	return runMigrationsFS(ctx, pool, os.DirFS(migrationsDir), ".")
}

func runMigrationsFS(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS, dir string) error {
	// Ensure schema_migrations table exists
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			version VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			execution_time_ms INTEGER,
			success BOOLEAN NOT NULL DEFAULT true,
			error_message TEXT,
			checksum VARCHAR(64)
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
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if isForwardMigration(e.Name()) {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	// Apply pending migrations
	for _, file := range files {
		version := strings.TrimSuffix(file, ".sql")
		if applied[version] {
			continue
		}

		path := file
		if dir != "." {
			path = dir + "/" + file
		}

		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
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
			"INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, $3) ON CONFLICT (version) DO NOTHING",
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

func isForwardMigration(name string) bool {
	if !strings.HasSuffix(name, ".sql") {
		return false
	}
	if strings.HasSuffix(name, "_rollback.sql") || strings.HasSuffix(name, ".down.sql") {
		return false
	}
	return true
}
