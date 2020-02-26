package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200225153733, Down20200225153733)
}

func Up20200225153733(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE denominations Change token_type coin_type bigint;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE denominations Change symbol asset_symbol varchar(255);")
	if err != nil {
		return err
	}
	return nil
}

func Down20200225153733(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE denominations Change coin_type token_type varchar(255);")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE denominations Change asset_symbol symbol varchar(255);")
	if err != nil {
		return err
	}
	return nil
}
