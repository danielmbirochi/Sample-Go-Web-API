package user_test

import (
	"testing"
	"time"

	"github.com/danielmbirochi/go-sample-service/business/auth"
	"github.com/danielmbirochi/go-sample-service/business/core/user"
	"github.com/danielmbirochi/go-sample-service/business/repository/schema"
	"github.com/danielmbirochi/go-sample-service/business/tests"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
)

func TestUser(t *testing.T) {
	log, db, teardown := tests.NewUnit(t)
	t.Cleanup(teardown)

	u := user.New(log, db)

	t.Log("Given the need to work with User records.")
	{
		testID := 0
		t.Logf("\tTest %d:\tWhen handling a single User.", testID)
		{
			ctx := tests.Context()
			now := time.Date(2021, time.October, 28, 0, 0, 0, 0, time.UTC)
			traceID := "00000000-0000-0000-0000-000000000001"

			nu := user.NewUser{
				Name:            "Daniel M",
				Email:           "dmbirochi@gmail.com.com",
				Roles:           []string{auth.RoleAdmin},
				Password:        "teste123",
				PasswordConfirm: "teste123",
			}

			if err := schema.DeleteAll(db); err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to delete all data : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to delete all data.", tests.Success, testID)

			usr, err := u.Create(ctx, traceID, nu, now)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to create user : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to create user.", tests.Success, testID)

			claims := auth.Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "go-sample-service project",
					Subject:   usr.ID,
					Audience:  []string{"testers"},
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(now),
				},
				Roles: []string{auth.RoleOperator},
			}

			savedU, err := u.GetById(ctx, traceID, claims, usr.ID)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to retrieve user by ID: %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to retrieve user by ID.", tests.Success, testID)

			if diff := cmp.Diff(usr, savedU); diff != "" {
				t.Fatalf("\t%s\tTest %d:\tShould get back the same user. Diff:\n%s", tests.Failed, testID, diff)
			}
			t.Logf("\t%s\tTest %d:\tShould get back the same user.", tests.Success, testID)

			upd := user.UpdateUser{
				Name:  tests.StringPointer("Daniel Miranda"),
				Email: tests.StringPointer("dani.mbirochi@gmail.com"),
			}

			claims = auth.Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "go-sample-service project",
					Audience:  []string{"testers"},
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(now),
				},
				Roles: []string{auth.RoleAdmin},
			}

			if err := u.Update(ctx, traceID, claims, usr.ID, upd, now); err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to update user : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to update user.", tests.Success, testID)

			savedU, err = u.GetByEmail(ctx, traceID, claims, *upd.Email)
			if err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to retrieve user : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to retrieve user.", tests.Success, testID)

			if savedU.Name != *upd.Name {
				t.Errorf("\t%s\tTest %d:\tShould be able to see updates to Name.", tests.Failed, testID)
				t.Logf("\t\tTest %d:\tGot: %v", testID, savedU.Name)
				t.Logf("\t\tTest %d:\tExp: %v", testID, *upd.Name)
			} else {
				t.Logf("\t%s\tTest %d:\tShould be able to see updates to Name.", tests.Success, testID)
			}

			if savedU.Email != *upd.Email {
				t.Errorf("\t%s\tTest %d:\tShould be able to see updates to Email.", tests.Failed, testID)
				t.Logf("\t\tTest %d:\tGot: %v", testID, savedU.Email)
				t.Logf("\t\tTest %d:\tExp: %v", testID, *upd.Email)
			} else {
				t.Logf("\t%s\tTest %d:\tShould be able to see updates to Email.", tests.Success, testID)
			}

			if err := u.Delete(ctx, traceID, usr.ID); err != nil {
				t.Fatalf("\t%s\tTest %d:\tShould be able to delete user : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould be able to delete user.", tests.Success, testID)

			_, err = u.GetById(ctx, traceID, claims, usr.ID)
			if errors.Cause(err) != user.ErrNotFound {
				t.Fatalf("\t%s\tTest %d:\tShould NOT be able to retrieve user : %s.", tests.Failed, testID, err)
			}
			t.Logf("\t%s\tTest %d:\tShould NOT be able to retrieve user.", tests.Success, testID)
		}
	}
}
