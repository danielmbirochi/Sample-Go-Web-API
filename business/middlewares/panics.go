package middleware

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/danielmbirochi/go-sample-service/foundation/web"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

// Panics recovers from panics propagated by the innerHandler and converts the panic
// to an error so it is handled in Errors middleware (outer middleware).
func Panics(log *zap.SugaredLogger) web.Middleware {

	m := func(innerHandler web.Handler) web.Handler {

		// Create the Panic handler that will be attached in the middleware onion chain.
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
			ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "business.middlewares.Panics")
			defer span.End()

			v, ok := ctx.Value(web.KeyValues).(*web.Values)
			if !ok {
				return web.NewShutdownError("web value missing from context")
			}

			// Defer a function to recover from a panic and set the err return
			// variable after the fact.
			defer func() {
				if r := recover(); r != nil {
					err = errors.Errorf("panic: %v", r)

					// Log the Go stack trace for this panic'd goroutine.
					log.Infof("%s :\n%s", v.TraceID, debug.Stack())
				}
			}()

			// Call the next Handler and set its return value in the err variable of this context.
			return innerHandler(ctx, w, r)
		}

		return h
	}

	return m
}
