// Package coreapp provides the application entry point for go-core.
//
// It lives in internal/ rather than the root package because the root package
// exports types (CacheStore, Config, etc.) that internal packages import.
// Placing the wiring here avoids circular dependencies.
package coreapp

import (
	"context"
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
	adminHandler.OperatorRoles = operatorRepo
	middleware.SetOperatorAccessLogger(func(rec operator.AccessRecord) {
		go func() {
			if err := operatorRepo.InsertAccessLog(context.Background(), rec); err != nil {
				log.Printf("operator access log insert: %v", err)
			}
		}()
	})

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
	adminHandler.Accounts = accountRepo
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
	guiHandler.AbortForbidden = middleware.AbortGUIForbidden
	guiHandler.AbortInternal = middleware.AbortGUIInternal

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
		adminRoutes.GET("/operator/access-logs", requireOp(operator.ResAdminIAM, operator.ActionRead), a.adminHandler.OperatorAccessLogs)
		adminRoutes.GET("/operator/access-logs/export", requireOp(operator.ResAdminIAM, operator.ActionRead), a.adminHandler.OperatorAccessLogsExport)
		adminRoutes.GET("/operator/iam-events", requireOp(operator.ResAdminIAM, operator.ActionRead), a.adminHandler.OperatorIAMEvents)
		adminRoutes.GET("/operator/iam-events/export", requireOp(operator.ResAdminIAM, operator.ActionRead), a.adminHandler.OperatorIAMEventsExport)
		adminRoutes.PUT("/operator/keys/:id/role", requireOp(operator.ResAdminIAM, operator.ActionWrite), a.adminHandler.OperatorKeyRole)
		adminRoutes.POST("/operator/accounts", requireOp(operator.ResAdminIAM, operator.ActionWrite), a.adminHandler.OperatorCreateAccount)
		adminRoutes.PUT("/operator/accounts/:id/role", requireOp(operator.ResAdminIAM, operator.ActionWrite), a.adminHandler.OperatorAccountRole)
		adminRoutes.POST("/operator/accounts/:id/disable", requireOp(operator.ResAdminIAM, operator.ActionWrite), a.adminHandler.OperatorDisableAccount)

		adminRoutes.GET("/operator/roster", requireOp(operator.ResAdminIAM, operator.ActionRead), a.adminHandler.OperatorRoster)
		adminRoutes.GET("/operator/roster/export", requireOp(operator.ResAdminIAM, operator.ActionRead), a.adminHandler.OperatorRosterExport)

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
		guiAuth.Use(middleware.GUIAuthMiddleware(a.accountService, a.operatorRepo, adminBasePath))
		guiAuth.Use(middleware.CSRFMiddleware(a.accountService))
		{
			guiAuth.GET("/", requireGUI(operator.ResDashboard, operator.ActionRead), a.guiHandler.Dashboard)
			guiAuth.GET("/dashboard/stats", requireGUI(operator.ResDashboard, operator.ActionRead), a.guiHandler.DashboardStats)
			guiAuth.GET("/dashboard/activity", requireGUI(operator.ResLogs, operator.ActionRead), a.guiHandler.DashboardActivity)
			guiAuth.GET("/logout", a.guiHandler.Logout)

			// Tenant management
			guiAuth.GET("/tenants", requireGUI(operator.ResTenants, operator.ActionRead), a.guiHandler.TenantPage)
			guiAuth.GET("/tenants/list", requireGUI(operator.ResTenants, operator.ActionRead), a.guiHandler.TenantList)
			guiAuth.GET("/tenants/new", requireGUI(operator.ResTenants, operator.ActionWrite), a.guiHandler.TenantCreateForm)
			guiAuth.POST("/tenants", requireGUI(operator.ResTenants, operator.ActionWrite), a.guiHandler.TenantCreate)
			guiAuth.GET("/tenants/form-cancel", requireGUI(operator.ResTenants, operator.ActionRead), a.guiHandler.TenantFormCancel)
			guiAuth.GET("/tenants/:id/edit", requireGUI(operator.ResTenants, operator.ActionWrite), a.guiHandler.TenantEditForm)
			guiAuth.PUT("/tenants/:id", requireGUI(operator.ResTenants, operator.ActionWrite), a.guiHandler.TenantUpdate)
			guiAuth.GET("/tenants/:id/delete", requireGUI(operator.ResTenants, operator.ActionWrite), a.guiHandler.TenantDeleteConfirm)
			guiAuth.DELETE("/tenants/:id", requireGUI(operator.ResTenants, operator.ActionWrite), a.guiHandler.TenantDelete)

			// Application management
			guiAuth.GET("/applications", requireGUI(operator.ResApplications, operator.ActionRead), a.guiHandler.AppPage)
			guiAuth.GET("/applications/list", requireGUI(operator.ResApplications, operator.ActionRead), a.guiHandler.AppList)
			guiAuth.GET("/applications/new", requireGUI(operator.ResApplications, operator.ActionWrite), a.guiHandler.AppCreateForm)
			guiAuth.POST("/applications", requireGUI(operator.ResApplications, operator.ActionWrite), a.guiHandler.AppCreate)
			guiAuth.GET("/applications/form-cancel", requireGUI(operator.ResApplications, operator.ActionRead), a.guiHandler.AppFormCancel)
			guiAuth.GET("/applications/:id/edit", requireGUI(operator.ResApplications, operator.ActionWrite), a.guiHandler.AppEditForm)
			guiAuth.PUT("/applications/:id", requireGUI(operator.ResApplications, operator.ActionWrite), a.guiHandler.AppUpdate)
			guiAuth.GET("/applications/:id/delete", requireGUI(operator.ResApplications, operator.ActionWrite), a.guiHandler.AppDeleteConfirm)
			guiAuth.DELETE("/applications/:id", requireGUI(operator.ResApplications, operator.ActionWrite), a.guiHandler.AppDelete)

			// OAuth config management
			guiAuth.GET("/oauth", requireGUI(operator.ResOAuth, operator.ActionRead), a.guiHandler.OAuthPage)
			guiAuth.GET("/oauth/list", requireGUI(operator.ResOAuth, operator.ActionRead), a.guiHandler.OAuthList)
			guiAuth.GET("/oauth/new", requireGUI(operator.ResOAuth, operator.ActionWrite), a.guiHandler.OAuthCreateForm)
			guiAuth.POST("/oauth", requireGUI(operator.ResOAuth, operator.ActionWrite), a.guiHandler.OAuthCreate)
			guiAuth.GET("/oauth/form-cancel", requireGUI(operator.ResOAuth, operator.ActionRead), a.guiHandler.OAuthFormCancel)
			guiAuth.GET("/oauth/:id/edit", requireGUI(operator.ResOAuth, operator.ActionWrite), a.guiHandler.OAuthEditForm)
			guiAuth.PUT("/oauth/:id", requireGUI(operator.ResOAuth, operator.ActionWrite), a.guiHandler.OAuthUpdate)
			guiAuth.GET("/oauth/:id/delete", requireGUI(operator.ResOAuth, operator.ActionWrite), a.guiHandler.OAuthDeleteConfirm)
			guiAuth.DELETE("/oauth/:id", requireGUI(operator.ResOAuth, operator.ActionWrite), a.guiHandler.OAuthDelete)
			guiAuth.PUT("/oauth/:id/toggle", requireGUI(operator.ResOAuth, operator.ActionWrite), a.guiHandler.OAuthToggleEnabled)

			// User management
			guiAuth.GET("/users", requireGUI(operator.ResUsers, operator.ActionRead), a.guiHandler.UserPage)
			guiAuth.GET("/users/list", requireGUI(operator.ResUsers, operator.ActionRead), a.guiHandler.UserList)
			guiAuth.GET("/users/export", requireGUI(operator.ResUsers, operator.ActionRead), a.guiHandler.UserExport)
			guiAuth.GET("/users/import/modal", requireGUI(operator.ResUsers, operator.ActionWrite), a.guiHandler.UserImportModal)
			guiAuth.POST("/users/import", requireGUI(operator.ResUsers, operator.ActionWrite), a.guiHandler.UserImport)
			guiAuth.GET("/users/:id", requireGUI(operator.ResUsers, operator.ActionRead), a.guiHandler.UserDetail)
			guiAuth.PUT("/users/:id/toggle", requireGUI(operator.ResUsers, operator.ActionWrite), a.guiHandler.UserToggleActive)
			guiAuth.PUT("/users/:id/unlock", requireGUI(operator.ResUsers, operator.ActionWrite), a.guiHandler.UserUnlock)
			guiAuth.GET("/users/social-accounts/:id/unlink", requireGUI(operator.ResUsers, operator.ActionWrite), a.guiHandler.SocialAccountUnlinkConfirm)
			guiAuth.DELETE("/users/social-accounts/:id", requireGUI(operator.ResUsers, operator.ActionWrite), a.guiHandler.SocialAccountUnlink)
			guiAuth.GET("/users/passkeys/:id/delete", requireGUI(operator.ResUsers, operator.ActionWrite), a.guiHandler.PasskeyDeleteConfirm)
			guiAuth.DELETE("/users/passkeys/:id", requireGUI(operator.ResUsers, operator.ActionWrite), a.guiHandler.PasskeyDelete)
			guiAuth.DELETE("/users/:id/trusted-devices/:device_id", requireGUI(operator.ResUsers, operator.ActionWrite), a.guiHandler.UserRevokeTrustedDevice)
			guiAuth.DELETE("/users/:id/trusted-devices", requireGUI(operator.ResUsers, operator.ActionWrite), a.guiHandler.UserRevokeAllTrustedDevices)

			// Activity logs viewer
			guiAuth.GET("/logs", requireGUI(operator.ResLogs, operator.ActionRead), a.guiHandler.LogsPage)
			guiAuth.GET("/logs/list", requireGUI(operator.ResLogs, operator.ActionRead), a.guiHandler.LogList)
			guiAuth.GET("/logs/export", requireGUI(operator.ResLogs, operator.ActionRead), a.guiHandler.LogExport)
			guiAuth.GET("/logs/:id", requireGUI(operator.ResLogs, operator.ActionRead), a.guiHandler.LogDetail)

			// API key management
			guiAuth.GET("/api-keys", requireGUI(operator.ResAPIKeys, operator.ActionRead), a.guiHandler.ApiKeysPage)
			guiAuth.GET("/api-keys/list", requireGUI(operator.ResAPIKeys, operator.ActionRead), a.guiHandler.ApiKeyList)
			guiAuth.GET("/api-keys/new", requireGUI(operator.ResAPIKeys, operator.ActionWrite), a.guiHandler.ApiKeyCreateForm)
			guiAuth.POST("/api-keys", requireGUI(operator.ResAPIKeys, operator.ActionWrite), a.guiHandler.ApiKeyCreate)
			guiAuth.GET("/api-keys/form-cancel", requireGUI(operator.ResAPIKeys, operator.ActionRead), a.guiHandler.ApiKeyFormCancel)
			guiAuth.GET("/api-keys/:id/edit", requireGUI(operator.ResAPIKeys, operator.ActionWrite), a.guiHandler.ApiKeyEditForm)
			guiAuth.PUT("/api-keys/:id", requireGUI(operator.ResAPIKeys, operator.ActionWrite), a.guiHandler.ApiKeyUpdate)
			guiAuth.GET("/api-keys/:id/usage", requireGUI(operator.ResAPIKeys, operator.ActionRead), a.guiHandler.ApiKeyUsagePage)
			guiAuth.GET("/api-keys/:id/revoke", requireGUI(operator.ResAPIKeys, operator.ActionWrite), a.guiHandler.ApiKeyRevokeConfirm)
			guiAuth.PUT("/api-keys/:id/revoke", requireGUI(operator.ResAPIKeys, operator.ActionWrite), a.guiHandler.ApiKeyRevoke)
			guiAuth.GET("/api-keys/:id/delete", requireGUI(operator.ResAPIKeys, operator.ActionWrite), a.guiHandler.ApiKeyDeleteConfirm)
			guiAuth.DELETE("/api-keys/:id", requireGUI(operator.ResAPIKeys, operator.ActionWrite), a.guiHandler.ApiKeyDelete)

			// Settings management
			guiAuth.GET("/settings", requireGUI(operator.ResSettings, operator.ActionRead), a.guiHandler.SettingsPage)
			guiAuth.GET("/settings/info", requireGUI(operator.ResSettings, operator.ActionRead), a.guiHandler.SettingsInfo)
			guiAuth.GET("/settings/section/:category", requireGUI(operator.ResSettings, operator.ActionRead), a.guiHandler.SettingsSection)
			guiAuth.PUT("/settings/:key", requireGUI(operator.ResSettings, operator.ActionWrite), a.guiHandler.SettingUpdate)
			guiAuth.DELETE("/settings/:key", requireGUI(operator.ResSettings, operator.ActionWrite), a.guiHandler.SettingReset)

			// System health monitoring
			guiAuth.GET("/monitoring", requireGUI(operator.ResMonitoring, operator.ActionRead), a.guiHandler.MonitoringPage)
			guiAuth.GET("/monitoring/health", requireGUI(operator.ResMonitoring, operator.ActionRead), a.guiHandler.MonitoringHealth)
			guiAuth.GET("/monitoring/metrics", requireGUI(operator.ResMonitoring, operator.ActionRead), a.guiHandler.MonitoringMetrics)

			// Email server management
			guiAuth.GET("/email-servers", requireGUI(operator.ResEmail, operator.ActionRead), a.guiHandler.EmailServersPage)
			guiAuth.GET("/email-servers/list", requireGUI(operator.ResEmail, operator.ActionRead), a.guiHandler.EmailServerList)
			guiAuth.GET("/email-servers/new", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailServerCreateForm)
			guiAuth.POST("/email-servers", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailServerCreate)
			guiAuth.GET("/email-servers/form-cancel", requireGUI(operator.ResEmail, operator.ActionRead), a.guiHandler.EmailServerFormCancel)
			guiAuth.GET("/email-servers/:id/edit", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailServerEditForm)
			guiAuth.PUT("/email-servers/:id", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailServerUpdate)
			guiAuth.GET("/email-servers/:id/delete", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailServerDeleteConfirm)
			guiAuth.DELETE("/email-servers/:id", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailServerDelete)
			guiAuth.POST("/email-servers/:id/test", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailServerSendTest)

			// Email template management
			guiAuth.GET("/email-templates", requireGUI(operator.ResEmail, operator.ActionRead), a.guiHandler.EmailTemplatesPage)
			guiAuth.GET("/email-templates/list", requireGUI(operator.ResEmail, operator.ActionRead), a.guiHandler.EmailTemplateList)
			guiAuth.GET("/email-templates/new", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTemplateCreateForm)
			guiAuth.POST("/email-templates", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTemplateCreate)
			guiAuth.GET("/email-templates/form-cancel", requireGUI(operator.ResEmail, operator.ActionRead), a.guiHandler.EmailTemplateFormCancel)
			guiAuth.GET("/email-templates/:id/edit", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTemplateEditForm)
			guiAuth.PUT("/email-templates/:id", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTemplateUpdate)
			guiAuth.GET("/email-templates/:id/delete", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTemplateDeleteConfirm)
			guiAuth.DELETE("/email-templates/:id", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTemplateDelete)
			guiAuth.POST("/email-templates/preview", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTemplatePreview)
			guiAuth.POST("/email-templates/editor-window", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTemplateEditorWindow)
			guiAuth.GET("/email-templates/:id/reset", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTemplateResetConfirm)
			guiAuth.POST("/email-templates/:id/reset", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTemplateReset)

			// Email template variables
			guiAuth.GET("/email-variables", requireGUI(operator.ResEmail, operator.ActionRead), a.guiHandler.EmailVariablesList)

			// Email types management
			guiAuth.GET("/email-types", requireGUI(operator.ResEmail, operator.ActionRead), a.guiHandler.EmailTypesPage)
			guiAuth.GET("/email-types/list", requireGUI(operator.ResEmail, operator.ActionRead), a.guiHandler.EmailTypeList)
			guiAuth.GET("/email-types/new", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTypeCreateForm)
			guiAuth.POST("/email-types", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTypeCreate)
			guiAuth.GET("/email-types/form-cancel", requireGUI(operator.ResEmail, operator.ActionRead), a.guiHandler.EmailTypeFormCancel)
			guiAuth.GET("/email-types/:id/edit", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTypeEditForm)
			guiAuth.PUT("/email-types/:id", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTypeUpdate)
			guiAuth.GET("/email-types/:id/delete", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTypeDeleteConfirm)
			guiAuth.DELETE("/email-types/:id", requireGUI(operator.ResEmail, operator.ActionWrite), a.guiHandler.EmailTypeDelete)

			// Roles management
			guiAuth.GET("/roles", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.RolesPage)
			guiAuth.GET("/roles/list", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.RoleList)
			guiAuth.GET("/roles/new", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.RoleCreateForm)
			guiAuth.POST("/roles", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.RoleCreate)
			guiAuth.GET("/roles/form-cancel", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.RoleFormCancel)
			guiAuth.GET("/roles/:id/edit", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.RoleEditForm)
			guiAuth.PUT("/roles/:id", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.RoleUpdate)
			guiAuth.GET("/roles/:id/delete", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.RoleDeleteConfirm)
			guiAuth.DELETE("/roles/:id", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.RoleDelete)
			guiAuth.GET("/roles/:id/permissions", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.RolePermissions)
			guiAuth.PUT("/roles/:id/permissions", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.RolePermissionsUpdate)

			// Permissions management
			guiAuth.GET("/permissions", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.PermissionsPage)
			guiAuth.GET("/permissions/list", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.PermissionList)
			guiAuth.GET("/permissions/new", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.PermissionCreateForm)
			guiAuth.POST("/permissions", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.PermissionCreate)
			guiAuth.GET("/permissions/form-cancel", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.PermissionFormCancel)

			// User roles management
			guiAuth.GET("/user-roles", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.UserRolesPage)
			guiAuth.GET("/user-roles/list", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.UserRoleList)
			guiAuth.GET("/user-roles/new", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.UserRoleCreateForm)
			guiAuth.POST("/user-roles", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.UserRoleCreate)
			guiAuth.PUT("/user-roles", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.UserRoleUpdate)
			guiAuth.GET("/user-roles/roles-for-app", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.UserRoleRolesForApp)
			guiAuth.GET("/user-roles/search-users", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.UserRoleSearchUsers)
			guiAuth.GET("/user-roles/revoke", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.UserRoleRevokeConfirm)
			guiAuth.DELETE("/user-roles", requireGUI(operator.ResEndUserRBAC, operator.ActionWrite), a.guiHandler.UserRoleRevoke)
			guiAuth.GET("/user-roles/form-cancel", requireGUI(operator.ResEndUserRBAC, operator.ActionRead), a.guiHandler.UserRoleFormCancel)

			// Session management
			guiAuth.GET("/sessions", requireGUI(operator.ResSessions, operator.ActionRead), a.guiHandler.SessionsPage)
			guiAuth.GET("/sessions/list", requireGUI(operator.ResSessions, operator.ActionRead), a.guiHandler.SessionList)
			guiAuth.GET("/sessions/:app_id/:session_id/detail", requireGUI(operator.ResSessions, operator.ActionRead), a.guiHandler.SessionDetail)
			guiAuth.DELETE("/sessions/:app_id/:session_id", requireGUI(operator.ResSessions, operator.ActionWrite), a.guiHandler.SessionRevoke)
			guiAuth.DELETE("/sessions/revoke-all-user", requireGUI(operator.ResSessions, operator.ActionWrite), a.guiHandler.SessionRevokeAllForUser)
			guiAuth.GET("/users/:id/sessions", requireGUI(operator.ResSessions, operator.ActionRead), a.guiHandler.UserSessions)

			// IP Rule management
			guiAuth.GET("/ip-rules", requireGUI(operator.ResIPRules, operator.ActionRead), a.guiHandler.IPRulePage)
			guiAuth.GET("/ip-rules/list", requireGUI(operator.ResIPRules, operator.ActionRead), a.guiHandler.IPRuleList)
			guiAuth.GET("/ip-rules/new", requireGUI(operator.ResIPRules, operator.ActionWrite), a.guiHandler.IPRuleCreateForm)
			guiAuth.POST("/ip-rules", requireGUI(operator.ResIPRules, operator.ActionWrite), a.guiHandler.IPRuleCreate)
			guiAuth.GET("/ip-rules/form-cancel", requireGUI(operator.ResIPRules, operator.ActionRead), a.guiHandler.IPRuleFormCancel)
			guiAuth.GET("/ip-rules/:id/edit", requireGUI(operator.ResIPRules, operator.ActionWrite), a.guiHandler.IPRuleEditForm)
			guiAuth.PUT("/ip-rules/:id", requireGUI(operator.ResIPRules, operator.ActionWrite), a.guiHandler.IPRuleUpdate)
			guiAuth.GET("/ip-rules/:id/delete", requireGUI(operator.ResIPRules, operator.ActionWrite), a.guiHandler.IPRuleDeleteConfirm)
			guiAuth.DELETE("/ip-rules/:id", requireGUI(operator.ResIPRules, operator.ActionWrite), a.guiHandler.IPRuleDelete)
			guiAuth.POST("/ip-rules/check", requireGUI(operator.ResIPRules, operator.ActionWrite), a.guiHandler.IPRuleCheckAccess)

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
			guiAuth.GET("/webhooks", requireGUI(operator.ResWebhooks, operator.ActionRead), a.guiHandler.WebhookPage)
			guiAuth.GET("/webhooks/list", requireGUI(operator.ResWebhooks, operator.ActionRead), a.guiHandler.WebhookList)
			guiAuth.GET("/webhooks/new", requireGUI(operator.ResWebhooks, operator.ActionWrite), a.guiHandler.WebhookCreateForm)
			guiAuth.POST("/webhooks", requireGUI(operator.ResWebhooks, operator.ActionWrite), a.guiHandler.WebhookCreate)
			guiAuth.GET("/webhooks/form-cancel", requireGUI(operator.ResWebhooks, operator.ActionRead), a.guiHandler.WebhookFormCancel)
			guiAuth.GET("/webhooks/:id/delete", requireGUI(operator.ResWebhooks, operator.ActionWrite), a.guiHandler.WebhookDeleteConfirm)
			guiAuth.DELETE("/webhooks/:id", requireGUI(operator.ResWebhooks, operator.ActionWrite), a.guiHandler.WebhookDelete)
			guiAuth.PUT("/webhooks/:id/toggle", requireGUI(operator.ResWebhooks, operator.ActionWrite), a.guiHandler.WebhookToggle)
			guiAuth.GET("/webhooks/:id/deliveries", requireGUI(operator.ResWebhooks, operator.ActionRead), a.guiHandler.WebhookDeliveries)

			// OIDC client management (GUI)
			guiAuth.GET("/oidc-clients", requireGUI(operator.ResOIDC, operator.ActionRead), a.guiHandler.OIDCClientsPage)
			guiAuth.GET("/oidc-clients/list", requireGUI(operator.ResOIDC, operator.ActionRead), a.guiHandler.OIDCClientList)
			guiAuth.GET("/oidc-clients/new", requireGUI(operator.ResOIDC, operator.ActionWrite), a.guiHandler.OIDCClientCreateForm)
			guiAuth.POST("/oidc-clients", requireGUI(operator.ResOIDC, operator.ActionWrite), a.guiHandler.OIDCClientCreate)
			guiAuth.GET("/oidc-clients/form-cancel", requireGUI(operator.ResOIDC, operator.ActionRead), a.guiHandler.OIDCClientFormCancel)
			guiAuth.GET("/oidc-clients/:id/edit", requireGUI(operator.ResOIDC, operator.ActionWrite), a.guiHandler.OIDCClientEditForm)
			guiAuth.PUT("/oidc-clients/:id", requireGUI(operator.ResOIDC, operator.ActionWrite), a.guiHandler.OIDCClientUpdate)
			guiAuth.GET("/oidc-clients/:id/delete", requireGUI(operator.ResOIDC, operator.ActionWrite), a.guiHandler.OIDCClientDeleteConfirm)
			guiAuth.DELETE("/oidc-clients/:id", requireGUI(operator.ResOIDC, operator.ActionWrite), a.guiHandler.OIDCClientDelete)
			guiAuth.POST("/oidc-clients/:id/rotate-secret", requireGUI(operator.ResOIDC, operator.ActionWrite), a.guiHandler.OIDCClientRotateSecret)

			// Session Group management
			guiAuth.GET("/session-groups", requireGUI(operator.ResSessionGroups, operator.ActionRead), a.guiHandler.SessionGroupPage)
			guiAuth.GET("/session-groups/list", requireGUI(operator.ResSessionGroups, operator.ActionRead), a.guiHandler.SessionGroupList)
			guiAuth.GET("/session-groups/new", requireGUI(operator.ResSessionGroups, operator.ActionWrite), a.guiHandler.SessionGroupCreateForm)
			guiAuth.POST("/session-groups", requireGUI(operator.ResSessionGroups, operator.ActionWrite), a.guiHandler.SessionGroupCreate)
			guiAuth.GET("/session-groups/form-cancel", requireGUI(operator.ResSessionGroups, operator.ActionRead), a.guiHandler.SessionGroupFormCancel)
			guiAuth.GET("/session-groups/:id/edit", requireGUI(operator.ResSessionGroups, operator.ActionWrite), a.guiHandler.SessionGroupEditForm)
			guiAuth.PUT("/session-groups/:id", requireGUI(operator.ResSessionGroups, operator.ActionWrite), a.guiHandler.SessionGroupUpdate)
			guiAuth.GET("/session-groups/:id/delete", requireGUI(operator.ResSessionGroups, operator.ActionWrite), a.guiHandler.SessionGroupDeleteConfirm)
			guiAuth.DELETE("/session-groups/:id", requireGUI(operator.ResSessionGroups, operator.ActionWrite), a.guiHandler.SessionGroupDelete)
			guiAuth.GET("/session-groups/:id/apps", requireGUI(operator.ResSessionGroups, operator.ActionRead), a.guiHandler.SessionGroupApps)
			guiAuth.POST("/session-groups/:id/apps", requireGUI(operator.ResSessionGroups, operator.ActionWrite), a.guiHandler.SessionGroupAddApp)
			guiAuth.DELETE("/session-groups/:id/apps/:app_id", requireGUI(operator.ResSessionGroups, operator.ActionWrite), a.guiHandler.SessionGroupRemoveApp)
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

func requireGUI(resource, action string) gin.HandlerFunc {
	return middleware.RequireGUIPermission(resource, action)
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
