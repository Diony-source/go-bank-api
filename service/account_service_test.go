// service/account_service_test.go
package service

import (
	"database/sql"
	"errors"
	"go-bank-api/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAccountRepoForAccountSvc struct{ mock.Mock }

func (m *mockAccountRepoForAccountSvc) CreateAccount(account *model.Account) error {
	args := m.Called(account)
	return args.Error(0)
}
func (m *mockAccountRepoForAccountSvc) DepositToAccount(id int, amount float64) (*model.Account, error) {
	args := m.Called(id, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}
func (m *mockAccountRepoForAccountSvc) GetAccountsByUserID(int) ([]*model.Account, error) {
	return nil, nil
}
func (m *mockAccountRepoForAccountSvc) GetAllAccounts() ([]*model.Account, error) { return nil, nil }
func (m *mockAccountRepoForAccountSvc) GetAccountForUpdate(*sql.Tx, int) (*model.Account, error) {
	return nil, nil
}
func (m *mockAccountRepoForAccountSvc) UpdateAccountBalance(*sql.Tx, int, float64) error { return nil }

func TestAccountService_DepositToAccount(t *testing.T) {
	mockRepo := new(mockAccountRepoForAccountSvc)
	accountService := NewAccountService(mockRepo)

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

func TestAccountService_CreateNewAccount(t *testing.T) {
	mockRepo := new(mockAccountRepoForAccountSvc)
	accountService := NewAccountService(mockRepo)

	userID := 1
	currency := "TRY"

	mockRepo.On("CreateAccount", mock.AnythingOfType("*model.Account")).Return(nil).Once()

	account, err := accountService.CreateNewAccount(userID, currency)

	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, userID, account.UserID)
	assert.Equal(t, currency, account.Currency)
	assert.True(t, account.AccountNumber > 0)
	mockRepo.AssertExpectations(t)
}
