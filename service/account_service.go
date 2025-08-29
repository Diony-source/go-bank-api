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

	"github.com/redis/go-redis/v9"
)

// AccountService now includes a Redis client for caching operations.
type AccountService struct {
	repo        repository.IAccountRepository
	redisClient *redis.Client
}

// NewAccountService is updated to accept a Redis client as a dependency.
func NewAccountService(repo repository.IAccountRepository, redisClient *redis.Client) *AccountService {
	return &AccountService{
		repo:        repo,
		redisClient: redisClient,
	}
}

// CreateNewAccount creates a new account, saves it to the database, and invalidates the user's account cache.
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

	// First, try to save the account to the database.
	// This is the source of truth; the cache is just a reflection.
	err = s.repo.CreateAccount(account)
	if err != nil {
		return nil, err
	}

	// If the database write is successful, we must invalidate the cache.
	// This ensures that the next read will fetch the fresh, complete list of accounts.
	cacheKey := fmt.Sprintf("accounts:%d", userID)
	s.redisClient.Del(context.Background(), cacheKey) // Fire-and-forget invalidation.

	return account, nil
}

// ListAccountsForUser lists accounts for a specific user, utilizing a cache-aside strategy.
func (s *AccountService) ListAccountsForUser(userID int) ([]*model.Account, error) {
	cacheKey := fmt.Sprintf("accounts:%d", userID)
	ctx := context.Background()

	// 1. Attempt to fetch from the cache first. This is the "fast path".
	cachedAccounts, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache hit. Deserialize and return immediately.
		// Avoids a costly database roundtrip.
		var accounts []*model.Account
		if err := json.Unmarshal([]byte(cachedAccounts), &accounts); err == nil {
			return accounts, nil
		}
		// If unmarshalling fails, we'll treat it as a cache miss and proceed to the database.
	}

	// 2. Cache miss. The data is not in the cache or is corrupted.
	// Fetch the canonical data from the database (the "source of truth").
	accounts, err := s.repo.GetAccountsByUserID(userID)
	if err != nil {
		return nil, err
	}

	// 3. Populate the cache. Serialize the fresh data and store it in Redis.
	// The next request for this user will now hit the cache.
	data, err := json.Marshal(accounts)
	if err == nil {
		// We set an expiration to prevent stale data from living forever.
		// 10 minutes is a reasonable TTL for this kind of data.
		s.redisClient.Set(ctx, cacheKey, data, 10*time.Minute)
	}

	return accounts, nil
}

// GetAllAccounts retrieves all accounts. Caching is not applied here as admin data may need to be fresh.
func (s *AccountService) GetAllAccounts() ([]*model.Account, error) {
	return s.repo.GetAllAccounts()
}

// DepositToAccount handles the business logic for depositing funds.
func (s *AccountService) DepositToAccount(accountID int, amount float64) (*model.Account, error) {
	if amount <= 0 {
		return nil, errors.New("deposit amount must be positive")
	}
	// NOTE: For a complete solution, this should also invalidate the user's account cache.
	// This is a good candidate for a future enhancement.
	return s.repo.DepositToAccount(accountID, amount)
}
