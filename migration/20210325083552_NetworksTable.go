package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210325083552, Down20210325083552)
}

func Up20210325083552(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS networks (
		id varchar(36) NOT NULL, 
		created_at timestamp NULL, 
		updated_at timestamp NULL, 
		asset_symbol varchar(100) NOT NULL,
		native_decimals int,
		coin_type bigint, 
		is_token tinyint(1) DEFAULT 0,
	native_asset varchar(100) NOT NULL, 
		network varchar(255) NOT NULL, 
		chain_denom_id varchar(255) NOT NULL, 
		address_provider varchar(150) NOT NULL, 
		withdraw_activity varchar(255) NOT NULL, 
		deposit_activity varchar(255) NOT NULL, 
		requires_memo tinyint(1) DEFAULT 0,
		is_batchable tinyint(1) DEFAULT 0 NOT NULL,
		is_multi_addresses tinyint(1) DEFAULT 0 NOT NULL,
		minimum_sweepable decimal(64,18) NOT NULL,
		sweep_fee bigint NULL,
		is_enabled tinyint(1) DEFAULT 1,
		
	
		PRIMARY KEY (id),
		CONSTRAINT uix_networks_asset_symbol_network UNIQUE (asset_symbol, network)
		);
		`)
	if err != nil {
		return err
	}
	return nil

}

func Down20210325083552(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("DROP TABLE IF EXISTS networks;")
	if err != nil {
		return err
	}
	return nil
}
