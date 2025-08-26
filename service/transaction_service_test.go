// service/transaction_service_test.go
package service

import (
	"context"
	"database/sql"
	"errors"
	"go-bank-api/logger"
	"go-bank-api/model"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestMain runs setup before any tests in this package are executed.
func TestMain(m *testing.M) {
	// Initialize the logger for the test environment.
	logger.Init()
	// Run all tests
	exitCode := m.Run()
	// Exit
	os.Exit(exitCode)
}

// MockAccountRepository is a mock for IAccountRepository.
type MockAccountRepository struct{ mock.Mock }

func (m *MockAccountRepository) GetAccountForUpdate(tx *sql.Tx, id int) (*model.Account, error) {
	args := m.Called(tx, id)
	// Handle nil case for failed lookups
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}
func (m *MockAccountRepository) UpdateAccountBalance(tx *sql.Tx, id int, bal float64) error {
	args := m.Called(tx, id, bal)
	return args.Error(0)
}

// Unused methods needed to satisfy the interface
func (m *MockAccountRepository) CreateAccount(*model.Account) error                { return nil }
func (m *MockAccountRepository) GetAccountsByUserID(int) ([]*model.Account, error) { return nil, nil }
func (m *MockAccountRepository) GetAllAccounts() ([]*model.Account, error)         { return nil, nil }
func (m *MockAccountRepository) DepositToAccount(int, float64) (*model.Account, error) {
	return nil, nil
}

// MockTransactionRepository is a mock for ITransactionRepository.
type MockTransactionRepository struct{ mock.Mock }

func (m *MockTransactionRepository) CreateTransaction(tx *sql.Tx, tr *model.Transaction) error {
	args := m.Called(tx, tr)
	return args.Error(0)
}

func TestTransactionService_TransferMoney(t *testing.T) {
	// Setup
	db, dbMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mockAccountRepo := new(MockAccountRepository)
	mockTxnRepo := new(MockTransactionRepository)

	transactionService := NewTransactionService(db, mockAccountRepo, mockTxnRepo)

	ctx := context.Background()
	userID := 1
	req := TransferRequest{
		FromAccountID: 1,
		ToAccountID:   2,
		Amount:        100.0,
	}

	fromAccount := &model.Account{ID: 1, UserID: 1, Balance: 500.0, Currency: "TRY"}
	toAccount := &model.Account{ID: 2, UserID: 2, Balance: 200.0, Currency: "TRY"}

	// --- Test Case 1: Successful Transfer ---
	t.Run("success", func(t *testing.T) {
		// Expectations
		dbMock.ExpectBegin()
		mockAccountRepo.On("GetAccountForUpdate", mock.Anything, req.FromAccountID).Return(fromAccount, nil).Once()
		mockAccountRepo.On("GetAccountForUpdate", mock.Anything, req.ToAccountID).Return(toAccount, nil).Once()
		mockAccountRepo.On("UpdateAccountBalance", mock.Anything, fromAccount.ID, fromAccount.Balance-req.Amount).Return(nil).Once()
		mockAccountRepo.On("UpdateAccountBalance", mock.Anything, toAccount.ID, toAccount.Balance+req.Amount).Return(nil).Once()
		mockTxnRepo.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*model.Transaction")).Return(nil).Once()
		dbMock.ExpectCommit()

		// Execution
		_, err := transactionService.TransferMoney(ctx, req, userID)

		// Assertions
		assert.NoError(t, err)
		mockAccountRepo.AssertExpectations(t)
		mockTxnRepo.AssertExpectations(t)
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	// --- Test Case 2: Insufficient Funds ---
	t.Run("insufficient funds", func(t *testing.T) {
		// Setup
		fromAccountPoor := &model.Account{ID: 1, UserID: 1, Balance: 50.0, Currency: "TRY"} // Not enough balance

		// Expectations
		dbMock.ExpectBegin()
		mockAccountRepo.On("GetAccountForUpdate", mock.Anything, req.FromAccountID).Return(fromAccountPoor, nil).Once()
		mockAccountRepo.On("GetAccountForUpdate", mock.Anything, req.ToAccountID).Return(toAccount, nil).Once()
		dbMock.ExpectRollback()

		// Execution
		_, err := transactionService.TransferMoney(ctx, req, userID)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, ErrInsufficientFunds, err)
		mockAccountRepo.AssertExpectations(t)
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	// --- Test Case 3: DB Commit Fails ---
	t.Run("commit error", func(t *testing.T) {
		// Expectations
		dbMock.ExpectBegin()
		mockAccountRepo.On("GetAccountForUpdate", mock.Anything, req.FromAccountID).Return(fromAccount, nil).Once()
		mockAccountRepo.On("GetAccountForUpdate", mock.Anything, req.ToAccountID).Return(toAccount, nil).Once()
		mockAccountRepo.On("UpdateAccountBalance", mock.Anything, fromAccount.ID, fromAccount.Balance-req.Amount).Return(nil).Once()
		mockAccountRepo.On("UpdateAccountBalance", mock.Anything, toAccount.ID, toAccount.Balance+req.Amount).Return(nil).Once()
		mockTxnRepo.On("CreateTransaction", mock.Anything, mock.AnythingOfType("*model.Transaction")).Return(nil).Once()
		dbMock.ExpectCommit().WillReturnError(errors.New("commit failed"))

		// Execution
		_, err := transactionService.TransferMoney(ctx, req, userID)

		// Assertions
		assert.Error(t, err)
		mockAccountRepo.AssertExpectations(t)
		mockTxnRepo.AssertExpectations(t)
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})
}
