package common

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ValidateAndDecode(w http.ResponseWriter, r *http.Request, payload interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return false
	}

	if err := validate.Struct(payload); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		http.Error(w, validationErrors.Error(), http.StatusBadRequest)
		return false
	}

	return true
}
