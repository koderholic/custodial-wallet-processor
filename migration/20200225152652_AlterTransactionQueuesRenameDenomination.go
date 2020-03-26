package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200225152652, Down20200225152652)
}

func Up20200225152652(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE transaction_queues Change denomination asset_symbol varchar(255);")
	if err != nil {
		return err
	}
	return nil
}

func Down20200225152652(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE transaction_queues Change asset_symbol denomination varchar(36);")
	if err != nil {
		return err
	}
	return nil
}
