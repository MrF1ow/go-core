package models

import "time"

// SchemaMigration tracks which database migrations have been applied
type SchemaMigration struct {
	ID              uint      `json:"id"`
	Version         string    `json:"version"` // YYYYMMDD_HHMMSS format
	Name            string    `json:"name"`
	AppliedAt       time.Time `json:"applied_at"`
	ExecutionTimeMs int       `json:"execution_time_ms"` // How long the migration took
	Success         bool      `json:"success"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	Checksum        string    `json:"checksum,omitempty"` // SHA256 of migration file
}

// TableName specifies the table name for GORM
func (SchemaMigration) TableName() string {
	return "schema_migrations"
}
