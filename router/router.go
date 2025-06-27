package router

import (
	"go-bank-api/handler"
	"net/http"
)

func NewRouter(userHandler *handler.UserHandler, accountHandler *handler.AccountHandler) http.Handler {
	mux := http.NewServeMux()

	// === Public Routes ===
	// These routes do not require a token.
	mux.Handle("/register", handler.ErrorHandlingMiddleware(userHandler.Register))
	mux.Handle("/login", handler.ErrorHandlingMiddleware(userHandler.Login))

	// === Protected API Routes ===
	// These routes require a valid JWT (Authorization header) for access.

	// Route for creating a new account
	accountListHandler := handler.ErrorHandlingMiddleware(accountHandler.ListAccounts)
	mux.Handle("GET /api/accounts", handler.AuthMiddleware(accountListHandler))

	accountCreateHandler := handler.ErrorHandlingMiddleware(accountHandler.CreateAccount)
	mux.Handle("POST /api/accounts", handler.AuthMiddleware(accountCreateHandler))

	return mux
}
