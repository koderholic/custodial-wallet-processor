package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210428174014, Down20210428174014)
}

func Up20210428174014(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE shared_addresses ADD network varchar(150);")
	if err != nil {
		return err
	}
	_, err2 := tx.Exec("UPDATE shared_addresses set network='BEP2' where asset_symbol IN ('BNB','BUSD','WRX');")
	if err2 != nil {
		return err2
	}
	return nil
}

func Down20210428174014(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE shared_addresses DROP COLUMN network;")
	if err != nil {
		return err
	}
	return nil
}
