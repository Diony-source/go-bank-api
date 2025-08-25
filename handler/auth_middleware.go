package handler

import (
	"context"
	"go-bank-api/common"
	"go-bank-api/config"
	"go-bank-api/model"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey defines a custom type for context keys to avoid collisions.
type contextKey string

const (
	UserIDKey   contextKey = "userID"
	UserRoleKey contextKey = "userRole"
)

// AuthMiddleware validates the JWT from the Authorization header.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			err := common.NewAppError(http.StatusUnauthorized, "Authorization header is required", nil)
			err.Send(w)
			return
		}

		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || strings.ToLower(headerParts[0]) != "bearer" {
			err := common.NewAppError(http.StatusUnauthorized, "Invalid authorization header format", nil)
			err.Send(w)
			return
		}

		tokenString := headerParts[1]
		claims := &model.AppClaims{}
		jwtKey := []byte(config.AppConfig.JWT.SecretKey)

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			appErr := common.NewAppError(http.StatusUnauthorized, "Invalid or expired token", err)
			appErr.Send(w)
			return
		}

		// If the token is valid, store user info in the request context.
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserRoleKey, claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminMiddleware checks if the user has admin privileges.
func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(UserRoleKey).(string)

		if !ok || role != string(model.RoleAdmin) { // Using the constant for safety
			err := common.NewAppError(http.StatusForbidden, "Access denied. Admin privileges required.", nil)
			err.Send(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}