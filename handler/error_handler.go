package handler

import (
	"go-bank-api/common"
	"net/http"
)

func ErrorHandlingMiddleware(next func(http.ResponseWriter, *http.Request) *common.AppError) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := next(w, r); err != nil {
			err.Send(w)
		}
	}
}
