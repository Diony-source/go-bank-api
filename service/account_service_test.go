// file: service/account_service_test.go

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go-bank-api/model"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockAccountRepoForAccountSvc provides a mock implementation of IAccountRepository.
type mockAccountRepoForAccountSvc struct{ mock.Mock }

func (m *mockAccountRepoForAccountSvc) CreateAccount(account *model.Account) error {
	args := m.Called(account)
	return args.Error(0)
}
func (m *mockAccountRepoForAccountSvc) GetLastAccountNumber() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockAccountRepoForAccountSvc) DepositToAccount(id int, amount float64) (*model.Account, error) {
	args := m.Called(id, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}
func (m *mockAccountRepoForAccountSvc) GetAccountsByUserID(userID int) ([]*model.Account, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Account), args.Error(1)
}
func (m *mockAccountRepoForAccountSvc) GetAllAccounts() ([]*model.Account, error) { return nil, nil }
func (m *mockAccountRepoForAccountSvc) GetAccountForUpdate(*sql.Tx, int) (*model.Account, error) {
	return nil, nil
}
func (m *mockAccountRepoForAccountSvc) UpdateAccountBalance(*sql.Tx, int, float64) error { return nil }

// mockRedisClient provides a mock for redis.Client to simulate Redis interactions.
type mockRedisClient struct {
	mock.Mock
	redis.Client
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewStringCmd(ctx)
	cmd.SetVal(args.String(0))
	cmd.SetErr(args.Error(1))
	return cmd
}
func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	m.Called(ctx, key, value, expiration)
	return redis.NewStatusCmd(ctx)
}
func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	m.Called(ctx, keys[0])
	return redis.NewIntCmd(ctx)
}

// --- Test Suites ---

func TestAccountService_CreateNewAccount(t *testing.T) {
	mockRepo := new(mockAccountRepoForAccountSvc)
	mockRedis := new(mockRedisClient)
	// FIX: The AccountService constructor requires a valid Redis client.
	// We pass the mock Redis client directly. Its embedded `redis.Client` field
	// is nil, but our mock methods (Get, Set, Del) will intercept calls.
	accountService := NewAccountService(mockRepo, &mockRedis.Client)

	userID := 1
	currency := "TRY"
	lastAccountNumber := int64(1000000025)
	cacheKey := fmt.Sprintf("accounts:%d", userID)

	// --- Expectations ---
	// 1. Verify correct account number generation.
	mockRepo.On("GetLastAccountNumber").Return(lastAccountNumber, nil).Once()
	expectedNewAccountNumber := lastAccountNumber + 1
	mockRepo.On("CreateAccount", mock.MatchedBy(func(acc *model.Account) bool {
		return acc.AccountNumber == expectedNewAccountNumber && acc.UserID == userID
	})).Return(nil).Once()
	// 2. VERIFY: A successful DB write must trigger a cache invalidation.
	mockRedis.On("Del", mock.Anything, cacheKey).Return().Once()

	// --- Execution ---
	account, err := accountService.CreateNewAccount(userID, currency)

	// --- Assertions ---
	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, expectedNewAccountNumber, account.AccountNumber)
	mockRepo.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

func TestAccountService_ListAccountsForUser_CacheHit(t *testing.T) {
	mockRepo := new(mockAccountRepoForAccountSvc)
	mockRedis := new(mockRedisClient)
	accountService := NewAccountService(mockRepo, &mockRedis.Client)

	userID := 2
	cacheKey := fmt.Sprintf("accounts:%d", userID)
	expectedAccounts := []*model.Account{{ID: 1, UserID: userID, Balance: 100.0}}
	cachedData, _ := json.Marshal(expectedAccounts)

	// SETUP: Prime the mock cache with the user's account data.
	mockRedis.On("Get", mock.Anything, cacheKey).Return(string(cachedData), nil).Once()

	accounts, err := accountService.ListAccountsForUser(userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedAccounts, accounts)
	// VERIFY: The database repository should not be invoked on a cache hit.
	mockRepo.AssertNotCalled(t, "GetAccountsByUserID")
	mockRedis.AssertExpectations(t)
}

func TestAccountService_ListAccountsForUser_CacheMiss(t *testing.T) {
	mockRepo := new(mockAccountRepoForAccountSvc)
	mockRedis := new(mockRedisClient)
	accountService := NewAccountService(mockRepo, &mockRedis.Client)

	userID := 3
	cacheKey := fmt.Sprintf("accounts:%d", userID)
	dbAccounts := []*model.Account{{ID: 2, UserID: userID, Balance: 500.0}}
	dbData, _ := json.Marshal(dbAccounts)

	// SETUP: Simulate a cache miss by returning redis.Nil.
	mockRedis.On("Get", mock.Anything, cacheKey).Return("", redis.Nil).Once()
	// VERIFY: A cache miss must trigger a database query.
	mockRepo.On("GetAccountsByUserID", userID).Return(dbAccounts, nil).Once()
	// VERIFY: The result from the database must be stored back into the cache.
	mockRedis.On("Set", mock.Anything, cacheKey, dbData, 10*time.Minute).Return().Once()

	accounts, err := accountService.ListAccountsForUser(userID)

	assert.NoError(t, err)
	assert.Equal(t, dbAccounts, accounts)
	mockRepo.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
}

func TestAccountService_DepositToAccount(t *testing.T) {
	mockRepo := new(mockAccountRepoForAccountSvc)
	mockRedis := new(mockRedisClient)
	// FIX: Always provide a valid, non-nil client, even if no calls are expected.
	accountService := NewAccountService(mockRepo, &mockRedis.Client)

	t.Run("success", func(t *testing.T) {
		accountID := 1
		amount := 100.0
		expectedAccount := &model.Account{ID: accountID, Balance: 100.0}
		mockRepo.On("DepositToAccount", accountID, amount).Return(expectedAccount, nil).Once()
		updatedAccount, err := accountService.DepositToAccount(accountID, amount)
		assert.NoError(t, err)
		assert.Equal(t, expectedAccount, updatedAccount)
		mockRepo.AssertExpectations(t)
	})
	t.Run("negative amount", func(t *testing.T) {
		_, err := accountService.DepositToAccount(2, -50.0)
		assert.Error(t, err)
		assert.Equal(t, "deposit amount must be positive", err.Error())
		mockRepo.AssertNotCalled(t, "DepositToAccount")
	})
	t.Run("repository error", func(t *testing.T) {
		expectedError := errors.New("db error")
		mockRepo.On("DepositToAccount", 3, 200.0).Return(nil, expectedError).Once()
		_, err := accountService.DepositToAccount(3, 200.0)
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		mockRepo.AssertExpectations(t)
	})
}
