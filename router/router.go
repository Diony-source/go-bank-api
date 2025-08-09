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
	mux.Handle("GET /api/accounts", handler.AuthMiddleware(handler.ErrorHandlingMiddleware(accountHandler.ListAccounts)))
	mux.Handle("POST /api/accounts", handler.AuthMiddleware(handler.ErrorHandlingMiddleware(accountHandler.CreateAccount)))

	// Auth and Admin protected routes
	mux.Handle("GET /api/admin/users", handler.AuthMiddleware(handler.AdminMiddleware(handler.ErrorHandlingMiddleware(userHandler.GetAllUsers))))

	// Health Check
	mux.HandleFunc("GET /health", handler.HealthCheck)

	return mux
}
