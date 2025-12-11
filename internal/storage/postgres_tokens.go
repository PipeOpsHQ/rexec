package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rexec/rexec/internal/models"
)

// GenerateAPIToken creates a new API token and returns the plain text token (only shown once)
func (s *PostgresStore) GenerateAPIToken(ctx context.Context, userID, name string, scopes []string, expiresAt *time.Time) (*models.APIToken, string, error) {
	// Generate random token (32 bytes = 64 hex chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}
	
	// Format: rexec_<random>
	plainToken := "rexec_" + hex.EncodeToString(tokenBytes)
	
	// Hash the token for storage
	hash := sha256.Sum256([]byte(plainToken))
	tokenHash := hex.EncodeToString(hash[:])
	
	// Token prefix for identification (first 12 chars including rexec_)
	tokenPrefix := plainToken[:12]
	
	// Default scopes if none provided
	if len(scopes) == 0 {
		scopes = []string{"read", "write"}
	}
	
	token := &models.APIToken{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		TokenHash:   tokenHash,
		TokenPrefix: tokenPrefix,
		Scopes:      scopes,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}
	
	query := `
		INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, scopes, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	_, err := s.db.ExecContext(ctx, query,
		token.ID,
		token.UserID,
		token.Name,
		token.TokenHash,
		token.TokenPrefix,
		pq.Array(token.Scopes),
		token.ExpiresAt,
		token.CreatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create token: %w", err)
	}
	
	return token, plainToken, nil
}

// ValidateAPIToken validates a token and returns the associated user ID
func (s *PostgresStore) ValidateAPIToken(ctx context.Context, plainToken string) (*models.APIToken, error) {
	// Hash the provided token
	hash := sha256.Sum256([]byte(plainToken))
	tokenHash := hex.EncodeToString(hash[:])
	
	query := `
		SELECT id, user_id, name, token_prefix, scopes, last_used_at, expires_at, created_at, revoked_at
		FROM api_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL
	`
	
	var token models.APIToken
	var scopes pq.StringArray
	var lastUsedAt, expiresAt, revokedAt sql.NullTime
	
	err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.Name,
		&token.TokenPrefix,
		&scopes,
		&lastUsedAt,
		&expiresAt,
		&revokedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid token")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	
	token.Scopes = []string(scopes)
	if lastUsedAt.Valid {
		token.LastUsedAt = &lastUsedAt.Time
	}
	if expiresAt.Valid {
		token.ExpiresAt = &expiresAt.Time
	}
	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}
	
	// Check expiration
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}
	
	// Update last used timestamp
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.db.ExecContext(updateCtx, "UPDATE api_tokens SET last_used_at = $1 WHERE id = $2", time.Now(), token.ID)
	}()
	
	return &token, nil
}

// GetAPITokensByUserID returns all tokens for a user (without hashes)
func (s *PostgresStore) GetAPITokensByUserID(ctx context.Context, userID string) ([]*models.APIToken, error) {
	query := `
		SELECT id, user_id, name, token_prefix, scopes, last_used_at, expires_at, created_at, revoked_at
		FROM api_tokens
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`
	
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}
	defer rows.Close()
	
	var tokens []*models.APIToken
	for rows.Next() {
		var token models.APIToken
		var scopes pq.StringArray
		var lastUsedAt, expiresAt, revokedAt sql.NullTime
		
		err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.Name,
			&token.TokenPrefix,
			&scopes,
			&lastUsedAt,
			&expiresAt,
			&revokedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan token: %w", err)
		}
		
		token.Scopes = []string(scopes)
		if lastUsedAt.Valid {
			token.LastUsedAt = &lastUsedAt.Time
		}
		if expiresAt.Valid {
			token.ExpiresAt = &expiresAt.Time
		}
		if revokedAt.Valid {
			token.RevokedAt = &revokedAt.Time
		}
		
		tokens = append(tokens, &token)
	}
	
	return tokens, nil
}

// RevokeAPIToken revokes a token
func (s *PostgresStore) RevokeAPIToken(ctx context.Context, userID, tokenID string) error {
	query := `
		UPDATE api_tokens 
		SET revoked_at = $1 
		WHERE id = $2 AND user_id = $3 AND revoked_at IS NULL
	`
	
	result, err := s.db.ExecContext(ctx, query, time.Now(), tokenID, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("token not found or already revoked")
	}
	
	return nil
}

// DeleteAPIToken permanently deletes a token
func (s *PostgresStore) DeleteAPIToken(ctx context.Context, userID, tokenID string) error {
	query := `DELETE FROM api_tokens WHERE id = $1 AND user_id = $2`
	
	result, err := s.db.ExecContext(ctx, query, tokenID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("token not found")
	}
	
	return nil
}
