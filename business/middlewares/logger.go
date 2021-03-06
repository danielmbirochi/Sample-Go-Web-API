package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/danielmbirochi/go-sample-service/foundation/web"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

// Logger writes some information about the request to the logs in the
func Logger(log *zap.SugaredLogger) web.Middleware {

	// This is the actual middleware function to be executed.
	m := func(innerHandler web.Handler) web.Handler {

		// Creates the request Logger handler that will be attached in the middleware chain (outer handler of the onion)
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "business.middlewares.Logger")
			defer span.End()

			// If the context is missing this value (integrity error), request the service
			// to be shutdown gracefully.
			v, ok := ctx.Value(web.KeyValues).(*web.Values)
			if !ok {
				return web.NewShutdownError("web value missing from context")
			}

			log.Infof("%s : started   : %s %s -> %s",
				v.TraceID,
				r.Method, r.URL.Path,
				r.RemoteAddr,
			)

			err := innerHandler(ctx, w, r)

			// format: TraceID : (200) GET /foo -> IP ADDR (latency)
			log.Infof("%s : completed : (%d) : %s %s -> %s (%s)",
				v.TraceID, v.StatusCode,
				r.Method, r.URL.Path,
				r.RemoteAddr, time.Since(v.Now),
			)

			// Return the error so it can be handled further up the chain.
			return err
		}

		return h
	}

	return m
}
