package repository

import (
	"database/sql"
	"go-bank-api/logger"
	"go-bank-api/model"

	"github.com/sirupsen/logrus"
)

type AccountRepository struct {
	DB *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{DB: db}
}

// CreateAccount adds a new account to the database.
func (r *AccountRepository) CreateAccount(account *model.Account) error {
	log := logger.Log.WithFields(logrus.Fields{
		"user_id":        account.UserID,
		"account_number": account.AccountNumber,
		"currency":       account.Currency,
	})
	log.Info("Executing query to create a new account")

	query := `INSERT INTO accounts (user_id, account_number, currency) VALUES ($1, $2, $3) RETURNING id, balance, created_at`
	err := r.DB.QueryRow(query, account.UserID, account.AccountNumber, account.Currency).Scan(&account.ID, &account.Balance, &account.CreatedAt)
	if err != nil {
		log.WithError(err).Error("Failed to execute create account query")
		return err
	}
	return nil
}
