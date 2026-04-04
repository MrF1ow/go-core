package geoip

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/JedidiahDigital/go-core/pkg/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IPRuleRepository handles database operations for IP rules
type IPRuleRepository struct {
	pool *pgxpool.Pool
}

// NewIPRuleRepository creates a new IP rule repository
func NewIPRuleRepository(pool *pgxpool.Pool) *IPRuleRepository {
	return &IPRuleRepository{pool: pool}
}

// scanIPRule scans a single IPRule from a pgx row.
func scanIPRule(row pgx.Row) (*models.IPRule, error) {
	var rule models.IPRule
	err := row.Scan(
		&rule.ID,
		&rule.AppID,
		&rule.RuleType,
		&rule.MatchType,
		&rule.Value,
		&rule.Description,
		&rule.IsActive,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// scanIPRules scans multiple IPRule rows.
func scanIPRules(rows pgx.Rows) ([]models.IPRule, error) {
	var rules []models.IPRule
	for rows.Next() {
		var rule models.IPRule
		if err := rows.Scan(
			&rule.ID,
			&rule.AppID,
			&rule.RuleType,
			&rule.MatchType,
			&rule.Value,
			&rule.Description,
			&rule.IsActive,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// ListByApp retrieves all active IP rules for an application
func (r *IPRuleRepository) ListByApp(appID uuid.UUID) ([]models.IPRule, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, app_id, rule_type, match_type, value, description, is_active, created_at, updated_at
		 FROM ip_rules WHERE app_id = $1 AND is_active = true
		 ORDER BY rule_type ASC, created_at ASC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIPRules(rows)
}

// ListAllByApp retrieves all IP rules for an application (including inactive)
func (r *IPRuleRepository) ListAllByApp(appID uuid.UUID) ([]models.IPRule, error) {
	rows, err := r.pool.Query(context.Background(),
		`SELECT id, app_id, rule_type, match_type, value, description, is_active, created_at, updated_at
		 FROM ip_rules WHERE app_id = $1
		 ORDER BY rule_type ASC, created_at DESC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIPRules(rows)
}

// GetByID retrieves a specific IP rule by ID.
// Returns nil, nil when the rule does not exist.
func (r *IPRuleRepository) GetByID(id uuid.UUID) (*models.IPRule, error) {
	row := r.pool.QueryRow(context.Background(),
		`SELECT id, app_id, rule_type, match_type, value, description, is_active, created_at, updated_at
		 FROM ip_rules WHERE id = $1`, id)
	rule, err := scanIPRule(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return rule, err
}

// Create creates a new IP rule
func (r *IPRuleRepository) Create(rule *models.IPRule) error {
	return r.pool.QueryRow(context.Background(),
		`INSERT INTO ip_rules (app_id, rule_type, match_type, value, description, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, app_id, rule_type, match_type, value, description, is_active, created_at, updated_at`,
		rule.AppID, rule.RuleType, rule.MatchType, rule.Value, rule.Description, rule.IsActive,
	).Scan(
		&rule.ID, &rule.AppID, &rule.RuleType, &rule.MatchType, &rule.Value,
		&rule.Description, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt,
	)
}

// Update updates an existing IP rule
func (r *IPRuleRepository) Update(rule *models.IPRule) error {
	_, err := r.pool.Exec(context.Background(),
		`UPDATE ip_rules SET rule_type = $1, match_type = $2, value = $3, description = $4, is_active = $5, updated_at = NOW()
		 WHERE id = $6`,
		rule.RuleType, rule.MatchType, rule.Value, rule.Description, rule.IsActive, rule.ID)
	return err
}

// Delete deletes an IP rule by ID
func (r *IPRuleRepository) Delete(id uuid.UUID) error {
	_, err := r.pool.Exec(context.Background(),
		`DELETE FROM ip_rules WHERE id = $1`, id)
	return err
}

// IPRuleEvaluator evaluates IP rules against client IPs for access control.
// It caches rules per application in memory with a configurable TTL.
type IPRuleEvaluator struct {
	repo    *IPRuleRepository
	geoip   *Service
	cache   map[uuid.UUID]*cachedRules
	cacheMu sync.RWMutex
	ttl     time.Duration
}

type cachedRules struct {
	rules     []models.IPRule
	fetchedAt time.Time
}

// AccessResult contains the result of an IP access evaluation
type AccessResult struct {
	Allowed     bool     `json:"allowed"`
	Reason      string   `json:"reason"`                 // Human-readable reason for the decision
	MatchedRule string   `json:"matched_rule,omitempty"` // ID of the rule that matched (if any)
	Country     string   `json:"country,omitempty"`      // Resolved country code for the IP
	GeoInfo     *GeoInfo `json:"geo_info,omitempty"`     // Full geographic info (if available)
}

// NewIPRuleEvaluator creates a new IP rule evaluator with caching
func NewIPRuleEvaluator(repo *IPRuleRepository, geoip *Service) *IPRuleEvaluator {
	return &IPRuleEvaluator{
		repo:  repo,
		geoip: geoip,
		cache: make(map[uuid.UUID]*cachedRules),
		ttl:   5 * time.Minute,
	}
}

// EvaluateAccess checks whether a client IP is allowed to access a given application.
//
// Evaluation logic:
//  1. Load active rules for the app (cached for 5 minutes)
//  2. If no rules exist, access is allowed (default open)
//  3. If allowlist rules exist: IP must match at least one allow rule
//  4. If only blocklist rules exist: IP must NOT match any block rule
//  5. For country rules: GeoIP lookup resolves the IP to a country code
func (e *IPRuleEvaluator) EvaluateAccess(appID uuid.UUID, clientIP string) AccessResult {
	rules := e.getRules(appID)

	if len(rules) == 0 {
		return AccessResult{Allowed: true, Reason: "no_rules_configured"}
	}

	// Resolve GeoIP info for the client (may be nil if service is unavailable)
	var geoInfo *GeoInfo
	if e.geoip != nil {
		geoInfo = e.geoip.Lookup(clientIP)
	}

	// Separate rules by type
	var allowRules, blockRules []models.IPRule
	for _, rule := range rules {
		switch rule.RuleType {
		case models.IPRuleTypeAllow:
			allowRules = append(allowRules, rule)
		case models.IPRuleTypeBlock:
			blockRules = append(blockRules, rule)
		}
	}

	country := ""
	if geoInfo != nil {
		country = geoInfo.Country
	}

	// If allowlist rules exist, IP must match at least one
	if len(allowRules) > 0 {
		for _, rule := range allowRules {
			if e.matchesRule(rule, clientIP, country) {
				return AccessResult{
					Allowed:     true,
					Reason:      "matched_allowlist",
					MatchedRule: rule.ID.String(),
					Country:     country,
					GeoInfo:     geoInfo,
				}
			}
		}
		// No allowlist match found - deny
		return AccessResult{
			Allowed: false,
			Reason:  "not_in_allowlist",
			Country: country,
			GeoInfo: geoInfo,
		}
	}

	// Only blocklist rules exist: check if IP matches any
	for _, rule := range blockRules {
		if e.matchesRule(rule, clientIP, country) {
			return AccessResult{
				Allowed:     false,
				Reason:      "matched_blocklist",
				MatchedRule: rule.ID.String(),
				Country:     country,
				GeoInfo:     geoInfo,
			}
		}
	}

	// No block rules matched - allow
	return AccessResult{
		Allowed: true,
		Reason:  "not_in_blocklist",
		Country: country,
		GeoInfo: geoInfo,
	}
}

// InvalidateCache clears the cached rules for an application.
// Call this when IP rules are created, updated, or deleted.
func (e *IPRuleEvaluator) InvalidateCache(appID uuid.UUID) {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	delete(e.cache, appID)
}

// getRules retrieves rules for an app from cache or database
func (e *IPRuleEvaluator) getRules(appID uuid.UUID) []models.IPRule {
	e.cacheMu.RLock()
	cached, exists := e.cache[appID]
	e.cacheMu.RUnlock()

	if exists && time.Since(cached.fetchedAt) < e.ttl {
		return cached.rules
	}

	// Fetch from database
	rules, err := e.repo.ListByApp(appID)
	if err != nil {
		log.Printf("IPRuleEvaluator: Failed to load rules for app %s: %v", appID, err)
		// On error, return empty (fail-open). In a high-security environment,
		// you might want fail-closed instead.
		return nil
	}

	// Update cache
	e.cacheMu.Lock()
	e.cache[appID] = &cachedRules{
		rules:     rules,
		fetchedAt: time.Now(),
	}
	e.cacheMu.Unlock()

	return rules
}

// matchesRule checks if a client IP matches a specific rule
func (e *IPRuleEvaluator) matchesRule(rule models.IPRule, clientIP, clientCountry string) bool {
	switch rule.MatchType {
	case models.IPMatchTypeIP:
		return clientIP == rule.Value

	case models.IPMatchTypeCIDR:
		return e.matchesCIDR(clientIP, rule.Value)

	case models.IPMatchTypeCountry:
		if clientCountry == "" {
			return false // Cannot match country without GeoIP data
		}
		return strings.EqualFold(clientCountry, rule.Value)

	default:
		return false
	}
}

// matchesCIDR checks if an IP address falls within a CIDR range
func (e *IPRuleEvaluator) matchesCIDR(ipStr, cidr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}

	return network.Contains(ip)
}

// ValidateRule validates an IP rule before creation/update
func ValidateRule(rule *models.IPRule) error {
	// Validate rule type
	if rule.RuleType != models.IPRuleTypeAllow && rule.RuleType != models.IPRuleTypeBlock {
		return fmt.Errorf("invalid rule_type: must be '%s' or '%s'", models.IPRuleTypeAllow, models.IPRuleTypeBlock)
	}

	// Validate match type and value
	switch rule.MatchType {
	case models.IPMatchTypeIP:
		if strings.Contains(rule.Value, "/") {
			return fmt.Errorf("value contains '/' — did you mean to select 'CIDR Range' as the match type? For a single IP use e.g. 192.168.1.1")
		}
		if net.ParseIP(rule.Value) == nil {
			return fmt.Errorf("invalid IP address: %s (expected format: 192.168.1.1 or 2001:db8::1)", rule.Value)
		}

	case models.IPMatchTypeCIDR:
		if !strings.Contains(rule.Value, "/") {
			return fmt.Errorf("value has no CIDR prefix (e.g. /24) — did you mean to select 'IP Address' as the match type? For a range use e.g. 10.0.0.0/8")
		}
		_, _, err := net.ParseCIDR(rule.Value)
		if err != nil {
			return fmt.Errorf("invalid CIDR notation: %s (%v)", rule.Value, err)
		}

	case models.IPMatchTypeCountry:
		// Basic validation: 2-letter ISO country code
		if len(rule.Value) != 2 {
			return fmt.Errorf("invalid country code: must be a 2-letter ISO code (e.g. US, DE, JP)")
		}
		rule.Value = strings.ToUpper(rule.Value)

	default:
		return fmt.Errorf("invalid match_type: must be '%s', '%s', or '%s'",
			models.IPMatchTypeIP, models.IPMatchTypeCIDR, models.IPMatchTypeCountry)
	}

	return nil
}
