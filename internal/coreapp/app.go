// Package coreapp provides the application entry point for go-core.
//
// It lives in internal/ rather than the root package because the root package
// exports types (CacheStore, Config, etc.) that internal packages import.
// Placing the wiring here avoids circular dependencies.
package coreapp

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	core "github.com/MrF1ow/go-core"
	"github.com/MrF1ow/go-core/internal/admin"
	"github.com/MrF1ow/go-core/internal/bruteforce"
	"github.com/MrF1ow/go-core/internal/database"
	"github.com/MrF1ow/go-core/internal/email"
	"github.com/MrF1ow/go-core/internal/geoip"
	"github.com/MrF1ow/go-core/internal/health"
	logService "github.com/MrF1ow/go-core/internal/log"
	"github.com/MrF1ow/go-core/internal/middleware"
	"github.com/MrF1ow/go-core/internal/oidc"
	"github.com/MrF1ow/go-core/internal/operator"
	"github.com/MrF1ow/go-core/internal/rbac"
	"github.com/MrF1ow/go-core/internal/redis"
	"github.com/MrF1ow/go-core/internal/session"
	"github.com/MrF1ow/go-core/internal/sessiongroup"
	"github.com/MrF1ow/go-core/internal/sms"
	"github.com/MrF1ow/go-core/internal/social"
	ssopkg "github.com/MrF1ow/go-core/internal/sso"
	"github.com/MrF1ow/go-core/internal/twofa"
	"github.com/MrF1ow/go-core/internal/user"
	"github.com/MrF1ow/go-core/internal/util"
	passkey "github.com/MrF1ow/go-core/internal/webauthn"
	"github.com/MrF1ow/go-core/internal/webhook"
	pkgjwt "github.com/MrF1ow/go-core/pkg/jwt"
	"github.com/MrF1ow/go-core/web"
	"github.com/MrF1ow/go-core/web/static"
)

// App holds the initialized services and handlers for the go-core module.
// Create via New() or NewWithDB().
type App struct {
	pool     *pgxpool.Pool
	config   core.Config
	ownsPool bool

	// Handlers
	userHandler     *user.Handler
	socialHandler   *social.Handler
	twofaHandler    *twofa.Handler
	logHandler      *logService.Handler
	sessionHandler  *session.Handler
	adminHandler    *admin.Handler
	rbacHandler     *rbac.Handler
	healthHandler   *health.Handler
	webauthnHandler *passkey.Handler
	guiHandler      *admin.GUIHandler
	ssoHandler      *ssopkg.Handler
	webhookHandler  *webhook.Handler
	oidcHandler     *oidc.Handler

	// Services that need references for wiring and cleanup
	rbacService         *rbac.Service
	accountService      *admin.AccountService
	adminRepo           *admin.Repository
	operatorRepo        *operator.Repository
	sessionGroupRevoker *sessiongroup.Revoker
	settingsService     *admin.SettingsService

	// Background services (for graceful shutdown)
	webhookService        *webhook.Service
	cleanupService        *logService.CleanupService
	apiKeyNotificationSvc *admin.ApiKeyNotificationService
	expiryService         *sessiongroup.ExpiryService

	logoData           []byte
	logoContentType    string
	faviconData        []byte
	faviconContentType string
}

// New creates a new App by connecting to the database (pgx) and initializing
// all services and handlers.
func New(cfg core.Config) (*App, error) {
	pool, err := database.ConnectPgx(
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)
	if err != nil {
		return nil, err
	}

	app, err := initialize(cfg, pool)
	if err != nil {
		pool.Close()
		return nil, err
	}
	app.ownsPool = true
	return app, nil
}

// NewWithDB creates a new App using an externally managed DB pool.
// The caller is responsible for closing the pool.
func NewWithDB(cfg core.Config, pool *pgxpool.Pool) (*App, error) {
	app, err := initialize(cfg, pool)
	if err != nil {
		return nil, err
	}
	app.ownsPool = false
	return app, nil
}

// initialize performs all service and handler construction and wiring.
// This is a direct transplant of cmd/api/main.go lines ~211-476.
func initialize(cfg core.Config, pool *pgxpool.Pool) (*App, error) {
	// Initialize foundation packages
	pkgjwt.Init(cfg.JWT.Secret, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)
	util.Init(cfg.FrontendURL, cfg.AppName)
	social.Init(cfg.Social.AllowedRedirectDomains, cfg.Social.DefaultRedirectURI)
	passkey.Init(cfg.WebAuthn.RPID, cfg.WebAuthn.RPName, cfg.WebAuthn.RPOrigins, cfg.FrontendURL, cfg.AppName, cfg.Admin.BaseURL)

	// Connect to Redis (via CacheStore abstraction)
	var cacheStore core.CacheStore
	if cfg.Redis != nil {
		rcs, err := core.NewRedisCacheStore(*cfg.Redis)
		if err != nil {
			return nil, err
		}
		cacheStore = rcs
	} else {
		cacheStore = core.NewMemoryCacheStore()
	}
	redis.Init(cacheStore)
	middleware.SetCacheStore(cacheStore)

	// Initialize GeoIP service (graceful degradation if not configured)
	geoIPService := geoip.NewService(cfg.GeoIP.DBPath)

	// Initialize Anomaly Detector (uses GeoIP if available)
	anomalyDetector := logService.NewAnomalyDetector(pool, geoIPService)

	// Initialize Brute-Force Protection Service
	bruteForceService := bruteforce.NewService(pool)

	// Initialize Activity Log Service
	logSvc := logService.InitializeLogService(pool, anomalyDetector)

	// Initialize IP Rule infrastructure
	ipRuleRepo := geoip.NewIPRuleRepository(pool)
	ipRuleEvaluator := geoip.NewIPRuleEvaluator(ipRuleRepo, geoIPService)

	// Initialize Activity Log Cleanup Service
	cleanupService := logService.InitializeCleanupService(pool)

	// Initialize Services and Handlers
	userRepo := user.NewRepository(pool)
	socialRepo := social.NewRepository(pool)
	logRepo := logService.NewRepository(pool)
	emailRepo := email.NewRepository(pool)
	emailService := email.NewService(emailRepo)
	rbacRepo := rbac.NewRepository(pool)
	rbacService := rbac.NewService(rbacRepo, cacheStore, cfg.JWT.AccessTokenTTL)
	rbacHandler := rbac.NewHandler(rbacService)
	userService := user.NewService(userRepo, emailService)
	userService.LookupRoles = rbacService.GetUserRoleNames
	userService.AssignDefaultRole = rbacService.AssignDefaultRole
	sessionService := session.NewService()
	userService.SessionService = sessionService
	socialService := social.NewService(userRepo, socialRepo)
	socialService.LookupRoles = rbacService.GetUserRoleNames
	socialService.AssignDefaultRole = rbacService.AssignDefaultRole
	socialService.SessionService = sessionService
	twofaService := twofa.NewService(userRepo, emailService)
	twofaService.AppName = cfg.AppName
	logQueryService := logService.NewQueryService(logRepo)

	// Initialize SMS sender (graceful degradation if not configured)
	smsSender := sms.NewSender(cfg.SMS)

	// Initialize Trusted Device Repository
	trustedDeviceRepo := twofa.NewTrustedDeviceRepository(pool)

	// Initialize Webhook Service
	webhookRepo := webhook.NewRepository(pool)
	webhookService := webhook.NewService(webhookRepo)
	webhookHandler := webhook.NewHandler(webhookService)

	// Wire WebhookService into domain services
	userService.WebhookService = webhookService
	twofaService.WebhookService = webhookService
	socialService.WebhookService = webhookService
	// Wire SMS sender and trusted device repo into twofa service
	twofaService.SMSSender = smsSender
	twofaService.TrustedDeviceRepo = trustedDeviceRepo
	// Wire SMS sender into user service so SMS 2FA codes are auto-sent on password login
	userService.SMSSender = smsSender
	userHandler := user.NewHandler(userService)
	socialHandler := social.NewHandler(socialService)
	// Wire twofa service into social handler so SMS 2FA codes are auto-sent on social login
	socialHandler.TwoFAService = twofaService
	// Wire trusted device validation callback into social handler
	socialHandler.ValidateTrustedDevice = func(plainToken string) (uuid.UUID, uuid.UUID, bool) {
		device, appErr := twofaService.ValidateTrustedDevice(plainToken)
		if appErr != nil {
			return uuid.Nil, uuid.Nil, false
		}
		return device.UserID, device.AppID, true
	}
	twofaHandler := twofa.NewHandler(twofaService)
	twofaHandler.LookupRoles = rbacService.GetUserRoleNames
	twofaHandler.SessionService = sessionService
	twofaHandler.AssignDefaultRole = rbacService.AssignDefaultRole
	// Wire trusted device repo into twofa handler
	twofaHandler.TrustedDeviceRepo = trustedDeviceRepo
	// Wire trusted device validation callback into user handler (avoids circular import)
	userHandler.ValidateTrustedDevice = func(plainToken string) (uuid.UUID, uuid.UUID, bool) {
		device, appErr := twofaService.ValidateTrustedDevice(plainToken)
		if appErr != nil {
			return uuid.Nil, uuid.Nil, false
		}
		return device.UserID, device.AppID, true
	}
	logHandler := logService.NewHandler(logQueryService)
	sessionHandler := session.NewHandler(sessionService)
	adminRepo := admin.NewRepository(pool)
	operatorRepo := operator.NewRepository(pool)
	adminHandler := admin.NewHandler(adminRepo, emailService)

	// Wire resolver callbacks so the email variable resolver can look up email types, users, and apps
	emailService.Resolver().LookupEmailType = emailRepo.GetEmailTypeByCode
	emailService.Resolver().LookupUser = userRepo.GetUserByID
	emailService.Resolver().LookupApp = adminRepo.GetAppByID

	// Wire AppLookup callback into all services/handlers that need application config
	userService.AppLookup = adminRepo.GetAppByID
	twofaService.AppLookup = adminRepo.GetAppByID
	socialService.AppLookup = adminRepo.GetAppByID
	twofaHandler.AppLookup = adminRepo.GetAppByID

	// Initialize Health & Metrics Handler
	smtpAddr := health.ResolveSMTPAddr(pool)
	healthHandler := health.NewHandler(pool, cacheStore, smtpAddr)

	// Initialize WebAuthn/Passkey Services and Handler
	webauthnRepo := passkey.NewRepository(pool)
	webauthnService := passkey.NewService(webauthnRepo, userRepo)
	webauthnHandler := passkey.NewHandler(webauthnService)
	webauthnHandler.LookupRoles = rbacService.GetUserRoleNames
	webauthnHandler.SessionService = sessionService
	webauthnHandler.AssignDefaultRole = rbacService.AssignDefaultRole
	webauthnService.AppLookup = adminRepo.GetAppByID
	webauthnHandler.AppLookup = adminRepo.GetAppByID

	// Initialize Admin GUI Services and Handler
	accountRepo := admin.NewAccountRepository(pool)
	accountService := admin.NewAccountService(accountRepo, emailService, cfg.Admin.SessionTTL)
	dashboardService := admin.NewDashboardService(pool)
	settingsRepo := admin.NewSettingsRepository(pool)
	redisAddr := ""
	if cfg.Redis != nil {
		redisAddr = cfg.Redis.Addr
	}
	settingsService := admin.NewSettingsService(settingsRepo, pool, redisAddr, cfg.Port, cfg.GinMode)
	guiHandler := admin.NewGUIHandler(accountService, dashboardService, adminRepo, settingsService, emailService, rbacService, webauthnService)
	guiHandler.OperatorRepo = operatorRepo
	guiHandler.AdminBaseURL = cfg.Admin.BaseURL
	guiHandler.AccessTokenTTL = cfg.JWT.AccessTokenTTL
	guiHandler.AdminSessionTTL = cfg.Admin.SessionTTL
	guiHandler.BasePath = cfg.Admin.AdminBasePath

	// Initialize SSO Handler
	ssoHandler := ssopkg.NewHandler(adminRepo, userRepo, sessionService)
	ssoHandler.LookupRoles = rbacService.GetUserRoleNames
	ssoHandler.AppLookup = adminRepo.GetAppByID

	// Create session group revoker for shared logout/expiry logic
	sessionGroupRevoker := sessiongroup.NewRevoker(adminRepo, userRepo, sessionService)

	// Wire SSO global logout
	userService.GroupLogoutFunc = func(appID, userEmail string) {
		sessionGroupRevoker.RevokeAllUserSessionsInGroup(appID, userEmail)
	}

	// Wire SettingsService resolver into twofa handler
	twofaHandler.SettingResolver = settingsService.GetResolvedValue

	// Wire admin lookup for passkey discoverable login
	webauthnService.AdminLookup = accountRepo.GetByID

	// Wire WebhookService into admin GUI handler
	guiHandler.WebhookService = webhookService
	webauthnHandler.WebhookService = webhookService

	// Initialize OIDC Provider (enabled via OIDC_ENABLED=true)
	var oidcHandler *oidc.Handler
	if cfg.OIDC.Enabled {
		oidcRepo := oidc.NewRepository(pool)
		oidcService := oidc.NewService(oidcRepo, rbacService.GetUserRoleNames, cfg.PublicURL, cfg.OIDC.AuthCodeTTL, cfg.OIDC.IDTokenTTL, cfg.JWT.AccessTokenTTL)
		oidcHandler = oidc.NewHandler(oidcService, oidcRepo, cfg.PublicURL, cfg.JWT.AccessTokenTTL, cfg.JWT.Secret)
		guiHandler.OIDCService = oidcService
		// Wire OIDC RP-initiated logout group logout
		oidcHandler.GroupLogoutFunc = func(appID, userEmail string) {
			sessionGroupRevoker.RevokeAllUserSessionsInGroup(appID, userEmail)
		}
		// Run an initial cleanup immediately on startup so stale codes
		// from before the last restart are purged without waiting a full hour.
		go func() {
			if err := oidcRepo.DeleteExpiredAuthCodes(); err != nil {
				log.Printf("[OIDC] startup cleanup of expired auth codes failed: %v", err)
			}
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				if err := oidcRepo.DeleteExpiredAuthCodes(); err != nil {
					log.Printf("[OIDC] failed to delete expired auth codes: %v", err)
				}
			}
		}()
	}

	// Wire IP rule evaluator and anomaly detector on login handlers
	userHandler.IPRuleEvaluator = ipRuleEvaluator
	userHandler.AnomalyDetector = anomalyDetector
	socialHandler.IPRuleEvaluator = ipRuleEvaluator
	socialHandler.AnomalyDetector = anomalyDetector
	twofaHandler.IPRuleEvaluator = ipRuleEvaluator
	twofaHandler.AnomalyDetector = anomalyDetector
	webauthnHandler.IPRuleEvaluator = ipRuleEvaluator
	webauthnHandler.AnomalyDetector = anomalyDetector

	// Wire brute-force protection service on login handlers
	userHandler.BruteForceService = bruteForceService
	guiHandler.BruteForceService = bruteForceService

	// Wire IP rule management on admin handlers
	adminHandler.IPRuleRepo = ipRuleRepo
	adminHandler.IPRuleEvaluator = ipRuleEvaluator
	adminHandler.GeoIPService = geoIPService
	adminHandler.TrustedDeviceRepo = trustedDeviceRepo
	guiHandler.IPRuleRepo = ipRuleRepo
	guiHandler.IPRuleEvaluator = ipRuleEvaluator
	guiHandler.GeoIPService = geoIPService
	guiHandler.TrustedDeviceRepo = trustedDeviceRepo

	// Wire health handler into admin GUI for the monitoring page
	guiHandler.HealthHandler = healthHandler

	// Wire anomaly notification callback
	logSvc.SetAnomalyCallback(func(appID, userID uuid.UUID, userEmail string, result logService.AnomalyResult) {
		if result.NotificationDetails == nil {
			return
		}
		nd := result.NotificationDetails
		switch result.NotificationType {
		case "new_device_login":
			if err := emailService.SendNewDeviceLoginEmail(appID, userEmail, &userID,
				nd.IPAddress, nd.Location, nd.Device, nd.LoginTime); err != nil {
				log.Printf("Anomaly notification (new_device_login) failed for user %s: %v", userEmail, err)
			}
		case "suspicious_activity":
			if err := emailService.SendSuspiciousActivityEmail(appID, userEmail, &userID,
				nd.IPAddress, nd.Location, nd.Device, nd.LoginTime, nd.AlertType, nd.Details); err != nil {
				log.Printf("Anomaly notification (suspicious_activity) failed for user %s: %v", userEmail, err)
			}
		}
	})

	// Initialize and start the API key expiry notification service
	apiKeyNotificationSvc := admin.NewApiKeyNotificationService(adminRepo, emailService, cfg.Admin.Email)
	apiKeyNotificationSvc.Start()

	var logoData []byte
	var logoContentType string
	var faviconData []byte
	var faviconContentType string
	b := cfg.Admin.Branding
	if b.LogoURL != "" && !isRemoteAssetURL(b.LogoURL) {
		var err error
		logoData, logoContentType, err = readBrandingFile(b.LogoURL)
		if err != nil {
			return nil, fmt.Errorf("failed to read branding logo file: %w", err)
		}
	}
	if b.FaviconURL != "" && !isRemoteAssetURL(b.FaviconURL) {
		var err error
		faviconData, faviconContentType, err = readBrandingFile(b.FaviconURL)
		if err != nil {
			return nil, fmt.Errorf("failed to read branding favicon file: %w", err)
		}
	}

	return &App{
		pool:                  pool,
		config:                cfg,
		userHandler:           userHandler,
		socialHandler:         socialHandler,
		twofaHandler:          twofaHandler,
		logHandler:            logHandler,
		sessionHandler:        sessionHandler,
		adminHandler:          adminHandler,
		rbacHandler:           rbacHandler,
		healthHandler:         healthHandler,
		webauthnHandler:       webauthnHandler,
		guiHandler:            guiHandler,
		ssoHandler:            ssoHandler,
		webhookHandler:        webhookHandler,
		oidcHandler:           oidcHandler,
		rbacService:           rbacService,
		accountService:        accountService,
		adminRepo:             adminRepo,
		operatorRepo:          operatorRepo,
		sessionGroupRevoker:   sessionGroupRevoker,
		settingsService:       settingsService,
		webhookService:        webhookService,
		cleanupService:        cleanupService,
		apiKeyNotificationSvc: apiKeyNotificationSvc,
		logoData:              logoData,
		logoContentType:       logoContentType,
		faviconData:           faviconData,
		faviconContentType:    faviconContentType,
	}, nil
}

// RegisterRoutes sets up the Gin template renderer, global middleware, and all
// route groups on the provided engine. This is a direct transplant of
// cmd/api/main.go lines ~478-1098.
func (a *App) RegisterRoutes(r *gin.Engine) {
	cfg := a.config

	// Resolve admin base path (default "/gui")
	adminBasePath := cfg.Admin.AdminBasePath
	if adminBasePath == "" {
		adminBasePath = "/gui"
	}

	// Initialize template renderer for GUI
	renderer, err := web.NewRenderer(adminBasePath)
	if err != nil {
		log.Fatalf("Failed to initialize template renderer: %v", err)
	}
	r.HTMLRender = renderer

	// Apply admin branding to renderer
	b := cfg.Admin.Branding
	logoServeURL := ""
	if b.LogoURL != "" && !isRemoteAssetURL(b.LogoURL) {
		logoServeURL = adminBasePath + "/branding/logo"
	}
	faviconServeURL := ""
	if b.FaviconURL != "" && !isRemoteAssetURL(b.FaviconURL) {
		faviconServeURL = adminBasePath + "/branding/favicon"
	}
	renderer.SetBranding(web.ResolveBranding(web.BrandingInput{
		OrgName:          b.OrgName,
		LogoURL:          b.LogoURL,
		PrimaryColor:     b.PrimaryColor,
		SecondaryColor:   b.SecondaryColor,
		BorderRadius:     b.BorderRadius,
		SidebarColor:     b.SidebarColor,
		SidebarTextColor: b.SidebarTextColor,
		CustomCSS:        b.CustomCSS,
		FaviconURL:       b.FaviconURL,
		LogoServeURL:     logoServeURL,
		FaviconServeURL:  faviconServeURL,
	}))

	// Add security headers middleware (before CORS so headers are always set)
	r.Use(middleware.SecurityHeadersMiddleware(adminBasePath))

	// Add CORS middleware
	r.Use(middleware.CORSMiddleware(cfg.CORS))
	r.Use(middleware.AppIDMiddleware(cfg.MultiTenant, adminBasePath))

	// Instrument all requests with Prometheus metrics
	r.Use(health.PrometheusMiddleware())

	// Public routes (with rate limiting)
	public := r.Group("/")
	{
		public.POST("/register", middleware.APIRegisterRateLimit(), a.userHandler.Register)
		public.POST("/login", middleware.APILoginRateLimit(), a.userHandler.Login)
		public.POST("/refresh-token", middleware.APIRefreshTokenRateLimit(), a.userHandler.RefreshToken)
		public.POST("/forgot-password", middleware.APIForgotPasswordRateLimit(), a.userHandler.ForgotPassword)
		public.POST("/reset-password", middleware.APIResetPasswordRateLimit(), a.userHandler.ResetPassword)
		public.GET("/verify-email", a.userHandler.VerifyEmail)
		public.POST("/resend-verification", middleware.APIResendVerificationRateLimit(), a.userHandler.ResendVerification)
		public.POST("/2fa/login-verify", middleware.API2FAVerifyRateLimit(), a.twofaHandler.VerifyLogin)
		public.POST("/2fa/email/resend", middleware.API2FAVerifyRateLimit(), a.twofaHandler.ResendEmail2FACode)
		public.POST("/2fa/sms/resend", middleware.API2FAVerifyRateLimit(), a.twofaHandler.ResendSMS2FACode)
		public.POST("/2fa/backup-email/resend", middleware.API2FAVerifyRateLimit(), a.twofaHandler.ResendBackupEmail2FACode)
		public.GET("/2fa/backup-email/verify", a.twofaHandler.VerifyBackupEmail)
		public.GET("/2fa/methods", a.twofaHandler.GetAvailableMethods)

		// Passkey 2FA login (public because it needs temp token)
		public.POST("/2fa/passkey/begin", middleware.APIPasskey2FARateLimit(), a.webauthnHandler.BeginPasskey2FA)
		public.POST("/2fa/passkey/finish", middleware.APIPasskey2FARateLimit(), a.webauthnHandler.FinishPasskey2FA)

		// Passwordless passkey login (public)
		public.POST("/passkey/login/begin", middleware.APIPasskeyLoginRateLimit(), a.webauthnHandler.BeginPasswordlessLogin)
		public.POST("/passkey/login/finish", middleware.APIPasskeyLoginRateLimit(), a.webauthnHandler.FinishPasswordlessLogin)

		// Magic link passwordless login (public)
		public.POST("/magic-link/request", middleware.APIMagicLinkRateLimit(), a.userHandler.RequestMagicLink)
		public.POST("/magic-link/verify", middleware.APIMagicLinkRateLimit(), a.userHandler.VerifyMagicLink)

		// Public app login configuration
		public.GET("/app-config/:app_id", a.adminHandler.GetAppLoginConfig)

		// Health check
		public.GET("/health", a.healthHandler.Health)
	}

	// Metrics endpoint (Admin API Key required)
	metricsGroup := r.Group("", a.adminAuth())
	{
		metricsGroup.GET("/metrics", requireOp(operator.ResMonitoring, operator.ActionRead), a.healthHandler.Metrics)
	}

	// Social authentication routes
	auth := r.Group("/auth")
	{
		auth.GET("/google/login", a.socialHandler.GoogleLogin)
		auth.GET("/google/callback", a.socialHandler.GoogleCallback)
		auth.GET("/facebook/login", a.socialHandler.FacebookLogin)
		auth.GET("/facebook/callback", a.socialHandler.FacebookCallback)
		auth.GET("/github/login", a.socialHandler.GithubLogin)
		auth.GET("/github/callback", a.socialHandler.GithubCallback)
		auth.POST("/merge/confirm", a.socialHandler.MergeConfirm)
		auth.GET("/google/link/callback", a.socialHandler.GoogleLinkCallback)
		auth.GET("/facebook/link/callback", a.socialHandler.FacebookLinkCallback)
		auth.GET("/github/link/callback", a.socialHandler.GithubLinkCallback)
	}

	// Account linking initiation routes (require JWT authentication)
	authLink := r.Group("/auth")
	authLink.Use(middleware.AuthMiddleware())
	{
		authLink.GET("/google/link", a.socialHandler.GoogleLink)
		authLink.GET("/facebook/link", a.socialHandler.FacebookLink)
		authLink.GET("/github/link", a.socialHandler.GithubLink)
	}

	// Protected routes (require JWT authentication)
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// User profile routes
		protected.GET("/profile", middleware.AuthorizePermission(a.rbacService, "user", "read"), a.userHandler.GetProfile)
		protected.PUT("/profile", middleware.AuthorizePermission(a.rbacService, "user", "write"), a.userHandler.UpdateProfile)
		protected.DELETE("/profile", middleware.AuthorizePermission(a.rbacService, "user", "delete"), a.userHandler.DeleteAccount)
		protected.PUT("/profile/email", middleware.AuthorizePermission(a.rbacService, "user", "write"), a.userHandler.UpdateEmail)
		protected.PUT("/profile/password", middleware.AuthorizePermission(a.rbacService, "user", "write"), a.userHandler.UpdatePassword)
		protected.POST("/profile/set-password", middleware.AuthorizePermission(a.rbacService, "user", "write"), a.userHandler.SetPassword)

		// Social account management routes
		protected.GET("/profile/social-accounts", middleware.AuthorizePermission(a.rbacService, "user", "read"), a.socialHandler.ListSocialAccounts)
		protected.DELETE("/profile/social-accounts/:id", middleware.AuthorizePermission(a.rbacService, "user", "write"), a.socialHandler.UnlinkSocialAccount)

		// Auth routes
		protected.GET("/auth/validate", a.userHandler.ValidateToken)
		protected.POST("/logout", a.userHandler.Logout)

		// 2FA management routes
		protected.POST("/2fa/generate", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.Generate2FA)
		protected.POST("/2fa/verify-setup", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.VerifySetup)
		protected.POST("/2fa/enable", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.Enable2FA)
		protected.POST("/2fa/disable", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.Disable2FA)
		protected.POST("/2fa/recovery-codes", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.GenerateRecoveryCodes)
		protected.POST("/2fa/email/enable", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.EnableEmail2FA)
		protected.POST("/2fa/sms/enable", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.EnableSMS2FA)

		// Backup email 2FA routes
		protected.POST("/2fa/backup-email", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.AddBackupEmail)
		protected.DELETE("/2fa/backup-email", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.RemoveBackupEmail)
		protected.GET("/2fa/backup-email/status", middleware.AuthorizePermission(a.rbacService, "settings", "read"), a.twofaHandler.BackupEmailStatus)
		protected.POST("/2fa/backup-email/enable", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.EnableBackupEmail2FA)
		protected.POST("/2fa/backup-email/disable", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.DisableBackupEmail2FA)

		// Phone management routes
		protected.POST("/phone", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.AddPhone)
		protected.POST("/phone/verify", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.VerifyPhone)
		protected.DELETE("/phone", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.RemovePhone)
		protected.GET("/phone/status", middleware.AuthorizePermission(a.rbacService, "settings", "read"), a.twofaHandler.PhoneStatus)

		// Trusted device management routes
		protected.GET("/2fa/trusted-devices", middleware.AuthorizePermission(a.rbacService, "settings", "read"), a.twofaHandler.ListTrustedDevices)
		protected.DELETE("/2fa/trusted-devices/:id", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.RevokeTrustedDevice)
		protected.DELETE("/2fa/trusted-devices", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.twofaHandler.RevokeAllTrustedDevices)

		// Passkey management routes
		protected.POST("/passkey/register/begin", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.webauthnHandler.BeginRegistration)
		protected.POST("/passkey/register/finish", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.webauthnHandler.FinishRegistration)
		protected.GET("/passkeys", middleware.AuthorizePermission(a.rbacService, "settings", "read"), a.webauthnHandler.ListCredentials)
		protected.PUT("/passkeys/:id", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.webauthnHandler.RenameCredential)
		protected.DELETE("/passkeys/:id", middleware.AuthorizePermission(a.rbacService, "settings", "write"), a.webauthnHandler.DeleteCredential)

		// Activity log routes
		protected.GET("/activity-logs", middleware.AuthorizePermission(a.rbacService, "log", "read"), a.logHandler.GetUserActivityLogs)
		protected.GET("/activity-logs/event-types", middleware.AuthorizePermission(a.rbacService, "log", "read"), a.logHandler.GetEventTypes)
		protected.GET("/activity-logs/export", middleware.AuthorizePermission(a.rbacService, "log", "read"), a.logHandler.ExportUserActivityLogs)
		protected.GET("/activity-logs/:id", middleware.AuthorizePermission(a.rbacService, "log", "read"), a.logHandler.GetActivityLogByID)

		// Session management routes
		protected.GET("/sessions", a.sessionHandler.ListSessions)
		protected.DELETE("/sessions/:id", a.sessionHandler.RevokeSession)
		protected.DELETE("/sessions", a.sessionHandler.RevokeAllSessions)
	}

	// SSO routes
	ssoPublic := r.Group("/sso")
	{
		ssoPublic.POST("/exchange", a.ssoHandler.Exchange)
		ssoPublic.GET("/peers", a.ssoHandler.GetPeers)
	}
	ssoProtected := r.Group("/sso")
	ssoProtected.Use(middleware.AuthMiddleware())
	{
		ssoProtected.POST("/token", middleware.APISSORateLimit(), a.ssoHandler.IssueToken)
	}

	// Admin routes (protected by Admin API Key)
	adminRoutes := r.Group("/admin")
	adminRoutes.Use(a.adminAuth())
	{
		adminRoutes.GET("/activity-logs", requireOp(operator.ResLogs, operator.ActionRead), a.logHandler.GetAllActivityLogs)
		adminRoutes.GET("/activity-logs/export", requireOp(operator.ResLogs, operator.ActionRead), a.logHandler.ExportAllActivityLogs)

		adminRoutes.POST("/tenants", requireOp(operator.ResTenants, operator.ActionWrite), a.adminHandler.CreateTenant)
		adminRoutes.GET("/tenants", requireOp(operator.ResTenants, operator.ActionRead), a.adminHandler.ListTenants)
		adminRoutes.POST("/apps", requireOp(operator.ResApplications, operator.ActionWrite), a.adminHandler.CreateApp)
		adminRoutes.GET("/apps/:id", requireOp(operator.ResApplications, operator.ActionRead), a.adminHandler.GetAppDetails)
		adminRoutes.POST("/apps/:id/oauth-config", requireOp(operator.ResOAuth, operator.ActionWrite), a.adminHandler.UpsertOAuthConfig)

		adminRoutes.GET("/email-types", requireOp(operator.ResEmail, operator.ActionRead), a.adminHandler.ListEmailTypes)
		adminRoutes.GET("/email-types/:code", requireOp(operator.ResEmail, operator.ActionRead), a.adminHandler.GetEmailType)
		adminRoutes.GET("/email-variables", requireOp(operator.ResEmail, operator.ActionRead), a.adminHandler.ListWellKnownVariables)
		adminRoutes.GET("/email-templates", requireOp(operator.ResEmail, operator.ActionRead), a.adminHandler.ListEmailTemplates)
		adminRoutes.GET("/email-templates/:id", requireOp(operator.ResEmail, operator.ActionRead), a.adminHandler.GetEmailTemplate)
		adminRoutes.POST("/email-templates", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.SaveEmailTemplate)
		adminRoutes.DELETE("/email-templates/:id", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.DeleteEmailTemplate)
		adminRoutes.POST("/email-templates/preview", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.PreviewEmailTemplate)
		adminRoutes.GET("/apps/:id/email-config", requireOp(operator.ResEmail, operator.ActionRead), a.adminHandler.GetEmailServerConfig)
		adminRoutes.PUT("/apps/:id/email-config", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.SaveEmailServerConfig)
		adminRoutes.DELETE("/apps/:id/email-config", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.DeleteEmailServerConfig)
		adminRoutes.POST("/apps/:id/email-test", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.SendTestEmail)
		adminRoutes.GET("/apps/:id/email-servers", requireOp(operator.ResEmail, operator.ActionRead), a.adminHandler.ListEmailServerConfigsByApp)

		adminRoutes.GET("/email-servers", requireOp(operator.ResEmail, operator.ActionRead), a.adminHandler.ListAllEmailServerConfigs)
		adminRoutes.GET("/email-servers/:id", requireOp(operator.ResEmail, operator.ActionRead), a.adminHandler.GetEmailServerConfigByID)
		adminRoutes.POST("/email-servers", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.CreateEmailServerConfig)
		adminRoutes.PUT("/email-servers/:id", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.UpdateEmailServerConfigByID)
		adminRoutes.DELETE("/email-servers/:id", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.DeleteEmailServerConfigByID)
		adminRoutes.POST("/email-servers/:id/test", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.SendTestEmailByConfigID)

		adminRoutes.POST("/email-types", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.CreateEmailType)
		adminRoutes.PUT("/email-types/:id", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.UpdateEmailType)
		adminRoutes.DELETE("/email-types/:id", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.DeleteEmailType)
		adminRoutes.POST("/apps/:id/send-email", requireOp(operator.ResEmail, operator.ActionWrite), a.adminHandler.SendCustomEmail)

		adminRoutes.GET("/rbac/roles", requireOp(operator.ResEndUserRBAC, operator.ActionRead), a.rbacHandler.ListRoles)
		adminRoutes.GET("/rbac/roles/:id", requireOp(operator.ResEndUserRBAC, operator.ActionRead), a.rbacHandler.GetRole)
		adminRoutes.POST("/rbac/roles", requireOp(operator.ResEndUserRBAC, operator.ActionWrite), a.rbacHandler.CreateRole)
		adminRoutes.PUT("/rbac/roles/:id", requireOp(operator.ResEndUserRBAC, operator.ActionWrite), a.rbacHandler.UpdateRole)
		adminRoutes.DELETE("/rbac/roles/:id", requireOp(operator.ResEndUserRBAC, operator.ActionWrite), a.rbacHandler.DeleteRole)
		adminRoutes.PUT("/rbac/roles/:id/permissions", requireOp(operator.ResEndUserRBAC, operator.ActionWrite), a.rbacHandler.SetRolePermissions)
		adminRoutes.GET("/rbac/permissions", requireOp(operator.ResEndUserRBAC, operator.ActionRead), a.rbacHandler.ListPermissions)
		adminRoutes.POST("/rbac/permissions", requireOp(operator.ResEndUserRBAC, operator.ActionWrite), a.rbacHandler.CreatePermission)
		adminRoutes.GET("/rbac/user-roles", requireOp(operator.ResEndUserRBAC, operator.ActionRead), a.rbacHandler.ListUserRoles)
		adminRoutes.POST("/rbac/user-roles", requireOp(operator.ResEndUserRBAC, operator.ActionWrite), a.rbacHandler.AssignRole)
		adminRoutes.DELETE("/rbac/user-roles", requireOp(operator.ResEndUserRBAC, operator.ActionWrite), a.rbacHandler.RevokeRole)
		adminRoutes.GET("/rbac/user-roles/user", requireOp(operator.ResEndUserRBAC, operator.ActionRead), a.rbacHandler.GetUserRoles)

		adminRoutes.GET("/apps/:id/ip-rules", requireOp(operator.ResIPRules, operator.ActionRead), a.adminHandler.ListIPRules)
		adminRoutes.POST("/apps/:id/ip-rules", requireOp(operator.ResIPRules, operator.ActionWrite), a.adminHandler.CreateIPRule)
		adminRoutes.GET("/apps/:id/ip-rules/:rule_id", requireOp(operator.ResIPRules, operator.ActionRead), a.adminHandler.GetIPRule)
		adminRoutes.PUT("/apps/:id/ip-rules/:rule_id", requireOp(operator.ResIPRules, operator.ActionWrite), a.adminHandler.UpdateIPRule)
		adminRoutes.DELETE("/apps/:id/ip-rules/:rule_id", requireOp(operator.ResIPRules, operator.ActionWrite), a.adminHandler.DeleteIPRule)
		adminRoutes.POST("/apps/:id/ip-rules/check", requireOp(operator.ResIPRules, operator.ActionWrite), a.adminHandler.CheckIPAccess)

		adminRoutes.GET("/webhooks", requireOp(operator.ResWebhooks, operator.ActionRead), a.webhookHandler.AdminListEndpoints)
		adminRoutes.GET("/webhooks/apps/:app_id", requireOp(operator.ResWebhooks, operator.ActionRead), a.webhookHandler.AdminListEndpointsByApp)
		adminRoutes.POST("/webhooks/apps/:app_id", requireOp(operator.ResWebhooks, operator.ActionWrite), a.webhookHandler.AdminCreateEndpoint)
		adminRoutes.PUT("/webhooks/:id/toggle", requireOp(operator.ResWebhooks, operator.ActionWrite), a.webhookHandler.AdminToggleEndpoint)
		adminRoutes.DELETE("/webhooks/:id", requireOp(operator.ResWebhooks, operator.ActionWrite), a.webhookHandler.AdminDeleteEndpoint)
		adminRoutes.GET("/webhooks/:id/deliveries", requireOp(operator.ResWebhooks, operator.ActionRead), a.webhookHandler.AdminListDeliveriesByEndpoint)
		adminRoutes.GET("/webhooks/apps/:app_id/deliveries", requireOp(operator.ResWebhooks, operator.ActionRead), a.webhookHandler.AdminListDeliveriesByApp)

		adminRoutes.GET("/users/export", requireOp(operator.ResUsers, operator.ActionRead), a.adminHandler.ExportUsers)
		adminRoutes.POST("/users/import", requireOp(operator.ResUsers, operator.ActionWrite), a.adminHandler.ImportUsers)

		adminRoutes.GET("/users/:id/trusted-devices", requireOp(operator.ResUsers, operator.ActionRead), a.adminHandler.AdminListTrustedDevices)
		adminRoutes.DELETE("/users/:id/trusted-devices/:device_id", requireOp(operator.ResUsers, operator.ActionWrite), a.adminHandler.AdminRevokeTrustedDevice)
		adminRoutes.DELETE("/users/:id/trusted-devices", requireOp(operator.ResUsers, operator.ActionWrite), a.adminHandler.AdminRevokeAllTrustedDevices)
	}

	// App API routes (protected by per-application API key)
	appRoutes := r.Group("/app/:id")
	appRoutes.Use(middleware.AppApiKeyMiddleware(a.adminRepo))
	appRoutes.Use(middleware.AppRouteGuardMiddleware())
	{
		appRoutes.GET("/email-config", a.adminHandler.GetEmailServerConfig)
		appRoutes.GET("/email-servers", a.adminHandler.ListEmailServerConfigsByApp)
		appRoutes.POST("/email-test", a.adminHandler.SendTestEmail)
		appRoutes.POST("/send-email", a.adminHandler.SendCustomEmail)

		// Webhook Management (App-scoped)
		appRoutes.GET("/webhooks", a.webhookHandler.AppListEndpoints)
		appRoutes.POST("/webhooks", a.webhookHandler.AppCreateEndpoint)
		appRoutes.PUT("/webhooks/:webhook_id/toggle", a.webhookHandler.AppToggleEndpoint)
		appRoutes.DELETE("/webhooks/:webhook_id", a.webhookHandler.AppDeleteEndpoint)
		appRoutes.GET("/webhooks/deliveries", a.webhookHandler.AppListDeliveries)
	}

	// GUI routes (Admin web interface)
	gui := r.Group(adminBasePath)
	{
		// Static assets (no auth required)
		gui.StaticFS("/static", static.HTTPFileSystem())

		if logoServeURL != "" {
			gui.GET("/branding/logo", func(c *gin.Context) {
				c.Header("Cache-Control", "public, max-age=86400, immutable")
				c.Data(http.StatusOK, a.logoContentType, a.logoData)
			})
		}
		if faviconServeURL != "" {
			gui.GET("/branding/favicon", func(c *gin.Context) {
				c.Header("Cache-Control", "public, max-age=86400, immutable")
				c.Data(http.StatusOK, a.faviconContentType, a.faviconData)
			})
		}

		// Login page and form submission (no auth required)
		gui.GET("/login", a.guiHandler.LoginPage)
		gui.POST("/login", middleware.LoginRateLimitMiddleware(), a.guiHandler.LoginSubmit)

		// Passkey login (no auth required)
		gui.POST("/passkey-login/begin", middleware.GUIPasskeyLoginRateLimit(), a.guiHandler.PasskeyLoginBegin)
		gui.POST("/passkey-login/finish", middleware.GUIPasskeyLoginRateLimit(), a.guiHandler.PasskeyLoginFinish)

		// Magic link login (no auth required)
		gui.POST("/magic-link-login", middleware.GUIMagicLinkRateLimit(), a.guiHandler.MagicLinkLoginRequest)
		gui.GET("/magic-link-login/verify", a.guiHandler.MagicLinkLoginVerify)

		// 2FA verification during login (no auth required)
		gui.GET("/2fa-verify", a.guiHandler.TwoFAVerifyPage)
		gui.POST("/2fa-verify", a.guiHandler.TwoFAVerifySubmit)
		gui.POST("/2fa-resend-email", a.guiHandler.TwoFAResendEmail)

		// Authenticated GUI routes
		guiAuth := gui.Group("/")
		guiAuth.Use(middleware.GUIAuthMiddleware(a.accountService, adminBasePath))
		guiAuth.Use(middleware.CSRFMiddleware(a.accountService))
		{
			guiAuth.GET("/", a.guiHandler.Dashboard)
			guiAuth.GET("/dashboard/stats", a.guiHandler.DashboardStats)
			guiAuth.GET("/dashboard/activity", a.guiHandler.DashboardActivity)
			guiAuth.GET("/logout", a.guiHandler.Logout)

			// Tenant management
			guiAuth.GET("/tenants", a.guiHandler.TenantPage)
			guiAuth.GET("/tenants/list", a.guiHandler.TenantList)
			guiAuth.GET("/tenants/new", a.guiHandler.TenantCreateForm)
			guiAuth.POST("/tenants", a.guiHandler.TenantCreate)
			guiAuth.GET("/tenants/form-cancel", a.guiHandler.TenantFormCancel)
			guiAuth.GET("/tenants/:id/edit", a.guiHandler.TenantEditForm)
			guiAuth.PUT("/tenants/:id", a.guiHandler.TenantUpdate)
			guiAuth.GET("/tenants/:id/delete", a.guiHandler.TenantDeleteConfirm)
			guiAuth.DELETE("/tenants/:id", a.guiHandler.TenantDelete)

			// Application management
			guiAuth.GET("/applications", a.guiHandler.AppPage)
			guiAuth.GET("/applications/list", a.guiHandler.AppList)
			guiAuth.GET("/applications/new", a.guiHandler.AppCreateForm)
			guiAuth.POST("/applications", a.guiHandler.AppCreate)
			guiAuth.GET("/applications/form-cancel", a.guiHandler.AppFormCancel)
			guiAuth.GET("/applications/:id/edit", a.guiHandler.AppEditForm)
			guiAuth.PUT("/applications/:id", a.guiHandler.AppUpdate)
			guiAuth.GET("/applications/:id/delete", a.guiHandler.AppDeleteConfirm)
			guiAuth.DELETE("/applications/:id", a.guiHandler.AppDelete)

			// OAuth config management
			guiAuth.GET("/oauth", a.guiHandler.OAuthPage)
			guiAuth.GET("/oauth/list", a.guiHandler.OAuthList)
			guiAuth.GET("/oauth/new", a.guiHandler.OAuthCreateForm)
			guiAuth.POST("/oauth", a.guiHandler.OAuthCreate)
			guiAuth.GET("/oauth/form-cancel", a.guiHandler.OAuthFormCancel)
			guiAuth.GET("/oauth/:id/edit", a.guiHandler.OAuthEditForm)
			guiAuth.PUT("/oauth/:id", a.guiHandler.OAuthUpdate)
			guiAuth.GET("/oauth/:id/delete", a.guiHandler.OAuthDeleteConfirm)
			guiAuth.DELETE("/oauth/:id", a.guiHandler.OAuthDelete)
			guiAuth.PUT("/oauth/:id/toggle", a.guiHandler.OAuthToggleEnabled)

			// User management
			guiAuth.GET("/users", a.guiHandler.UserPage)
			guiAuth.GET("/users/list", a.guiHandler.UserList)
			guiAuth.GET("/users/export", a.guiHandler.UserExport)
			guiAuth.GET("/users/import/modal", a.guiHandler.UserImportModal)
			guiAuth.POST("/users/import", a.guiHandler.UserImport)
			guiAuth.GET("/users/:id", a.guiHandler.UserDetail)
			guiAuth.PUT("/users/:id/toggle", a.guiHandler.UserToggleActive)
			guiAuth.PUT("/users/:id/unlock", a.guiHandler.UserUnlock)
			guiAuth.GET("/users/social-accounts/:id/unlink", a.guiHandler.SocialAccountUnlinkConfirm)
			guiAuth.DELETE("/users/social-accounts/:id", a.guiHandler.SocialAccountUnlink)
			guiAuth.GET("/users/passkeys/:id/delete", a.guiHandler.PasskeyDeleteConfirm)
			guiAuth.DELETE("/users/passkeys/:id", a.guiHandler.PasskeyDelete)
			guiAuth.DELETE("/users/:id/trusted-devices/:device_id", a.guiHandler.UserRevokeTrustedDevice)
			guiAuth.DELETE("/users/:id/trusted-devices", a.guiHandler.UserRevokeAllTrustedDevices)

			// Activity logs viewer
			guiAuth.GET("/logs", a.guiHandler.LogsPage)
			guiAuth.GET("/logs/list", a.guiHandler.LogList)
			guiAuth.GET("/logs/export", a.guiHandler.LogExport)
			guiAuth.GET("/logs/:id", a.guiHandler.LogDetail)

			// API key management
			guiAuth.GET("/api-keys", a.guiHandler.ApiKeysPage)
			guiAuth.GET("/api-keys/list", a.guiHandler.ApiKeyList)
			guiAuth.GET("/api-keys/new", a.guiHandler.ApiKeyCreateForm)
			guiAuth.POST("/api-keys", a.guiHandler.ApiKeyCreate)
			guiAuth.GET("/api-keys/form-cancel", a.guiHandler.ApiKeyFormCancel)
			guiAuth.GET("/api-keys/:id/edit", a.guiHandler.ApiKeyEditForm)
			guiAuth.PUT("/api-keys/:id", a.guiHandler.ApiKeyUpdate)
			guiAuth.GET("/api-keys/:id/usage", a.guiHandler.ApiKeyUsagePage)
			guiAuth.GET("/api-keys/:id/revoke", a.guiHandler.ApiKeyRevokeConfirm)
			guiAuth.PUT("/api-keys/:id/revoke", a.guiHandler.ApiKeyRevoke)
			guiAuth.GET("/api-keys/:id/delete", a.guiHandler.ApiKeyDeleteConfirm)
			guiAuth.DELETE("/api-keys/:id", a.guiHandler.ApiKeyDelete)

			// Settings management
			guiAuth.GET("/settings", a.guiHandler.SettingsPage)
			guiAuth.GET("/settings/info", a.guiHandler.SettingsInfo)
			guiAuth.GET("/settings/section/:category", a.guiHandler.SettingsSection)
			guiAuth.PUT("/settings/:key", a.guiHandler.SettingUpdate)
			guiAuth.DELETE("/settings/:key", a.guiHandler.SettingReset)

			// System health monitoring
			guiAuth.GET("/monitoring", a.guiHandler.MonitoringPage)
			guiAuth.GET("/monitoring/health", a.guiHandler.MonitoringHealth)
			guiAuth.GET("/monitoring/metrics", a.guiHandler.MonitoringMetrics)

			// Email server management
			guiAuth.GET("/email-servers", a.guiHandler.EmailServersPage)
			guiAuth.GET("/email-servers/list", a.guiHandler.EmailServerList)
			guiAuth.GET("/email-servers/new", a.guiHandler.EmailServerCreateForm)
			guiAuth.POST("/email-servers", a.guiHandler.EmailServerCreate)
			guiAuth.GET("/email-servers/form-cancel", a.guiHandler.EmailServerFormCancel)
			guiAuth.GET("/email-servers/:id/edit", a.guiHandler.EmailServerEditForm)
			guiAuth.PUT("/email-servers/:id", a.guiHandler.EmailServerUpdate)
			guiAuth.GET("/email-servers/:id/delete", a.guiHandler.EmailServerDeleteConfirm)
			guiAuth.DELETE("/email-servers/:id", a.guiHandler.EmailServerDelete)
			guiAuth.POST("/email-servers/:id/test", a.guiHandler.EmailServerSendTest)

			// Email template management
			guiAuth.GET("/email-templates", a.guiHandler.EmailTemplatesPage)
			guiAuth.GET("/email-templates/list", a.guiHandler.EmailTemplateList)
			guiAuth.GET("/email-templates/new", a.guiHandler.EmailTemplateCreateForm)
			guiAuth.POST("/email-templates", a.guiHandler.EmailTemplateCreate)
			guiAuth.GET("/email-templates/form-cancel", a.guiHandler.EmailTemplateFormCancel)
			guiAuth.GET("/email-templates/:id/edit", a.guiHandler.EmailTemplateEditForm)
			guiAuth.PUT("/email-templates/:id", a.guiHandler.EmailTemplateUpdate)
			guiAuth.GET("/email-templates/:id/delete", a.guiHandler.EmailTemplateDeleteConfirm)
			guiAuth.DELETE("/email-templates/:id", a.guiHandler.EmailTemplateDelete)
			guiAuth.POST("/email-templates/preview", a.guiHandler.EmailTemplatePreview)
			guiAuth.POST("/email-templates/editor-window", a.guiHandler.EmailTemplateEditorWindow)
			guiAuth.GET("/email-templates/:id/reset", a.guiHandler.EmailTemplateResetConfirm)
			guiAuth.POST("/email-templates/:id/reset", a.guiHandler.EmailTemplateReset)

			// Email template variables
			guiAuth.GET("/email-variables", a.guiHandler.EmailVariablesList)

			// Email types management
			guiAuth.GET("/email-types", a.guiHandler.EmailTypesPage)
			guiAuth.GET("/email-types/list", a.guiHandler.EmailTypeList)
			guiAuth.GET("/email-types/new", a.guiHandler.EmailTypeCreateForm)
			guiAuth.POST("/email-types", a.guiHandler.EmailTypeCreate)
			guiAuth.GET("/email-types/form-cancel", a.guiHandler.EmailTypeFormCancel)
			guiAuth.GET("/email-types/:id/edit", a.guiHandler.EmailTypeEditForm)
			guiAuth.PUT("/email-types/:id", a.guiHandler.EmailTypeUpdate)
			guiAuth.GET("/email-types/:id/delete", a.guiHandler.EmailTypeDeleteConfirm)
			guiAuth.DELETE("/email-types/:id", a.guiHandler.EmailTypeDelete)

			// Roles management
			guiAuth.GET("/roles", a.guiHandler.RolesPage)
			guiAuth.GET("/roles/list", a.guiHandler.RoleList)
			guiAuth.GET("/roles/new", a.guiHandler.RoleCreateForm)
			guiAuth.POST("/roles", a.guiHandler.RoleCreate)
			guiAuth.GET("/roles/form-cancel", a.guiHandler.RoleFormCancel)
			guiAuth.GET("/roles/:id/edit", a.guiHandler.RoleEditForm)
			guiAuth.PUT("/roles/:id", a.guiHandler.RoleUpdate)
			guiAuth.GET("/roles/:id/delete", a.guiHandler.RoleDeleteConfirm)
			guiAuth.DELETE("/roles/:id", a.guiHandler.RoleDelete)
			guiAuth.GET("/roles/:id/permissions", a.guiHandler.RolePermissions)
			guiAuth.PUT("/roles/:id/permissions", a.guiHandler.RolePermissionsUpdate)

			// Permissions management
			guiAuth.GET("/permissions", a.guiHandler.PermissionsPage)
			guiAuth.GET("/permissions/list", a.guiHandler.PermissionList)
			guiAuth.GET("/permissions/new", a.guiHandler.PermissionCreateForm)
			guiAuth.POST("/permissions", a.guiHandler.PermissionCreate)
			guiAuth.GET("/permissions/form-cancel", a.guiHandler.PermissionFormCancel)

			// User roles management
			guiAuth.GET("/user-roles", a.guiHandler.UserRolesPage)
			guiAuth.GET("/user-roles/list", a.guiHandler.UserRoleList)
			guiAuth.GET("/user-roles/new", a.guiHandler.UserRoleCreateForm)
			guiAuth.POST("/user-roles", a.guiHandler.UserRoleCreate)
			guiAuth.PUT("/user-roles", a.guiHandler.UserRoleUpdate)
			guiAuth.GET("/user-roles/roles-for-app", a.guiHandler.UserRoleRolesForApp)
			guiAuth.GET("/user-roles/search-users", a.guiHandler.UserRoleSearchUsers)
			guiAuth.GET("/user-roles/revoke", a.guiHandler.UserRoleRevokeConfirm)
			guiAuth.DELETE("/user-roles", a.guiHandler.UserRoleRevoke)
			guiAuth.GET("/user-roles/form-cancel", a.guiHandler.UserRoleFormCancel)

			// Session management
			guiAuth.GET("/sessions", a.guiHandler.SessionsPage)
			guiAuth.GET("/sessions/list", a.guiHandler.SessionList)
			guiAuth.GET("/sessions/:app_id/:session_id/detail", a.guiHandler.SessionDetail)
			guiAuth.DELETE("/sessions/:app_id/:session_id", a.guiHandler.SessionRevoke)
			guiAuth.DELETE("/sessions/revoke-all-user", a.guiHandler.SessionRevokeAllForUser)
			guiAuth.GET("/users/:id/sessions", a.guiHandler.UserSessions)

			// IP Rule management
			guiAuth.GET("/ip-rules", a.guiHandler.IPRulePage)
			guiAuth.GET("/ip-rules/list", a.guiHandler.IPRuleList)
			guiAuth.GET("/ip-rules/new", a.guiHandler.IPRuleCreateForm)
			guiAuth.POST("/ip-rules", a.guiHandler.IPRuleCreate)
			guiAuth.GET("/ip-rules/form-cancel", a.guiHandler.IPRuleFormCancel)
			guiAuth.GET("/ip-rules/:id/edit", a.guiHandler.IPRuleEditForm)
			guiAuth.PUT("/ip-rules/:id", a.guiHandler.IPRuleUpdate)
			guiAuth.GET("/ip-rules/:id/delete", a.guiHandler.IPRuleDeleteConfirm)
			guiAuth.DELETE("/ip-rules/:id", a.guiHandler.IPRuleDelete)
			guiAuth.POST("/ip-rules/check", a.guiHandler.IPRuleCheckAccess)

			// My Account & 2FA management
			guiAuth.GET("/my-account", a.guiHandler.MyAccountPage)
			guiAuth.POST("/my-account/email", a.guiHandler.MyAccountUpdateEmail)
			guiAuth.POST("/my-account/password", a.guiHandler.MyAccountChangePassword)
			guiAuth.POST("/my-account/2fa/generate", a.guiHandler.MyAccount2FAGenerateTOTP)
			guiAuth.POST("/my-account/2fa/verify-totp", a.guiHandler.MyAccount2FAVerifyTOTP)
			guiAuth.POST("/my-account/2fa/enable-email", a.guiHandler.MyAccount2FAEnableEmail)
			guiAuth.POST("/my-account/2fa/disable", a.guiHandler.MyAccount2FADisable)
			guiAuth.GET("/my-account/2fa/status", a.guiHandler.MyAccount2FAStatus)
			guiAuth.POST("/my-account/2fa/regenerate-codes", a.guiHandler.MyAccount2FARegenerateCodes)

			// Passkey management (admin self-service)
			guiAuth.GET("/my-account/passkeys/status", a.guiHandler.MyAccountPasskeyStatus)
			guiAuth.POST("/my-account/passkeys/register/begin", a.guiHandler.MyAccountPasskeyBeginRegister)
			guiAuth.POST("/my-account/passkeys/register/finish", a.guiHandler.MyAccountPasskeyFinishRegister)
			guiAuth.DELETE("/my-account/passkeys/:id", a.guiHandler.MyAccountPasskeyDelete)
			guiAuth.POST("/my-account/passkeys/:id/rename", a.guiHandler.MyAccountPasskeyRename)

			// Magic link management (admin self-service)
			guiAuth.GET("/my-account/magic-link/status", a.guiHandler.MyAccountMagicLinkStatus)
			guiAuth.POST("/my-account/magic-link/toggle", a.guiHandler.MyAccountMagicLinkToggle)

			// Backup email management (admin self-service)
			guiAuth.GET("/my-account/backup-email/status", a.guiHandler.MyAccountBackupEmailStatus)
			guiAuth.POST("/my-account/backup-email", a.guiHandler.MyAccountSetBackupEmail)
			guiAuth.DELETE("/my-account/backup-email", a.guiHandler.MyAccountRemoveBackupEmail)

			// Trusted device management (admin self-service)
			guiAuth.GET("/my-account/trusted-devices", a.guiHandler.MyAccountTrustedDevices)
			guiAuth.DELETE("/my-account/trusted-devices/:device_id", a.guiHandler.MyAccountRevokeTrustedDevice)

			// Webhook management
			guiAuth.GET("/webhooks", a.guiHandler.WebhookPage)
			guiAuth.GET("/webhooks/list", a.guiHandler.WebhookList)
			guiAuth.GET("/webhooks/new", a.guiHandler.WebhookCreateForm)
			guiAuth.POST("/webhooks", a.guiHandler.WebhookCreate)
			guiAuth.GET("/webhooks/form-cancel", a.guiHandler.WebhookFormCancel)
			guiAuth.GET("/webhooks/:id/delete", a.guiHandler.WebhookDeleteConfirm)
			guiAuth.DELETE("/webhooks/:id", a.guiHandler.WebhookDelete)
			guiAuth.PUT("/webhooks/:id/toggle", a.guiHandler.WebhookToggle)
			guiAuth.GET("/webhooks/:id/deliveries", a.guiHandler.WebhookDeliveries)

			// OIDC client management (GUI)
			guiAuth.GET("/oidc-clients", a.guiHandler.OIDCClientsPage)
			guiAuth.GET("/oidc-clients/list", a.guiHandler.OIDCClientList)
			guiAuth.GET("/oidc-clients/new", a.guiHandler.OIDCClientCreateForm)
			guiAuth.POST("/oidc-clients", a.guiHandler.OIDCClientCreate)
			guiAuth.GET("/oidc-clients/form-cancel", a.guiHandler.OIDCClientFormCancel)
			guiAuth.GET("/oidc-clients/:id/edit", a.guiHandler.OIDCClientEditForm)
			guiAuth.PUT("/oidc-clients/:id", a.guiHandler.OIDCClientUpdate)
			guiAuth.GET("/oidc-clients/:id/delete", a.guiHandler.OIDCClientDeleteConfirm)
			guiAuth.DELETE("/oidc-clients/:id", a.guiHandler.OIDCClientDelete)
			guiAuth.POST("/oidc-clients/:id/rotate-secret", a.guiHandler.OIDCClientRotateSecret)

			// Session Group management
			guiAuth.GET("/session-groups", a.guiHandler.SessionGroupPage)
			guiAuth.GET("/session-groups/list", a.guiHandler.SessionGroupList)
			guiAuth.GET("/session-groups/new", a.guiHandler.SessionGroupCreateForm)
			guiAuth.POST("/session-groups", a.guiHandler.SessionGroupCreate)
			guiAuth.GET("/session-groups/form-cancel", a.guiHandler.SessionGroupFormCancel)
			guiAuth.GET("/session-groups/:id/edit", a.guiHandler.SessionGroupEditForm)
			guiAuth.PUT("/session-groups/:id", a.guiHandler.SessionGroupUpdate)
			guiAuth.GET("/session-groups/:id/delete", a.guiHandler.SessionGroupDeleteConfirm)
			guiAuth.DELETE("/session-groups/:id", a.guiHandler.SessionGroupDelete)
			guiAuth.GET("/session-groups/:id/apps", a.guiHandler.SessionGroupApps)
			guiAuth.POST("/session-groups/:id/apps", a.guiHandler.SessionGroupAddApp)
			guiAuth.DELETE("/session-groups/:id/apps/:app_id", a.guiHandler.SessionGroupRemoveApp)
		}
	}

	// OIDC Provider routes (enabled only when OIDC_ENABLED=true)
	if a.oidcHandler != nil {
		// Global OIDC discovery redirect to default app
		r.GET("/.well-known/openid-configuration", func(c *gin.Context) {
			c.Redirect(302, "/oidc/"+cfg.OIDC.DefaultAppID+"/.well-known/openid-configuration")
		})

		// Per-app OIDC endpoints
		oidcGroup := r.Group("/oidc/:app_id")
		{
			oidcGroup.GET("/.well-known/openid-configuration", a.oidcHandler.WellKnownConfiguration)
			oidcGroup.GET("/.well-known/jwks.json", a.oidcHandler.JWKS)
			oidcGroup.GET("/authorize", middleware.OIDCAuthorizeRateLimit(), a.oidcHandler.Authorize)
			oidcGroup.POST("/authorize", middleware.OIDCAuthorizeRateLimit(), a.oidcHandler.AuthorizeSubmit)
			oidcGroup.POST("/token", middleware.OIDCTokenRateLimit(), a.oidcHandler.Token)
			oidcGroup.GET("/userinfo", middleware.OIDCUserInfoRateLimit(), a.oidcHandler.UserInfo)
			oidcGroup.POST("/userinfo", middleware.OIDCUserInfoRateLimit(), a.oidcHandler.UserInfo)
			oidcGroup.POST("/introspect", middleware.OIDCIntrospectRateLimit(), a.oidcHandler.Introspect)
			oidcGroup.POST("/revoke", middleware.OIDCRevokeRateLimit(), a.oidcHandler.Revoke)
			oidcGroup.GET("/end_session", a.oidcHandler.EndSession)
			oidcGroup.POST("/end_session", a.oidcHandler.EndSession)
		}

		// Admin OIDC client management (JSON API, protected by Admin API key)
		adminOIDC := r.Group("/admin/oidc/apps/:id/clients")
		adminOIDC.Use(a.adminAuth())
		{
			adminOIDC.POST("", requireOp(operator.ResOIDC, operator.ActionWrite), a.oidcHandler.AdminCreateClient)
			adminOIDC.GET("", requireOp(operator.ResOIDC, operator.ActionRead), a.oidcHandler.AdminListClients)
			adminOIDC.GET("/:cid", requireOp(operator.ResOIDC, operator.ActionRead), a.oidcHandler.AdminGetClient)
			adminOIDC.PUT("/:cid", requireOp(operator.ResOIDC, operator.ActionWrite), a.oidcHandler.AdminUpdateClient)
			adminOIDC.DELETE("/:cid", requireOp(operator.ResOIDC, operator.ActionWrite), a.oidcHandler.AdminDeleteClient)
			adminOIDC.POST("/:cid/rotate-secret", requireOp(operator.ResOIDC, operator.ActionWrite), a.oidcHandler.AdminRotateClientSecret)
		}
	}

	// Start session group expiry detection service
	a.expiryService = sessiongroup.NewExpiryService(a.sessionGroupRevoker, sessiongroup.NewConfig(
		cfg.Session.GroupExpiryEnabled,
		cfg.Session.GroupExpiryScanInterval,
		cfg.Session.GroupKeyspaceNotifEnabled,
		cfg.Session.RedisNotifyKeyspaceEvents,
	))
	a.expiryService.Start()
}

func (a *App) adminAuth() gin.HandlerFunc {
	return middleware.AdminAuthMiddleware(a.config.Admin.APIKey, a.adminRepo, a.operatorRepo)
}

func requireOp(resource, action string) gin.HandlerFunc {
	return middleware.RequireOperatorPermission(resource, action)
}

// Close performs graceful shutdown of all background services and, if the pool
// was created by New(), closes the database connection pool.
func (a *App) Close() {
	if a.apiKeyNotificationSvc != nil {
		a.apiKeyNotificationSvc.Shutdown()
	}
	if a.webhookService != nil {
		a.webhookService.Shutdown()
	}
	if a.cleanupService != nil {
		a.cleanupService.Shutdown()
	}
	if a.expiryService != nil {
		a.expiryService.Stop()
	}
	if a.ownsPool && a.pool != nil {
		a.pool.Close()
	}
}

func isRemoteAssetURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func readBrandingFile(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path is AdminBrandingConfig LogoURL/FaviconURL, validated by validateBrandingFile at app.New.
	if err != nil {
		return nil, "", err
	}
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".svg"):
		return data, "image/svg+xml", nil
	case strings.HasSuffix(lower, ".ico"):
		return data, "image/x-icon", nil
	default:
		return data, http.DetectContentType(data), nil
	}
}
