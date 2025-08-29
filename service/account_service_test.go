// file: service/account_service_test.go

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"go-bank-api/model"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockAccountRepo provides a mock for IAccountRepository.
type mockAccountRepo struct{ mock.Mock }

func (m *mockAccountRepo) CreateAccount(a *model.Account) error { return m.Called(a).Error(0) }
func (m *mockAccountRepo) GetLastAccountNumber() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockAccountRepo) DepositToAccount(id int, a float64) (*model.Account, error) {
	args := m.Called(id, a)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}
func (m *mockAccountRepo) GetAccountsByUserID(id int) ([]*model.Account, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Account), args.Error(1)
}
func (m *mockAccountRepo) GetAllAccounts() ([]*model.Account, error)                { return nil, nil }
func (m *mockAccountRepo) GetAccountForUpdate(*sql.Tx, int) (*model.Account, error) { return nil, nil }
func (m *mockAccountRepo) UpdateAccountBalance(*sql.Tx, int, float64) error         { return nil }

// mockCacheClient provides a mock for ICacheClient, implementing the interface directly.
type mockCacheClient struct{ mock.Mock }

func (m *mockCacheClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	cmd := redis.NewStringCmd(ctx)
	cmd.SetVal(args.String(0))
	cmd.SetErr(args.Error(1))
	return cmd
}
func (m *mockCacheClient) Set(ctx context.Context, key string, val interface{}, exp time.Duration) *redis.StatusCmd {
	m.Called(ctx, key, val, exp)
	return redis.NewStatusCmd(ctx)
}
func (m *mockCacheClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	m.Called(ctx, keys[0])
	return redis.NewIntCmd(ctx)
}

// --- Test Suites ---

func TestAccountService_CreateNewAccount(t *testing.T) {
	mockRepo := new(mockAccountRepo)
	mockCache := new(mockCacheClient)
	accountService := NewAccountService(mockRepo, mockCache) // Inject the mock directly.

	userID := 1
	cacheKey := fmt.Sprintf("accounts:%d", userID)
	mockRepo.On("GetLastAccountNumber").Return(int64(1000000025), nil).Once()
	mockRepo.On("CreateAccount", mock.Anything).Return(nil).Once()
	mockCache.On("Del", mock.Anything, cacheKey).Return().Once()

	_, err := accountService.CreateNewAccount(userID, "TRY")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestAccountService_ListAccountsForUser_CacheHit(t *testing.T) {
	mockRepo := new(mockAccountRepo)
	mockCache := new(mockCacheClient)
	accountService := NewAccountService(mockRepo, mockCache)

	userID := 2
	cacheKey := fmt.Sprintf("accounts:%d", userID)
	expectedAccounts := []*model.Account{{ID: 1, UserID: userID}}
	cachedData, _ := json.Marshal(expectedAccounts)

	mockCache.On("Get", mock.Anything, cacheKey).Return(string(cachedData), nil).Once()

	accounts, err := accountService.ListAccountsForUser(userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedAccounts, accounts)
	mockRepo.AssertNotCalled(t, "GetAccountsByUserID")
	mockCache.AssertExpectations(t)
}

func TestAccountService_ListAccountsForUser_CacheMiss(t *testing.T) {
	mockRepo := new(mockAccountRepo)
	mockCache := new(mockCacheClient)
	accountService := NewAccountService(mockRepo, mockCache)

	userID := 3
	cacheKey := fmt.Sprintf("accounts:%d", userID)
	dbAccounts := []*model.Account{{ID: 2, UserID: userID}}
	dbData, _ := json.Marshal(dbAccounts)

	mockCache.On("Get", mock.Anything, cacheKey).Return("", redis.Nil).Once()
	mockRepo.On("GetAccountsByUserID", userID).Return(dbAccounts, nil).Once()
	mockCache.On("Set", mock.Anything, cacheKey, dbData, 10*time.Minute).Return().Once()

	accounts, err := accountService.ListAccountsForUser(userID)

	assert.NoError(t, err)
	assert.Equal(t, dbAccounts, accounts)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestAccountService_DepositToAccount(t *testing.T) {
	mockRepo := new(mockAccountRepo)
	mockCache := new(mockCacheClient)
	accountService := NewAccountService(mockRepo, mockCache)

	t.Run("success", func(t *testing.T) {
		mockRepo.On("DepositToAccount", 1, 100.0).Return(&model.Account{ID: 1}, nil).Once()
		_, err := accountService.DepositToAccount(1, 100.0)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
	t.Run("negative amount", func(t *testing.T) {
		_, err := accountService.DepositToAccount(2, -50.0)
		assert.Error(t, err)
		mockRepo.AssertNotCalled(t, "DepositToAccount")
	})
}
