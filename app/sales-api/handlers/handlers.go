// Package handlers containes the full set of handler functions and routes supported by the http api.
package handlers

import (
	"log"
	"net/http"
	"os"

	"github.com/danielmbirochi/go-sample-service/foundation/web"
)

// API construct an http.Handler with all application routes defined.
func API(build string, shutdown chan os.Signal, log *log.Logger) *web.App {
	app := web.NewApp(shutdown)

	app.Handle(http.MethodGet, "/healthcheck", readiness)

	return app
}
