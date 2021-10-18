package handlers

import (
	"encoding/json"
	"net/http"
)

func readiness(w http.ResponseWriter, r *http.Request) {
	message := struct {
		Message string
	}{
		Message: "Application is running ...",
	}
	json.NewEncoder(w).Encode(message)
}
