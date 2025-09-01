package handler

import (
	"encoding/json"
	"go-bank-api/common"
	"go-bank-api/logger"
	"go-bank-api/model"
	"go-bank-api/service"
	"net/http"
	"strconv"

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

// CreateAccount godoc
// @Summary      Create a new bank account
// @Description  Creates a new bank account for the authenticated user. Supported currencies: TRY, USD, EUR.
// @Tags         accounts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body object{currency=string} true "Account Creation Request with currency"
// @Success      201  {object}  model.Account
// @Failure      400  {object}  common.AppError "Invalid request body or unsupported currency"
// @Failure      401  {object}  common.AppError "Unauthorized: Invalid or missing token"
// @Failure      500  {object}  common.AppError "Internal server error while creating account"
// @Router       /api/accounts [post]
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

// ListAccounts godoc
// @Summary      List user's accounts
// @Description  Retrieves a list of bank accounts for the currently authenticated user.
// @Tags         accounts
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   model.Account
// @Failure      401  {object}  common.AppError "Unauthorized: Invalid or missing token"
// @Failure      500  {object}  common.AppError "Internal server error while retrieving accounts"
// @Router       /api/accounts [get]
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

// GetAllAccounts godoc
// @Summary      Get all accounts (Admin)
// @Description  Retrieves a list of all bank accounts in the system. Admin access required.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   model.Account
// @Failure      401  {object}  common.AppError "Unauthorized: Invalid or missing token"
// @Failure      403  {object}  common.AppError "Forbidden: User does not have admin privileges"
// @Failure      500  {object}  common.AppError "Internal server error while retrieving all accounts"
// @Router       /api/admin/accounts [get]
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

// DepositToAccount godoc
// @Summary      Deposit funds into an account (Admin)
// @Description  Deposits a specified amount into a user's account. Admin access is required.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        accountId path int true "Account ID to deposit funds into"
// @Param        request body model.DepositRequest true "Deposit Amount"
// @Success      200  {object}  model.Account "The updated account details"
// @Failure      400  {object}  common.AppError "Invalid account ID or request body"
// @Failure      401  {object}  common.AppError "Unauthorized: Invalid or missing token"
// @Failure      403  {object}  common.AppError "Forbidden: User does not have admin privileges"
// @Failure      404  {object}  common.AppError "Account with the specified ID not found"
// @Failure      500  {object}  common.AppError "Internal server error while processing the deposit"
// @Router       /api/admin/accounts/{accountId}/deposit [post]
func (h *AccountHandler) DepositToAccount(w http.ResponseWriter, r *http.Request) *common.AppError {
	// Extract account ID from URL path.
	accountIDStr := r.PathValue("accountId")
	accountID, err := strconv.Atoi(accountIDStr)
	if err != nil {
		return common.NewAppError(http.StatusBadRequest, "Invalid account ID in URL path", err)
	}

	// Decode and validate the request body.
	var req model.DepositRequest
	if err := common.ValidateAndDecode(r, &req); err != nil {
		return err
	}

	log := logger.Log.WithFields(logrus.Fields{
		"target_account_id": accountID,
		"amount":            req.Amount,
		"admin_user_id":     r.Context().Value(UserIDKey),
	})
	log.Info("Admin deposit request received")

	// Call the service to perform the deposit.
	updatedAccount, err := h.service.DepositToAccount(accountID, req.Amount)
	if err != nil {
		// Map service-level errors to appropriate HTTP status codes.
		if err.Error() == "account not found" {
			return common.NewAppError(http.StatusNotFound, err.Error(), err)
		}
		return common.NewAppError(http.StatusInternalServerError, "Could not process deposit", err)
	}

	log.Info("Deposit successful")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedAccount)

	return nil
}
