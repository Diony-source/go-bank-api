package handler

import (
	"encoding/json"
	"go-bank-api/common"
	"go-bank-api/logger"
	"go-bank-api/service"
	"net/http"

	"github.com/sirupsen/logrus"
)

type AccountHandler struct {
	service *service.AccountService
}

func NewAccountHandler(service *service.AccountService) *AccountHandler {
	return &AccountHandler{service: service}
}

// CreateAccount handles the request to create a new bank account.
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) *common.AppError {
	var req struct {
		Currency string `json:"currency" validate:"required,oneof=TRY USD EUR"`
	}
	if !common.ValidateAndDecode(w, r, &req) {
		return nil
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

// ListAccounts lists the accounts according to the incoming request.
func (h *AccountHandler) ListAccounts(w http.ResponseWriter, r *http.Request) *common.AppError {
	userID, ok := r.Context().Value(UserIDKey).(int)
	if !ok {
		return common.NewAppError(http.StatusUnauthorized, "Invalid user ID in token", nil)
	}
	userRole, ok := r.Context().Value(UserRoleKey).(string)
	if !ok {
		return common.NewAppError(http.StatusUnauthorized, "Invalid user role in token", nil)
	}

	log := logger.Log.WithFields(logrus.Fields{
		"user_id": userID,
		"role":    userRole,
	})
	log.Info("List accounts request received")

	accounts, err := h.service.ListAccountsForUser(userID, userRole)
	if err != nil {
		return common.NewAppError(http.StatusInternalServerError, "Could not retrieve accounts", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(accounts)

	return nil
}
