// Package app provides the public entry point for the go-core authentication
// module. Consumers call New to initialize, RegisterRoutes to mount handlers
// onto a Gin engine, and Close to shut down gracefully.
package app

import (
	"fmt"

	"github.com/gin-gonic/gin"

	core "github.com/MrF1ow/go-core"
	"github.com/MrF1ow/go-core/internal/coreapp"
	"github.com/MrF1ow/go-core/internal/middleware"
)

// App is the public entry point for the go-core authentication module.
// All internal services are hidden behind an opaque struct — consumers
// interact only through RegisterRoutes and Close.
type App struct {
	app *coreapp.App
}

// New validates the configuration and initializes the go-core module.
// It creates and owns its own database connection pool.
func New(cfg core.Config) (*App, error) {
	if err := core.ValidateConfig(cfg); err != nil {
		return nil, err
	}

	internal, err := coreapp.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("core: initialization failed: %w", err)
	}

	return &App{app: internal}, nil
}

// RegisterRoutes mounts all go-core HTTP routes (auth, admin, OIDC, etc.)
// onto the provided Gin engine. The consumer owns the engine and server lifecycle.
func (a *App) RegisterRoutes(r *gin.Engine) {
	a.app.RegisterRoutes(r)
}

// AuthMiddleware returns a Gin handler that enforces JWT authentication.
// Drop it into any route or group the consumer controls:
//
//	r.GET("/profile", coreApp.AuthMiddleware(), profileHandler)
//	api := r.Group("/api", coreApp.AuthMiddleware())
func (a *App) AuthMiddleware() gin.HandlerFunc {
	return middleware.AuthMiddleware()
}

// Close gracefully shuts down all background services and the database
// connection pool owned by the module.
func (a *App) Close() {
	a.app.Close()
}
