// Package core defines the configuration types and shared interfaces for the
// go-core authentication module. Consuming applications build a Config struct
// and pass it to app.New.
package core

import "time"

// Config is the top-level configuration for the go-core module.
// Consuming applications construct this struct however they want (env vars,
// files, flags, etc.). The core module never reads environment variables itself.
type Config struct {
	Database    DatabaseConfig
	Redis       *RedisConfig // nil = use in-memory store
	JWT         JWTConfig
	Email       *EmailConfig // nil = no email sending
	CORS        CORSConfig
	OIDC        OIDCConfig
	WebAuthn    WebAuthnConfig
	SMS         SMSConfig
	Admin       AdminConfig
	Social      SocialConfig
	GeoIP       GeoIPConfig
	Session     SessionConfig
	MultiTenant bool
	PublicURL   string
	FrontendURL string
	AppName     string
	Port        string
	GinMode     string
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// RedisConfig holds Redis connection parameters.
// Set Config.Redis to nil to use an in-memory store instead.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// JWTConfig holds JSON Web Token settings.
type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// EmailConfig holds SMTP / email sending settings.
// Set Config.Email to nil to disable all email sending.
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
}

// CORSConfig holds Cross-Origin Resource Sharing settings.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposeHeaders    []string
	MaxAge           time.Duration
	AllowCredentials bool
}

// OIDCConfig holds OpenID Connect provider settings.
type OIDCConfig struct {
	Enabled      bool
	DefaultAppID string
	IDTokenTTL   time.Duration
	AuthCodeTTL  time.Duration
}

// WebAuthnConfig holds WebAuthn / passkey relying-party settings.
type WebAuthnConfig struct {
	RPID      string
	RPName    string
	RPOrigins []string
}

// SMSConfig holds SMS provider credentials.
type SMSConfig struct {
	Provider         string
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string
}

// AdminBrandingConfig customizes the admin dashboard appearance.
// All fields are optional — zero values produce the default Bootstrap look.
type AdminBrandingConfig struct {
	OrgName          string
	LogoURL          string
	PrimaryColor     string
	SecondaryColor   string
	BorderRadius     string
	SidebarColor     string
	SidebarTextColor string
}

// AdminConfig holds admin GUI / API settings.
type AdminConfig struct {
	APIKey     string
	Email      string
	SessionTTL time.Duration
	BaseURL    string
	Branding   AdminBrandingConfig
}

// SocialConfig holds OAuth2 social-login settings.
type SocialConfig struct {
	AllowedRedirectDomains []string
	DefaultRedirectURI     string
}

// GeoIPConfig holds GeoIP database settings.
type GeoIPConfig struct {
	DBPath string
}

// SessionConfig holds session and trusted-device settings.
type SessionConfig struct {
	TrustedDeviceCookieSameSite string
	GroupExpiryEnabled          bool
	GroupExpiryScanInterval     time.Duration
	GroupKeyspaceNotifEnabled   bool
	RedisNotifyKeyspaceEvents   string // value of REDIS_NOTIFY_KEYSPACE_EVENTS for expiry service
}

// DefaultConfig returns a Config populated with sensible defaults that match
// the application's built-in Viper defaults.
func DefaultConfig() Config {
	return Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "",
			DBName:   "go_core",
			SSLMode:  "disable",
		},
		Redis: &RedisConfig{
			Addr:     "localhost:6379",
			Password: "",
			DB:       1,
		},
		JWT: JWTConfig{
			Secret:          "",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 720 * time.Hour,
		},
		Email: nil,
		CORS: CORSConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
				"http://localhost:5173",
				"http://localhost:5174",
				"http://localhost:5175",
				"http://localhost:8080",
			},
			AllowedMethods: []string{
				"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD",
			},
			AllowedHeaders: []string{
				"Origin",
				"Content-Type",
				"Content-Length",
				"Accept-Encoding",
				"X-CSRF-Token",
				"Authorization",
				"Accept",
				"Cache-Control",
				"X-Requested-With",
				"X-App-ID",
			},
			ExposeHeaders: []string{
				"Content-Length",
				"Access-Control-Allow-Origin",
				"Access-Control-Allow-Headers",
				"Content-Type",
			},
			MaxAge:           12 * time.Hour,
			AllowCredentials: true,
		},
		OIDC: OIDCConfig{
			Enabled:      false,
			DefaultAppID: "00000000-0000-0000-0000-000000000001",
			IDTokenTTL:   60 * time.Minute,
			AuthCodeTTL:  10 * time.Minute,
		},
		WebAuthn: WebAuthnConfig{
			RPID:      "localhost",
			RPName:    "",
			RPOrigins: []string{},
		},
		SMS: SMSConfig{
			Provider:         "",
			TwilioAccountSID: "",
			TwilioAuthToken:  "",
			TwilioFromNumber: "",
		},
		Admin: AdminConfig{
			APIKey:     "",
			Email:      "",
			SessionTTL: 24 * time.Hour,
			BaseURL:    "",
		},
		Social: SocialConfig{
			AllowedRedirectDomains: []string{},
			DefaultRedirectURI:     "",
		},
		GeoIP: GeoIPConfig{
			DBPath: "",
		},
		Session: SessionConfig{
			TrustedDeviceCookieSameSite: "none",
			GroupExpiryEnabled:          false,
			GroupExpiryScanInterval:     5 * time.Minute,
			GroupKeyspaceNotifEnabled:   false,
		},
		MultiTenant: false,
		PublicURL:   "http://localhost:8080",
		FrontendURL: "http://localhost:5173",
		AppName:     "",
		Port:        "8080",
		GinMode:     "debug",
	}
}
