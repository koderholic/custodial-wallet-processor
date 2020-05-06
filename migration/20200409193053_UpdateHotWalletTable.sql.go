package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200409193053, Down20200409193053)
}

func Up20200409193053(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, errStmt1 := tx.Exec("ALTER TABLE hot_wallet_assets ADD COLUMN `reserved_balance` bigint NULL AFTER `is_disabled`;")
	if errStmt1 != nil {
		return errStmt1
	}
	_, errStmt2 := tx.Exec("ALTER TABLE hot_wallet_assets ADD COLUMN `last_deposit_created_at` timestamp NULL AFTER `reserved_balance`;")
	if errStmt2 != nil {
		return errStmt2
	}
	_, errStmt3 := tx.Exec("ALTER TABLE hot_wallet_assets ADD COLUMN `last_withdrawal_created_at` timestamp NULL AFTER `last_deposit_created_at`;")
	if errStmt3 != nil {
		return errStmt3
	}
	return nil
}

func Down20200409193053(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
