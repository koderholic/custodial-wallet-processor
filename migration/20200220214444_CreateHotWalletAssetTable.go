package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200220214444, Down20200220214444)
}

func Up20200220214444(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS hot_wallet_assets (
		id varchar(36) NOT NULL, 
		created_at timestamp NULL, 
		updated_at timestamp NULL, 
		address varchar(255) NOT NULL, 
		asset_symbol varchar(255) NOT NULL, 
		balance bigint, 
		is_disabled tinyint(1) DEFAULT 0, 
	
		PRIMARY KEY (id), 
		CONSTRAINT uix_hot_wallet_assets_asset_symbol UNIQUE (asset_symbol)
	);`)
    if err != nil {
        return err
    }
	return nil
}

func Down20200220214444(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("DROP TABLE IF EXISTS hot_wallet_assets;")
    if err != nil {
        return err
    }
	return nil
}
