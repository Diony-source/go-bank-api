package handler

import (
	"encoding/json"
	"go-bank-api/common"
	"go-bank-api/logger"
	"go-bank-api/service"
	"net/http"

	"github.com/sirupsen/logrus"
)

// AccountHandler holds dependencies for account-related handlers.
type AccountHandler struct {
	service *service.AccountService
}

// NewAccountHandler creates a new AccountHandler with its dependencies.
func NewAccountHandler(service *service.AccountService) *AccountHandler {
	return &AccountHandler{service: service}
}

// CreateAccount handles the request to create a new bank account.
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) *common.AppError {
	var req struct {
		Currency string `json:"currency" validate:"required,oneof=TRY USD EUR"`
	}
	if err := common.ValidateAndDecode(r, &req); err != nil {
		return err
	}

	userID, ok := r.Context().Value(UserIDKey).(int)
	if !ok {
		return common.NewAppError(http.StatusUnauthorized, "Invalid user ID in token", nil)
	}

	log := logger.Log.WithFields(logrus.Fields{
		"user_id":  userID,
		"currency": req.Currency,
	})
	log.Info("Create account request received")

	account, err := h.service.CreateNewAccount(userID, req.Currency)
	if err != nil {
		return common.NewAppError(http.StatusInternalServerError, "Could not create account", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(account)

	return nil
}

// ListAccounts lists the accounts for the currently authenticated user.
func (h *AccountHandler) ListAccounts(w http.ResponseWriter, r *http.Request) *common.AppError {
	userID, ok := r.Context().Value(UserIDKey).(int)
	if !ok {
		return common.NewAppError(http.StatusUnauthorized, "Invalid user ID in token", nil)
	}

	log := logger.Log.WithField("user_id", userID)
	log.Info("List user's own accounts request received")

	accounts, err := h.service.ListAccountsForUser(userID)
	if err != nil {
		return common.NewAppError(http.StatusInternalServerError, "Could not retrieve accounts", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(accounts)

	return nil
}

// GetAllAccounts lists all accounts in the system. Admin only.
func (h *AccountHandler) GetAllAccounts(w http.ResponseWriter, r *http.Request) *common.AppError {
	adminID, _ := r.Context().Value(UserIDKey).(int)
	log := logger.Log.WithField("admin_user_id", adminID)
	log.Info("Admin request to list all accounts received")

	accounts, err := h.service.GetAllAccounts()
	if err != nil {
		return common.NewAppError(http.StatusInternalServerError, "Could not retrieve all accounts", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(accounts)

	return nil
}
