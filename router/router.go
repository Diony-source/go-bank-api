package router

import (
	"go-bank-api/handler"
	"net/http"
)

func NewRouter(userHandler *handler.UserHandler) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/register", handler.ErrorHandlingMiddleware(userHandler.Register))
	mux.Handle("/login", handler.ErrorHandlingMiddleware(userHandler.Login))

	return mux
}
