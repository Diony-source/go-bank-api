// file: model/request.go

package model

// RegisterRequest defines the payload for creating a new user.
// It includes validation tags to ensure data integrity at the entry point.
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginRequest defines the payload for user authentication.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// UpdateUserRoleRequest defines the payload for updating a user's role.
// Using a dedicated struct instead of an inline anonymous struct in the handler
// improves code clarity, reusability, and compatibility with tooling like swag.
type UpdateUserRoleRequest struct {
	Role Role `json:"role" validate:"required,oneof=admin user"`
}
