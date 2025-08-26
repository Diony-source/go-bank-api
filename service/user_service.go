package service

import (
	"errors"
	"go-bank-api/model"
	"go-bank-api/repository"
)

// UserService now depends on the IUserRepository interface, not the concrete struct.
type UserService struct {
	userRepo repository.IUserRepository // UPDATED
}

// NewUserService accepts the interface, allowing for mocks to be injected.
func NewUserService(userRepo repository.IUserRepository) *UserService { // UPDATED
	return &UserService{userRepo: userRepo}
}

// UpdateUserRole validates the role and calls the repository to update it.
func (s *UserService) UpdateUserRole(userID int, newRole model.Role) error {
	// We ensure that only valid roles can be assigned.
	if newRole != model.RoleAdmin && newRole != model.RoleUser {
		return errors.New("invalid role specified")
	}

	return s.userRepo.UpdateUserRole(userID, string(newRole))
}
