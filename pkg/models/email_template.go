package models

import (
	"time"

	"github.com/google/uuid"
)

// EmailTemplate stores email templates that can be per-application or global defaults.
// Resolution order: app-specific (app_id set) -> global default (app_id NULL) -> hardcoded fallback.
type EmailTemplate struct {
	ID             uuid.UUID  `json:"id"`
	AppID          *uuid.UUID `json:"app_id"` // NULL = global default template
	EmailTypeID    uuid.UUID  `json:"email_type_id"`
	Name           string     `json:"name"`
	Subject        string     `json:"subject"`
	BodyHTML       string     `json:"body_html"`
	BodyText       string     `json:"body_text"`
	TemplateEngine string     `json:"template_engine"` // go_template | placeholder | raw_html
	FromEmail      string     `json:"from_email,omitempty"`               // Optional sender override
	FromName       string     `json:"from_name,omitempty"`                // Optional sender name override
	ServerConfigID *uuid.UUID `json:"server_config_id,omitempty"`                            // Optional link to specific SMTP config
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relations
	EmailType    EmailType          `json:"email_type,omitempty"`
	ServerConfig *EmailServerConfig `json:"server_config,omitempty"`
}

// TableName specifies the table name for EmailTemplate.
func (EmailTemplate) TableName() string {
	return "email_templates"
}

// Template engine constants
const (
	TemplateEngineGoTemplate  = "go_template" // Go html/template syntax: {{.VarName}}
	TemplateEnginePlaceholder = "placeholder" // Simple replacement: {var_name}
	TemplateEngineRawHTML     = "raw_html"    // Raw HTML with {{.VarName}} substitution
)
