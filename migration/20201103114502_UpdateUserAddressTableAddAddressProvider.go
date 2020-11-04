package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20201103114502, Down20201103114502)
}

func Up20201103114502(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE user_addresses ADD column address_provider VARCHAR(100) NOT NULL DEFAULT 'Bundle' AFTER `v2_address`")
	if err != nil {
		return err
	}
	return nil
}

func Down20201103114502(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE user_addresses DROP column address_provider")
	if err != nil {
		return err
	}
	return nil
}
