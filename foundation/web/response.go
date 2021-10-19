package web

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

// Encode business layer outputs to send back to clients.
func Respond(ctx context.Context, w http.ResponseWriter, data interface{}, statusCode int) error {

	// Set the status code for the request logger middleware.
	// If the context is missing the pointer to the injected Values type,
	// it`ll request the service to be shutdown gracefully.
	v, ok := ctx.Value(KeyValues).(*Values)
	if !ok {
		return NewShutdownError("web value missing from context")
	}
	v.StatusCode = statusCode

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if _, err := w.Write(jsonData); err != nil {
		return err
	}

	return nil
}

// RespondError sends an error response back to clients.
func RespondError(ctx context.Context, w http.ResponseWriter, err error) error {

	// If the error was of the type *Error, the handler has
	// a specific status code and error to return. That means,
	// it is a trusted error, so we can return it back to clients.
	if webErr, ok := errors.Cause(err).(*Error); ok {
		erRes := ErrorResponse{
			Error:  webErr.Err.Error(),
			Fields: webErr.Fields,
		}
		if err := Respond(ctx, w, erRes, webErr.Status); err != nil {
			return err
		}

		return nil
	}

	// If is not a trusted error, the handler sent back a 500 status code within
	// a boilerplate message
	erRes := ErrorResponse{
		Error: http.StatusText(http.StatusInternalServerError),
	}
	if err := Respond(ctx, w, erRes, http.StatusInternalServerError); err != nil {
		return err
	}

	return nil
}
