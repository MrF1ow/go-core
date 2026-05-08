package social

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MrF1ow/go-core/internal/sqlcgen"
	"github.com/MrF1ow/go-core/pkg/models"
)

type Repository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

func (r *Repository) CreateSocialAccount(socialAccount *models.SocialAccount) error {
	if socialAccount.ID == uuid.Nil {
		socialAccount.ID = uuid.New()
	}
	now := time.Now().UTC()
	if socialAccount.CreatedAt.IsZero() {
		socialAccount.CreatedAt = now
	}
	if socialAccount.UpdatedAt.IsZero() {
		socialAccount.UpdatedAt = now
	}
	return r.queries.CreateSocialAccount(context.Background(), sqlcgen.CreateSocialAccountParams{
		ID:             socialAccount.ID,
		AppID:          socialAccount.AppID,
		UserID:         socialAccount.UserID,
		Provider:       socialAccount.Provider,
		ProviderUserID: socialAccount.ProviderUserID,
		Email:          socialAccount.Email,
		Name:           socialAccount.Name,
		FirstName:      socialAccount.FirstName,
		LastName:       socialAccount.LastName,
		ProfilePicture: socialAccount.ProfilePicture,
		Username:       socialAccount.Username,
		Locale:         socialAccount.Locale,
		RawData:        []byte(socialAccount.RawData),
		AccessToken:    socialAccount.AccessToken,
		RefreshToken:   socialAccount.RefreshToken,
		ExpiresAt:      timePtrToTimestamptz(socialAccount.ExpiresAt),
		CreatedAt:      socialAccount.CreatedAt,
		UpdatedAt:      socialAccount.UpdatedAt,
	})
}

func (r *Repository) GetOAuthProviderConfig(appID string, provider string) (*models.OAuthProviderConfig, error) {
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetOAuthProviderConfig(context.Background(), sqlcgen.GetOAuthProviderConfigParams{
		AppID:    appUUID,
		Provider: provider,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	cfg := toModelOAuthProviderConfig(row)
	return &cfg, nil
}

func (r *Repository) GetSocialAccountByProviderAndUserID(appID, provider, providerUserID string) (*models.SocialAccount, error) {
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetSocialAccountByProviderAndUserID(context.Background(), sqlcgen.GetSocialAccountByProviderAndUserIDParams{
		AppID:          appUUID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	sa := toModelSocialAccount(row)
	return &sa, nil
}

func (r *Repository) GetSocialAccountsByUserID(userID string) ([]models.SocialAccount, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.GetSocialAccountsByUserID(context.Background(), uid)
	if err != nil {
		return nil, err
	}
	return toModelSocialAccounts(rows), nil
}

func (r *Repository) UpdateSocialAccount(socialAccount *models.SocialAccount) error {
	return r.queries.UpdateSocialAccount(context.Background(), sqlcgen.UpdateSocialAccountParams{
		ID:             socialAccount.ID,
		AppID:          socialAccount.AppID,
		UserID:         socialAccount.UserID,
		Provider:       socialAccount.Provider,
		ProviderUserID: socialAccount.ProviderUserID,
		Email:          socialAccount.Email,
		Name:           socialAccount.Name,
		FirstName:      socialAccount.FirstName,
		LastName:       socialAccount.LastName,
		ProfilePicture: socialAccount.ProfilePicture,
		Username:       socialAccount.Username,
		Locale:         socialAccount.Locale,
		RawData:        []byte(socialAccount.RawData),
		AccessToken:    socialAccount.AccessToken,
		RefreshToken:   socialAccount.RefreshToken,
		ExpiresAt:      timePtrToTimestamptz(socialAccount.ExpiresAt),
	})
}

func (r *Repository) UpdateSocialAccountTokens(id string, accessToken, refreshToken string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.UpdateSocialAccountTokens(context.Background(), sqlcgen.UpdateSocialAccountTokensParams{
		ID:           uid,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (r *Repository) DeleteSocialAccount(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.DeleteSocialAccount(context.Background(), uid)
}

func (r *Repository) GetSocialAccountByID(id string) (*models.SocialAccount, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetSocialAccountByID(context.Background(), uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	sa := toModelSocialAccount(row)
	return &sa, nil
}

func (r *Repository) CountSocialAccountsByUserID(userID string) (int64, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, err
	}
	return r.queries.CountSocialAccountsByUserID(context.Background(), uid)
}

// ── type conversions ────────────────────────────────────────────────────────

func toModelSocialAccount(row sqlcgen.SocialAccount) models.SocialAccount {
	return models.SocialAccount{
		ID:             row.ID,
		AppID:          row.AppID,
		UserID:         row.UserID,
		Provider:       row.Provider,
		ProviderUserID: row.ProviderUserID,
		Email:          row.Email,
		Name:           row.Name,
		FirstName:      row.FirstName,
		LastName:       row.LastName,
		ProfilePicture: row.ProfilePicture,
		Username:       row.Username,
		Locale:         row.Locale,
		RawData:        json.RawMessage(row.RawData),
		AccessToken:    row.AccessToken,
		RefreshToken:   row.RefreshToken,
		ExpiresAt:      timestamptzToTimePtr(row.ExpiresAt),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func toModelSocialAccounts(rows []sqlcgen.SocialAccount) []models.SocialAccount {
	out := make([]models.SocialAccount, len(rows))
	for i, row := range rows {
		out[i] = toModelSocialAccount(row)
	}
	return out
}

func toModelOAuthProviderConfig(row sqlcgen.OauthProviderConfig) models.OAuthProviderConfig {
	return models.OAuthProviderConfig{
		ID:           row.ID,
		AppID:        row.AppID,
		Provider:     row.Provider,
		ClientID:     row.ClientID,
		ClientSecret: row.ClientSecret,
		RedirectURL:  row.RedirectUrl,
		IsEnabled:    row.IsEnabled,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func timePtrToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timestamptzToTimePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
