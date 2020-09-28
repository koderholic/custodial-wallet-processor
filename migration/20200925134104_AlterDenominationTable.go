package migration

import (
	"database/sql"

	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200925134104, Down20200925134104)
}

func Up20200925134104(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE denominations ADD is_batchable tinyint(1) DEFAULT 0 NOT NULL AFTER `requires_memo`;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE denominations ADD minimum_sweepable decimal(64,18) NOT NULL AFTER `is_batchable`;")
	if err != nil {
		return err
	}
	return nil
}

func Down20200925134104(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE denominations DROP COLUMN is_batchable;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE denominations DROP COLUMN minimum_sweepable;")
	if err != nil {
		return err
	}
	return nil
}
