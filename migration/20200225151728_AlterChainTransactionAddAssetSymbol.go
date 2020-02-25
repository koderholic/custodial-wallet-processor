package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200225151728, Down20200225151728)
}

func Up20200225151728(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE chain_transactions ADD asset_symbol varchar(255);")
	if err != nil {
		return err
	}
	return nil
}

func Down20200225151728(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE chain_transactions DROP COLUMN asset_symbol;")
	if err != nil {
		return err
	}
	return nil
}
