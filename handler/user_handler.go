package handler

import (
	"encoding/json"
	"go-bank-api/common"
	"go-bank-api/logger"
	"go-bank-api/model"
	"go-bank-api/repository"
	"go-bank-api/service"
	"net/http"

	"github.com/sirupsen/logrus"
)

type UserHandler struct {
	Repo *repository.UserRepository
}

func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{Repo: repo}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) *common.AppError {
	var req model.RegisterRequest
	if err := common.ValidateAndDecode(r, &req); err != nil {
		return err
	}

	log := logger.Log.WithFields(logrus.Fields{"username": req.Username, "email": req.Email})
	log.Info("User registration attempt started")

	hashedPassword, err := service.HashPassword(req.Password)
	if err != nil {
		return common.NewAppError(http.StatusInternalServerError, "Could not process request", err)
	}

	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
	}

	if err := h.Repo.CreateUser(user); err != nil {
		return common.NewAppError(http.StatusInternalServerError, "Could not create user", err)
	}

	log.WithField("user_id", user.ID).Info("User registered successfully")
	w.WriteHeader(http.StatusCreated)
	user.Password = ""
	json.NewEncoder(w).Encode(user)

	return nil
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) *common.AppError {
	var req model.LoginRequest
	if err := common.ValidateAndDecode(r, &req); err != nil {
		return err
	}

	log := logger.Log.WithField("email", req.Email)
	log.Info("User login attempt started")

	user, err := h.Repo.GetUserByEmail(req.Email)
	if err != nil {
		return common.NewAppError(http.StatusUnauthorized, "Invalid email or password", err)
	}

	if !service.CheckPasswordHash(req.Password, user.Password) {
		return common.NewAppError(http.StatusUnauthorized, "Invalid email or password", nil)
	}

	tokenString, err := service.GenerateJWT(user)
	if err != nil {
		return common.NewAppError(http.StatusInternalServerError, "Could not generate token", err)
	}

	log.Info("User logged in successfully")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})

	return nil
}

// GetAllUsers lists all users in the system. Admin only.
func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) *common.AppError {
	logger.Log.Info("Admin request to list all users received")

	users, err := h.Repo.GetAllUsers()
	if err != nil {
		return common.NewAppError(http.StatusInternalServerError, "Could not retrieve users", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(users)

	return nil
}
