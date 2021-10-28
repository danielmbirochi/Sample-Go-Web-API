// Package tests contains supporting code for running tests.
package tests

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/danielmbirochi/go-sample-service/business/repository/schema"
	"github.com/danielmbirochi/go-sample-service/foundation/database"
	"github.com/danielmbirochi/go-sample-service/foundation/web"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Success and failure markers.
const (
	success = "\u2713"
	failed  = "\u2717"
)

// Configs for running tests.
var (
	dbImage = "postgres:13.4-alpine"
	dbPort  = "5432"
	dbArgs  = []string{"-U", "testuser", "-e", "POSTGRES_PASSWORD=mysecretpassword", "testdb"}
	AdminID = "32bc1165-24t2-61a7-af3e-9da4agf2h1p1"
	UserID  = "14hg2372-66e5-34e9-jl8d-6ga1tuf7l3r4"
)

// NewUnit creates a test database. It sets the proper db migrations.
// It returns the logger, the database and a teardown function.
func NewUnit(t *testing.T) (*log.Logger, *sqlx.DB, func()) {
	c := startContainer(t, dbImage, dbPort, dbArgs...)

	cfg := database.Config{
		User:       "testuser",
		Password:   "mysecretpassword",
		Hostname:   c.Host,
		Name:       "testdb",
		DisableTLS: true,
	}
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatalf("opening database connection: %v", err)
	}

	t.Log("waiting for database to be ready ...")

	// Wait for the database to be ready. Wait 100ms longer between each attempt.
	// Do not try more than 20 times.
	var pingError error
	maxAttempts := 20
	for attempts := 1; attempts <= maxAttempts; attempts++ {
		pingError = db.Ping()
		if pingError == nil {
			break
		}
		time.Sleep(time.Duration(attempts) * 100 * time.Millisecond)
	}

	if pingError != nil {
		dumpContainerLogs(t, c.ID)
		stopContainer(t, c.ID)
		t.Fatalf("database never ready: %v", pingError)
	}

	if err := schema.Migrate(db); err != nil {
		stopContainer(t, c.ID)
		t.Fatalf("migrating error: %s", err)
	}

	// teardown is the function that should be invoked when the caller is done
	// with the database.
	teardown := func() {
		t.Helper()
		db.Close()
		stopContainer(t, c.ID)
	}

	log := log.New(os.Stdout, "TEST : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	return log, db, teardown
}

// Context returns an app level context for testing.
func Context() context.Context {
	values := web.Values{
		TraceID: uuid.New().String(),
		Now:     time.Now(),
	}

	return context.WithValue(context.Background(), web.KeyValues, &values)
}

// StringPointer is a helper to get a *string from a string for helping on running tests.
func StringPointer(s string) *string {
	return &s
}

// IntPointer is a helper to get a *int from a int for helping on running tests.
func IntPointer(i int) *int {
	return &i
}
