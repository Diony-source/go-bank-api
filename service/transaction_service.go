package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go-bank-api/logger"
	"go-bank-api/model"
	"go-bank-api/repository"

	"github.com/sirupsen/logrus"
)

var (
	ErrSenderAccountNotFound   = errors.New("sender account not found")
	ErrReceiverAccountNotFound = errors.New("receiver account not found")
	ErrSameAccountTransfer     = errors.New("cannot transfer money to the same account")
	ErrPermissionDenied        = errors.New("you can only transfer money from your own account")
	ErrInsufficientFunds       = errors.New("insufficient funds")
	ErrCurrencyMismatch        = errors.New("currency mismatch between accounts")
	ErrInvalidAmount           = errors.New("transfer amount must be greater than zero")
	ErrAccountNotFound         = errors.New("account not found")
)

type TransactionService struct {
	db              *sql.DB
	accountRepo     repository.IAccountRepository
	transactionRepo repository.ITransactionRepository
}

func NewTransactionService(db *sql.DB, accountRepo repository.IAccountRepository, transactionRepo repository.ITransactionRepository) *TransactionService {
	return &TransactionService{
		db:              db,
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

// TransferRequest defines the structure for a money transfer. from_account_id is now sourced from the URL.
type TransferRequest struct {
	ToAccountID int     `json:"to_account_id" validate:"required"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
}

func (s *TransactionService) TransferMoney(ctx context.Context, userID, fromAccountID int, req TransferRequest) (*model.Transaction, error) {
	log := logger.Log.WithFields(logrus.Fields{
		"from_account_id": fromAccountID,
		"to_account_id":   req.ToAccountID,
		"amount":          req.Amount,
		"user_id":         userID,
	})

	log.Info("Starting money transfer process")

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	if fromAccountID == req.ToAccountID {
		return nil, ErrSameAccountTransfer
	}
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	fromAccount, err := s.accountRepo.GetAccountForUpdate(tx, fromAccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSenderAccountNotFound
		}
		return nil, err
	}

	toAccount, err := s.accountRepo.GetAccountForUpdate(tx, req.ToAccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrReceiverAccountNotFound
		}
		return nil, err
	}

	if fromAccount.UserID != userID {
		return nil, ErrPermissionDenied
	}
	if fromAccount.Balance < req.Amount {
		return nil, ErrInsufficientFunds
	}
	if fromAccount.Currency != toAccount.Currency {
		return nil, ErrCurrencyMismatch
	}

	err = s.accountRepo.UpdateAccountBalance(tx, fromAccount.ID, fromAccount.Balance-req.Amount)
	if err != nil {
		return nil, fmt.Errorf("could not update sender balance: %w", err)
	}

	err = s.accountRepo.UpdateAccountBalance(tx, toAccount.ID, toAccount.Balance+req.Amount)
	if err != nil {
		return nil, fmt.Errorf("could not update receiver balance: %w", err)
	}

	transaction := &model.Transaction{
		FromAccountID: fromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	err = s.transactionRepo.CreateTransaction(tx, transaction)
	if err != nil {
		return nil, fmt.Errorf("could not create transaction record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	log.Info("Transaction completed successfully")
	return transaction, nil
}

// ListTransactionsForAccount retrieves the transaction history for a specific account.
func (s *TransactionService) ListTransactionsForAccount(ctx context.Context, userID, accountID int) ([]*model.Transaction, error) {
	log := logger.Log.WithFields(logrus.Fields{
		"requesting_user_id": userID,
		"target_account_id":  accountID,
	})

	// Authorization check: User must own the account.
	account, err := s.accountRepo.GetAccountByID(accountID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	if account.UserID != userID {
		log.Warn("Permission denied for accessing account's transaction history")
		return nil, ErrPermissionDenied
	}

	// Authorization passed. Now, fetch the history.
	return s.transactionRepo.GetTransactionsByAccountID(accountID)
}
