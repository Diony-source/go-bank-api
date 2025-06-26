package common

import (
	"encoding/json"
	"go-bank-api/logger"
	"net/http"

	"github.com/sirupsen/logrus"
)

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func (e *AppError) Send(w http.ResponseWriter) {
	if e.Err != nil {
		logger.Log.WithFields(logrus.Fields{
			"status_code":    e.Code,
			"internal_error": e.Err.Error(),
		}).Error(e.Message)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(e.Code)
	json.NewEncoder(w).Encode(e)
}
