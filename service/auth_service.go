package service

import (
	"fmt"
	"go-bank-api/config"
	"go-bank-api/logger"
	"go-bank-api/model"
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

func GenerateJWT(user *model.User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Token geçerliliğini 24 saate çıkaralım.

	// Yeni AppClaims yapımızı kullanarak "claims" oluşturuyoruz.
	claims := &model.AppClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Email,
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Token'ı AppClaims ve imzalama metodu ile oluşturuyoruz.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Token'ı gizli anahtarımızla imzalayıp string'e çeviriyoruz.
	tokenString, err := token.SignedString(getJwtKey())
	if err != nil {
		logger.Log.WithError(err).WithField("email", user.Email).Error("Failed to sign JWT")
		return "", fmt.Errorf("failed to sign token string: %w", err)
	}

	return tokenString, nil
}
