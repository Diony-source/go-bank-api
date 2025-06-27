package model

import "time"

type Account struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	AccountNumber int64     `json:"account_number"`
	Balance       float64   `json:"balance"`
	Currency      string    `json:"currency"`
	CreatedAt     time.Time `json:"created_at"`
}
