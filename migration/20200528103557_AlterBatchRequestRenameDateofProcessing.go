package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200528103557, Down20200528103557)
}

func Up20200528103557(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE batch_requests Change date_ofprocessing date_of_processing DATETIME NULL;")
	if err != nil {
		return err
	}
	return nil
}

func Down20200528103557(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE batch_requests Change date_of_processing date_ofprocessing DATETIME NULL;")
	if err != nil {
		return err
	}
	return nil
}
