package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20201103114552, Down20201103114552)
}

func Up20201103114552(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE denominations ADD column address_provider VARCHAR(100) NOT NULL DEFAULT 'Bundle' AFTER `main_coin_asset_symbol`")
	if err != nil {
		return err
	}
	return nil
}

func Down20201103114552(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE denominations DROP column address_provider")
	if err != nil {
		return err
	}
	return nil
}
