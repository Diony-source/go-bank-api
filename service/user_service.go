package service

import (
	"errors"
	"go-bank-api/model"
	"go-bank-api/repository"
)

// UserService handles user-related business logic.
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService creates a new UserService.
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// UpdateUserRole validates the role and calls the repository to update it.
func (s *UserService) UpdateUserRole(userID int, newRole model.Role) error {
	// We ensure that only valid roles can be assigned.
	if newRole != model.RoleAdmin && newRole != model.RoleUser {
		return errors.New("invalid role specified")
	}
	// In the future, more complex logic can be added here.
	// e.g., "The last admin cannot demote themselves."

	return s.userRepo.UpdateUserRole(userID, string(newRole))
}
