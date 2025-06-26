package model

import "github.com/golang-jwt/jwt/v5"

type AppClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}
