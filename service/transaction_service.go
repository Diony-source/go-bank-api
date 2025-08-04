// File: service/transaction_service.go
package service

import (
	"database/sql"
	"errors"
	"fmt"
	"go-bank-api/model"
	"go-bank-api/repository"
)

type TransactionService struct {
	db              *sql.DB
	accountRepo     *repository.AccountRepository
	transactionRepo *repository.TransactionRepository
}

func NewTransactionService(db *sql.DB, accountRepo *repository.AccountRepository, transactionRepo *repository.TransactionRepository) *TransactionService {
	return &TransactionService{
		db:              db,
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

func (s *TransactionService) TransferMoney(fromAccountID, toAccountID, userID int, amount float64) (*model.Transaction, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	fromAccount, err := s.accountRepo.GetAccountForUpdate(tx, fromAccountID)
	if err != nil {
		return nil, errors.New("sender account not found")
	}

	toAccount, err := s.accountRepo.GetAccountForUpdate(tx, toAccountID)
	if err != nil {
		return nil, errors.New("receiver account not found")
	}

	if fromAccount.UserID != userID {
		return nil, errors.New("you can only transfer money from your own account")
	}
	if fromAccount.Balance < amount {
		return nil, errors.New("insufficient funds")
	}
	if fromAccount.Currency != toAccount.Currency {
		return nil, errors.New("currency mismatch between accounts")
	}

	err = s.accountRepo.UpdateAccountBalance(tx, fromAccountID, fromAccount.Balance-amount)
	if err != nil {
		return nil, fmt.Errorf("could not update sender balance: %w", err)
	}

	err = s.accountRepo.UpdateAccountBalance(tx, toAccountID, toAccount.Balance+amount)
	if err != nil {
		return nil, fmt.Errorf("could not update receiver balance: %w", err)
	}

	transaction := &model.Transaction{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        amount,
	}
	err = s.transactionRepo.CreateTransaction(tx, transaction)
	if err != nil {
		return nil, fmt.Errorf("could not create transaction record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return transaction, nil
}
