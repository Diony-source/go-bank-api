// file: model/token.go

package model

import "time"

// RefreshToken holds the data for a refresh token in the database.
type RefreshToken struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	TokenHash string    `json:"-"` // The hash is not exposed in JSON responses.
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
