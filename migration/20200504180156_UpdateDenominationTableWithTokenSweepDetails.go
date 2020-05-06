package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200504180156, Down20200504180156)
}

func Up20200504180156(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, errStmt1 := tx.Exec("ALTER TABLE denominations ADD COLUMN `is_token` tinyint(1) DEFAULT 0 AFTER `is_enabled`;")
	if errStmt1 != nil {
		return errStmt1
	}
	_, errStmt2 := tx.Exec("ALTER TABLE denominations ADD COLUMN `main_coin_asset_symbol` VARCHAR(255) NULL AFTER `is_token`;")
	if errStmt2 != nil {
		return errStmt2
	}
	_, errStmt3 := tx.Exec("ALTER TABLE denominations ADD COLUMN `sweep_fee` bigint NULL AFTER `main_coin_asset_symbol`;")
	if errStmt3 != nil {
		return errStmt3
	}
	return nil
}

func Down20200504180156(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
