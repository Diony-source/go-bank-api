package handler

import (
	"encoding/json"
	"net/http"
)

// HealthCheck godoc
// @Summary      Show the status of server
// @Description  get the status of server
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "API is healthy and running"})
}
