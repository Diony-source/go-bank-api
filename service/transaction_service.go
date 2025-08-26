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

// Pre-defined errors for cleaner error handling in the handler layer
var (
	ErrSenderAccountNotFound   = errors.New("sender account not found")
	ErrReceiverAccountNotFound = errors.New("receiver account not found")
	ErrSameAccountTransfer     = errors.New("cannot transfer money to the same account")
	ErrPermissionDenied        = errors.New("you can only transfer money from your own account")
	ErrInsufficientFunds       = errors.New("insufficient funds")
	ErrCurrencyMismatch        = errors.New("currency mismatch between accounts")
	ErrInvalidAmount           = errors.New("transfer amount must be greater than zero")
)

type TransactionService struct {
	db              *sql.DB
	accountRepo     repository.IAccountRepository     // UPDATED
	transactionRepo repository.ITransactionRepository // UPDATED
}

func NewTransactionService(db *sql.DB, accountRepo repository.IAccountRepository, transactionRepo repository.ITransactionRepository) *TransactionService { // UPDATED
	return &TransactionService{
		db:              db,
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

// TransferRequest defines the structure for a money transfer
type TransferRequest struct {
	FromAccountID int     `json:"from_account_id" validate:"required"`
	ToAccountID   int     `json:"to_account_id" validate:"required"`
	Amount        float64 `json:"amount" validate:"required,gt=0"`
}

func (s *TransactionService) TransferMoney(ctx context.Context, req TransferRequest, userID int) (*model.Transaction, error) {
	log := logger.Log.WithFields(logrus.Fields{
		"from_account_id": req.FromAccountID,
		"to_account_id":   req.ToAccountID,
		"amount":          req.Amount,
		"user_id":         userID,
	})

	log.Info("Starting money transfer process")

	// Start a new database transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.WithError(err).Error("Could not begin database transaction")
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	// Defer a rollback in case anything goes wrong
	defer tx.Rollback()

	// --- VALIDATION AND BUSINESS LOGIC ---
	if req.FromAccountID == req.ToAccountID {
		return nil, ErrSameAccountTransfer
	}
	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	// Lock the sender and receiver accounts to prevent race conditions
	log.Info("Locking sender account for update")
	fromAccount, err := s.accountRepo.GetAccountForUpdate(tx, req.FromAccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSenderAccountNotFound
		}
		return nil, err
	}

	log.Info("Locking receiver account for update")
	toAccount, err := s.accountRepo.GetAccountForUpdate(tx, req.ToAccountID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrReceiverAccountNotFound
		}
		return nil, err
	}

	// Check for business rule violations
	if fromAccount.UserID != userID {
		return nil, ErrPermissionDenied
	}
	if fromAccount.Balance < req.Amount {
		return nil, ErrInsufficientFunds
	}
	if fromAccount.Currency != toAccount.Currency {
		return nil, ErrCurrencyMismatch
	}

	// --- PERFORM THE TRANSFER ---
	log.Info("Updating sender account balance")
	err = s.accountRepo.UpdateAccountBalance(tx, fromAccount.ID, fromAccount.Balance-req.Amount)
	if err != nil {
		log.WithError(err).Error("Could not update sender balance")
		return nil, fmt.Errorf("could not update sender balance: %w", err)
	}

	log.Info("Updating receiver account balance")
	err = s.accountRepo.UpdateAccountBalance(tx, toAccount.ID, toAccount.Balance+req.Amount)
	if err != nil {
		log.WithError(err).Error("Could not update receiver balance")
		return nil, fmt.Errorf("could not update receiver balance: %w", err)
	}

	// Record the transaction
	transaction := &model.Transaction{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	log.Info("Creating transaction record")
	err = s.transactionRepo.CreateTransaction(tx, transaction)
	if err != nil {
		log.WithError(err).Error("Could not create transaction record")
		return nil, fmt.Errorf("could not create transaction record: %w", err)
	}

	// If everything is successful, commit the transaction
	log.Info("Committing the transaction")
	if err := tx.Commit(); err != nil {
		log.WithError(err).Error("Could not commit transaction")
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	log.Info("Transaction completed successfully")
	return transaction, nil
}
