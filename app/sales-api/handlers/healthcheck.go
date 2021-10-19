package handlers

import (
	"context"
	"net/http"

	"github.com/danielmbirochi/go-sample-service/foundation/web"
)

type check struct {
	build string
}

func (c check) readiness(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

	statusCode := http.StatusOK

	health := struct {
		Version string `json:"version"`
		Status  string `json:"status"`
	}{
		Version: c.build,
		Status:  "Application is running ...",
	}

	return web.Respond(ctx, w, health, statusCode)
}
