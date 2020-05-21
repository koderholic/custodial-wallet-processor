package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200521164546, Down20200521164546)
}

func Up20200521164546(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE batch_requests Change denomination_id asset_symbol varchar(100);")
	if err != nil {
		 return err
	}
	return nil
}

func Down20200521164546(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE batch_requests Change asset_symbol denomination_id varchar(36);")
	if err != nil {
		 return err
	}
	return nil
}
