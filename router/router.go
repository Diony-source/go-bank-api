package router

import (
	"go-bank-api/handler"
	"net/http"
)

// NewRouter sets up all application routes and their corresponding handlers.
func NewRouter(userHandler *handler.UserHandler, accountHandler *handler.AccountHandler, transactionHandler *handler.TransactionHandler) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.Handle("POST /register", handler.ErrorHandlingMiddleware(userHandler.Register))
	mux.Handle("POST /login", handler.ErrorHandlingMiddleware(userHandler.Login))

	// Authenticated routes
	mux.Handle("GET /api/accounts", handler.AuthMiddleware(handler.ErrorHandlingMiddleware(accountHandler.ListAccounts)))
	mux.Handle("POST /api/accounts", handler.AuthMiddleware(handler.ErrorHandlingMiddleware(accountHandler.CreateAccount)))
	mux.Handle("POST /api/transfers", handler.AuthMiddleware(handler.ErrorHandlingMiddleware(transactionHandler.CreateTransfer)))

	// Admin-only routes
	mux.Handle("GET /api/admin/users", handler.AuthMiddleware(handler.AdminMiddleware(handler.ErrorHandlingMiddleware(userHandler.GetAllUsers))))
	mux.Handle("PATCH /api/admin/users/{id}/role",
		handler.AuthMiddleware(
			handler.AdminMiddleware(
				handler.ErrorHandlingMiddleware(userHandler.UpdateUserRole),
			),
		),
	)
	mux.Handle("GET /api/admin/accounts",
		handler.AuthMiddleware(
			handler.AdminMiddleware(
				handler.ErrorHandlingMiddleware(accountHandler.GetAllAccounts),
			),
		),
	)

	// Health Check
	mux.HandleFunc("GET /health", handler.HealthCheck)

	return mux
}