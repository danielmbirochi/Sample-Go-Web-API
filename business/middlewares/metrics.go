package middleware

import (
	"context"
	"expvar"
	"net/http"
	"runtime"

	"github.com/danielmbirochi/go-sample-service/foundation/web"
)

// m contains the global program of metrics for the app.
var m = struct {
	gr  *expvar.Int
	req *expvar.Int
	err *expvar.Int
}{
	gr:  expvar.NewInt("goroutines"),
	req: expvar.NewInt("requests"),
	err: expvar.NewInt("errors"),
}

// Metrics updates the program counters metrics.
func Metrics() web.Middleware {

	m := func(innerHandler web.Handler) web.Handler {

		// Create the handler that will be attached in the middleware onion chain.
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

			err := innerHandler(ctx, w, r)

			m.req.Add(1)

			// Update the counter for the number of active goroutines every 100 requests.
			if m.req.Value()%100 == 0 {
				m.gr.Set(int64(runtime.NumGoroutine()))
			}

			if err != nil {
				m.err.Add(1)
			}

			// Return the error so it can be handled further up the chain.
			return err
		}

		return h
	}

	return m
}
