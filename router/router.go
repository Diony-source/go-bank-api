package router

import (
	"go-bank-api/handler"
	"net/http"
)

func NewRouter(userHandler *handler.UserHandler, accountHandler *handler.AccountHandler) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.Handle("POST /register", handler.ErrorHandlingMiddleware(userHandler.Register))
	mux.Handle("POST /login", handler.ErrorHandlingMiddleware(userHandler.Login))

	// Auth protected User routes
	// This endpoint now correctly serves ONLY the user's own accounts for ANY authenticated user (user or admin).
	mux.Handle("GET /api/accounts", handler.AuthMiddleware(handler.ErrorHandlingMiddleware(accountHandler.ListAccounts)))
	mux.Handle("POST /api/accounts", handler.AuthMiddleware(handler.ErrorHandlingMiddleware(accountHandler.CreateAccount)))

	// Auth and Admin protected routes
	mux.Handle("GET /api/admin/users", handler.AuthMiddleware(handler.AdminMiddleware(handler.ErrorHandlingMiddleware(userHandler.GetAllUsers))))
	mux.Handle("PATCH /api/admin/users/{id}/role",
		handler.AuthMiddleware(
			handler.AdminMiddleware(
				handler.ErrorHandlingMiddleware(userHandler.UpdateUserRole),
			),
		),
	)

	// NEW: Admin-only route to get all accounts in the system.
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
