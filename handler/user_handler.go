// file: handler/user_handler.go

package handler

import (
	"database/sql"
	"encoding/json"
	"go-bank-api/common"
	"go-bank-api/logger"
	"go-bank-api/model"
	"go-bank-api/repository"
	"go-bank-api/service"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"
)

// UserHandler holds dependencies for user-related handlers.
// It now includes AuthService to handle complex authentication logic like token generation.
type UserHandler struct {
	userRepo    repository.IUserRepository
	userService *service.UserService
	authService *service.AuthService // <-- NEW DEPENDENCY
}

// NewUserHandler creates a new UserHandler with its dependencies.
// The signature is updated to accept an AuthService instance.
func NewUserHandler(userRepo repository.IUserRepository, userService *service.UserService, authService *service.AuthService) *UserHandler {
	return &UserHandler{
		userRepo:    userRepo,
		userService: userService,
		authService: authService,
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a new user account
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        user body model.RegisterRequest true "User Registration Info"
// @Success      201  {object}  model.User
// @Failure      400  {object}  common.AppError
// @Failure      500  {object}  common.AppError
// @Router       /register [post]
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) *common.AppError {
	var req model.RegisterRequest
	if err := common.ValidateAndDecode(r, &req); err != nil {
		return err
	}

	log := logger.Log.WithFields(logrus.Fields{"username": req.Username, "email": req.Email})
	log.Info("User registration attempt started")

	// Hashing logic is now encapsulated within AuthService.
	hashedPassword, err := h.authService.HashPassword(req.Password)
	if err != nil {
		return common.NewAppError(http.StatusInternalServerError, "Could not process request", err)
	}

	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
	}

	// The repository interface for user creation is now used.
	if err := h.userRepo.CreateUser(user); err != nil {
		return common.NewAppError(http.StatusInternalServerError, "Could not create user", err)
	}

	log.WithField("user_id", user.ID).Info("User registered successfully")
	w.WriteHeader(http.StatusCreated)
	user.Password = "" // Ensure password is not returned.
	json.NewEncoder(w).Encode(user)

	return nil
}

// Login godoc
// @Summary      User login
// @Description  Authenticates a user and returns a JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials body model.LoginRequest true "User Credentials"
// @Success      200  {object}  map[string]string "{"token": "..."}"
// @Failure      400  {object}  common.AppError "Invalid request body"
// @Failure      401  {object}  common.AppError "Invalid email or password"
// @Failure      500  {object}  common.AppError "Internal server error"
// @Router       /login [post]
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) *common.AppError {
	var req model.LoginRequest
	if err := common.ValidateAndDecode(r, &req); err != nil {
		return err
	}

	log := logger.Log.WithField("email", req.Email)
	log.Info("User login attempt started")

	// All authentication and token generation logic is now delegated to the AuthService.
	tokenPair, err := h.authService.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		// AuthService returns a generic error for security, which we map to Unauthorized.
		return common.NewAppError(http.StatusUnauthorized, "Invalid email or password", err)
	}

	log.Info("User logged in successfully, token pair generated")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// The response now includes both access and refresh tokens.
	json.NewEncoder(w).Encode(tokenPair)

	return nil
}

// GetAllUsers godoc
// @Summary      Get all users
// @Description  Retrieves a list of all users. Admin access required.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   model.User
// @Failure      401  {object}  common.AppError "Unauthorized"
// @Failure      403  {object}  common.AppError "Forbidden"
// @Failure      500  {object}  common.AppError "Internal server error"
// @Router       /api/admin/users [get]
func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) *common.AppError {
	logger.Log.Info("Admin request to list all users received")

	users, err := h.userRepo.GetAllUsers()
	if err != nil {
		return common.NewAppError(http.StatusInternalServerError, "Could not retrieve users", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(users)

	return nil
}

// UpdateUserRole godoc
// @Summary      Update a user's role
// @Description  Updates the role of a specific user. This is an admin-only endpoint.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID to be updated"
// @Param        role body      model.UpdateUserRoleRequest true "The new role for the user"
// @Success      200  {object}  map[string]string "{"message": "User role updated successfully"}"
// @Failure      400  {object}  common.AppError "Invalid user ID in URL path or invalid request body"
// @Failure      401  {object}  common.AppError "Unauthorized: Invalid or missing token"
// @Failure      403  {object}  common.AppError "Forbidden: User does not have admin privileges"
// @Failure      404  {object}  common.AppError "User with the specified ID not found"
// @Failure      500  {object}  common.AppError "Internal server error while updating user role"
// @Router       /api/admin/users/{id}/role [patch]
func (h *UserHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) *common.AppError {
	userIDStr := r.PathValue("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return common.NewAppError(http.StatusBadRequest, "Invalid user ID in URL path", err)
	}

	var req model.UpdateUserRoleRequest
	if err := common.ValidateAndDecode(r, &req); err != nil {
		return err
	}

	log := logger.Log.WithFields(logrus.Fields{"user_id_to_update": userID, "new_role": req.Role})
	log.Info("Admin request to update user role received")

	if err := h.userService.UpdateUserRole(userID, req.Role); err != nil {
		if err == sql.ErrNoRows {
			return common.NewAppError(http.StatusNotFound, "User with the specified ID not found", err)
		}
		return common.NewAppError(http.StatusInternalServerError, "Could not update user role", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User role updated successfully"})
	return nil
}
