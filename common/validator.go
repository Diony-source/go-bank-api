package common

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ValidateAndDecode(r *http.Request, payload interface{}) *AppError {
	if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
		return NewAppError(http.StatusBadRequest, "Invalid request body", err)
	}

	if err := validate.Struct(payload); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		return NewAppError(http.StatusBadRequest, validationErrors.Error(), err)
	}

	return nil
}

