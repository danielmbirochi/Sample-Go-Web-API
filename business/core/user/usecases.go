// Pakcage user contains usecases for CRUD operations.
package user

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/danielmbirochi/go-sample-service/business/auth"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
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
	store *sqlx.DB
}

// New is a factory method for constructing user service.
func New(log *log.Logger, sqlxDB *sqlx.DB) UserService {
	return UserService{
		store: sqlxDB,
	}
}

// Create inserts a new user into the database.
func (us UserService) Create(ctx context.Context, nu NewUser, now time.Time) (User, error) {

	hash, err := bcrypt.GenerateFromPassword([]byte(nu.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, errors.Wrap(err, "generating password hash")
	}

	u := User{
		ID:           uuid.New().String(),
		Name:         nu.Name,
		Email:        nu.Email,
		PasswordHash: hash,
		Roles:        nu.Roles,
		DateCreated:  now.UTC(),
		DateUpdated:  now.UTC(),
	}

	const q = `INSERT INTO users
		(user_id, name, email, password_hash, roles, date_created, date_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if _, err = us.store.ExecContext(ctx, q, u.ID, u.Name, u.Email, u.PasswordHash, u.Roles, u.DateCreated, u.DateUpdated); err != nil {
		return User{}, errors.Wrap(err, "inserting user")
	}

	return u, nil
}

// Update replaces a user document in the database.
func (us UserService) Update(ctx context.Context, claims auth.Claims, id string, uu UpdateUser, now time.Time) error {
	u, err := us.GetById(ctx, claims, id)
	if err != nil {
		return err
	}

	if uu.Name != nil {
		u.Name = *uu.Name
	}
	if uu.Email != nil {
		u.Email = *uu.Email
	}
	if uu.Roles != nil {
		u.Roles = uu.Roles
	}
	if uu.Password != nil {
		pw, err := bcrypt.GenerateFromPassword([]byte(*uu.Password), bcrypt.DefaultCost)
		if err != nil {
			return errors.Wrap(err, "generating password hash")
		}
		u.PasswordHash = pw
	}
	u.DateUpdated = now

	const q = `UPDATE users SET
		"name" = $2,
		"email" = $3,
		"roles" = $4,
		"password_hash" = $5,
		"date_updated" = $6
		WHERE user_id = $1`

	if _, err = us.store.ExecContext(ctx, q, id, u.Name, u.Email, u.Roles, u.PasswordHash, u.DateUpdated); err != nil {
		return errors.Wrap(err, "updating user")
	}

	return nil
}

// Delete removes a user from the database.
func (us UserService) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalidID
	}

	const q = `DELETE FROM users WHERE user_id = $1`

	if _, err := us.store.ExecContext(ctx, q, id); err != nil {
		return errors.Wrapf(err, "deleting user %s", id)
	}

	return nil
}

// List retrieves a list of existing users from the database.
// PS: List func needs to be updated for supporting data pagination.
func (us UserService) List(ctx context.Context) ([]User, error) {
	const q = `SELECT * FROM users`

	users := []User{}
	if err := us.store.SelectContext(ctx, &users, q); err != nil {
		return nil, errors.Wrap(err, "selecting users")
	}

	return users, nil
}

// GetById gets the specified user from the database.
func (us UserService) GetById(ctx context.Context, claims auth.Claims, userID string) (User, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return User{}, ErrInvalidID
	}

	// If you are not an admin and looking to retrieve someone other than yourself.
	if !claims.HasRole(auth.RoleAdmin) && claims.Subject != userID {
		return User{}, ErrForbidden
	}

	const q = `SELECT * FROM users WHERE user_id = $1`

	var u User
	if err := us.store.GetContext(ctx, &u, q, userID); err != nil {
		if err == sql.ErrNoRows {
			return User{}, ErrNotFound
		}
		return User{}, errors.Wrapf(err, "selecting user %q", userID)
	}

	return u, nil
}

// Authenticate finds a user by their email and verifies their password. On
// success it returns a Claims value representing this user. The claims can be
// used to generate a token for future authentication.
func (us UserService) Authenticate(ctx context.Context, now time.Time, email, password string) (auth.Claims, error) {
	const q = `SELECT * FROM users WHERE email = $1`

	var u User
	if err := us.store.GetContext(ctx, &u, q, email); err != nil {

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
			Issuer:    "service project",
			Subject:   u.ID,
			Audience:  []string{"students"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Roles: u.Roles,
	}

	return claims, nil
}
