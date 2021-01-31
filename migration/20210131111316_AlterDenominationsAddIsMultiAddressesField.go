package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20210131111316, Down20210131111316)
}

func Up20210131111316(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE denominations ADD is_multi_addresses tinyint(1) DEFAULT 0 NOT NULL after `is_batchable`;")
	if err != nil {
		return err
	}
	return nil
}

func Down20210131111316(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE denominations DROP COLUMN is_multi_addresses;")
	if err != nil {
		return err
	}
	return nil
}
