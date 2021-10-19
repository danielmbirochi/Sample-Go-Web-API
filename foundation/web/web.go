// Package web contains gluecode for abstracting observability, middlewares and error handling around http Handlers
package web

import (
	"context"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/dimfeld/httptreemux/v5"
	"github.com/google/uuid"
)

// ctxKey represents the type of value for the context key.
type ctxKey int

// KeyValues is how request metadata (type Values) are stored/retrieved.
const KeyValues ctxKey = 1

// Values represent metadata attached to requests for debugging purposes.
type Values struct {
	TraceID    string
	Now        time.Time
	StatusCode int
}

// Type Handler is an adapter to allow the use of custom method signature as native http.HandlerFunc
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

// Type App is the entrypoint into the web application, it configures context for http handlers and hooks up
// os.Signal from application inner layers. This can be extended for further behaviors.
type App struct {
	*httptreemux.ContextMux
	shutdown chan os.Signal
	mw       []Middleware
}

// Factory method for creating concrete App that handles http routes handling
func NewApp(shutdown chan os.Signal, mw ...Middleware) *App {
	app := App{
		ContextMux: httptreemux.NewContextMux(),
		shutdown:   shutdown,
		mw:         mw,
	}

	return &app
}

// Handle encapsulates concrete http.HandleFunc calls
// to abstract requests observability and error handling
func (a *App) Handle(method string, path string, handler Handler, mw ...Middleware) {

	handler = wrapMiddleware(mw, handler)

	handler = wrapMiddleware(a.mw, handler)

	h := func(w http.ResponseWriter, r *http.Request) {

		// Injects unique identifier & timestamp into request context to be processed
		v := Values{
			TraceID: uuid.New().String(),
			Now:     time.Now(),
		}
		ctx := context.WithValue(r.Context(), KeyValues, &v)

		if err := handler(ctx, w, r); err != nil {
			a.SignalShutdown()
			return
		}
	}

	a.ContextMux.Handle(method, path, h)
}

// SignalShutdown is for gracefully shutdown the application process hooking up OS signals
func (a *App) SignalShutdown() {
	a.shutdown <- syscall.SIGTERM
}
