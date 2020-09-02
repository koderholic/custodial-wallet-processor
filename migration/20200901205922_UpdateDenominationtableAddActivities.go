package migration

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200901205922, Down20200901205922)
}

func Up20200901205922(tx *sql.Tx) error {
	_, errStmt1 := tx.Exec("ALTER TABLE denominations ADD COLUMN `trade_activity` VARCHAR(50) AFTER `main_coin_asset_symbol`;")
	if errStmt1 != nil {
		return errStmt1
	}
	_, errStmt2 := tx.Exec("ALTER TABLE denominations ADD COLUMN `deposit_activity` VARCHAR(50) NULL AFTER `trade_activity`;")
	if errStmt2 != nil {
		return errStmt2
	}
	_, errStmt3 := tx.Exec("ALTER TABLE denominations ADD COLUMN `withdraw_activity` VARCHAR(50) NULL AFTER `deposit_activity`;")
	if errStmt3 != nil {
		return errStmt3
	}
	_, errStmt4 := tx.Exec("ALTER TABLE denominations ADD COLUMN `transfer_activity` VARCHAR(50) NULL AFTER `withdraw_activity`;")
	if errStmt4 != nil {
		return errStmt4
	}
	return nil
}

func Down20200901205922(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
