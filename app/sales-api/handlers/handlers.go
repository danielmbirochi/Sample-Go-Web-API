// Package handlers containes the full set of handler functions and routes supported by the http api.
package handlers

import (
	"log"
	"net/http"
	"os"

	"github.com/danielmbirochi/go-sample-service/business/auth"
	middleware "github.com/danielmbirochi/go-sample-service/business/middlewares"
	"github.com/danielmbirochi/go-sample-service/foundation/web"
)

// API construct an http.Handler with all application routes defined.
func API(build string, shutdown chan os.Signal, log *log.Logger, a *auth.Auth) *web.App {
	app := web.NewApp(shutdown, middleware.Logger(log), middleware.Errors(log), middleware.Metrics(), middleware.Panics(log))

	// Register the healthcheck endpoint
	c := check{
		build: build,
	}
	app.Handle(http.MethodGet, "/v1/healthcheck", c.readiness)

	return app
}
