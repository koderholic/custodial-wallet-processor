package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200507002410, Down20200507002410)
}

func Up20200507002410(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE user_addresses ADD memo varchar(15);")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE user_addresses ADD v2_address varchar(255) AFTER `address`;")
	if err != nil {
		return err
	}
	return nil
}

func Down20200507002410(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE user_addresses DROP COLUMN memo;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE user_addresses DROP COLUMN v2_address;")
	if err != nil {
		return err
	}
	return nil
}
