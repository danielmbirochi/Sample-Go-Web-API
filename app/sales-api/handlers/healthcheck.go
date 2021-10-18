package handlers

import (
	"context"
	"encoding/json"
	"net/http"
)

func readiness(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	message := struct {
		Message string
	}{
		Message: "Application is running ...",
	}
	return json.NewEncoder(w).Encode(message)
}
