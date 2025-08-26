package repository

import (
	"database/sql"
	"go-bank-api/logger"
	"go-bank-api/model"

	"github.com/sirupsen/logrus"
)

// ITransactionRepository defines the contract for transaction database operations.
type ITransactionRepository interface {
	CreateTransaction(tx *sql.Tx, transaction *model.Transaction) error
}

// TransactionRepository implements ITransactionRepository.
type TransactionRepository struct {
	DB *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{DB: db}
}

func (r *TransactionRepository) CreateTransaction(tx *sql.Tx, transaction *model.Transaction) error {
	log := logger.Log.WithFields(logrus.Fields{
		"from_account_id": transaction.FromAccountID,
		"to_account_id":   transaction.ToAccountID,
		"amount":          transaction.Amount,
	})
	log.Info("Executing query to create a new transaction")

	query := `INSERT INTO transactions (from_account_id, to_account_id, amount) VALUES ($1, $2, $3) RETURNING id, created_at`
	err := tx.QueryRow(query, transaction.FromAccountID, transaction.ToAccountID, transaction.Amount).Scan(&transaction.ID, &transaction.CreatedAt)
	if err != nil {
		log.WithError(err).Error("Failed to execute create transaction query")
		return err
	}
	return nil
}
