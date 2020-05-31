package migration

import (
	"database/sql"
	"github.com/pressly/goose"
)

func init() {
	goose.AddMigration(Up20200520194845, Down20200520194845)
}

func Up20200520194845(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	_, err := tx.Exec("ALTER TABLE transaction_queues ADD batch_id varchar(36) AFTER `memo`;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE transaction_queues ADD INDEX (batch_id);")
	if err != nil {
		return err
	}
	return nil
}

func Down20200520194845(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	_, err := tx.Exec("ALTER TABLE transaction_queues DROP COLUMN batch_id;")
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE transaction_queues DROP INDEX (batch_id);")
	if err != nil {
		return err
	}
	return nil
}
