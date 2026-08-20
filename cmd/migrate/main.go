package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	core "github.com/MrF1ow/go-core"
	"github.com/MrF1ow/go-core/internal/database"
)

func main() {
	status := flag.Bool("status", false, "list applied migrations and exit")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	port := 5432
	if v := os.Getenv("DB_PORT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("DB_PORT must be an integer: %v", err)
		}
		port = n
	}

	pool, err := database.ConnectPgx(
		envOr("DB_HOST", "localhost"),
		port,
		envOr("DB_USER", "postgres"),
		os.Getenv("DB_PASSWORD"),
		envOr("DB_NAME", "go_core"),
		envOr("DB_SSLMODE", "disable"),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if *status {
		if err := printStatus(ctx, pool); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := core.RunCoreMigrations(ctx, pool); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func printStatus(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `
		SELECT version, name, applied_at
		FROM schema_migrations
		ORDER BY version`)
	if err != nil {
		return fmt.Errorf("query schema_migrations (run make migrate-up first): %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var version, name string
		var appliedAt time.Time
		if err := rows.Scan(&version, &name, &appliedAt); err != nil {
			return err
		}
		fmt.Printf("%s\t%s\t%s\n", version, name, appliedAt.UTC().Format(time.RFC3339))
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count == 0 {
		fmt.Println("No migrations applied.")
	}
	return nil
}
