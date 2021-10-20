package middleware

import (
	"context"
	"log"
	"net/http"

	"github.com/danielmbirochi/go-sample-service/foundation/web"
)

// Errors handles errors coming out of the call chain (propagated by the innerHandler). It detects normal
// application errors which are used to respond to the client in a uniform way (trusted errors).
// Unexpected errors (status >= 500) are logged (untrusted errors).
func Errors(log *log.Logger) web.Middleware {

	m := func(innerHandler web.Handler) web.Handler {

		// Creates the Error handler that will be attached in the middleware onion chain (outer handler of the onion)
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

			// If the context is missing this value (integrity error), request the service
			// to be shutdown gracefully.
			v, ok := ctx.Value(web.KeyValues).(*web.Values)
			if !ok {
				return web.NewShutdownError("web value missing from context")
			}

			// Run the handler chain and catch any propagated error.
			if err := innerHandler(ctx, w, r); err != nil {

				log.Printf("%s : ERROR     : %v", v.TraceID, err)

				// Send the error back to the client. If this call throws any error, it`s going to
				// return the untrusted error up to the chain (i.e. network errors)
				if err := web.RespondError(ctx, w, err); err != nil {
					return err
				}

				// If the innerHandler returns a shutdown error, return it
				// up to the chain for the application be shutdown gracefully.
				if ok := web.IsShutdown(err); ok {
					return err
				}
			}

			// The error has been handled so we can stop propagating it.
			return nil
		}

		return h
	}

	return m
}
