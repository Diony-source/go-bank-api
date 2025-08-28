// repository/account_repository.go
package repository

import (
	"database/sql"
	"go-bank-api/logger"
	"go-bank-api/model"

	"github.com/sirupsen/logrus"
)

// IAccountRepository defines the contract for account database operations.
type IAccountRepository interface {
	CreateAccount(account *model.Account) error
	GetAccountsByUserID(userID int) ([]*model.Account, error)
	GetAllAccounts() ([]*model.Account, error)
	GetAccountForUpdate(tx *sql.Tx, accountID int) (*model.Account, error)
	UpdateAccountBalance(tx *sql.Tx, accountID int, newBalance float64) error
	DepositToAccount(accountID int, amount float64) (*model.Account, error)
	GetLastAccountNumber() (int, error)
}

// AccountRepository implements IAccountRepository.
type AccountRepository struct {
	DB *sql.DB
}

// NewAccountRepository creates a new AccountRepository.
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

// GetAccountsByUserID retrieves all accounts for a specific user.
func (r *AccountRepository) GetAccountsByUserID(userID int) ([]*model.Account, error) {
	log := logger.Log.WithField("user_id", userID)
	log.Info("Executing query to get accounts by user ID")

	query := `SELECT id, user_id, account_number, balance, currency, created_at FROM accounts WHERE user_id = $1`
	rows, err := r.DB.Query(query, userID)
	if err != nil {
		log.WithError(err).Error("Failed to execute query for accounts by user ID")
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var acc model.Account
		if err := rows.Scan(&acc.ID, &acc.UserID, &acc.AccountNumber, &acc.Balance, &acc.Currency, &acc.CreatedAt); err != nil {
			log.WithError(err).Error("Failed to scan account row")
			return nil, err
		}
		accounts = append(accounts, &acc)
	}
	return accounts, nil
}

// GetAllAccounts retrieves all accounts from the database. Admin only.
func (r *AccountRepository) GetAllAccounts() ([]*model.Account, error) {
	log := logger.Log
	log.Info("Executing query to get all accounts")

	query := `SELECT id, user_id, account_number, balance, currency, created_at FROM accounts`
	rows, err := r.DB.Query(query)
	if err != nil {
		log.WithError(err).Error("Failed to execute query for all accounts")
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var acc model.Account
		if err := rows.Scan(&acc.ID, &acc.UserID, &acc.AccountNumber, &acc.Balance, &acc.Currency, &acc.CreatedAt); err != nil {
			log.WithError(err).Error("Failed to scan account row")
			return nil, err
		}
		accounts = append(accounts, &acc)
	}
	return accounts, nil
}

// GetAccountForUpdate locks and retrieves an account row within a transaction.
func (r *AccountRepository) GetAccountForUpdate(tx *sql.Tx, accountID int) (*model.Account, error) {
	log := logger.Log.WithField("account_id", accountID)
	log.Info("Executing query to get account for update")

	account := &model.Account{}
	query := `SELECT id, user_id, balance, currency FROM accounts WHERE id = $1 FOR UPDATE`
	err := tx.QueryRow(query, accountID).Scan(&account.ID, &account.UserID, &account.Balance, &account.Currency)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("Account not found for update")
		} else {
			log.WithError(err).Error("Failed to execute get account for update query")
		}
		return nil, err
	}
	return account, nil
}

// UpdateAccountBalance updates an account's balance within a transaction.
func (r *AccountRepository) UpdateAccountBalance(tx *sql.Tx, accountID int, newBalance float64) error {
	log := logger.Log.WithFields(logrus.Fields{
		"account_id":  accountID,
		"new_balance": newBalance,
	})
	log.Info("Executing query to update account balance")

	query := `UPDATE accounts SET balance = $1 WHERE id = $2`
	_, err := tx.Exec(query, newBalance, accountID)
	if err != nil {
		log.WithError(err).Error("Failed to execute update account balance query")
		return err
	}
	return nil
}

// DepositToAccount adds a specified amount to an account's balance.
func (r *AccountRepository) DepositToAccount(accountID int, amount float64) (*model.Account, error) {
	log := logger.Log.WithFields(logrus.Fields{
		"account_id": accountID,
		"amount":     amount,
	})
	log.Info("Executing query to deposit funds")

	var updatedAccount model.Account
	query := `
		UPDATE accounts 
		SET balance = balance + $1 
		WHERE id = $2 
		RETURNING id, user_id, account_number, balance, currency, created_at`

	err := r.DB.QueryRow(query, amount, accountID).Scan(
		&updatedAccount.ID,
		&updatedAccount.UserID,
		&updatedAccount.AccountNumber,
		&updatedAccount.Balance,
		&updatedAccount.Currency,
		&updatedAccount.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Info("Account not found for deposit")
			return nil, err
		}
		log.WithError(err).Error("Failed to execute deposit query")
		return nil, err
	}

	log.Info("Funds deposited successfully")
	return &updatedAccount, nil
}

// GetLastAccountNumber retrieves the highest account number from the database.
func (r *AccountRepository) GetLastAccountNumber() (int64, error) {
	log := logger.Log
	log.Info("Executing query to get the last account number")

	var lastAccountNumber sql.NullInt64 // Use sql.NullInt64 to handle the case where the table is empty.
	query := `SELECT MAX(account_number) FROM accounts`
	err := r.DB.QueryRow(query).Scan(&lastAccountNumber)

	if err != nil {
		log.WithError(err).Error("Failed to execute query for the last account number")
		return 0, err
	}

	if !lastAccountNumber.Valid {
		// No accounts exist yet, so we can start from a base number.
		return 1000000000, nil // Start from a fixed number
	}

	return lastAccountNumber.Int64, nil
}
