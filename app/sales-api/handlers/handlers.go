// Package handlers containes the full set of handler functions and routes supported by the http api.
package handlers

import (
	"log"
	"net/http"
	"os"

	middleware "github.com/danielmbirochi/go-sample-service/business/middlewares"
	"github.com/danielmbirochi/go-sample-service/foundation/web"
)

// API construct an http.Handler with all application routes defined.
func API(build string, shutdown chan os.Signal, log *log.Logger) *web.App {
	app := web.NewApp(shutdown, middleware.Logger(log))

	// Register the healthcheck endpoint
	c := check{
		build: build,
	}
	app.Handle(http.MethodGet, "/v1/healthcheck", c.readiness)

	return app
}
