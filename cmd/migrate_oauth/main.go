package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	ctx := context.Background()

	port := 5432
	if v := os.Getenv("DB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		os.Getenv("DB_HOST"),
		port,
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Default App ID from migration 20260105_add_multi_tenancy.sql
	appID := "00000000-0000-0000-0000-000000000001"

	// Ensure tenant exists
	_, err = pool.Exec(ctx, `
		INSERT INTO tenants (id, name, created_at, updated_at)
		VALUES ($1, 'Default Tenant', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`,
		appID,
	)
	if err != nil {
		log.Fatalf("Failed to ensure tenant: %v", err)
	}

	// Ensure app exists
	_, err = pool.Exec(ctx, `
		INSERT INTO applications (id, tenant_id, name, description, created_at, updated_at)
		VALUES ($1, $1, 'Default App', 'Created by migration script', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`,
		appID,
	)
	if err != nil {
		log.Fatalf("Failed to ensure application: %v", err)
	}

	// Fetch app name for logging
	var appName string
	err = pool.QueryRow(ctx, `SELECT name FROM applications WHERE id = $1`, appID).Scan(&appName)
	if err != nil {
		log.Fatalf("Failed to fetch application: %v", err)
	}
	log.Printf("Using App: %s (%s)", appName, appID)

	// Upsert OAuth provider configs
	providers := []struct {
		Name        string
		EnvID       string
		EnvSecret   string
		EnvRedirect string
	}{
		{"google", "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "GOOGLE_REDIRECT_URL"},
		{"facebook", "FACEBOOK_CLIENT_ID", "FACEBOOK_CLIENT_SECRET", "FACEBOOK_REDIRECT_URL"},
		{"github", "GITHUB_CLIENT_ID", "GITHUB_CLIENT_SECRET", "GITHUB_REDIRECT_URL"},
	}

	for _, p := range providers {
		clientID := os.Getenv(p.EnvID)
		clientSecret := os.Getenv(p.EnvSecret)
		redirectURL := os.Getenv(p.EnvRedirect)

		if clientID == "" || clientSecret == "" {
			log.Printf("Skipping %s: missing ID or Secret in env", p.Name)
			continue
		}

		tag, err := pool.Exec(ctx, `
			INSERT INTO oauth_provider_configs (app_id, provider, client_id, client_secret, redirect_url, is_enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())
			ON CONFLICT (app_id, provider) DO UPDATE SET
				client_id = EXCLUDED.client_id,
				client_secret = EXCLUDED.client_secret,
				redirect_url = EXCLUDED.redirect_url,
				is_enabled = true,
				updated_at = NOW()`,
			appID, p.Name, clientID, clientSecret, redirectURL,
		)
		if err != nil {
			log.Printf("Failed to upsert %s config: %v", p.Name, err)
			continue
		}

		if tag.RowsAffected() > 0 {
			log.Printf("Upserted %s config", p.Name)
		}
	}

	log.Println("Migration completed.")
}
