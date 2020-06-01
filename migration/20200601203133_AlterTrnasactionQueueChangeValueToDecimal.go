package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200601203133, Down20200601203133)
}

func Up20200601203133(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE transaction_queues Change value value decimal(64,0);")
	if err != nil {
		return err
	}
	return nil
}

func Down20200601203133(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE transaction_queues Change value value bigint;")
	if err != nil {
		return err
	}
	return nil
}
