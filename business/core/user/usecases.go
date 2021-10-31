// Pakcage user contains usecases for CRUD operations.
package user

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/danielmbirochi/go-sample-service/business/auth"
	"github.com/danielmbirochi/go-sample-service/foundation/database"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrNotFound is used when a specific User is requested but does not exist.
	ErrNotFound = errors.New("not found")

	// ErrInvalidID occurs when an ID is not in a valid form (UUID).
	ErrInvalidID = errors.New("ID is not in its proper form")

	// ErrAuthenticationFailure occurs when a user attempts to authenticate but
	// anything goes wrong.
	ErrAuthenticationFailure = errors.New("authentication failed")

	// ErrForbidden occurs when a user tries to do something that is forbidden to them according to our access control policies.
	ErrForbidden = errors.New("attempted action is not allowed")
)

type UserService struct {
	db  *sqlx.DB
	log *log.Logger
}

// New is a factory method for constructing user service.
func New(log *log.Logger, sqlxDB *sqlx.DB) UserService {
	return UserService{
		db:  sqlxDB,
		log: log,
	}
}

// Create inserts a new user into the database.
func (us UserService) Create(ctx context.Context, traceID string, nu NewUser, now time.Time) (User, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "business.core.user.Create")
	defer span.End()

	hash, err := bcrypt.GenerateFromPassword([]byte(nu.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, errors.Wrap(err, "generating password hash")
	}

	usr := User{
		ID:           uuid.New().String(),
		Name:         nu.Name,
		Email:        nu.Email,
		PasswordHash: hash,
		Roles:        nu.Roles,
		DateCreated:  now.UTC(),
		DateUpdated:  now.UTC(),
	}

	const q = `
	INSERT INTO users
		(user_id, name, email, password_hash, roles, date_created, date_updated)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	us.log.Printf("%s : %s : query : %s", traceID, "user.Create",
		database.Log(q, usr.ID, usr.Name, usr.Email, usr.PasswordHash, usr.Roles, usr.DateCreated, usr.DateUpdated),
	)

	if _, err = us.db.ExecContext(ctx, q, usr.ID, usr.Name, usr.Email, usr.PasswordHash, usr.Roles, usr.DateCreated, usr.DateUpdated); err != nil {
		return User{}, errors.Wrap(err, "inserting user")
	}

	return usr, nil
}

// Update replaces a user document in the database.
func (us UserService) Update(ctx context.Context, traceID string, claims auth.Claims, id string, uu UpdateUser, now time.Time) error {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "business.core.user.Update")
	defer span.End()

	usr, err := us.GetById(ctx, traceID, claims, id)
	if err != nil {
		return err
	}

	if uu.Name != nil {
		usr.Name = *uu.Name
	}
	if uu.Email != nil {
		usr.Email = *uu.Email
	}
	if uu.Roles != nil {
		usr.Roles = uu.Roles
	}
	if uu.Password != nil {
		pw, err := bcrypt.GenerateFromPassword([]byte(*uu.Password), bcrypt.DefaultCost)
		if err != nil {
			return errors.Wrap(err, "generating password hash")
		}
		usr.PasswordHash = pw
	}
	usr.DateUpdated = now

	const q = `
	UPDATE users 
		SET
			"name" = $2,
			"email" = $3,
			"roles" = $4,
			"password_hash" = $5,
			"date_updated" = $6
		WHERE user_id = $1
	`

	us.log.Printf("%s : %s : query : %s", traceID, "user.Update",
		database.Log(q, usr.ID, usr.Name, usr.Email, usr.Roles, usr.PasswordHash, usr.DateCreated, usr.DateUpdated),
	)

	if _, err = us.db.ExecContext(ctx, q, id, usr.Name, usr.Email, usr.Roles, usr.PasswordHash, usr.DateUpdated); err != nil {
		return errors.Wrap(err, "updating user")
	}

	return nil
}

// Delete removes a user from the database.
func (us UserService) Delete(ctx context.Context, traceID string, id string) error {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "business.core.user.Delete")
	defer span.End()

	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalidID
	}

	const q = `
	DELETE 
		FROM users 
			WHERE user_id = $1
	`

	us.log.Printf("%s : %s : query : %s", traceID, "user.Delete",
		database.Log(q, id),
	)

	if _, err := us.db.ExecContext(ctx, q, id); err != nil {
		return errors.Wrapf(err, "deleting user %s", id)
	}

	return nil
}

// List retrieves a list of existing users from the database.
// PS: List func needs to be updated for supporting data pagination.
func (us UserService) List(ctx context.Context, traceID string, pageNumber int, rowsPerPage int) ([]User, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "business.core.user.List")
	defer span.End()

	data := struct {
		Offset      int `db:"offset"`
		RowsPerPage int `db:"rows_per_page"`
	}{
		Offset:      (pageNumber - 1) * rowsPerPage,
		RowsPerPage: rowsPerPage,
	}

	const q = `
	SELECT * 
		FROM users
	ORDER BY user_id
	OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY
	`

	us.log.Printf("%s : %s : query : %s", traceID, "user.List",
		database.Log(q, data),
	)

	var users []User
	if err := database.NamedQuerySlice(ctx, us.db, q, data, &users); err != nil {
		if err == database.ErrNotFound {
			return nil, database.ErrNotFound
		}
		return nil, errors.Wrap(err, "selecting users")
	}

	return users, nil
}

// GetById gets the specified user from the database.
func (us UserService) GetById(ctx context.Context, traceID string, claims auth.Claims, userID string) (User, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "business.core.user.GetById")
	defer span.End()

	if _, err := uuid.Parse(userID); err != nil {
		return User{}, ErrInvalidID
	}

	// Only admins and the own user can access such record.
	if !claims.HasRole(auth.RoleAdmin) && claims.Subject != userID {
		return User{}, ErrForbidden
	}

	const q = `
	SELECT * 
		FROM users 
			WHERE user_id = $1
	`

	us.log.Printf("%s : %s : query : %s", traceID, "user.GetById",
		database.Log(q, userID),
	)

	var u User
	if err := us.db.GetContext(ctx, &u, q, userID); err != nil {
		if err == sql.ErrNoRows {
			return User{}, ErrNotFound
		}
		return User{}, errors.Wrapf(err, "selecting user %q", userID)
	}

	return u, nil
}

// GetByEmail gets the specified user from the database.
func (us UserService) GetByEmail(ctx context.Context, traceID string, claims auth.Claims, email string) (User, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "business.core.user.GetByEmail")
	defer span.End()

	const q = `
	SELECT * 
		FROM users 
			WHERE email = $1
	`

	us.log.Printf("%s : %s : query : %s", traceID, "user.GetByEmail",
		database.Log(q, email),
	)

	var usr User
	if err := us.db.GetContext(ctx, &usr, q, email); err != nil {
		if err == sql.ErrNoRows {
			return User{}, ErrNotFound
		}
		return User{}, errors.Wrapf(err, "selecting user %q", email)
	}

	// Only admins and the own user can access such record.
	if !claims.HasRole(auth.RoleAdmin) && claims.Subject != usr.ID {
		return User{}, ErrForbidden
	}

	return usr, nil
}

// Authenticate finds a user by their email and verifies their password. On
// success it returns a Claims value representing this user. The claims can be
// used to generate a token for future authentication.
func (us UserService) Authenticate(ctx context.Context, traceID string, now time.Time, email, password string) (auth.Claims, error) {
	ctx, span := otel.GetTracerProvider().Tracer("").Start(ctx, "business.core.user.Authenticate")
	defer span.End()

	const q = `
	SELECT * 
		FROM users 
			WHERE email = $1
	`

	us.log.Printf("%s: %s: %s", traceID, "user.Authenticate",
		database.Log(q, email),
	)

	var u User
	if err := us.db.GetContext(ctx, &u, q, email); err != nil {

		// Normally we would return ErrNotFound in this scenario but we do not want
		// to leak to an unauthenticated user which emails are in the system.
		if err == sql.ErrNoRows {
			return auth.Claims{}, ErrAuthenticationFailure
		}

		return auth.Claims{}, errors.Wrap(err, "selecting single user")
	}

	// Compare the provided password with the saved hash. Use the bcrypt
	// comparison function so it is cryptographically secure.
	if err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)); err != nil {
		return auth.Claims{}, ErrAuthenticationFailure
	}

	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "go-sample-service",
			Subject:   u.ID,
			Audience:  []string{"students"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Roles: u.Roles,
	}

	return claims, nil
}
