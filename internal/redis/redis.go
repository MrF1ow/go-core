package redis

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	core "github.com/JedidiahDigital/go-core"
	goredis "github.com/go-redis/redis/v8"
)

var store core.CacheStore
var ctx = context.Background()

// Init sets the cache store used by all token/session operations.
func Init(s core.CacheStore) {
	store = s
}

// IsInitialized reports whether a cache store has been configured.
func IsInitialized() bool {
	return store != nil
}

// GetRedisClient returns the underlying *redis.Client if the store is Redis-backed, or nil.
// This is needed for Redis-specific features like PubSub that cannot be abstracted behind CacheStore.
func GetRedisClient() *goredis.Client {
	type clientGetter interface {
		Client() *goredis.Client
	}
	if cg, ok := store.(clientGetter); ok {
		return cg.Client()
	}
	return nil
}

// Ping checks the health of the underlying cache store.
func Ping() error {
	if store == nil {
		return fmt.Errorf("cache store not initialized")
	}
	return store.Ping(ctx)
}

// SetRefreshToken stores a refresh token with its expiration
func SetRefreshToken(appID, userID, token string, expiration time.Duration) error {
	key := fmt.Sprintf("app:%s:refresh_token:%s", appID, userID)
	return store.Set(ctx, key, token, expiration)
}

// GetRefreshToken retrieves a refresh token
func GetRefreshToken(appID, userID string) (string, error) {
	key := fmt.Sprintf("app:%s:refresh_token:%s", appID, userID)
	return store.Get(ctx, key)
}

// RevokeRefreshToken deletes a refresh token (effectively blacklisting it)
func RevokeRefreshToken(appID, userID, token string) error {
	key := fmt.Sprintf("app:%s:refresh_token:%s", appID, userID)
	val, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return nil // Token already gone or never existed
	} else if err != nil {
		return err
	}

	if val == token {
		return store.Delete(ctx, key)
	}
	return nil // Token found but doesn't match, might be an older token
}

// IsRefreshTokenRevoked checks if a refresh token is revoked (by checking if it exists)
func IsRefreshTokenRevoked(appID, userID, token string) (bool, error) {
	key := fmt.Sprintf("app:%s:refresh_token:%s", appID, userID)
	val, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return true, nil // Token not found, so it's considered revoked or expired
	} else if err != nil {
		return false, err
	}
	return val != token, nil // If value doesn't match, it means a new token was issued, old one is implicitly revoked
}

// SetEmailVerificationToken stores an email verification token and a reverse lookup key (userID → token).
// The reverse lookup allows invalidating old tokens when a new one is issued.
func SetEmailVerificationToken(appID, userID, token string, expiration time.Duration) error {
	key := fmt.Sprintf("app:%s:email_verify:%s", appID, token)
	if err := store.Set(ctx, key, userID, expiration); err != nil {
		return err
	}
	// Store reverse lookup: userID → token (so we can find and invalidate old tokens)
	reverseKey := fmt.Sprintf("app:%s:email_verify_user:%s", appID, userID)
	return store.Set(ctx, reverseKey, token, expiration)
}

// GetEmailVerificationToken retrieves an email verification token
func GetEmailVerificationToken(appID, token string) (string, error) {
	key := fmt.Sprintf("app:%s:email_verify:%s", appID, token)
	return store.Get(ctx, key)
}

// GetEmailVerificationTokenByUserID retrieves the current verification token for a user (reverse lookup).
func GetEmailVerificationTokenByUserID(appID, userID string) (string, error) {
	key := fmt.Sprintf("app:%s:email_verify_user:%s", appID, userID)
	return store.Get(ctx, key)
}

// DeleteEmailVerificationToken deletes an email verification token and its reverse lookup key.
func DeleteEmailVerificationToken(appID, token string) error {
	key := fmt.Sprintf("app:%s:email_verify:%s", appID, token)
	// Look up the userID so we can also clean up the reverse key
	userID, err := store.Get(ctx, key)
	if err == nil && userID != "" {
		reverseKey := fmt.Sprintf("app:%s:email_verify_user:%s", appID, userID)
		if err := store.Delete(ctx, reverseKey); err != nil {
			log.Printf("Warning: Failed to delete email verification reverse lookup %s: %v", reverseKey, err)
		}
	}
	return store.Delete(ctx, key)
}

// SetPasswordResetToken stores a password reset token
func SetPasswordResetToken(appID, userID, token string, expiration time.Duration) error {
	key := fmt.Sprintf("app:%s:password_reset:%s", appID, token)
	return store.Set(ctx, key, userID, expiration)
}

// GetPasswordResetToken retrieves a password reset token
func GetPasswordResetToken(appID, token string) (string, error) {
	key := fmt.Sprintf("app:%s:password_reset:%s", appID, token)
	return store.Get(ctx, key)
}

// DeletePasswordResetToken deletes a password reset token
func DeletePasswordResetToken(appID, token string) error {
	key := fmt.Sprintf("app:%s:password_reset:%s", appID, token)
	return store.Delete(ctx, key)
}

// Magic Link related functions

// SetMagicLinkToken stores a magic link token and a reverse lookup key (userID → token).
// The reverse lookup allows invalidating old tokens when a new one is issued.
func SetMagicLinkToken(appID, userID, token string, expiration time.Duration) error {
	// Invalidate any existing magic link token for this user (only one active at a time)
	reverseKey := fmt.Sprintf("app:%s:magic_link_user:%s", appID, userID)
	oldToken, err := store.Get(ctx, reverseKey)
	if err == nil && oldToken != "" {
		oldKey := fmt.Sprintf("app:%s:magic_link:%s", appID, oldToken)
		if err := store.Delete(ctx, oldKey); err != nil {
			log.Printf("Warning: Failed to delete old magic link token for user %s in app %s: %v", userID, appID, err)
		}
	}

	// Store token → userID mapping
	key := fmt.Sprintf("app:%s:magic_link:%s", appID, token)
	if err := store.Set(ctx, key, userID, expiration); err != nil {
		return err
	}
	// Store reverse lookup: userID → token
	return store.Set(ctx, reverseKey, token, expiration)
}

// GetMagicLinkToken retrieves the userID associated with a magic link token
func GetMagicLinkToken(appID, token string) (string, error) {
	key := fmt.Sprintf("app:%s:magic_link:%s", appID, token)
	return store.Get(ctx, key)
}

// DeleteMagicLinkToken deletes a magic link token and its reverse lookup key (single-use).
func DeleteMagicLinkToken(appID, token string) error {
	key := fmt.Sprintf("app:%s:magic_link:%s", appID, token)
	// Look up the userID so we can also clean up the reverse key
	userID, err := store.Get(ctx, key)
	if err == nil && userID != "" {
		reverseKey := fmt.Sprintf("app:%s:magic_link_user:%s", appID, userID)
		if err := store.Delete(ctx, reverseKey); err != nil {
			log.Printf("Warning: Failed to delete magic link reverse lookup %s: %v", reverseKey, err)
		}
	}
	return store.Delete(ctx, key)
}

// 2FA related functions

// SetTempTwoFASecret stores a temporary 2FA secret during setup
func SetTempTwoFASecret(appID, userID, secret string, expiration time.Duration) error {
	key := fmt.Sprintf("app:%s:temp_2fa_secret:%s", appID, userID)
	return store.Set(ctx, key, secret, expiration)
}

// GetTempTwoFASecret retrieves a temporary 2FA secret
func GetTempTwoFASecret(appID, userID string) (string, error) {
	key := fmt.Sprintf("app:%s:temp_2fa_secret:%s", appID, userID)
	return store.Get(ctx, key)
}

// DeleteTempTwoFASecret deletes a temporary 2FA secret
func DeleteTempTwoFASecret(appID, userID string) error {
	key := fmt.Sprintf("app:%s:temp_2fa_secret:%s", appID, userID)
	return store.Delete(ctx, key)
}

// SetTempUserSession stores a temporary user session for 2FA login
func SetTempUserSession(appID, tempToken, userID string, expiration time.Duration) error {
	key := fmt.Sprintf("app:%s:temp_session:%s", appID, tempToken)
	return store.Set(ctx, key, userID, expiration)
}

// GetTempUserSession retrieves a temporary user session
func GetTempUserSession(appID, tempToken string) (string, error) {
	key := fmt.Sprintf("app:%s:temp_session:%s", appID, tempToken)
	return store.Get(ctx, key)
}

// DeleteTempUserSession deletes a temporary user session
func DeleteTempUserSession(appID, tempToken string) error {
	key := fmt.Sprintf("app:%s:temp_session:%s", appID, tempToken)
	return store.Delete(ctx, key)
}

// Access Token Blacklisting Functions

// BlacklistAccessToken adds an access token to the blacklist with its remaining TTL
func BlacklistAccessToken(appID, tokenString string, userID string, expiration time.Duration) error {
	key := fmt.Sprintf("app:%s:blacklist_token:%s", appID, tokenString)
	return store.Set(ctx, key, userID, expiration)
}

// IsAccessTokenBlacklisted checks if an access token is blacklisted
func IsAccessTokenBlacklisted(appID, tokenString string) (bool, error) {
	key := fmt.Sprintf("app:%s:blacklist_token:%s", appID, tokenString)
	_, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return false, nil // Token not found in blacklist
	} else if err != nil {
		return false, err // Cache error
	}
	return true, nil // Token found in blacklist
}

// BlacklistAllUserTokens blacklists all tokens for a specific user (useful for password changes, account compromise)
func BlacklistAllUserTokens(appID, userID string, expiration time.Duration) error {
	key := fmt.Sprintf("app:%s:blacklist_user:%s", appID, userID)
	return store.Set(ctx, key, "all_tokens_revoked", expiration)
}

// IsUserTokensBlacklisted checks if all tokens for a user are blacklisted
func IsUserTokensBlacklisted(appID, userID string) (bool, error) {
	key := fmt.Sprintf("app:%s:blacklist_user:%s", appID, userID)
	_, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return false, nil // User tokens not blacklisted
	} else if err != nil {
		return false, err // Cache error
	}
	return true, nil // All user tokens are blacklisted
}

// ClearUserTokenBlacklist removes the user-wide token blacklist entry.
// Called when a user successfully authenticates with fresh credentials (e.g. new login
// after a password reset) so that newly issued tokens are not blocked by the stale
// post-reset blacklist.
func ClearUserTokenBlacklist(appID, userID string) error {
	key := fmt.Sprintf("app:%s:blacklist_user:%s", appID, userID)
	return store.Delete(ctx, key)
}

// ==================== Session Management Functions ====================

// CreateSession stores a new session as a hash with metadata.
// Key pattern: app:{appID}:session:{sessionID}
// Also adds the sessionID to the user's session index set.
func CreateSession(appID, sessionID, userID, refreshToken, ip, userAgent string, ttl time.Duration) error {
	key := fmt.Sprintf("app:%s:session:%s", appID, sessionID)
	fields := map[string]any{
		"user_id":       userID,
		"refresh_token": refreshToken,
		"ip":            ip,
		"user_agent":    userAgent,
		"created_at":    time.Now().UTC().Format(time.RFC3339),
		"last_active":   time.Now().UTC().Format(time.RFC3339),
	}
	if err := store.HSet(ctx, key, fields); err != nil {
		return err
	}
	if err := store.Expire(ctx, key, ttl); err != nil {
		return err
	}
	// Add to user session index
	indexKey := fmt.Sprintf("app:%s:user_sessions:%s", appID, userID)
	if err := store.SAdd(ctx, indexKey, sessionID); err != nil {
		return err
	}
	// Set a generous TTL on the index (longer than any single session) to prevent stale keys
	if err := store.Expire(ctx, indexKey, ttl+24*time.Hour); err != nil {
		log.Printf("Warning: Failed to set TTL on user session index %s: %v", indexKey, err)
	}

	// Add to app-level session index (for admin dashboard enumeration)
	appIndexKey := fmt.Sprintf("app:%s:all_sessions", appID)
	if err := store.SAdd(ctx, appIndexKey, sessionID); err != nil {
		log.Printf("Warning: Failed to add session %s to app index %s: %v", sessionID, appIndexKey, err)
	}
	if err := store.Expire(ctx, appIndexKey, ttl+24*time.Hour); err != nil {
		log.Printf("Warning: Failed to set TTL on app session index %s: %v", appIndexKey, err)
	}

	// Store session metadata for expiration detection
	metaKey := fmt.Sprintf("session_meta:%s:%s:%s", appID, userID, sessionID)
	if err := store.Set(ctx, metaKey, "1", ttl); err != nil {
		// Log but don't fail session creation
		log.Printf("Warning: Failed to create session metadata key: %v", err)
	}

	return nil
}

// GetSession retrieves all fields of a session hash.
func GetSession(appID, sessionID string) (map[string]string, error) {
	key := fmt.Sprintf("app:%s:session:%s", appID, sessionID)
	result, err := store.HGetAll(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, core.ErrCacheKeyNotFound
	}
	return result, nil
}

// GetSessionRefreshToken retrieves only the refresh_token field from a session.
func GetSessionRefreshToken(appID, sessionID string) (string, error) {
	key := fmt.Sprintf("app:%s:session:%s", appID, sessionID)
	return store.HGet(ctx, key, "refresh_token")
}

// UpdateSessionRefreshToken updates the refresh token stored in a session hash.
func UpdateSessionRefreshToken(appID, sessionID, newRefreshToken string) error {
	key := fmt.Sprintf("app:%s:session:%s", appID, sessionID)
	return store.HSet(ctx, key, map[string]any{"refresh_token": newRefreshToken})
}

// ResetSessionTTL resets the TTL on a session hash key.
// Call this on every token rotation so the session lifetime slides forward
// with the newly issued refresh token instead of expiring at the original login time.
func ResetSessionTTL(appID, sessionID string, ttl time.Duration) error {
	key := fmt.Sprintf("app:%s:session:%s", appID, sessionID)
	return store.Expire(ctx, key, ttl)
}

// TouchSession updates the last_active timestamp of a session.
func TouchSession(appID, sessionID string) error {
	key := fmt.Sprintf("app:%s:session:%s", appID, sessionID)
	return store.HSet(ctx, key, map[string]any{"last_active": time.Now().UTC().Format(time.RFC3339)})
}

// DeleteSession removes a session hash and removes it from the user and app session indexes.
func DeleteSession(appID, sessionID, userID string) error {
	key := fmt.Sprintf("app:%s:session:%s", appID, sessionID)
	if err := store.Delete(ctx, key); err != nil {
		return err
	}
	// Remove from user session index
	indexKey := fmt.Sprintf("app:%s:user_sessions:%s", appID, userID)
	if err := store.SRem(ctx, indexKey, sessionID); err != nil {
		log.Printf("Warning: Failed to remove session %s from user index %s: %v", sessionID, indexKey, err)
	}
	// Remove from app-level session index
	appIndexKey := fmt.Sprintf("app:%s:all_sessions", appID)
	if err := store.SRem(ctx, appIndexKey, sessionID); err != nil {
		log.Printf("Warning: Failed to remove session %s from app index %s: %v", sessionID, appIndexKey, err)
	}
	// Delete session metadata key
	metaKey := fmt.Sprintf("session_meta:%s:%s:%s", appID, userID, sessionID)
	if err := store.Delete(ctx, metaKey); err != nil {
		log.Printf("Warning: Failed to delete session metadata key %s: %v", metaKey, err)
	}
	return nil
}

// GetUserSessionIDs returns all session IDs for a user from the session index set.
// It performs lazy cleanup: any session ID in the set that no longer exists is removed.
func GetUserSessionIDs(appID, userID string) ([]string, error) {
	indexKey := fmt.Sprintf("app:%s:user_sessions:%s", appID, userID)
	sessionIDs, err := store.SMembers(ctx, indexKey)
	if err != nil {
		return nil, err
	}

	// Lazy cleanup: verify each session still exists
	var validIDs []string
	for _, sid := range sessionIDs {
		sessionKey := fmt.Sprintf("app:%s:session:%s", appID, sid)
		exists, err := store.Exists(ctx, sessionKey)
		if err != nil {
			continue // Skip on error, don't remove
		}
		if !exists {
			// Session expired, remove from index
			if err := store.SRem(ctx, indexKey, sid); err != nil {
				log.Printf("Warning: Failed to remove expired session %s from user index %s: %v", sid, indexKey, err)
			}
			continue
		}
		validIDs = append(validIDs, sid)
	}

	return validIDs, nil
}

// DeleteAllUserSessions removes all sessions for a user except the one specified by exceptSessionID.
// If exceptSessionID is empty, all sessions are removed.
func DeleteAllUserSessions(appID, userID, exceptSessionID string) error {
	sessionIDs, err := GetUserSessionIDs(appID, userID)
	if err != nil {
		return err
	}

	for _, sid := range sessionIDs {
		if sid == exceptSessionID {
			continue
		}
		sessionKey := fmt.Sprintf("app:%s:session:%s", appID, sid)
		if err := store.Delete(ctx, sessionKey); err != nil {
			log.Printf("Warning: Failed to delete session %s during bulk cleanup: %v", sessionKey, err)
		}
		// Remove from app-level session index
		appIndexKey := fmt.Sprintf("app:%s:all_sessions", appID)
		if err := store.SRem(ctx, appIndexKey, sid); err != nil {
			log.Printf("Warning: Failed to remove session %s from app index %s during bulk cleanup: %v", sid, appIndexKey, err)
		}
	}

	// Clean up the index
	indexKey := fmt.Sprintf("app:%s:user_sessions:%s", appID, userID)
	if exceptSessionID == "" {
		if err := store.Delete(ctx, indexKey); err != nil {
			log.Printf("Warning: Failed to delete user session index %s: %v", indexKey, err)
		}
	} else {
		// Rebuild the set with only the excepted session
		if err := store.Delete(ctx, indexKey); err != nil {
			log.Printf("Warning: Failed to delete user session index %s for rebuild: %v", indexKey, err)
		}
		if err := store.SAdd(ctx, indexKey, exceptSessionID); err != nil {
			log.Printf("Warning: Failed to re-add excepted session %s to index %s: %v", exceptSessionID, indexKey, err)
		}
	}

	return nil
}

// SessionExists checks whether a session hash key exists.
func SessionExists(appID, sessionID string) (bool, error) {
	key := fmt.Sprintf("app:%s:session:%s", appID, sessionID)
	return store.Exists(ctx, key)
}

// GetAppSessionIDs returns all session IDs for an app from the app-level session index.
// Performs lazy cleanup: removes IDs whose session hash has expired.
func GetAppSessionIDs(appID string) ([]string, error) {
	indexKey := fmt.Sprintf("app:%s:all_sessions", appID)
	sessionIDs, err := store.SMembers(ctx, indexKey)
	if err != nil {
		return nil, err
	}

	var validIDs []string
	for _, sid := range sessionIDs {
		sessionKey := fmt.Sprintf("app:%s:session:%s", appID, sid)
		exists, err := store.Exists(ctx, sessionKey)
		if err != nil {
			continue
		}
		if !exists {
			if err := store.SRem(ctx, indexKey, sid); err != nil {
				log.Printf("Warning: Failed to remove expired session %s from app index %s: %v", sid, indexKey, err)
			}
			continue
		}
		validIDs = append(validIDs, sid)
	}
	return validIDs, nil
}

// CountAppSessions returns the count of active sessions for an app.
func CountAppSessions(appID string) (int64, error) {
	indexKey := fmt.Sprintf("app:%s:all_sessions", appID)
	members, err := store.SMembers(ctx, indexKey)
	if err != nil {
		return 0, err
	}
	return int64(len(members)), nil
}

// GetAllSessionsForApp returns full session metadata for all active sessions in an app.
// Each returned map contains: session_id, user_id, ip, user_agent, created_at, last_active.
// The refresh_token field is intentionally excluded for security.
func GetAllSessionsForApp(appID string) ([]map[string]string, error) {
	sessionIDs, err := GetAppSessionIDs(appID)
	if err != nil {
		return nil, err
	}

	var sessions []map[string]string
	for _, sid := range sessionIDs {
		data, err := GetSession(appID, sid)
		if err != nil {
			continue
		}
		data["session_id"] = sid
		// Remove refresh_token from admin-visible data
		delete(data, "refresh_token")
		sessions = append(sessions, data)
	}
	return sessions, nil
}

// Admin Session Functions

// SetAdminSession stores an admin session
func SetAdminSession(sessionID, adminID string, expiration time.Duration) error {
	key := fmt.Sprintf("admin:session:%s", sessionID)
	return store.Set(ctx, key, adminID, expiration)
}

// GetAdminSession retrieves an admin session, returning the admin ID
func GetAdminSession(sessionID string) (string, error) {
	key := fmt.Sprintf("admin:session:%s", sessionID)
	return store.Get(ctx, key)
}

// DeleteAdminSession removes an admin session
func DeleteAdminSession(sessionID string) error {
	key := fmt.Sprintf("admin:session:%s", sessionID)
	return store.Delete(ctx, key)
}

// Admin CSRF Functions

// SetCSRFToken stores a CSRF token for an admin session
func SetCSRFToken(sessionID, token string, expiration time.Duration) error {
	key := fmt.Sprintf("admin:csrf:%s", sessionID)
	return store.Set(ctx, key, token, expiration)
}

// GetCSRFToken retrieves the CSRF token for an admin session
func GetCSRFToken(sessionID string) (string, error) {
	key := fmt.Sprintf("admin:csrf:%s", sessionID)
	return store.Get(ctx, key)
}

// Admin Login Rate Limiting Functions

// IncrLoginAttempts increments the login attempt counter for an IP and sets a 60-second TTL.
// Returns the new count after increment.
func IncrLoginAttempts(ip string) (int64, error) {
	key := fmt.Sprintf("admin:login_attempts:%s", ip)
	return store.Increment(ctx, key, 60*time.Second)
}

// GetLoginAttempts returns the current login attempt count for an IP
func GetLoginAttempts(ip string) (int64, error) {
	key := fmt.Sprintf("admin:login_attempts:%s", ip)
	val, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// SetLoginLockout sets a lockout flag for an IP with the given expiration
func SetLoginLockout(ip string, expiration time.Duration) error {
	key := fmt.Sprintf("admin:login_lockout:%s", ip)
	return store.Set(ctx, key, "locked", expiration)
}

// IsLoginLocked checks if an IP is currently locked out
func IsLoginLocked(ip string) (bool, error) {
	key := fmt.Sprintf("admin:login_lockout:%s", ip)
	_, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// ClearLoginAttempts removes the attempt counter and lockout for an IP (called on successful login)
func ClearLoginAttempts(ip string) error {
	attemptsKey := fmt.Sprintf("admin:login_attempts:%s", ip)
	lockoutKey := fmt.Sprintf("admin:login_lockout:%s", ip)
	return store.Delete(ctx, attemptsKey, lockoutKey)
}

// Email 2FA Code Functions

// Set2FAEmailCode stores a 2FA email verification code with a 5-minute expiration.
func Set2FAEmailCode(appID, userID, code string) error {
	key := fmt.Sprintf("app:%s:2fa_email:%s", appID, userID)
	return store.Set(ctx, key, code, 5*time.Minute)
}

// Get2FAEmailCode retrieves a stored 2FA email verification code.
func Get2FAEmailCode(appID, userID string) (string, error) {
	key := fmt.Sprintf("app:%s:2fa_email:%s", appID, userID)
	return store.Get(ctx, key)
}

// Delete2FAEmailCode removes a 2FA email verification code after successful verification.
func Delete2FAEmailCode(appID, userID string) error {
	key := fmt.Sprintf("app:%s:2fa_email:%s", appID, userID)
	return store.Delete(ctx, key)
}

// ClearRateLimitKeys removes the generic rate-limit attempt counter and lockout
// for a given prefix + identifier. Used by the generic RateLimitMiddleware.
func ClearRateLimitKeys(keyPrefix, identifier string) error {
	attemptsKey := fmt.Sprintf("rl:%s:attempts:%s", keyPrefix, identifier)
	lockoutKey := fmt.Sprintf("rl:%s:lockout:%s", keyPrefix, identifier)
	return store.Delete(ctx, attemptsKey, lockoutKey)
}

// WebAuthn Challenge Functions

// SetWebAuthnRegistrationChallenge stores a WebAuthn registration challenge session.
func SetWebAuthnRegistrationChallenge(appID, userID, sessionJSON string, expiration time.Duration) error {
	key := fmt.Sprintf("app:%s:webauthn_reg:%s", appID, userID)
	return store.Set(ctx, key, sessionJSON, expiration)
}

// GetWebAuthnRegistrationChallenge retrieves a WebAuthn registration challenge session.
func GetWebAuthnRegistrationChallenge(appID, userID string) (string, error) {
	key := fmt.Sprintf("app:%s:webauthn_reg:%s", appID, userID)
	return store.Get(ctx, key)
}

// DeleteWebAuthnRegistrationChallenge removes a WebAuthn registration challenge session.
func DeleteWebAuthnRegistrationChallenge(appID, userID string) error {
	key := fmt.Sprintf("app:%s:webauthn_reg:%s", appID, userID)
	return store.Delete(ctx, key)
}

// SetWebAuthnLoginChallenge stores a WebAuthn login/assertion challenge session.
// The identifier can be a userID (for 2FA) or a sessionID (for passwordless).
func SetWebAuthnLoginChallenge(appID, identifier, sessionJSON string, expiration time.Duration) error {
	key := fmt.Sprintf("app:%s:webauthn_login:%s", appID, identifier)
	return store.Set(ctx, key, sessionJSON, expiration)
}

// GetWebAuthnLoginChallenge retrieves a WebAuthn login/assertion challenge session.
func GetWebAuthnLoginChallenge(appID, identifier string) (string, error) {
	key := fmt.Sprintf("app:%s:webauthn_login:%s", appID, identifier)
	return store.Get(ctx, key)
}

// DeleteWebAuthnLoginChallenge removes a WebAuthn login/assertion challenge session.
func DeleteWebAuthnLoginChallenge(appID, identifier string) error {
	key := fmt.Sprintf("app:%s:webauthn_login:%s", appID, identifier)
	return store.Delete(ctx, key)
}

// Admin 2FA Functions

// SetAdmin2FATempSecret stores a temporary TOTP secret during admin 2FA setup (10-minute TTL).
func SetAdmin2FATempSecret(adminID, secret string) error {
	key := fmt.Sprintf("admin:2fa_temp_secret:%s", adminID)
	return store.Set(ctx, key, secret, 10*time.Minute)
}

// GetAdmin2FATempSecret retrieves a temporary TOTP secret during admin 2FA setup.
func GetAdmin2FATempSecret(adminID string) (string, error) {
	key := fmt.Sprintf("admin:2fa_temp_secret:%s", adminID)
	return store.Get(ctx, key)
}

// DeleteAdmin2FATempSecret removes the temporary TOTP secret after setup is complete.
func DeleteAdmin2FATempSecret(adminID string) error {
	key := fmt.Sprintf("admin:2fa_temp_secret:%s", adminID)
	return store.Delete(ctx, key)
}

// SetAdmin2FATempSession stores a partial login session awaiting 2FA verification (10-minute TTL).
// The value is the admin account ID.
func SetAdmin2FATempSession(tempToken, adminID string) error {
	key := fmt.Sprintf("admin:2fa_temp_session:%s", tempToken)
	return store.Set(ctx, key, adminID, 10*time.Minute)
}

// GetAdmin2FATempSession retrieves the admin ID from a temporary 2FA login session.
func GetAdmin2FATempSession(tempToken string) (string, error) {
	key := fmt.Sprintf("admin:2fa_temp_session:%s", tempToken)
	return store.Get(ctx, key)
}

// DeleteAdmin2FATempSession removes a temporary 2FA login session after verification.
func DeleteAdmin2FATempSession(tempToken string) error {
	key := fmt.Sprintf("admin:2fa_temp_session:%s", tempToken)
	return store.Delete(ctx, key)
}

// SetAdmin2FAEmailCode stores a 2FA email verification code for an admin (5-minute TTL).
func SetAdmin2FAEmailCode(adminID, code string) error {
	key := fmt.Sprintf("admin:2fa_email:%s", adminID)
	return store.Set(ctx, key, code, 5*time.Minute)
}

// GetAdmin2FAEmailCode retrieves a stored 2FA email verification code for an admin.
func GetAdmin2FAEmailCode(adminID string) (string, error) {
	key := fmt.Sprintf("admin:2fa_email:%s", adminID)
	return store.Get(ctx, key)
}

// DeleteAdmin2FAEmailCode removes a 2FA email verification code after successful verification.
func DeleteAdmin2FAEmailCode(adminID string) error {
	key := fmt.Sprintf("admin:2fa_email:%s", adminID)
	return store.Delete(ctx, key)
}

// Admin Magic Link Functions

// SetAdminMagicLinkToken stores a magic link token and a reverse lookup key (adminID → token).
// The reverse lookup allows invalidating old tokens when a new one is issued.
func SetAdminMagicLinkToken(adminID, token string, expiration time.Duration) error {
	// Invalidate any existing magic link token for this admin (only one active at a time)
	reverseKey := fmt.Sprintf("admin:magic_link_user:%s", adminID)
	oldToken, err := store.Get(ctx, reverseKey)
	if err == nil && oldToken != "" {
		oldKey := fmt.Sprintf("admin:magic_link:%s", oldToken)
		if err := store.Delete(ctx, oldKey); err != nil {
			log.Printf("Warning: Failed to delete old admin magic link token for admin %s: %v", adminID, err)
		}
	}

	// Store token → adminID mapping
	key := fmt.Sprintf("admin:magic_link:%s", token)
	if err := store.Set(ctx, key, adminID, expiration); err != nil {
		return err
	}
	// Store reverse lookup: adminID → token
	return store.Set(ctx, reverseKey, token, expiration)
}

// GetAdminMagicLinkToken retrieves the adminID associated with a magic link token.
func GetAdminMagicLinkToken(token string) (string, error) {
	key := fmt.Sprintf("admin:magic_link:%s", token)
	return store.Get(ctx, key)
}

// DeleteAdminMagicLinkToken deletes a magic link token and its reverse lookup key (single-use).
func DeleteAdminMagicLinkToken(token string) error {
	key := fmt.Sprintf("admin:magic_link:%s", token)
	// Look up the adminID so we can also clean up the reverse key
	adminID, err := store.Get(ctx, key)
	if err == nil && adminID != "" {
		reverseKey := fmt.Sprintf("admin:magic_link_user:%s", adminID)
		if err := store.Delete(ctx, reverseKey); err != nil {
			log.Printf("Warning: Failed to delete admin magic link reverse lookup %s: %v", reverseKey, err)
		}
	}
	return store.Delete(ctx, key)
}

// ==================== Failed Login Tracking (Brute-Force Detection) ====================

// IncrFailedLogin increments the failed login counter for a given app + identifier (email or IP).
// The counter auto-expires after the given window duration.
// Returns the new count after increment.
func IncrFailedLogin(appID, identifier string, window time.Duration) (int64, error) {
	key := fmt.Sprintf("app:%s:failed_login:%s", appID, identifier)
	return store.Increment(ctx, key, window)
}

// GetFailedLoginCount returns the current failed login count for a given app + identifier.
func GetFailedLoginCount(appID, identifier string) (int64, error) {
	key := fmt.Sprintf("app:%s:failed_login:%s", appID, identifier)
	val, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// ResetFailedLogins clears the failed login counter for a given app + identifier.
// Call this on successful login.
func ResetFailedLogins(appID, identifier string) error {
	key := fmt.Sprintf("app:%s:failed_login:%s", appID, identifier)
	return store.Delete(ctx, key)
}

// ==================== Notification Cooldown ====================

// SetNotificationCooldown sets a cooldown flag to prevent spamming notification emails.
// Key pattern: notify_cooldown:{appID}:{userID}:{notificationType}
func SetNotificationCooldown(appID, userID, notificationType string, cooldown time.Duration) error {
	key := fmt.Sprintf("notify_cooldown:%s:%s:%s", appID, userID, notificationType)
	return store.Set(ctx, key, "1", cooldown)
}

// IsNotificationOnCooldown checks whether a notification cooldown is active for a user.
func IsNotificationOnCooldown(appID, userID, notificationType string) (bool, error) {
	key := fmt.Sprintf("notify_cooldown:%s:%s:%s", appID, userID, notificationType)
	_, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// ==================== Account Lockout Tier Tracking ====================

// IncrLockoutTier increments the lockout tier for a given app + email and sets the TTL.
// The tier determines which escalating lockout duration to use (e.g., tier 0 = 15m, tier 1 = 30m, etc.).
// Returns the new tier value (1-based after increment).
func IncrLockoutTier(appID, email string, ttl time.Duration) (int64, error) {
	key := fmt.Sprintf("app:%s:lockout_tier:%s", appID, email)
	tier, err := store.Increment(ctx, key, ttl)
	if err != nil {
		return 0, err
	}
	// Always refresh TTL on each lockout so the tier escalation window resets
	if err := store.Expire(ctx, key, ttl); err != nil {
		log.Printf("Warning: Failed to refresh TTL on lockout tier key %s: %v", key, err)
	}
	return tier, nil
}

// GetLockoutTier returns the current lockout tier for a given app + email.
// Returns 0 if no tier is set (user has not been locked out recently).
func GetLockoutTier(appID, email string) (int64, error) {
	key := fmt.Sprintf("app:%s:lockout_tier:%s", appID, email)
	val, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// ResetLockoutTier clears the lockout tier for a given app + email.
// Called by admin when manually unlocking an account.
func ResetLockoutTier(appID, email string) error {
	key := fmt.Sprintf("app:%s:lockout_tier:%s", appID, email)
	return store.Delete(ctx, key)
}

// ==================== Progressive Delay Tier Tracking ====================

// IncrDelayTier increments the delay tier for a given app + identifier (email or IP).
// The tier determines the exponential backoff delay applied before login processing.
// Returns the new tier value (1-based after increment).
func IncrDelayTier(appID, identifier string, ttl time.Duration) (int64, error) {
	key := fmt.Sprintf("app:%s:delay_tier:%s", appID, identifier)
	return store.Increment(ctx, key, ttl)
}

// GetDelayTier returns the current delay tier for a given app + identifier.
// Returns 0 if no tier is set (no recent failed attempts).
func GetDelayTier(appID, identifier string) (int64, error) {
	key := fmt.Sprintf("app:%s:delay_tier:%s", appID, identifier)
	val, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// ResetDelayTier clears the delay tier for a given app + identifier.
// Called on successful login to reset progressive delays.
func ResetDelayTier(appID, identifier string) error {
	key := fmt.Sprintf("app:%s:delay_tier:%s", appID, identifier)
	return store.Delete(ctx, key)
}

// ─── OIDC browser session (login cookie) ───────────────────────────────────────

// SetOIDCBrowserSession stores an opaque session token → userID mapping used by
// the OIDC login cookie. The token is a random value, never the user UUID.
func SetOIDCBrowserSession(appID, sessionToken, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("app:%s:oidc_browser:%s", appID, sessionToken)
	return store.Set(ctx, key, userID, ttl)
}

// GetOIDCBrowserSession resolves an opaque OIDC browser session token to a userID.
// Returns ("", nil) when the session does not exist.
func GetOIDCBrowserSession(appID, sessionToken string) (string, error) {
	key := fmt.Sprintf("app:%s:oidc_browser:%s", appID, sessionToken)
	val, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return "", nil
	}
	return val, err
}

// DeleteOIDCBrowserSession removes the OIDC browser session (e.g. on logout).
func DeleteOIDCBrowserSession(appID, sessionToken string) error {
	key := fmt.Sprintf("app:%s:oidc_browser:%s", appID, sessionToken)
	return store.Delete(ctx, key)
}

// ==================== Backup Email Verification ====================

// SetBackupEmailVerificationToken stores a token → (userID, pendingEmail) mapping used during
// backup email verification. The token is a random URL-safe value emailed to the backup address.
func SetBackupEmailVerificationToken(appID, userID, token, pendingEmail string, expiration time.Duration) error {
	// token → "userID|pendingEmail"
	key := fmt.Sprintf("app:%s:backup_email_verify:%s", appID, token)
	value := userID + "|" + pendingEmail
	return store.Set(ctx, key, value, expiration)
}

// GetBackupEmailVerificationToken retrieves the userID and pending email for a backup email verification token.
func GetBackupEmailVerificationToken(appID, token string) (userID, pendingEmail string, err error) {
	key := fmt.Sprintf("app:%s:backup_email_verify:%s", appID, token)
	val, err := store.Get(ctx, key)
	if err != nil {
		return "", "", err
	}
	// Split on first "|" only
	idx := strings.Index(val, "|")
	if idx < 0 {
		return val, "", nil
	}
	return val[:idx], val[idx+1:], nil
}

// DeleteBackupEmailVerificationToken removes a backup email verification token after use.
func DeleteBackupEmailVerificationToken(appID, token string) error {
	key := fmt.Sprintf("app:%s:backup_email_verify:%s", appID, token)
	return store.Delete(ctx, key)
}

// ==================== SMS / Phone Verification Codes ====================

// SetPhoneVerificationCode stores a 6-digit code used to verify a new phone number.
func SetPhoneVerificationCode(appID, userID, code string, expiration time.Duration) error {
	key := fmt.Sprintf("app:%s:phone_verify:%s", appID, userID)
	return store.Set(ctx, key, code, expiration)
}

// GetPhoneVerificationCode retrieves a phone verification code.
func GetPhoneVerificationCode(appID, userID string) (string, error) {
	key := fmt.Sprintf("app:%s:phone_verify:%s", appID, userID)
	return store.Get(ctx, key)
}

// DeletePhoneVerificationCode removes a phone verification code after successful use.
func DeletePhoneVerificationCode(appID, userID string) error {
	key := fmt.Sprintf("app:%s:phone_verify:%s", appID, userID)
	return store.Delete(ctx, key)
}

// Set2FASMSCode stores a 6-digit SMS 2FA / recovery code during login (5-minute TTL).
func Set2FASMSCode(appID, userID, code string) error {
	key := fmt.Sprintf("app:%s:2fa_sms:%s", appID, userID)
	return store.Set(ctx, key, code, 5*time.Minute)
}

// Get2FASMSCode retrieves a stored SMS 2FA code.
func Get2FASMSCode(appID, userID string) (string, error) {
	key := fmt.Sprintf("app:%s:2fa_sms:%s", appID, userID)
	return store.Get(ctx, key)
}

// Delete2FASMSCode removes an SMS 2FA code after successful verification (one-time use).
func Delete2FASMSCode(appID, userID string) error {
	key := fmt.Sprintf("app:%s:2fa_sms:%s", appID, userID)
	return store.Delete(ctx, key)
}

// SetBackupEmail2FACode stores a 6-digit code sent to the backup email during login (5-minute TTL).
func SetBackupEmail2FACode(appID, userID, code string) error {
	key := fmt.Sprintf("app:%s:2fa_backup_email:%s", appID, userID)
	return store.Set(ctx, key, code, 5*time.Minute)
}

// GetBackupEmail2FACode retrieves a stored backup-email 2FA code.
func GetBackupEmail2FACode(appID, userID string) (string, error) {
	key := fmt.Sprintf("app:%s:2fa_backup_email:%s", appID, userID)
	return store.Get(ctx, key)
}

// DeleteBackupEmail2FACode removes a backup-email 2FA code after successful verification.
func DeleteBackupEmail2FACode(appID, userID string) error {
	key := fmt.Sprintf("app:%s:2fa_backup_email:%s", appID, userID)
	return store.Delete(ctx, key)
}

// ─── OIDC granted scopes (per session) ─────────────────────────────────────────

// SetOIDCGrantedScopes stores the space-separated scopes that were granted for
// a given OIDC session. Used by the UserInfo endpoint to gate which claims are
// returned without embedding scopes in the JWT itself.
func SetOIDCGrantedScopes(appID, sessionID, scopes string, ttl time.Duration) error {
	key := fmt.Sprintf("app:%s:oidc_scopes:%s", appID, sessionID)
	return store.Set(ctx, key, scopes, ttl)
}

// GetOIDCGrantedScopes retrieves the space-separated scopes for an OIDC session.
// Returns ("", nil) when not found (e.g. token issued before this feature).
func GetOIDCGrantedScopes(appID, sessionID string) (string, error) {
	key := fmt.Sprintf("app:%s:oidc_scopes:%s", appID, sessionID)
	val, err := store.Get(ctx, key)
	if err == core.ErrCacheKeyNotFound {
		return "", nil
	}
	return val, err
}

// ============================================================================
// Account Merge Token helpers
//
// A merge token is a short-lived (15 min) Redis entry that stores all the
// information needed to link a social provider account to an existing user
// account.  It is created when a social-login callback detects that the
// provider email matches an existing user who does not yet have that social
// account linked, so the frontend can prompt the user to confirm the merge
// by supplying their existing password.
//
// Key layout: app:{appID}:merge_token:{mergeToken}  →  JSON-encoded payload
// ============================================================================

// SetMergeToken stores a merge token with a JSON-encoded payload and the given TTL.
func SetMergeToken(appID, mergeToken, payload string, expiration time.Duration) error {
	key := fmt.Sprintf("app:%s:merge_token:%s", appID, mergeToken)
	return store.Set(ctx, key, payload, expiration)
}

// GetMergeToken retrieves the JSON payload for a merge token.
// Returns ("", core.ErrCacheKeyNotFound) when the token does not exist or has expired.
func GetMergeToken(appID, mergeToken string) (string, error) {
	key := fmt.Sprintf("app:%s:merge_token:%s", appID, mergeToken)
	return store.Get(ctx, key)
}

// DeleteMergeToken removes a merge token after it has been consumed.
func DeleteMergeToken(appID, mergeToken string) error {
	key := fmt.Sprintf("app:%s:merge_token:%s", appID, mergeToken)
	return store.Delete(ctx, key)
}

// ============================================================================
// SSO (Shared Session) Token helpers
//
// An SSO token is a short-lived (60 s), single-use opaque token that encodes
// the source app, the session group, and the authenticated user.  It is issued
// by POST /sso/token and consumed exactly once by POST /sso/exchange on the
// target application to mint a new app-scoped token pair without re-auth.
//
// Key layout: sso:token:{token}  →  "{groupID}|{sourceAppID}|{userID}"
// ============================================================================

const ssoTokenTTL = 60 * time.Second

// SetSSOToken stores a new SSO exchange token with a 60-second TTL.
func SetSSOToken(token, groupID, sourceAppID, userID string) error {
	key := fmt.Sprintf("sso:token:%s", token)
	value := groupID + "|" + sourceAppID + "|" + userID
	return store.Set(ctx, key, value, ssoTokenTTL)
}

// GetSSOToken retrieves the group, source app, and user encoded in an SSO token.
// Returns core.ErrCacheKeyNotFound when the token does not exist or has expired.
func GetSSOToken(token string) (groupID, sourceAppID, userID string, err error) {
	key := fmt.Sprintf("sso:token:%s", token)
	val, err := store.Get(ctx, key)
	if err != nil {
		return "", "", "", err
	}
	parts := strings.SplitN(val, "|", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("malformed SSO token value")
	}
	return parts[0], parts[1], parts[2], nil
}

// DeleteSSOToken removes an SSO token after it has been consumed (single-use).
func DeleteSSOToken(token string) error {
	key := fmt.Sprintf("sso:token:%s", token)
	return store.Delete(ctx, key)
}

// ============================================================================
// Session Metadata Functions for Expiration Detection
// ============================================================================

// ParseSessionMetaKey extracts appID, userID, and sessionID from a session_meta key
func ParseSessionMetaKey(metaKey string) (appID, userID, sessionID string, err error) {
	// Remove the "session_meta:" prefix
	if !strings.HasPrefix(metaKey, "session_meta:") {
		return "", "", "", fmt.Errorf("not a session_meta key")
	}

	parts := strings.Split(metaKey[len("session_meta:"):], ":")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("malformed session_meta key")
	}

	return parts[0], parts[1], parts[2], nil
}

// GetExpiredSessionMetaKeys returns all session_meta keys that have expired (TTL <= 0)
func GetExpiredSessionMetaKeys() ([]string, error) {
	var expiredKeys []string

	// Use SCAN to find all session_meta keys
	var cursor uint64
	for {
		keys, nextCursor, err := store.Scan(ctx, cursor, "session_meta:*", 100)
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			// Check TTL
			ttl, err := store.TTL(ctx, key)
			if err != nil {
				continue
			}
			// TTL <= 0 means expired or no TTL
			if ttl <= 0 {
				expiredKeys = append(expiredKeys, key)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return expiredKeys, nil
}
