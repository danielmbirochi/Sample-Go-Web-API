// Package tests contains supporting code for running tests.
package tests

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/danielmbirochi/go-sample-service/business/auth"
	"github.com/danielmbirochi/go-sample-service/business/core/user"
	"github.com/danielmbirochi/go-sample-service/business/data/schema"
	"github.com/danielmbirochi/go-sample-service/foundation/database"
	"github.com/danielmbirochi/go-sample-service/foundation/docker"
	"github.com/danielmbirochi/go-sample-service/foundation/logger"
	"github.com/danielmbirochi/go-sample-service/foundation/web"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Success and failure markers.
const (
	Success = "\u2713"
	Failed  = "\u2717"
)

// Configs for running tests.
var (
	dbImage = "postgres:13.4-alpine"
	dbPort  = "5432"
	dbArgs  = []string{"-e", "POSTGRES_PASSWORD=mysecretpassword", "-e", "POSTGRES_USER=testuser", "-e", "POSTGRES_DB=testdb"}
	AdminID = "32bc1165-24t2-61a7-af3e-9da4agf2h1p1"
	UserID  = "14hg2372-66e5-34e9-jl8d-6ga1tuf7l3r4"
)

// NewUnit creates a test database. It sets the proper db migrations.
// It returns the logger, the database and a teardown function.
func NewUnit(t *testing.T) (*zap.SugaredLogger, *sqlx.DB, func()) {
	c := docker.StartContainer(t, dbImage, dbPort, dbArgs...)

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
		docker.DumpContainerLogs(t, c.ID)
		docker.StopContainer(t, c.ID)
		t.Fatalf("database never ready: %v", pingError)
	}

	if err := schema.Migrate(db); err != nil {
		docker.StopContainer(t, c.ID)
		t.Fatalf("migrating error: %s", err)
	}

	// teardown is the function that should be invoked when the caller is done
	// with the database.
	teardown := func() {
		t.Helper()
		db.Close()
		docker.StopContainer(t, c.ID)
	}

	log, err := logger.New("TEST")
	if err != nil {
		t.Fatalf("logger error: %s", err)
	}

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

// Test owns state for running and shutting down integration tests.
type Test struct {
	TraceID  string
	DB       *sqlx.DB
	Log      *zap.SugaredLogger
	Auth     *auth.Auth
	KID      string
	Teardown func()

	t *testing.T
}

// NewIntegration creates a database, seeds it, constructs an authenticator.
func NewIntegration(t *testing.T) *Test {
	log, db, teardown := NewUnit(t)

	if err := schema.Seed(db); err != nil {
		t.Fatal(err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	keyID := "4754d86b-7a6d-4df5-9c65-224741361492"
	lookup := func(publicKID string) (*rsa.PublicKey, error) {
		switch publicKID {
		case keyID:
			return &privateKey.PublicKey, nil
		}
		return nil, fmt.Errorf("no public key found for the specified kid: %s", publicKID)
	}

	auth, err := auth.New("RS256", lookup, auth.Keys{keyID: privateKey})
	if err != nil {
		t.Fatal(err)
	}

	test := Test{
		TraceID:  "00000000-0000-0000-0000-000000000001",
		DB:       db,
		Log:      log,
		Auth:     auth,
		KID:      keyID,
		t:        t,
		Teardown: teardown,
	}

	return &test
}

// Token generates an authenticated token.
func (test *Test) Token(email, pass string) string {
	test.t.Log("Generating token for test ...")

	u := user.New(test.Log, test.DB)
	claims, err := u.Authenticate(context.Background(), test.TraceID, time.Now(), email, pass)
	if err != nil {
		test.t.Fatal(err)
	}

	token, err := test.Auth.GenerateToken(test.KID, claims)
	if err != nil {
		test.t.Fatal(err)
	}

	return token
}
