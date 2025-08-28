// file: repository/token_repository.go

package repository

import (
	"database/sql"
	"go-bank-api/logger"
	"go-bank-api/model"

	"github.com/sirupsen/logrus"
)

// ITokenRepository defines the contract for refresh token database operations.
type ITokenRepository interface {
	Create(token *model.RefreshToken) error
	GetByTokenHash(tokenHash string) (*model.RefreshToken, error)
	DeleteByUserID(userID int) error
}

// TokenRepository implements ITokenRepository.
type TokenRepository struct {
	DB *sql.DB
}

// NewTokenRepository creates a new TokenRepository.
func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{DB: db}
}

// Create inserts a new refresh token record into the database.
func (r *TokenRepository) Create(token *model.RefreshToken) error {
	log := logger.Log.WithFields(logrus.Fields{
		"user_id":    token.UserID,
		"expires_at": token.ExpiresAt,
	})
	log.Info("Executing query to create a new refresh token")

	query := `INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING id, created_at`
	err := r.DB.QueryRow(query, token.UserID, token.TokenHash, token.ExpiresAt).Scan(&token.ID, &token.CreatedAt)
	if err != nil {
		log.WithError(err).Error("Failed to execute create refresh token query")
		return err
	}
	return nil
}

// GetByTokenHash retrieves a refresh token by its hashed value.
func (r *TokenRepository) GetByTokenHash(tokenHash string) (*model.RefreshToken, error) {
	log := logger.Log.WithField("token_hash", tokenHash)
	log.Info("Executing query to get refresh token by hash")

	token := &model.RefreshToken{}
	query := `SELECT id, user_id, token_hash, expires_at, created_at FROM refresh_tokens WHERE token_hash = $1`
	err := r.DB.QueryRow(query, tokenHash).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.CreatedAt)
	if err != nil {
		if err != sql.ErrNoRows {
			log.WithError(err).Error("Failed to execute get refresh token by hash query")
		}
		return nil, err // Return sql.ErrNoRows if not found
	}
	return token, nil
}

// DeleteByUserID deletes all refresh tokens for a specific user.
// This is used for logging out from all sessions.
func (r *TokenRepository) DeleteByUserID(userID int) error {
	log := logger.Log.WithField("user_id", userID)
	log.Info("Executing query to delete all refresh tokens for a user")

	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.DB.Exec(query, userID)
	if err != nil {
		log.WithError(err).Error("Failed to execute delete refresh tokens query")
		return err
	}
	return nil
}
