package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210507174719, Down20210507174719)
}

func Up20210507174719(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err8 := tx.Exec("ALTER table shared_addresses DROP INDEX uix_shared_addresses_asset_symbol;")
	if err8 != nil {
		return err8
	}
	_, err9 := tx.Exec("ALTER table shared_addresses ADD CONSTRAINT uix_shared_addresses_asset_symbol_network UNIQUE (asset_symbol,network);")
	if err9 != nil {
		return err9
	}
	return nil
}

func Down20210507174719(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err8 := tx.Exec("ALTER table shared_addresses DROP INDEX uix_shared_addresses_asset_symbol_network;")
	if err8 != nil {
		return err8
	}
	_, err9 := tx.Exec("ALTER table shared_addresses ADD CONSTRAINT uix_shared_addresses_asset_symbol UNIQUE (asset_symbol);")
	if err9 != nil {
		return err9
	}
	return nil
}
