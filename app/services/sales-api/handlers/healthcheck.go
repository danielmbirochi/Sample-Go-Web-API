package handlers

import (
	"context"
	"net/http"

	"github.com/danielmbirochi/go-sample-service/foundation/database"
	"github.com/danielmbirochi/go-sample-service/foundation/web"
	"github.com/jmoiron/sqlx"
)

type check struct {
	build string
	db    *sqlx.DB
}

func (c check) readiness(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

	status := "Application is running ..."
	statusCode := http.StatusOK
	if err := database.StatusCheck(ctx, c.db); err != nil {
		status = "db not ready"
		statusCode = http.StatusInternalServerError
	}

	health := struct {
		Version string `json:"version"`
		Status  string `json:"status"`
	}{
		Version: c.build,
		Status:  status,
	}

	return web.Respond(ctx, w, health, statusCode)
}
