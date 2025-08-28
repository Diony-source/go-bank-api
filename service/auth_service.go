// file: service/auth_service.go

package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"go-bank-api/config"
	"go-bank-api/logger"
	"go-bank-api/model"
	"go-bank-api/repository"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles the business logic for authentication, including token generation and validation.
// It depends on user and token repositories to interact with the database.
type AuthService struct {
	userRepo  repository.IUserRepository
	tokenRepo repository.ITokenRepository
}

// NewAuthService creates a new AuthService with its dependencies.
func NewAuthService(userRepo repository.IUserRepository, tokenRepo repository.ITokenRepository) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
	}
}

// TokenPair represents a pair of access and refresh tokens.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// HashPassword generates a bcrypt hash of the password.
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to hash password")
		return "", err
	}
	return string(bytes), nil
}

// CheckPasswordHash compares a password with its bcrypt hash.
func (s *AuthService) CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateAccessToken creates a new short-lived JWT access token.
func (s *AuthService) generateAccessToken(user *model.User) (string, error) {
	jwtKey := []byte(config.AppConfig.JWT.SecretKey)
	expirationTime := time.Now().Add(15 * time.Minute) // Short-lived

	claims := &model.AppClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Email,
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		logger.Log.WithError(err).WithField("email", user.Email).Error("Failed to sign access token")
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, nil
}

// generateRefreshToken creates a new long-lived, cryptographically secure refresh token.
func (s *AuthService) generateRefreshToken(userID int) (string, *model.RefreshToken, error) {
	// 1. Generate a random, secure token string
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random bytes for refresh token: %w", err)
	}
	tokenString := base64.URLEncoding.EncodeToString(randomBytes)

	// 2. Hash the token string for database storage
	hash := sha256.Sum256([]byte(tokenString))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	// 3. Create the model for the database
	refreshToken := &model.RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // Long-lived (7 days)
	}

	// The raw token string is returned to the user, the hashed version is stored.
	return tokenString, refreshToken, nil
}

// AuthenticateUser validates user credentials and generates a new token pair.
// It also cleans up any old refresh tokens and stores the new one.
func (s *AuthService) AuthenticateUser(email, password string) (*TokenPair, error) {
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !s.CheckPasswordHash(password, user.Password) {
		return nil, errors.New("invalid credentials")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("could not generate access token: %w", err)
	}

	refreshTokenString, refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("could not generate refresh token: %w", err)
	}

	if err := s.tokenRepo.DeleteByUserID(user.ID); err != nil {
		logger.Log.WithError(err).WithField("user_id", user.ID).Warn("Failed to delete old refresh tokens for user")
	}
	if err := s.tokenRepo.Create(refreshToken); err != nil {
		return nil, fmt.Errorf("could not store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
	}, nil
}

// RefreshAccessToken validates a refresh token and issues a new access token if valid.
func (s *AuthService) RefreshAccessToken(refreshTokenString string) (string, error) {
	hash := sha256.Sum256([]byte(refreshTokenString))
	tokenHash := base64.URLEncoding.EncodeToString(hash[:])

	refreshToken, err := s.tokenRepo.GetByTokenHash(tokenHash)
	if err != nil {
		return "", errors.New("invalid refresh token")
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		return "", errors.New("expired refresh token")
	}

	user, err := s.userRepo.GetUserByID(refreshToken.UserID)
	if err != nil {
		return "", errors.New("user not found for token")
	}

	newAccessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", fmt.Errorf("could not generate new access token: %w", err)
	}

	return newAccessToken, nil
}

// LogoutUser invalidates a user's session by deleting their refresh tokens.
func (s *AuthService) LogoutUser(userID int) error {
	if err := s.tokenRepo.DeleteByUserID(userID); err != nil {
		logger.Log.WithError(err).WithField("user_id", userID).Error("Failed to delete refresh tokens during logout")
		return fmt.Errorf("could not log out: %w", err)
	}
	return nil
}
