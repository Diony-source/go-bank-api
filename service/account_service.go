package service

import (
	"go-bank-api/model"
	"go-bank-api/repository"
	"math/rand"
	"time"
)

type AccountService struct {
	repo *repository.AccountRepository
}

func NewAccountService(repo *repository.AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

// CreateNewAccount creates a new account using the repository.
func (s *AccountService) CreateNewAccount(userID int, currency string) (*model.Account, error) {
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

// ListAccountsForUser lists accounts based on the user's role.
func (s *AccountService) ListAccountsForUser(userID int, userRole string) ([]*model.Account, error) {
	if userRole == "admin" {
		return s.repo.GetAllAccounts()
	}
	return s.repo.GetAccountsByUserID(userID)
}
