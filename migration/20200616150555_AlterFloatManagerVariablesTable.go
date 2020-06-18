package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200616150555, Down20200616150555)
}

func Up20200616150555(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE float_manager_variables ADD surplus decimal(64,18) after `deficit`;")
	if err != nil {
		return err
	}
	return nil
}

func Down20200616150555(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE float_manager_variables DROP COLUMN surplus;")
	if err != nil {
		return err
	}
	return nil
}
