// service/account_service.go
package service

import (
	"errors"
	"go-bank-api/model"
	"go-bank-api/repository"
)

type AccountService struct {
	repo repository.IAccountRepository
}

func NewAccountService(repo repository.IAccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

// CreateNewAccount creates a new account using the repository.
// It now generates a sequential and unique account number.
func (s *AccountService) CreateNewAccount(userID int, currency string) (*model.Account, error) {
	lastAccountNumber, err := s.repo.GetLastAccountNumber()
	if err != nil {
		return nil, err
	}

	newAccountNumber := lastAccountNumber + 1

	account := &model.Account{
		UserID:        userID,
		AccountNumber: newAccountNumber,
		Currency:      currency,
	}

	err = s.repo.CreateAccount(account)
	if err != nil {
		return nil, err
	}

	return account, nil
}

// ListAccountsForUser lists accounts for a specific user.
func (s *AccountService) ListAccountsForUser(userID int) ([]*model.Account, error) {
	return s.repo.GetAccountsByUserID(userID)
}

// GetAllAccounts retrieves all accounts. Admin only.
func (s *AccountService) GetAllAccounts() ([]*model.Account, error) {
	return s.repo.GetAllAccounts()
}

// DepositToAccount handles the business logic for depositing funds.
func (s *AccountService) DepositToAccount(accountID int, amount float64) (*model.Account, error) {
	if amount <= 0 {
		return nil, errors.New("deposit amount must be positive")
	}

	return s.repo.DepositToAccount(accountID, amount)
}
