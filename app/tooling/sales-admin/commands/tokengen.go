package commands

import (
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/danielmbirochi/go-sample-service/business/auth"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

// TokenGen generates a JWT for the specified user.
func TokenGen(id string, privateKeyFile string, algorithm string) error {
	if id == "" || privateKeyFile == "" || algorithm == "" {
		fmt.Println("help: gentoken <id> <private_key_file> <algorithm>")
		fmt.Println("algorithm: RS256, HS256")
		return nil
	}

	privatePEM, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return errors.Wrap(err, "reading PEM private key file")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privatePEM)
	if err != nil {
		return errors.Wrap(err, "parsing PEM into private key")
	}

	// In a production system, a key id (KID) would be assigned to
	// a key pair that will be used for generating JWT with a set of Claims. A
	// keyLookupFunc is provided to perform the task of retrieving a key pair for
	// the given KID.
	//
	// Here we`re using an arbitrary KID and the hardcoded key lookup function to fetch the right
	// key pair given a KID.
	keyID := "32bc1165-24t2-61a7-af3e-9da4agf2h1p1"
	keyLookupFunc := func(publicKID string) (*rsa.PublicKey, error) {
		switch publicKID {
		case keyID:
			return &privateKey.PublicKey, nil
		}
		return nil, fmt.Errorf("no public key found for the specified kid: %s", publicKID)
	}

	a, err := auth.New(algorithm, keyLookupFunc, auth.Keys{keyID: privateKey})
	if err != nil {
		return errors.Wrap(err, "constructing auth")
	}

	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "ADMIN",
			Subject:   "TestToken",
			Audience:  []string{"some_audience"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(8760 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Roles: []string{auth.RoleAdmin},
	}

	token, err := a.GenerateToken(keyID, claims)
	if err != nil {
		return errors.Wrap(err, "generating token")
	}

	fmt.Printf("-----BEGIN TOKEN-----\n%s\n-----END TOKEN-----\n", token)
	return nil
}
