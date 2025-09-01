// file: service/account_service.go

package service

import (
	"context"
	"database/sql"
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

// DepositToAccount handles the business logic for depositing funds into a specific account.
// It ensures the operation is valid and, upon success, invalidates the relevant user's cache.
func (s *AccountService) DepositToAccount(accountID int, amount float64) (*model.Account, error) {
	if amount <= 0 {
		return nil, errors.New("deposit amount must be positive")
	}

	// 1. Perform the database operation. The repository returns the updated account,
	// which critically includes the UserID needed for cache invalidation.
	updatedAccount, err := s.repo.DepositToAccount(accountID, amount)
	if err != nil {
		// Translate potential DB errors into service-level errors.
		if err == sql.ErrNoRows {
			return nil, errors.New("account not found")
		}
		return nil, err
	}

	// 2. If the DB write is successful, invalidate the cache for the account's owner.
	// This removes the technical debt and ensures data consistency.
	cacheKey := fmt.Sprintf("accounts:%d", updatedAccount.UserID)
	s.cacheClient.Del(context.Background(), cacheKey)

	return updatedAccount, nil
}
