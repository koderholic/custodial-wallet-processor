package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210223210843, Down20210223210843)
}

func Up20210223210843(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE hot_wallet_assets MODIFY is_disabled tinyint(1) DEFAULT 0;")
	if err != nil {
		return err
	}
	_, err2 := tx.Exec("UPDATE hot_wallet_assets SET is_disabled=false;")
	if err2 != nil {
		return err2
	}
	return nil
}

func Down20210223210843(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
