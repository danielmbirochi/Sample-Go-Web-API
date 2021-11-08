// This program performs administrative tasks for the sales app.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ardanlabs/conf"
	"github.com/danielmbirochi/go-sample-service/app/tooling/sales-admin/commands"
	"github.com/danielmbirochi/go-sample-service/foundation/database"
	"github.com/pkg/errors"
)

var build = "develop"

func main() {
	if err := run(); err != nil {
		if err != nil {
			log.Printf("error: %s", err)
		}
		os.Exit(1)
	}
}

func run() error {

	// =========================================================================
	// Configuration

	var cfg struct {
		conf.Version
		Args conf.Args
		DB   struct {
			User       string `conf:"default:testuser"`
			Password   string `conf:"default:mysecretpassword,mask"`
			Hostname   string `conf:"default:0.0.0.0"`
			Name       string `conf:"default:testdb"`
			DisableTLS bool   `conf:"default:false"`
		}
	}
	cfg.Version.SVN = build
	cfg.Version.Desc = "This is an Admin CLI Tooling for the Go Sample Service. It is used for performing administrative tasks."

	const prefix = "SALES"
	if err := conf.Parse(os.Args[1:], prefix, &cfg); err != nil {
		switch err {
		case conf.ErrHelpWanted:
			usage, err := conf.Usage(prefix, &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config usage")
			}
			fmt.Println(usage)
			fmt.Println("\n\n========================== SUPPORTED FLAGS ==========================")
			fmt.Println("\n-keygen: generate a set of private/public key files")
			fmt.Println("\n-tokengen: generate a JWT for a user with claims")
			fmt.Println("\n-migrate: create the schema in the database")
			fmt.Println("\n-seed: add data to the database")
			return nil
		case conf.ErrVersionWanted:
			version, err := conf.VersionString(prefix, &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config version")
			}
			fmt.Println(version)
			return nil
		}
		return errors.Wrap(err, "parsing config")
	}

	out, err := conf.String(&cfg)
	if err != nil {
		return errors.Wrap(err, "generating config for output")
	}
	log.Printf("main: Config :\n%v\n", out)

	// =========================================================================
	// Commands

	cfg.DB.DisableTLS = true

	dbConfig := database.Config{
		User:       cfg.DB.User,
		Password:   cfg.DB.Password,
		Hostname:   cfg.DB.Hostname,
		Name:       cfg.DB.Name,
		DisableTLS: cfg.DB.DisableTLS,
	}

	switch cfg.Args.Num(0) {
	case "keygen":
		if err := commands.KeyGen(); err != nil {
			return errors.Wrap(err, "key genereration")
		}

	case "tokengen":
		email := cfg.Args.Num(1)
		privateKeyFile := "/Users/miranda/Documents/estudos/go/go-sample-service/private.pem"
		algorithm := "RS256"
		if err := commands.TokenGen(email, privateKeyFile, algorithm); err != nil {
			return errors.Wrap(err, "generating token")
		}

	case "migrate":
		if err := commands.Migrate(dbConfig); err != nil {
			return errors.Wrap(err, "migrating database")
		}

	case "seed":
		if err := commands.Seed(dbConfig); err != nil {
			return errors.Wrap(err, "seeding database")
		}

	default:
		fmt.Println("\n\n========================== SUPPORTED FLAGS ==========================")
		fmt.Println("\n-keygen: generate a set of private/public key files")
		fmt.Println("\n-tokengen: generate a JWT for a user with claims")
		fmt.Println("\n-migrate: create the schema in the database")
		fmt.Println("\n-seed: add data to the database")
		return nil
	}

	return nil
}
