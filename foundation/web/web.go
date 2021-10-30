// Package web contains gluecode for abstracting observability, middlewares and error handling around http Handlers
package web

import (
	"context"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/dimfeld/httptreemux/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
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
	mux      *httptreemux.ContextMux
	otmux    http.Handler
	shutdown chan os.Signal
	mw       []Middleware
}

// Factory method for creating concrete App that handles http routes handling
func NewApp(shutdown chan os.Signal, mw ...Middleware) *App {

	mux := httptreemux.NewContextMux()

	return &App{
		mux:      mux,
		otmux:    otelhttp.NewHandler(mux, "request"),
		shutdown: shutdown,
		mw:       mw,
	}
}

// Handle encapsulates concrete http.HandleFunc calls
// to abstract requests observability and error handling
func (a *App) Handle(method string, path string, handler Handler, mw ...Middleware) {

	// handler is the most inner handler to be executed
	handler = wrapMiddleware(mw, handler)

	handler = wrapMiddleware(a.mw, handler)

	h := func(w http.ResponseWriter, r *http.Request) {

		// Start or expand a distributed trace.
		ctx := r.Context()
		ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, r.URL.Path)
		defer span.End()

		// Injects the spanned traceID & timestamp into request context to be processed
		v := Values{
			TraceID: span.SpanContext().TraceID().String(),
			Now:     time.Now(),
		}
		ctx = context.WithValue(ctx, KeyValues, &v)

		// Starts the execution of the Middleware chain
		if err := handler(ctx, w, r); err != nil {
			a.SignalShutdown()
			return
		}
	}

	a.mux.Handle(method, path, h)
}

// SignalShutdown is for gracefully shutdown the application process hooking up OS signals
func (a *App) SignalShutdown() {
	a.shutdown <- syscall.SIGTERM
}

// ServeHTTP is the entry point for all http traffic and allows
// the opentelemetry mux to run first to handle tracing. The opentelemetry
// mux then calls the application mux to handle application traffic.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.otmux.ServeHTTP(w, r)
}
