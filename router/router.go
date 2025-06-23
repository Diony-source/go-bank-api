package router

import (
	"go-bank-api/handler"
	"net/http"
)

func NewRouter(userHandler *handler.UserHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/register", userHandler.Register)
	return mux
}
