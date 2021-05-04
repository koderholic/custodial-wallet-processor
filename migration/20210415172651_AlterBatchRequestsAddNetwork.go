package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210415172651, Down20210415172651)
}

func Up20210415172651(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE batch_requests ADD network varchar(150);")
	if err != nil {
		return err
	}
	_, err4 := tx.Exec("UPDATE batch_requests set network='BTC' where asset_symbol IN ('BTC');")
	if err4 != nil {
		return err4
	}
	_, err5 := tx.Exec("UPDATE batch_requests set network='BCH' where asset_symbol IN ('BCH');")
	if err5 != nil {
		return err5
	}
	return nil
}

func Down20210415172651(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE batch_requests DROP COLUMN network;")
	if err != nil {
		return err
	}
	return nil
}
