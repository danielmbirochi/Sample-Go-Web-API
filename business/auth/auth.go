// Package auth provides authentication and authorization support.
package auth

import (
	"crypto/rsa"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

// These are the expected values for Claims.Roles.
const (
	RoleAdmin    = "ADMIN"
	RoleMaster   = "MASTER"
	RoleOperator = "OPERATOR"
)

// ctxKey represents the type of value for the context key.
type ctxKey int

// Key is used to store/retrieve a Claims value from a context.Context.
const Key ctxKey = 1

// Claims represents the authorization claims transmitted via a JWT.
type Claims struct {
	jwt.RegisteredClaims
	Roles []string `json:"roles"`
}

// Valid is called for validating parsed tokens.
// It wraps original jwt.RegisteredClaims.Valid method and provides
// custom wrapped error.
func (c Claims) Valid() error {
	if err := c.RegisteredClaims.Valid(); err != nil {
		return errors.Wrap(err, "validating standard claims")
	}

	return nil
}

// HasRole returns true if the claims has at least one of the provided roles.
func (c Claims) HasRole(roles ...string) bool {
	for _, has := range c.Roles {
		for _, want := range roles {
			if has == want {
				return true
			}
		}
	}
	return false
}

// Keys represents an in memory store of keys.
type Keys map[string]*rsa.PrivateKey

// KeyLookupFunc defines the signature of a function to lookup public keys.
//
// In a production system, a key id (KID) is used to retrieve the correct
// public key to parse a JWT for auth and claims. A key lookup function is
// provided for retrieving a public key for a given KID.
//
// A key lookup function is required for creating an Authenticator (type Auth).
//
// * Private keys should be rotated. During the transition period, tokens
// signed with the old and new keys can coexist by looking up the correct
// public key by KID.
//
// * KID to public key resolution is usually accomplished via a public JWKS
// endpoint. See https://auth0.com/docs/jwks for more details.
type PublicKeyLookup func(publicKID string) (*rsa.PublicKey, error)

// Auth is used to authenticate clients. It can generate a token for a
// set of user claims and recreate the claims by parsing the token.
type Auth struct {
	algorithm     string
	lookupKeyFunc func(t *jwt.Token) (interface{}, error)
	parser        *jwt.Parser
	keys          Keys
}

// New creates an *Authenticator.
func New(algorithm string, lookupFunc PublicKeyLookup, keys Keys) (*Auth, error) {

	lookupKey := func(t *jwt.Token) (interface{}, error) {
		kid, ok := t.Header["kid"]
		if !ok {
			return nil, errors.New("token header is missing key id (kid")
		}
		kidID, ok := kid.(string)
		if !ok {
			return nil, errors.New("kid must be string")
		}

		return lookupFunc(kidID)
	}

	// Create the token parser. The Algorithm used to sign the JWT must be validated
	// to avoid critical vulnerabilities:
	// For more information see: https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/
	parser := jwt.Parser{
		ValidMethods: []string{algorithm},
	}

	a := Auth{
		algorithm:     algorithm,
		lookupKeyFunc: lookupKey,
		parser:        &parser,
		keys:          keys,
	}

	return &a, nil
}

// AddKey adds a private key and kid to the local store.
func (a *Auth) AddKey(privateKey *rsa.PrivateKey, kid string) {
	a.keys[kid] = privateKey
}

// RemoveKey removes from local storage a key based on the provided kid.
func (a *Auth) RemoveKey(kid string) {
	delete(a.keys, kid)
}

// GenerateToken generates a JWT using the provided claims based on a given KID
func (a *Auth) GenerateToken(kid string, claims Claims) (string, error) {
	method := jwt.GetSigningMethod(a.algorithm)

	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = kid

	privateKey, ok := a.keys[kid]
	if !ok {
		return "", errors.New("kid lookup failed")
	}

	tokenStr, err := token.SignedString(privateKey)
	if err != nil {
		return "", errors.Wrap(err, "signing token")
	}

	return tokenStr, nil
}

// ValidateToken recreates the Claims used to generate a token.
// It verifies that the token was signed a valid key.
func (a *Auth) ValidateToken(tokenStr string) (Claims, error) {
	var claims Claims
	token, err := a.parser.ParseWithClaims(tokenStr, &claims, a.lookupKeyFunc)
	if err != nil {
		return Claims{}, errors.Wrap(err, "parsing token")
	}

	if !token.Valid {
		return Claims{}, errors.New("invalid token")
	}

	return claims, nil
}
