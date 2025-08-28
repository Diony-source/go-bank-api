// file: service/account_service_test.go

package service

import (
	"database/sql"
	"errors"
	"go-bank-api/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockAccountRepoForAccountSvc is a mock implementation of IAccountRepository for testing the account service.
type mockAccountRepoForAccountSvc struct{ mock.Mock }

func (m *mockAccountRepoForAccountSvc) CreateAccount(account *model.Account) error {
	args := m.Called(account)
	return args.Error(0)
}

func (m *mockAccountRepoForAccountSvc) GetLastAccountNumber() (int64, error) {
	args := m.Called()
	// We cast to int64 because the mock framework returns interface{}
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockAccountRepoForAccountSvc) DepositToAccount(id int, amount float64) (*model.Account, error) {
	args := m.Called(id, amount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

// --- Unused methods that are required to satisfy the interface contract ---
func (m *mockAccountRepoForAccountSvc) GetAccountsByUserID(int) ([]*model.Account, error) {
	return nil, nil
}
func (m *mockAccountRepoForAccountSvc) GetAllAccounts() ([]*model.Account, error) { return nil, nil }
func (m *mockAccountRepoForAccountSvc) GetAccountForUpdate(*sql.Tx, int) (*model.Account, error) {
	return nil, nil
}
func (m *mockAccountRepoForAccountSvc) UpdateAccountBalance(*sql.Tx, int, float64) error { return nil }

// TestAccountService_CreateNewAccount tests the sequential account number generation logic.
func TestAccountService_CreateNewAccount(t *testing.T) {
	mockRepo := new(mockAccountRepoForAccountSvc)
	accountService := NewAccountService(mockRepo)

	userID := 1
	currency := "TRY"
	// This is the last account number currently in the "database"
	lastAccountNumber := int64(1000000025)

	// We expect the GetLastAccountNumber function to be called.
	mockRepo.On("GetLastAccountNumber").Return(lastAccountNumber, nil).Once()

	// We expect the CreateAccount function to be called with the NEXT account number.
	expectedNewAccountNumber := lastAccountNumber + 1
	mockRepo.On("CreateAccount", mock.MatchedBy(func(acc *model.Account) bool {
		return acc.AccountNumber == expectedNewAccountNumber && acc.UserID == userID
	})).Return(nil).Once()

	// Execute the service method
	account, err := accountService.CreateNewAccount(userID, currency)

	// Assert the results
	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, expectedNewAccountNumber, account.AccountNumber)
	// Ensure all mock expectations were met
	mockRepo.AssertExpectations(t)
}

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
