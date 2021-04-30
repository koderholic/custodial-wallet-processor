package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210331084110, Down20210331084110)
}

func Up20210331084110(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE denominations ADD COLUMN `default_network` VARCHAR(100) AFTER `is_multi_addresses`;")
	if err != nil {
		return err
	}
	return nil
}

func Down20210331084110(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE denominations DROP COLUMN default_network;")
	if err != nil {
		return err
	}
	return nil
}
