package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200512174138, Down20200512174138)
}

func Up20200512174138(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS shared_addresses (
		id varchar(36) NOT NULL,
		user_id varchar(36) NOT NULL, 
		address varchar(100) NOT NULL, 
		asset_symbol varchar(100) NOT NULL, 
		coin_type bigint, 
		created_at timestamp NULL, 
		updated_at timestamp NULL, 
		
		PRIMARY KEY (id), 
		CONSTRAINT uix_shared_addresses_asset_symbol UNIQUE (asset_symbol))`)
	if err != nil {
		return err
	}
	return nil
}

func Down20200512174138(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("DROP TABLE IF EXISTS shared_addresses;")
	if err != nil {
		return err
	}
	return nil
}
