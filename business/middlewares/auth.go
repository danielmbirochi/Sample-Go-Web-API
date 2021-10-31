package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/danielmbirochi/go-sample-service/business/auth"
	"github.com/danielmbirochi/go-sample-service/foundation/web"
	"go.opentelemetry.io/otel"
)

// ErrForbidden is returned when an authenticated user does not have
// a sufficient role for an action.
var ErrForbidden = web.NewRequestError(
	errors.New("not authorized"),
	http.StatusForbidden,
)

// Authenticate middleware validates a JWT from the `Authorization` http header.
func Authenticate(a *auth.Auth) web.Middleware {

	// Middleware func
	m := func(innerHandler web.Handler) web.Handler {

		// Handler func
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "business.middlewares.Authenticate")
			defer span.End()

			// Expecting header: Authorization: bearer <token>
			authHeader := r.Header.Get("authorization")

			// Parse the authorization header.
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				err := errors.New("expected authorization header format: bearer <token>")
				return web.NewRequestError(err, http.StatusUnauthorized)
			}

			// Validate the token
			token := parts[1]
			claims, err := a.ValidateToken(token)
			if err != nil {
				return web.NewRequestError(err, http.StatusUnauthorized)
			}

			// Add claims to the context..
			ctx = context.WithValue(ctx, auth.Key, claims)

			return innerHandler(ctx, w, r)
		}

		return h
	}

	return m
}

// Authorize middleware validates that an authenticated user has at least one role
// from a specified list of roles.
func Authorize(roles ...string) web.Middleware {

	// Middleware func
	m := func(innerHandler web.Handler) web.Handler {

		// Handler func
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "business.middlewares.Authorize")
			defer span.End()

			// If the context is missing this value (integrity error) return failure.
			claims, ok := ctx.Value(auth.Key).(auth.Claims)
			if !ok {
				return errors.New("auth claims missing from context")
			}

			if !claims.HasRole(roles...) {
				return ErrForbidden
			}

			return innerHandler(ctx, w, r)
		}

		return h
	}

	return m
}
