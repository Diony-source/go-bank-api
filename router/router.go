package router

import (
	"go-bank-api/handler"
	"net/http"
)

func NewRouter(userHandler *handler.UserHandler, accountHandler *handler.AccountHandler) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("POST /register", handler.ErrorHandlingMiddleware(userHandler.Register))
	mux.Handle("POST /login", handler.ErrorHandlingMiddleware(userHandler.Login))

	accountListHandler := handler.ErrorHandlingMiddleware(accountHandler.ListAccounts)
	mux.Handle("GET /api/accounts", handler.AuthMiddleware(accountListHandler))

	accountCreateHandler := handler.ErrorHandlingMiddleware(accountHandler.CreateAccount)
	mux.Handle("POST /api/accounts", handler.AuthMiddleware(accountCreateHandler))

	getAllUsersHandler := handler.ErrorHandlingMiddleware(userHandler.GetAllUsers)
	adminProtectedHandler := handler.AdminMiddleware(getAllUsersHandler)
	authProtectedHandler := handler.AuthMiddleware(adminProtectedHandler)
	mux.Handle("GET /api/admin/users", authProtectedHandler)

	return mux
}
