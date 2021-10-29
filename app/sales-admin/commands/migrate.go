package commands

import (
	"fmt"

	"github.com/danielmbirochi/go-sample-service/business/data/schema"
	"github.com/danielmbirochi/go-sample-service/foundation/database"
	"github.com/pkg/errors"
)

// Migrate creates the schema in the database.
func Migrate(cfg database.Config) error {
	db, err := database.Open(cfg)
	if err != nil {
		return errors.Wrap(err, "connect database")
	}
	defer db.Close()

	if err := schema.Migrate(db); err != nil {
		return errors.Wrap(err, "migrate database")
	}

	fmt.Println("\nmigrations complete")
	return nil
}
