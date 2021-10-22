// This program performs administrative tasks for the sales app.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ardanlabs/conf"
	"github.com/danielmbirochi/go-sample-service/app/sales-admin/commands"
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

	switch cfg.Args.Num(0) {
	case "keygen":
		if err := commands.KeyGen(); err != nil {
			return errors.Wrap(err, "key genereration")
		}

	default:
		fmt.Println("\n\n========================== SUPPORTED FLAGS ==========================")
		fmt.Println("\n-keygen: generate a set of private/public key files")

		return nil
	}

	return nil
}
