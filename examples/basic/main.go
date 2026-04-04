package main

import (
	"log"

	"github.com/gin-gonic/gin"

	core "github.com/JedidiahDigital/go-core"
	"github.com/JedidiahDigital/go-core/app"
)

// This is a minimal example showing how to use go-core as an auth module
// in your own application. Customize the Config fields to match your environment.
//
// Prerequisites: PostgreSQL running with the migrations applied.
// Optionally: Redis for production caching (nil = in-memory fallback).
//
// Run: go run ./examples/basic

func main() {
	cfg := core.DefaultConfig()

	// Required: database connection
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	cfg.Database.DBName = "go_core"
	cfg.Database.User = "postgres"
	cfg.Database.Password = "postgres"

	// Required: JWT signing secret (min 32 characters)
	cfg.JWT.Secret = "change-me-to-a-real-secret-at-least-32-chars"

	// Optional: set nil to use in-memory cache (fine for dev)
	// cfg.Redis = nil

	coreApp, err := app.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize go-core: %v", err)
	}
	defer coreApp.Close()

	r := gin.Default()

	// Mount all go-core routes (auth, admin GUI, OIDC, etc.)
	coreApp.RegisterRoutes(r)

	// Add your own routes alongside go-core
	r.GET("/api/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello from your app!"})
	})

	// Protect your own routes with go-core's auth middleware
	protected := r.Group("/api", coreApp.AuthMiddleware())
	{
		protected.GET("/me", func(c *gin.Context) {
			c.JSON(200, gin.H{"userID": c.GetString("userID")})
		})
	}

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
