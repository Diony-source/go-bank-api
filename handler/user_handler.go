package handler

import (
	"encoding/json"
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

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.WithError(err).Error("Invalid request body for user registration")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log := logger.Log.WithFields(logrus.Fields{
		"username": req.Username,
		"email":    req.Email,
	})
	log.Info("User registration attempt started")

	hashedPassword, err := service.HashPassword(req.Password)
	if err != nil {
		log.Error("Password hashing failed during registration")
		http.Error(w, "Error processing request", http.StatusInternalServerError)
		return
	}

	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
	}

	if err := h.Repo.CreateUser(user); err != nil {
		log.Error("User creation in database failed")
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	log.WithField("user_id", user.ID).Info("User registered successfully")
	w.WriteHeader(http.StatusCreated)
	user.Password = ""
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.WithError(err).Error("Invalid request body for user login")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log := logger.Log.WithField("email", req.Email)
	log.Info("User login attempt started")

	user, err := h.Repo.GetUserByEmail(req.Email)
	if err != nil {
		log.Warn("Login failed: invalid credentials (user not found or db error)")
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	if !service.CheckPasswordHash(req.Password, user.Password) {
		log.Warn("Login failed: password mismatch")
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	tokenString, err := service.GenerateJWT(user.Email)
	if err != nil {
		log.Error("JWT generation failed during login")
		http.Error(w, "Error processing request", http.StatusInternalServerError)
		return
	}

	log.Info("User logged in successfully")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}
