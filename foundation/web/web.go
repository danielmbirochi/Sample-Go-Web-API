// Package web contains gluecode for abstracting observability, middlewares and error handling around http Handlers
package web

import (
	"context"
	"net/http"
	"os"
	"syscall"

	"github.com/dimfeld/httptreemux/v5"
)

// Type Handler is an adapter to allow the use of custom method signature as native http.HandlerFunc
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// Type App is the entrypoint into the web application, it configures context for http handlers and hooks up
// os.Signal from application inner layers. This can be extended for further behaviors.
type App struct {
	*httptreemux.ContextMux
	shutdown chan os.Signal
}

// Factory method for creating concrete App that handles http routes handling
func NewApp(shutdown chan os.Signal) *App {
	app := App{
		ContextMux: httptreemux.NewContextMux(),
		shutdown:   shutdown,
	}

	return &app
}

// Handle encapsulates concrete http.HandleFunc calls
// to abstract requests observability and error handling
func (a *App) Handle(method string, path string, handler Handler) {
	h := func(w http.ResponseWriter, r *http.Request) {

		// do some stuff here...

		if err := handler(r.Context(), w, r); err != nil {
			a.SignalShutdown()
			return
		}

		// do some stuff here...
	}

	a.ContextMux.Handle(method, path, h)
}

// SignalShutdown is for gracefully shutdown the application process hooking up OS signals
func (a *App) SignalShutdown() {
	a.shutdown <- syscall.SIGTERM
}
