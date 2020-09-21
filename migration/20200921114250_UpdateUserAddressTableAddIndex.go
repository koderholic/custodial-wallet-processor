package migration

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200921114250, Down20200921114250)
}

func Up20200921114250(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE user_addresses ADD INDEX v2_address_memo (v2_address, memo);")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE user_addresses ADD INDEX address (address);")
	if err != nil {
		return err
	}
	return nil
}

func Down20200921114250(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE user_addresses DROP INDEX v2_address_memo;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE user_addresses DROP INDEX address;")
	if err != nil {
		return err
	}
	// This code is executed when the migration is rolled back.
	return nil
}
