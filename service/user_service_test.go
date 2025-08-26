// service/user_service_test.go
package service

import (
	"errors"
	"go-bank-api/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) CreateUser(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}
func (m *mockUserRepo) GetUserByEmail(email string) (*model.User, error) {
	args := m.Called(email)
	return args.Get(0).(*model.User), args.Error(1)
}
func (m *mockUserRepo) GetAllUsers() ([]*model.User, error) {
	args := m.Called()
	return args.Get(0).([]*model.User), args.Error(1)
}
func (m *mockUserRepo) UpdateUserRole(userID int, newRole string) error {
	args := m.Called(userID, newRole)
	return args.Error(0)
}

func TestUserService_UpdateUserRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRepo := new(mockUserRepo)
		mockRepo.On("UpdateUserRole", 1, "admin").Return(nil).Once()

		userService := NewUserService(mockRepo)
		err := userService.UpdateUserRole(1, model.RoleAdmin)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := new(mockUserRepo)
		expectedError := errors.New("database error")
		mockRepo.On("UpdateUserRole", 2, "user").Return(expectedError).Once()

		userService := NewUserService(mockRepo)
		err := userService.UpdateUserRole(2, model.RoleUser)

		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid role", func(t *testing.T) {
		mockRepo := new(mockUserRepo)
		userService := NewUserService(mockRepo)

		err := userService.UpdateUserRole(3, "invalid_role")

		assert.Error(t, err)
		assert.Equal(t, "invalid role specified", err.Error())
		mockRepo.AssertNotCalled(t, "UpdateUserRole")
	})
}
