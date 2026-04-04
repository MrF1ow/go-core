package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	core "github.com/JedidiahDigital/go-core"
)

// CORSMiddleware creates and configures CORS middleware from the provided config.
func CORSMiddleware(cfg core.CORSConfig) gin.HandlerFunc {
	maxAge := cfg.MaxAge
	if maxAge == 0 {
		maxAge = 12 * time.Hour
	}

	config := cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     cfg.AllowedMethods,
		AllowHeaders:     cfg.AllowedHeaders,
		ExposeHeaders:    cfg.ExposeHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           maxAge,
	}

	return cors.New(config)
}
