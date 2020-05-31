package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200529141352, Down20200529141352)
}

func Up20200529141352(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE user_addresses ADD address_type varchar(50) NULL after `address`;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE user_addresses SET address_type='Segwit' WHERE address_type IS NULL AND address LIKE 'bc1%';")
	if err != nil {
		return err
	}
	return nil
}

func Down20200529141352(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE user_addresses DROP COLUMN address_type;")
	if err != nil {
		return err
	}
	return nil
}
