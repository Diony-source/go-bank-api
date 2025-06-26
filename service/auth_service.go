package service

import (
	"fmt"
	"go-bank-api/config"
	"go-bank-api/logger"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func getJwtKey() []byte {
	return []byte(config.AppConfig.JWT.SecretKey)
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to hash password")
		return "", err
	}
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateJWT(email string) (string, error) {
	expirationTime := time.Now().Add(1 * time.Hour)

	claims := &jwt.RegisteredClaims{
		Subject:   email,
		ExpiresAt: jwt.NewNumericDate(expirationTime),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(getJwtKey())
	if err != nil {
		logger.Log.WithError(err).WithField("email", email).Error("Failed to sign JWT")
		return "", fmt.Errorf("failed to sign token string: %w", err)
	}

	return tokenString, nil
}
