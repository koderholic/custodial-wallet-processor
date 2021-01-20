package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210119170342, Down20210119170342)
}

func Up20210119170342(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE user_addresses ADD column is_primary_address tinyint(1) DEFAULT 0 NOT NULL AFTER address_provider;")
	if err != nil {
		return err
	}
	return nil
	_, err2 := tx.Exec("UPDATE user_addresses SET is_primary_address=true;")
	if err2 != nil {
		return err2
	}
	return nil
}

func Down20210119170342(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE user_addresses DROP COLUMN is_primary_address;")
	if err != nil {
		return err
	}
	return nil
}
