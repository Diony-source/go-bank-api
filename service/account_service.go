// file: service/account_service.go

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-bank-api/model"
	"go-bank-api/repository"
	"time"
)

// AccountService depends on the ICacheClient interface, not a concrete Redis client.
type AccountService struct {
	repo        repository.IAccountRepository
	cacheClient ICacheClient // DEPENDENCY INVERSION
}

// NewAccountService is updated to accept the ICacheClient interface.
func NewAccountService(repo repository.IAccountRepository, cacheClient ICacheClient) *AccountService {
	return &AccountService{
		repo:        repo,
		cacheClient: cacheClient,
	}
}

// CreateNewAccount creates a new account and invalidates the user's account cache.
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

	if err = s.repo.CreateAccount(account); err != nil {
		return nil, err
	}

	// Invalidate the cache to ensure data consistency on the next read.
	cacheKey := fmt.Sprintf("accounts:%d", userID)
	s.cacheClient.Del(context.Background(), cacheKey)

	return account, nil
}

// ListAccountsForUser lists accounts for a specific user, utilizing a cache-aside strategy.
func (s *AccountService) ListAccountsForUser(userID int) ([]*model.Account, error) {
	cacheKey := fmt.Sprintf("accounts:%d", userID)
	ctx := context.Background()

	// 1. Attempt to fetch from the cache first (fast path).
	cachedAccounts, err := s.cacheClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var accounts []*model.Account
		if err := json.Unmarshal([]byte(cachedAccounts), &accounts); err == nil {
			return accounts, nil
		}
	}

	// 2. Cache miss. Fetch from the source of truth (database).
	accounts, err := s.repo.GetAccountsByUserID(userID)
	if err != nil {
		return nil, err
	}

	// 3. Populate the cache for subsequent requests.
	data, err := json.Marshal(accounts)
	if err == nil {
		s.cacheClient.Set(ctx, cacheKey, data, 10*time.Minute)
	}

	return accounts, nil
}

func (s *AccountService) GetAllAccounts() ([]*model.Account, error) {
	return s.repo.GetAllAccounts()
}

func (s *AccountService) DepositToAccount(accountID int, amount float64) (*model.Account, error) {
	if amount <= 0 {
		return nil, errors.New("deposit amount must be positive")
	}
	// TODO: This operation should also invalidate the user's account cache.
	return s.repo.DepositToAccount(accountID, amount)
}
