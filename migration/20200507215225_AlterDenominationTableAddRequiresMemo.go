package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200507215225, Down20200507215225)
}

func Up20200507215225(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE denominations ADD requires_memo tinyint(1) DEFAULT 0 NOT NULL after `coin_type`;")
	if err != nil {
		return err
	}
	return nil
}

func Down20200507215225(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE denominations DROP COLUMN requires_memo;")
	if err != nil {
		return err
	}
	return nil
}
