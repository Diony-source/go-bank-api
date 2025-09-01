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
	GetTransactionsByAccountID(accountID int) ([]*model.Transaction, error) // Correct signature
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

// GetTransactionsByAccountID retrieves all transactions for a specific account, returning a slice of pointers.
func (r *TransactionRepository) GetTransactionsByAccountID(accountID int) ([]*model.Transaction, error) {
	log := logger.Log.WithField("account_id", accountID)
	log.Info("Executing query to get transactions by account ID")

	query := `
		SELECT id, from_account_id, to_account_id, amount, created_at 
		FROM transactions 
		WHERE from_account_id = $1 OR to_account_id = $1
		ORDER BY created_at DESC`

	rows, err := r.DB.Query(query, accountID)
	if err != nil {
		log.WithError(err).Error("Failed to execute query for transactions by account ID")
		return nil, err
	}
	defer rows.Close()

	var transactions []*model.Transaction // Correct type: slice of pointers
	for rows.Next() {
		var t model.Transaction
		if err := rows.Scan(&t.ID, &t.FromAccountID, &t.ToAccountID, &t.Amount, &t.CreatedAt); err != nil {
			log.WithError(err).Error("Failed to scan transaction row")
			return nil, err
		}
		transactions = append(transactions, &t) // Correctly append the pointer
	}

	return transactions, nil
}
