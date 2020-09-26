package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200925134104, Down20200925134104)
}

func Up20200925134104(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return nil
}

func Down20200925134104(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
