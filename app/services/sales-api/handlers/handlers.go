// Package handlers containes the full set of handler functions and routes supported by the http api.
package handlers

import (
	"expvar"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/danielmbirochi/go-sample-service/business/auth"
	"github.com/danielmbirochi/go-sample-service/business/core/user"
	middleware "github.com/danielmbirochi/go-sample-service/business/middlewares"
	"github.com/danielmbirochi/go-sample-service/foundation/web"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// API construct an http.Handler with all application routes defined.
func API(build string, shutdown chan os.Signal, log *zap.SugaredLogger, a *auth.Auth, db *sqlx.DB) *web.App {
	app := web.NewApp(shutdown, middleware.Logger(log), middleware.Errors(log), middleware.Metrics(), middleware.Panics(log))

	// Register the healthcheck endpoint
	c := check{
		build: build,
		db:    db,
	}
	app.Handle(http.MethodGet, "/v1/healthcheck", c.readiness)

	// Register endpoints for accessing user service.
	uh := usersHandler{
		usecases: user.New(log, db),
		auth:     a,
	}
	app.Handle(http.MethodGet, "/v1/users/:page/:rows", uh.list, middleware.Authenticate(a), middleware.Authorize(auth.RoleAdmin))
	app.Handle(http.MethodGet, "/v1/users/token/:kid", uh.token)
	app.Handle(http.MethodGet, "/v1/users/:id", uh.queryByID, middleware.Authenticate(a))
	app.Handle(http.MethodPost, "/v1/users", uh.create, middleware.Authenticate(a), middleware.Authorize(auth.RoleAdmin))
	app.Handle(http.MethodPut, "/v1/users/:id", uh.update, middleware.Authenticate(a), middleware.Authorize(auth.RoleAdmin))
	app.Handle(http.MethodDelete, "/v1/users/:id", uh.delete, middleware.Authenticate(a), middleware.Authorize(auth.RoleAdmin))

	return app
}

// DebugStandardLibraryMux registers all the debug routes from the std library
// into a new mux. This is done to avoid the usage of DefaultServerMux, since a
// dependency could injects a handler into it.
func DebugStandardLibraryMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/vars", expvar.Handler())

	return mux
}
