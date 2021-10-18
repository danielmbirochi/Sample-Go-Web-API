// Package handlers containes the full set of handler functions and routes supported by the http api.
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/dimfeld/httptreemux"
)

// API construct an http.Handler with all application routes defined.
func API(build string, shutdown chan os.Signal, log *log.Logger) *httptreemux.ContextMux {
	mux := httptreemux.NewContextMux()

	h := func(w http.ResponseWriter, r *http.Request) {
		message := struct {
			Message string
		}{
			Message: "Application is running ...",
		}
		json.NewEncoder(w).Encode(message)
	}

	mux.Handle(http.MethodGet, "/healthcheck", h)

	return mux
}
