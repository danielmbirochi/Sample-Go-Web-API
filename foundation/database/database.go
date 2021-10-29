// Package database provides support for database interaction.
package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Postgres driver
)

var (
	ErrNotFound              = errors.New("not found")
	ErrInvalidID             = errors.New("Invalid ID format")
	ErrAuthenticationFailure = errors.New("authentication failed")
	ErrForbidden             = errors.New("action forbidden")
)

// Config is the required props for connecting to database.
type Config struct {
	User       string
	Password   string
	Hostname   string
	Name       string
	DisableTLS bool
}

// Open function configures and opens a database connection.
func Open(cfg Config) (*sqlx.DB, error) {
	sslMode := "require"
	if cfg.DisableTLS {
		sslMode = "disable"
	}

	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Hostname,
		Path:     cfg.Name,
		RawQuery: q.Encode(),
	}
	fmt.Println("DB_URI", u.String())

	return sqlx.Open("postgres", u.String())
}

// StatusCheck returns nil if it can successfully talk to the database engine.
func StatusCheck(ctx context.Context, db *sqlx.DB) error {

	// As the db "Ping" method can return a false-positive, we`re running
	// this query to force a round trip to the database.
	const q = `SELECT true`
	var tmp bool
	return db.QueryRowContext(ctx, q).Scan(&tmp)
}

// NamedQuerySlice is a helper function for executing queries that return a
// collection of data to be unmarshaled into a slice.
func NamedQuerySlice(ctx context.Context, db *sqlx.DB, query string, data interface{}, dest interface{}) error {

	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Slice {
		return errors.New("must provide a pointer to a slice")
	}

	rows, err := db.NamedQueryContext(ctx, query, data)
	if err != nil {
		return err
	}

	slice := val.Elem()
	for rows.Next() {
		v := reflect.New(slice.Type().Elem())
		if err := rows.StructScan(v.Interface()); err != nil {
			return err
		}
		slice.Set(reflect.Append(slice, v.Elem()))
	}

	return nil
}

// NamedQueryStruct is a helper function for executing queries that return a
// single value to be unmarshalled into a struct type.
func NamedQueryStruct(ctx context.Context, db *sqlx.DB, query string, data interface{}, dest interface{}) error {

	rows, err := db.NamedQueryContext(ctx, query, data)
	if err != nil {
		return err
	}
	if !rows.Next() {
		return ErrNotFound
	}

	if err := rows.StructScan(dest); err != nil {
		return err
	}

	return nil
}

// Log provides a parsed print version of the query and parameters (sqlx does not provide it).
// PS: This function is traversing linearly to the query string so it is not efficient
// for the task at hand.
func Log(query string, args ...interface{}) string {
	for i, arg := range args {
		n := fmt.Sprintf("$%d", i+1)

		var a string
		switch v := arg.(type) {
		case string:
			a = fmt.Sprintf("%q", v)
		case []byte:
			a = string(v)
		case []string:
			a = strings.Join(v, ",")
		default:
			a = fmt.Sprintf("%v", v)
		}

		query = strings.Replace(query, n, a, 1)
	}

	return query
}
