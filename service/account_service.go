// service/account_service.go
package service

import (
	"errors"
	"go-bank-api/model"
	"go-bank-api/repository"
	"math/rand"
	"time"
)

type AccountService struct {
	repo repository.IAccountRepository
}

func NewAccountService(repo repository.IAccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

// CreateNewAccount creates a new account using the repository.
func (s *AccountService) CreateNewAccount(userID int, currency string) (*model.Account, error) {
	// Note: For production, a more robust account number generation is needed.
	rand.Seed(time.Now().UnixNano())
	accountNumber := int64(rand.Intn(9000000000) + 1000000000)

	account := &model.Account{
		UserID:        userID,
		AccountNumber: accountNumber,
		Currency:      currency,
	}

	err := s.repo.CreateAccount(account)
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
