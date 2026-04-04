package core

import "fmt"

// ValidateConfig checks that all required Config fields are set.
// Returns a descriptive error for the first missing or invalid field.
// Called by app.New() before any initialization or connections.
func ValidateConfig(cfg Config) error {
	if cfg.Database.Host == "" {
		return fmt.Errorf("core: Config.Database.Host is required")
	}
	if cfg.Database.Port <= 0 {
		return fmt.Errorf("core: Config.Database.Port must be > 0")
	}
	if cfg.Database.DBName == "" {
		return fmt.Errorf("core: Config.Database.DBName is required")
	}
	if cfg.Database.User == "" {
		return fmt.Errorf("core: Config.Database.User is required")
	}
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("core: Config.JWT.Secret is required")
	}
	if len(cfg.JWT.Secret) < 32 {
		return fmt.Errorf("core: Config.JWT.Secret must be at least 32 characters")
	}
	return nil
}
