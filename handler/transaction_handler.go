package handler

import (
	"encoding/json"
	"go-bank-api/common"
	"go-bank-api/service"
	"net/http"
	"strconv"
)

// TransactionHandler holds dependencies for transaction-related handlers.
type TransactionHandler struct {
	service *service.TransactionService
}

// NewTransactionHandler creates a new TransactionHandler with its dependencies.
func NewTransactionHandler(s *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: s}
}

// CreateTransfer godoc
// @Summary      Transfer money from a specific account
// @Description  Handles the transfer of a specified amount from a specific account to another. The user must own the 'from' account.
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        fromAccountId path int true "The ID of the account to transfer funds from"
// @Param        transfer body service.TransferRequest true "Details of the financial transfer (to_account_id, amount)"
// @Success      201  {object}  model.Transaction
// @Failure      400  {object}  common.AppError "Bad Request (e.g., invalid ID, insufficient funds, etc.)"
// @Failure      401  {object}  common.AppError "Unauthorized: Invalid or missing token"
// @Failure      403  {object}  common.AppError "Forbidden: User does not own the source account"
// @Failure      404  {object}  common.AppError "Sender or receiver account not found"
// @Failure      500  {object}  common.AppError "Internal server error while processing transfer"
// @Router       /api/accounts/{fromAccountId}/transfers [post]
func (h *TransactionHandler) CreateTransfer(w http.ResponseWriter, r *http.Request) *common.AppError {
	// Extract user ID from token.
	userID, ok := r.Context().Value(UserIDKey).(int)
	if !ok {
		return common.NewAppError(http.StatusUnauthorized, "Invalid user ID in token", nil)
	}

	// Extract 'from' account ID from the URL path. This is more RESTful and secure.
	fromAccountIDStr := r.PathValue("fromAccountId")
	fromAccountID, err := strconv.Atoi(fromAccountIDStr)
	if err != nil {
		return common.NewAppError(http.StatusBadRequest, "Invalid source account ID in URL path", err)
	}

	// Decode the request body (which now only contains 'to_account_id' and 'amount').
	var req service.TransferRequest
	if err := common.ValidateAndDecode(r, &req); err != nil {
		return err
	}

	// Call the updated service method.
	transaction, err := h.service.TransferMoney(r.Context(), userID, fromAccountID, req)
	if err != nil {
		// Map specific business logic errors to appropriate HTTP status codes.
		switch err {
		case service.ErrSenderAccountNotFound, service.ErrReceiverAccountNotFound:
			return common.NewAppError(http.StatusNotFound, err.Error(), err)
		case service.ErrPermissionDenied:
			return common.NewAppError(http.StatusForbidden, err.Error(), err)
		case service.ErrInsufficientFunds, service.ErrCurrencyMismatch, service.ErrSameAccountTransfer, service.ErrInvalidAmount:
			return common.NewAppError(http.StatusBadRequest, err.Error(), err)
		default:
			return common.NewAppError(http.StatusInternalServerError, "Could not process transfer", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
	return nil
}

// ListTransactionsForAccount godoc
// @Summary      List account transaction history
// @Description  Retrieves the transaction history for a specific account owned by the authenticated user.
// @Tags         transactions
// @Produce      json
// @Security     BearerAuth
// @Param        accountId path int true "The ID of the account to retrieve transactions for"
// @Success      200  {array}   model.Transaction "A list of transactions for the account"
// @Failure      400  {object}  common.AppError "Invalid account ID in URL path"
// @Failure      401  {object}  common.AppError "Unauthorized: Invalid or missing token"
// @Failure      403  {object}  common.AppError "Forbidden: User does not own the specified account"
// @Failure      404  {object}  common.AppError "Account with the specified ID not found"
// @Failure      500  {object}  common.AppError "Internal server error while retrieving transactions"
// @Router       /api/accounts/{accountId}/transactions [get]
func (h *TransactionHandler) ListTransactionsForAccount(w http.ResponseWriter, r *http.Request) *common.AppError {
	// Extract user ID from the token context.
	userID, ok := r.Context().Value(UserIDKey).(int)
	if !ok {
		return common.NewAppError(http.StatusUnauthorized, "Invalid user ID in token", nil)
	}

	// Extract account ID from the URL path.
	accountIDStr := r.PathValue("accountId")
	accountID, err := strconv.Atoi(accountIDStr)
	if err != nil {
		return common.NewAppError(http.StatusBadRequest, "Invalid account ID in URL path", err)
	}

	// Call the service to get the transactions, which includes the authorization check.
	transactions, err := h.service.ListTransactionsForAccount(r.Context(), userID, accountID)
	if err != nil {
		switch err {
		case service.ErrAccountNotFound:
			return common.NewAppError(http.StatusNotFound, err.Error(), err)
		case service.ErrPermissionDenied:
			return common.NewAppError(http.StatusForbidden, err.Error(), err)
		default:
			return common.NewAppError(http.StatusInternalServerError, "Could not retrieve transactions", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)
	return nil
}
