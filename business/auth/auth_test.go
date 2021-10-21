package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/danielmbirochi/go-sample-service/business/auth"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

const (
	success = "\u2713"
	failed  = "\u2717"
)

func TestAuthenticator(t *testing.T) {
	t.Log("Given the need to be able to authenticate and authorize access.")
	{
		testID := 0
		t.Logf("\tTest %d:\tWhen handling a single user.", testID)
		{
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to parse the private key from pem: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to parse the private key from pem.", success, testID)

			// Sample kid
			const keyID = "54bb2165-71e1-41a6-af3e-7da4a0e1e2c1"
			keyLookupFunc := func(publicKID string) (*rsa.PublicKey, error) {
				if publicKID != keyID {
					return nil, errors.New("no public key found")
				}
				return &privateKey.PublicKey, nil
			}

			a, err := auth.New("RS256", keyLookupFunc, auth.Keys{keyID: privateKey})
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to create an authenticator: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to create an authenticator.", success, testID)

			claims := auth.Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "test issuer",
					Subject:   "0x01",
					Audience:  []string{"some_audience"},
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(8760 * time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
				Roles: []string{auth.RoleAdmin},
			}

			token, err := a.GenerateToken(keyID, claims)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to generate a JWT: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to generate a JWT.", success, testID)

			parsedClaims, err := a.ValidateToken(token)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to parse the claims: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to parse the claims.", success, testID)

			if exp, got := len(claims.Roles), len(parsedClaims.Roles); exp != got {
				t.Logf("\t\tTest %d:\texp: %d", testID, exp)
				t.Logf("\t\tTest %d:\tgot: %d", testID, got)
				t.Fatalf("\t%s\tTest %d:\tShould have the expexted number of roles: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould have the expexted number of roles.", success, testID)

			if exp, got := claims.Roles[0], parsedClaims.Roles[0]; exp != got {
				t.Logf("\t\tTest %d:\texp: %v", testID, exp)
				t.Logf("\t\tTest %d:\tgot: %v", testID, got)
				t.Fatalf("\t%s\tTest %d:\tShould have the expexted roles: %v", failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould have the expexted roles.", success, testID)
		}
	}
}
